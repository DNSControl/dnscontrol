package scaleway

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/DNSControl/dnscontrol/v4/models"
	"github.com/DNSControl/dnscontrol/v4/pkg/providers"
	domain "github.com/scaleway/scaleway-sdk-go/api/domain/v2beta1"
	"github.com/scaleway/scaleway-sdk-go/scw"
)

/*
Scaleway Domains & DNS provider.

Required credentials in creds.json:
   - access_key   Scaleway access key (e.g. SCWXXXXXXXXXXXXXXXXX)
   - secret_key   Scaleway secret key (UUID)

Optional:
   - project_id   Default project ID to scope the client to (UUID)
*/

type scalewayProvider struct {
	client *domain.API
}

var features = providers.DocumentationNotes{
	// The default for unlisted capabilities is 'Cannot'.
	// See providers/capabilities.go for the entire list of capabilities.
	providers.CanAutoDNSSEC:          providers.Cannot(),
	providers.CanConcur:              providers.Unimplemented(),
	providers.CanGetZones:            providers.Can(),
	providers.CanUseAlias:            providers.Can(),
	providers.CanUseCAA:              providers.Can(),
	providers.CanUseDHCID:            providers.Cannot(),
	providers.CanUseDNAME:            providers.Can(),
	providers.CanUseDS:               providers.Cannot(),
	providers.CanUseDSForChildren:    providers.Cannot(),
	providers.CanUseHTTPS:            providers.Can(),
	providers.CanUseLOC:              providers.Cannot(),
	providers.CanUseNAPTR:            providers.Can(),
	providers.CanUsePTR:              providers.Can(),
	providers.CanUseSOA:              providers.Cannot(),
	providers.CanUseSRV:              providers.Can(),
	providers.CanUseSSHFP:            providers.Can(),
	providers.CanUseSVCB:             providers.Can(),
	providers.CanUseTLSA:             providers.Can(),
	providers.DocCreateDomains:       providers.Cannot("Zones must already exist in the Scaleway console."),
	providers.DocDualHost:            providers.Cannot(),
	providers.DocOfficiallySupported: providers.Cannot(),
}

func init() {
	const providerName = "SCALEWAY"
	const providerMaintainer = "@alessiopcc"
	fns := providers.DspFuncs{
		Initializer:   newScaleway,
		RecordAuditor: AuditRecords,
	}
	providers.RegisterDomainServiceProviderType(providerName, fns, features)
	providers.RegisterMaintainer(providerName, providerMaintainer)
}

func newScaleway(settings map[string]string, _ json.RawMessage) (providers.DNSServiceProvider, error) {
	accessKey := settings["access_key"]
	secretKey := settings["secret_key"]
	if accessKey == "" || secretKey == "" {
		return nil, errors.New("SCALEWAY: access_key and secret_key are required in creds.json")
	}

	opts := []scw.ClientOption{
		scw.WithAuth(accessKey, secretKey),
		scw.WithUserAgent("dnscontrol"),
	}
	if projectID := settings["project_id"]; projectID != "" {
		opts = append(opts, scw.WithDefaultProjectID(projectID))
	}

	scwClient, err := scw.NewClient(opts...)
	if err != nil {
		return nil, fmt.Errorf("SCALEWAY: could not create client: %w", err)
	}

	return &scalewayProvider{client: domain.NewAPI(scwClient)}, nil
}

// GetNameservers returns the nameservers for a domain.
func (s *scalewayProvider) GetNameservers(zone string) ([]*models.Nameserver, error) {
	resp, err := s.client.ListDNSZoneNameservers(&domain.ListDNSZoneNameserversRequest{
		DNSZone: zone,
	})
	if err != nil {
		return nil, fmt.Errorf("SCALEWAY: could not fetch nameservers for %q: %w", zone, err)
	}

	names := make([]string, 0, len(resp.Ns))
	for _, ns := range resp.Ns {
		names = append(names, ns.Name)
	}
	return models.ToNameservers(names)
}

// ListZones returns the zones (domains) managed by this account.
func (s *scalewayProvider) ListZones() ([]string, error) {
	const pageSize = uint32(100)
	page := int32(1)
	var zones []string
	for {
		p := page
		ps := pageSize
		resp, err := s.client.ListDNSZones(&domain.ListDNSZonesRequest{
			Page:     &p,
			PageSize: &ps,
		})
		if err != nil {
			return nil, fmt.Errorf("SCALEWAY: could not list zones: %w", err)
		}
		for _, z := range resp.DNSZones {
			// A Scaleway "DNS Zone" entry consists of a Subdomain and a Domain.
			// The fully qualified zone is `<Subdomain>.<Domain>`, except when
			// Subdomain is empty (apex of the registered domain).
			name := z.Domain
			if z.Subdomain != "" {
				name = z.Subdomain + "." + z.Domain
			}
			zones = append(zones, name)
		}
		if uint32(len(zones)) >= resp.TotalCount {
			break
		}
		page++
	}
	return zones, nil
}
