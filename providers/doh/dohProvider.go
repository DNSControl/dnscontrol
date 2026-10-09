package doh

import (
	"encoding/json"
	"fmt"
	"slices"
	"strings"

	"github.com/DNSControl/dnscontrol/v5/models"
	"github.com/DNSControl/dnscontrol/v5/pkg/providers"
)

/*

DNS over HTTPS 'Registrar':

Info required in `creds.json`:
   - host                DNS over HTTPS host (eg 9.9.9.9)
*/

func init() {
	providers.Register[*dohProvider]("DNSOVERHTTPS", providers.Definition{
		FriendlyName:   "DNS over HTTPS",
		Maintainer:     "@mikenz",
		SupportedTypes: []string{},
		CanConcur:      providers.Can(),
	})
}

// Initialize initializes a fresh provider instance.
func (c *dohProvider) Initialize(m map[string]string, _ json.RawMessage, _ *providers.CreateOptions) error {
	*c = dohProvider{
		host: m["host"],
	}
	if c.host == "" {
		c.host = "dns.google"
	}
	return nil
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
