package doh

import (
	"fmt"
	"slices"
	"strings"

	"github.com/DNSControl/dnscontrol/v4/models"
	"github.com/DNSControl/dnscontrol/v4/pkg/providers"
)

/*

DNS over HTTPS 'Registrar':

Info required in `creds.json`:
   - host                DNS over HTTPS host (eg 9.9.9.9)
*/

var features = providers.DocumentationNotes{
	// The default for unlisted capabilities is 'Cannot'.
	// See providers/capabilities.go for the entire list of capabilities.
	providers.CanConcur: providers.Unimplemented(),
}

func init() {
	const providerName = "DNSOVERHTTPS"
	const providerMaintainer = "@mikenz"
	providers.RegisterRegistrarType(providerName, newDNSOverHTTPS, features)
	providers.RegisterMaintainer(providerName, providerMaintainer)
}

func newDNSOverHTTPS(m map[string]string) (providers.Registrar, error) {
	api := &dohProvider{
		host: m["host"],
	}
	if api.host == "" {
		api.host = "dns.google"
	}
	return api, nil
}

// GetRegistrarCorrections gathers corrections that would bring n to match dc.
func (c *dohProvider) GetRegistrarCorrections(dc *models.DomainConfig) ([]*models.Correction, error) {
	nss, err := c.getNameservers(dc.Name)
	if err != nil {
		return nil, err
	}
	foundNameservers := strings.Join(nss, ",")

	expected := make([]string, 0, len(dc.Nameservers))
	for _, ns := range dc.Nameservers {
		expected = append(expected, strings.ToLower(strings.TrimRight(ns.Name, ".")))
	}
	slices.Sort(expected)
	expected = slices.Compact(expected)
	expectedNameservers := strings.Join(expected, ",")

	if foundNameservers == expectedNameservers {
		return nil, nil
	}

	return []*models.Correction{
		{
			Msg: fmt.Sprintf("Update nameservers %s -> %s", foundNameservers, expectedNameservers),
			F: func() error {
				return c.updateNameservers(dc.Name)
			},
		},
	}, nil
}
