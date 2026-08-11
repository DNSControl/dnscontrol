package linode

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strings"

	"github.com/DNSControl/dnscontrol/v4/models"
	"github.com/DNSControl/dnscontrol/v4/pkg/diff"
	"github.com/DNSControl/dnscontrol/v4/pkg/providers"
	dnsutilv1 "github.com/miekg/dns/dnsutil"
	"golang.org/x/oauth2"
)

/*

Linode API DNS provider:

Info required in `creds.json`:
   - token

*/

// Allowed values from the Linode API
// https://www.linode.com/docs/api/domains/#domains-list__responses
var allowedTTLValues = []uint32{
	0,       // Default, currently 1209600 seconds
	300,     // 5 minutes
	3600,    // 1 hour
	7200,    // 2 hours
	14400,   // 4 hours
	28800,   // 8 hours
	57600,   // 16 hours
	86400,   // 1 day
	172800,  // 2 days
	345600,  // 4 days
	604800,  // 1 week
	1209600, // 2 weeks
	2419200, // 4 weeks
}

var srvRegexp = regexp.MustCompile(`^_(?P<Service>\w+)\.\_(?P<Protocol>\w+)$`)

// linodeProvider is the handle for this provider.
type linodeProvider struct {
	client      *http.Client
	baseURL     *url.URL
	domainIndex map[string]int
}

var defaultNameServerNames = []string{
	"ns1.linode.com",
	"ns2.linode.com",
	"ns3.linode.com",
	"ns4.linode.com",
	"ns5.linode.com",
}

// NewLinode creates the provider.
func NewLinode(m map[string]string, metadata json.RawMessage) (providers.DNSServiceProvider, error) {
	if m["token"] == "" {
		return nil, errors.New("missing Linode token")
	}

	ctx := context.Background()
	client := oauth2.NewClient(
		ctx,
		oauth2.StaticTokenSource(&oauth2.Token{AccessToken: m["token"]}),
	)

	baseURL, err := url.Parse(defaultBaseURL)
	if err != nil {
		return nil, errors.New("invalid base URL for Linode")
	}

	api := &linodeProvider{client: client, baseURL: baseURL}

	// Get a domain to validate the token
	if err := api.fetchDomainList(); err != nil {
		return nil, err
	}

	return api, nil
}

var features = providers.DocumentationNotes{
	// The default for unlisted capabilities is 'Cannot'.
	// See providers/capabilities.go for the entire list of capabilities.
	providers.CanConcur:              providers.Unimplemented(),
	providers.CanGetZones:            providers.Can(),
	providers.CanOnlyDiff1Features:   providers.Can(),
	providers.CanUseCAA:              providers.Can("Linode doesn't support changing the CAA flag"),
	providers.CanUseSRV:              providers.Can("Linode requires non-zero priority"),
	providers.CanUseLOC:              providers.Cannot(),
	providers.DocDualHost:            providers.Cannot(),
	providers.DocOfficiallySupported: providers.Cannot(),
}

func init() {
	const providerName = "LINODE"
	const providerMaintainer = "@koesie10"
	// SRV support is in this provider, but Linode doesn't seem to support it properly
	fns := providers.DspFuncs{
		Initializer:   NewLinode,
		RecordAuditor: AuditRecords,
	}
	providers.RegisterDomainServiceProviderType(providerName, fns, features)
	providers.RegisterMaintainer(providerName, providerMaintainer)
	providers.RegisterCredsMetadata(providerName, providers.CredsMetadata{
		DisplayName: "Linode",
		Kind:        providers.KindDNS,
		DocsURL:     "https://docs.dnscontrol.org/provider/linode",
		PortalURL:   "https://cloud.linode.com/profile/tokens", // TODO: Verify
		Fields: []providers.CredsField{
			{
				Key:      "token",
				Label:    "API token",
				Help:     "Your Linode Personal Access Token.",
				Secret:   true,
				Required: true,
			},
		},
	})
}

