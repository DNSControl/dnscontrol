package dynadot

import (
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/DNSControl/dnscontrol/v5/models"
	"github.com/DNSControl/dnscontrol/v5/pkg/providers"
)

/*

Dynadot Registrator:

Info required in `creds.json`:
   - key API Key

*/

func init() {
	providers.Register[*dynadotProvider]("DYNADOT", providers.Definition{
		FriendlyName:   "Dynadot",
		Maintainer:     "@e-im",
		SupportedTypes: []string{},
		CanConcur:      providers.Unimplemented(),
	})
}

// Initialize initializes a fresh provider instance.
func (c *dynadotProvider) Initialize(m map[string]string, _ json.RawMessage, _ *providers.CreateOptions) error {
	c.key = m["key"]
	if c.key == "" {
		return errors.New("missing Dynadot key")
	}

	return nil
}

func (c *dynadotProvider) GetRegistrarCorrections(dc *models.DomainConfig) ([]*models.Correction, error) {
	nss, err := c.getNameservers(dc.Name)
	if err != nil {
		return nil, err
	}
	foundNameservers := strings.Join(nss, ",")

	expected := []string{}
	for _, ns := range dc.Nameservers {
		name := strings.TrimRight(ns.Name, ".")
		expected = append(expected, name)
	}
	sort.Strings(expected)
	expectedNameservers := strings.Join(expected, ",")

	if foundNameservers != expectedNameservers {
		return []*models.Correction{
			{
				Msg: fmt.Sprintf("Update nameservers (%s) -> (%s)", foundNameservers, expectedNameservers),
				F: func() error {
					return c.updateNameservers(expected, dc.Name)
				},
			},
		}, nil
	}
	return nil, nil
}
