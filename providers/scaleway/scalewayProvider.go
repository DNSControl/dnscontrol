package scaleway

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/DNSControl/dnscontrol/v5/models"
	"github.com/DNSControl/dnscontrol/v5/pkg/providers"
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

func init() {
	providers.Register[*scalewayProvider]("SCALEWAY", providers.Definition{
		FriendlyName: "Scaleway",
		Maintainer:   "@alessiopcc",
		SupportedTypes: []string{
			"Default",
			"ALIAS",
			"DNAME",
			"HTTPS",
			"IMPORT_TRANSFORM",
			"NAPTR",
			"SSHFP",
			"SVCB",
			"TLSA",
		},
		CanAutoDNSSEC:          providers.Cannot(),
		CanConcur:              providers.Unimplemented(),
		CanUseDSForChildren:    providers.Cannot(),
		DocDualHost:            providers.Cannot(),
		DocOfficiallySupported: providers.Cannot(),
		// Features retains annotations for record types and interface-derived facts.
		Features: providers.DocumentationNotes{
			providers.DocCreateDomains: providers.Cannot("Zones must already exist in the Scaleway console."),
		},
	})
}

// Initialize initializes a fresh provider instance.
func (s *scalewayProvider) Initialize(settings map[string]string, _ json.RawMessage, _ *providers.CreateOptions) error {
	accessKey := settings["access_key"]
	secretKey := settings["secret_key"]
	if accessKey == "" || secretKey == "" {
		return errors.New("SCALEWAY: access_key and secret_key are required in creds.json")
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
		return fmt.Errorf("SCALEWAY: could not create client: %w", err)
	}

	*s = scalewayProvider{client: domain.NewAPI(scwClient)}
	return nil
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