// GetNameservers returns the nameservers for a domain.
func (api *linodeProvider) GetNameservers(domain string) ([]*models.Nameserver, error) {
	return models.ToNameservers(defaultNameServerNames)
}

// GetZoneRecords gets the records of a zone and returns them in RecordConfig format.
func (api *linodeProvider) GetZoneRecords(dc *models.DomainConfig) (models.Records, error) {
	domain := dc.Name

	if api.domainIndex == nil {
		if err := api.fetchDomainList(); err != nil {
			return nil, err
		}
	}
	domainID, ok := api.domainIndex[domain]
	if !ok {
		return nil, fmt.Errorf("'%s' not a zone in Linode account", domain)
	}

	return api.getRecordsForDomain(domainID, domain)
}

// GetZoneRecordsCorrections returns a list of corrections that will turn existing records into dc.Records.
func (api *linodeProvider) GetZoneRecordsCorrections(dc *models.DomainConfig, existingRecords models.Records) ([]*models.Correction, int, error) {
	// Linode doesn't allow selecting an arbitrary TTL, only a set of predefined values
	// We need to make sure we don't change it every time if it is as close as it's going to get
	// The documentation says that it will always round up to the next highest value: 300 -> 300, 301 -> 3600.
	// https://www.linode.com/docs/api/domains/#domains-list__responses
	for _, record := range dc.Records {
		record.TTL = fixTTL(record.TTL)
	}

	if api.domainIndex == nil {
		if err := api.fetchDomainList(); err != nil {
			return nil, 0, err
		}
	}
	domainID, ok := api.domainIndex[dc.Name]
	if !ok {
		return nil, 0, fmt.Errorf("'%s' not a zone in Linode account", dc.Name)
	}

	toReport, create, del, modify, actualChangeCount, err := diff.NewCompat(dc).IncrementalDiff(existingRecords)
	if err != nil {
		return nil, 0, err
	}
	// Start corrections with the reports
	corrections := diff.GenerateMessageCorrections(toReport)

	// Deletes first so changing type works etc.
	for _, m := range del {
		id := m.Existing.Original.(*domainRecord).ID
		if id == 0 { // Skip ID 0, these are the default nameservers always present
			continue
		}
		corr := &models.Correction{
			Msg: fmt.Sprintf("%s, Linode ID: %d", m.String(), id),
			F: func() error {
				return api.deleteRecord(domainID, id)
			},
		}
		corrections = append(corrections, corr)
	}
	for _, m := range create {
		req, err := toReq(dc, m.Desired)
		if err != nil {
			return nil, 0, err
		}
		j, err := json.Marshal(req)
		if err != nil {
			return nil, 0, err
		}
		corr := &models.Correction{
			Msg: fmt.Sprintf("%s: %s", m.String(), string(j)),
			F: func() error {
				record, err := api.createRecord(domainID, req)
				if err != nil {
					return err
				}
				// TTL isn't saved when creating a record, so we will need to modify it immediately afterwards
				return api.modifyRecord(domainID, record.ID, req)
			},
		}
		corrections = append(corrections, corr)
	}
	for _, m := range modify {
		id := m.Existing.Original.(*domainRecord).ID
		if id == 0 { // Skip ID 0, these are the default nameservers always present
			continue
		}
		req, err := toReq(dc, m.Desired)
		if err != nil {
			return nil, 0, err
		}
		j, err := json.Marshal(req)
		if err != nil {
			return nil, 0, err
		}
		corr := &models.Correction{
			Msg: fmt.Sprintf("%s, Linode ID: %d: %s", m.String(), id, string(j)),
			F: func() error {
				return api.modifyRecord(domainID, id, req)
			},
		}
		corrections = append(corrections, corr)
	}

	return corrections, actualChangeCount, nil
}

