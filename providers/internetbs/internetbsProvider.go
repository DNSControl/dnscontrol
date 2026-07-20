package internetbs

import (
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/DNSControl/dnscontrol/v4/models"
	"github.com/DNSControl/dnscontrol/v4/pkg/providers"
)

/*

Internet.bs Registrator:

Info required in `creds.json`:
   - api-key  ApiKey
   - password  Your account password

*/

var features = providers.DocumentationNotes{
	// The default for unlisted capabilities is 'Cannot'.
	// See providers/capabilities.go for the entire list of capabilities.
	providers.CanConcur: providers.Unimplemented(),
}

func init() {
	const providerName = "INTERNETBS"
	const providerMaintainer = "@pragmaton"
	providers.RegisterRegistrarType(providerName, newInternetBs, features)
	providers.RegisterMaintainer(providerName, providerMaintainer)
}

func newInternetBs(m map[string]string) (providers.Registrar, error) {
	api := &internetbsProvider{}

	api.key, api.password = m["api-key"], m["password"]
	if api.key == "" || api.password == "" {
		return nil, errors.New("missing Internet.bs api-key and password")
	}

	return api, nil
}

// GetRegistrarCorrections gathers corrections that would bring n to match dc.
func (c *internetbsProvider) GetRegistrarCorrections(dc *models.DomainConfig) ([]*models.Correction, error) {
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
