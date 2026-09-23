package mittwald

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	mittwaldv2 "github.com/mittwald/api-client-go/mittwaldv2"
	generatedv2 "github.com/mittwald/api-client-go/mittwaldv2/generated/clients"
	"github.com/mittwald/api-client-go/mittwaldv2/generated/clients/domainclientv2"
	"github.com/mittwald/api-client-go/mittwaldv2/generated/clients/projectclientv2"
	"github.com/mittwald/api-client-go/mittwaldv2/generated/schemas/dnsv2"
	"golang.org/x/net/idna"
)

// apiTimeout bounds one attempt of a request; waiting for the rate limit
// between attempts is not part of it.
const apiTimeout = 60 * time.Second

// projectPageSize is the number of projects requested per page.
const projectPageSize = 100

// api wraps the mStudio client and remembers which project holds a domain.
type api struct {
	client generatedv2.Client

	mu            sync.Mutex
	projectByZone map[string]string // root domain -> project ID, filled by findProject
}

func newAPI(token string) (*api, error) {
	client, err := mittwaldv2.New(context.Background(),
		mittwaldv2.WithHTTPClient(newAPIRunner(&http.Client{Timeout: apiTimeout})), // must come first
		mittwaldv2.WithAccessToken(token),
	)
	if err != nil {
		return nil, err
	}
	return &api{client: client}, nil
}

// listProjectZones returns every DNS zone of a project: the root zones of its
// domains and one zone per name below them. The API names a zone in Unicode;
// the names returned here are ASCII (Punycode), as dnscontrol uses them.
func (a *api) listProjectZones(projectID string) ([]dnsv2.Zone, error) {
	ctx := context.Background()
	zones, resp, err := a.client.Domain().ListDNSZones(ctx, domainclientv2.ListDNSZonesRequest{ProjectID: projectID})
	closeBody(resp)
	if err != nil {
		return nil, fmt.Errorf("listing DNS zones of project %s: %w", projectID, err)
	}
	for i := range *zones {
		z := &(*zones)[i]
		if z.Domain, err = idna.ToASCII(z.Domain); err != nil {
			return nil, fmt.Errorf("zone %s: %w", z.Id, err)
		}
	}
	return *zones, nil
}

// listProjectIDs returns the IDs of all projects the token can access.
func (a *api) listProjectIDs() ([]string, error) {
	var ids []string
	limit := int64(projectPageSize)
	for skip := int64(0); ; skip += limit {
		page, resp, err := a.client.Project().ListProjects(context.Background(), projectclientv2.ListProjectsRequest{Limit: &limit, Skip: &skip})
		closeBody(resp)
		if err != nil {
			return nil, fmt.Errorf("listing projects: %w", err)
		}
		for _, p := range *page {
			ids = append(ids, p.Id)
		}
		if int64(len(*page)) < limit {
			return ids, nil
		}
	}
}

// findProject returns the ID of the project whose DNS zones contain the
// root zone of domain.
func (a *api) findProject(domain string) (string, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if id, ok := a.projectByZone[domain]; ok {
		return id, nil
	}
	// The domain list names the project of a domain in one request.
	if id, err := a.projectFromDomainList(domain); err != nil || id != "" {
		return id, err
	}
	// A domain missing from that list is looked for in the zones of every project.
	ids, err := a.listProjectIDs()
	if err != nil {
		return "", err
	}
	a.projectByZone = map[string]string{}
	for _, id := range ids {
		zones, err := a.listProjectZones(id)
		if err != nil {
			return "", err
		}
		for _, z := range zones {
			a.projectByZone[z.Domain] = id
		}
	}
	id, ok := a.projectByZone[domain]
	if !ok {
		return "", fmt.Errorf("domain %q not found in any mStudio project of this token", domain)
	}
	return id, nil
}

// projectFromDomainList returns the ID of the project that holds domain
// according to the token's domain list, or "" when the list does not have it.
func (a *api) projectFromDomainList(domain string) (string, error) {
	search := domain
	if u, err := idna.ToUnicode(domain); err == nil {
		search = u // the API names domains in Unicode
	}
	domains, resp, err := a.client.Domain().ListDomains(context.Background(), domainclientv2.ListDomainsRequest{DomainSearchName: &search})
	closeBody(resp)
	if err != nil {
		return "", fmt.Errorf("listing domains: %w", err)
	}
	for _, d := range *domains {
		if name, err := idna.ToASCII(d.Domain); err == nil && name == domain && d.ProjectId != "" {
			if a.projectByZone == nil {
				a.projectByZone = map[string]string{}
			}
			a.projectByZone[domain] = d.ProjectId
			return d.ProjectId, nil
		}
	}
	return "", nil
}

// zonesOf returns the root zone of domain and every zone below it, keyed by
// the fully qualified name without trailing dot.
func (a *api) zonesOf(domain string) (map[string]dnsv2.Zone, error) {
	projectID, err := a.findProject(domain)
	if err != nil {
		return nil, err
	}
	all, err := a.listProjectZones(projectID)
	if err != nil {
		return nil, err
	}
	zones := map[string]dnsv2.Zone{}
	for _, z := range all {
		if z.Domain == domain || strings.HasSuffix(z.Domain, "."+domain) {
			zones[z.Domain] = z
		}
	}
	if _, ok := zones[domain]; !ok {
		return nil, fmt.Errorf("root zone of %q not found", domain)
	}
	return zones, nil
}

// createZone creates the zone for name (relative to the root zone) and returns its ID.
func (a *api) createZone(rootID, name string) (string, error) {
	ctx := context.Background()
	created, resp, err := a.client.Domain().CreateDNSZone(ctx, domainclientv2.CreateDNSZoneRequest{
		Body: domainclientv2.CreateDNSZoneRequestBody{Name: name, ParentZoneId: rootID},
	})
	closeBody(resp)
	if err != nil {
		return "", fmt.Errorf("creating zone %q: %w", name, err)
	}
	return created.Id, nil
}

func (a *api) deleteZone(zoneID string) error {
	ctx := context.Background()
	resp, err := a.client.Domain().DeleteDNSZone(ctx, domainclientv2.DeleteDNSZoneRequest{DNSZoneID: zoneID})
	closeBody(resp)
	if err != nil {
		return fmt.Errorf("deleting zone %s: %w", zoneID, err)
	}
	return nil
}

// setRecordSet replaces one record set of a zone. An empty body unsets it.
func (a *api) setRecordSet(zoneID string, set domainclientv2.UpdateRecordSetRequestPathRecordSet, body domainclientv2.UpdateRecordSetRequestBody) error {
	ctx := context.Background()
	resp, err := a.client.Domain().UpdateRecordSet(ctx, domainclientv2.UpdateRecordSetRequest{
		DNSZoneID: zoneID,
		RecordSet: set,
		Body:      body,
	})
	closeBody(resp)
	if err != nil {
		return fmt.Errorf("setting %s records of zone %s: %w", set, zoneID, err)
	}
	return nil
}

func (a *api) unsetRecordSet(zoneID string, set domainclientv2.UpdateRecordSetRequestPathRecordSet) error {
	return a.setRecordSet(zoneID, set, domainclientv2.UpdateRecordSetRequestBody{AlternativeRecordUnset: &dnsv2.RecordUnset{}})
}

// closeBody releases the connection of a response; the client leaves that to the caller.
func closeBody(resp *http.Response) {
	if resp != nil && resp.Body != nil {
		_, _ = io.Copy(io.Discard, resp.Body)
		_ = resp.Body.Close()
	}
}