func (api *linodeProvider) getRecordsForDomain(domainID int, domain string) (models.Records, error) {
	records, err := api.getRecords(domainID)
	if err != nil {
		return nil, err
	}

	existingRecords := make([]*models.RecordConfig, len(records), len(records)+len(defaultNameServerNames))
	for i := range records {
		existingRecords[i], err = toRc(domain, &records[i])
		if err != nil {
			return nil, err
		}
	}

	// Linode always has read-only NS servers, but these are not mentioned in the API response
	// https://github.com/linode/manager/blob/edd99dc4e1be5ab8190f243c3dbf8b830716255e/src/constants.js#L184
	for _, name := range defaultNameServerNames {
		rc := &models.RecordConfig{
			Type:     "NS",
			Original: &domainRecord{},
		}
		rc.SetLabelFromFQDN(domain, domain)
		if err := rc.SetTarget(name); err != nil {
			return nil, err
		}

		existingRecords = append(existingRecords, rc)
	}

	return existingRecords, nil
}

func toRc(domain string, r *domainRecord) (*models.RecordConfig, error) {
	rc := &models.RecordConfig{
		Type:         r.Type,
		TTL:          r.TTLSec,
		MxPreference: r.Priority,
		SrvPriority:  r.Priority,
		SrvWeight:    r.Weight,
		SrvPort:      r.Port,
		CaaTag:       r.Tag,
		Original:     r,
	}
	rc.SetLabel(r.Name, domain)

	var err error
	switch rtype := r.Type; rtype { // #rtype_variations
	case "CNAME", "MX", "NS", "SRV":
		err = rc.SetTarget(dnsutilv1.AddOrigin(r.Target+".", domain))
	case "CAA":
		// Linode doesn't support CAA flags and just returns the tag and value separately
		err = rc.SetTarget(r.Target)
	default:
		err = rc.PopulateFromString(r.Type, r.Target, domain)
	}
	return rc, err
}

func toReq(dc *models.DomainConfig, rc *models.RecordConfig) (*recordEditRequest, error) {
	req := &recordEditRequest{
		Type:     rc.Type,
		Name:     rc.GetLabel(),
		Target:   rc.GetTargetField(),
		TTL:      int(rc.TTL),
		Priority: 0,
		Port:     int(rc.SrvPort),
		Weight:   int(rc.SrvWeight),
	}

	// Linode doesn't use "@", it uses an empty name
	if req.Name == "@" {
		req.Name = ""
	}

	// Linode uses the same property for MX and SRV priority
	switch rc.Type { // #rtype_variations
	case "A", "AAAA", "NS", "PTR", "TXT", "SOA", "TLSA":
		// Nothing special.
	case "MX":
		req.Priority = int(rc.MxPreference)
		req.Target = fixTarget(req.Target, dc.Name)

		// Linode doesn't use "." for a null MX record, it uses an empty name
		if req.Target == "." {
			req.Target = ""
		}
	case "SRV":
		req.Priority = int(rc.SrvPriority)

		// From softlayer provider
		// This is to support SRV, it doesn't work yet for Linode
		result := srvRegexp.FindStringSubmatch(req.Name)

		if len(result) != 3 {
			return nil, fmt.Errorf("SRV Record must match format \"_service._protocol\" not %s", req.Name)
		}

		serviceName, protocol := result[1], strings.ToLower(result[2])

		req.Protocol = protocol
		req.Service = serviceName
		req.Name = ""
	case "CNAME":
		req.Target = fixTarget(req.Target, dc.Name)
	case "CAA":
		req.Tag = rc.CaaTag
	default:
		return nil, fmt.Errorf("linode.toReq rtype %q unimplemented", rc.Type)
	}

	return req, nil
}

func fixTarget(target, domain string) string {
	// Linode always wants a fully qualified target name
	if target[len(target)-1] == '.' {
		return target[:len(target)-1]
	}
	return fmt.Sprintf("%s.%s", target, domain)
}

func fixTTL(ttl uint32) uint32 {
	// if the TTL is larger than the largest allowed value, return the largest allowed value
	if ttl > allowedTTLValues[len(allowedTTLValues)-1] {
		return allowedTTLValues[len(allowedTTLValues)-1]
	}

	for _, v := range allowedTTLValues {
		if v >= ttl {
			return v
		}
	}

	return allowedTTLValues[0]
}
