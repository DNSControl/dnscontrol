package dynadot

import (
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/DNSControl/dnscontrol/v4/models"
	"github.com/DNSControl/dnscontrol/v4/pkg/providers"
)

/*

Dynadot Registrator:

Info required in `creds.json`:
   - key API Key

*/

var features = providers.DocumentationNotes{
	// The default for unlisted capabilities is 'Cannot'.
	// See providers/capabilities.go for the entire list of capabilities.
	providers.CanConcur: providers.Unimplemented(),
}

func init() {
	const providerName = "DYNADOT"
	const providerMaintainer = "@e-im"
	providers.RegisterRegistrarType(providerName, newDynadot, features)
	providers.RegisterMaintainer(providerName, providerMaintainer)
}

func newDynadot(m map[string]string) (providers.Registrar, error) {
	d := &dynadotProvider{}

	d.key = m["key"]
	if d.key == "" {
		return nil, errors.New("missing Dynadot key")
	}

	return d, nil
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
