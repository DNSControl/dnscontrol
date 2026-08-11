package easyname

import (
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/DNSControl/dnscontrol/v4/models"
	"github.com/DNSControl/dnscontrol/v4/pkg/providers"
)

type easynameProvider struct {
	apikey   string
	apiauth  string
	signSalt string
	domains  map[string]easynameDomain
}

var features = providers.DocumentationNotes{
	// The default for unlisted capabilities is 'Cannot'.
	// See providers/capabilities.go for the entire list of capabilities.
	providers.CanConcur: providers.Unimplemented(),
}

func init() {
	const providerName = "EASYNAME"
	const providerMaintainer = "@tresni"
	providers.RegisterRegistrarType(providerName, newEasyname, features)
	providers.RegisterMaintainer(providerName, providerMaintainer)
}

func newEasyname(m map[string]string) (providers.Registrar, error) {
	api := &easynameProvider{}

	if m["email"] == "" || m["userid"] == "" || m["apikey"] == "" || m["authsalt"] == "" || m["signsalt"] == "" {
		return nil, errors.New("missing easyname email, userid, apikey, authsalt and/or signsalt")
	}

	api.apikey, api.signSalt = m["apikey"], m["signsalt"]
	composed := fmt.Sprintf(m["authsalt"], m["userid"], m["email"])
	api.apiauth = hashEncodeString(composed)

	return api, nil
}

// GetRegistrarCorrections gathers corrections that would bring n to match dc.
func (c *easynameProvider) GetRegistrarCorrections(dc *models.DomainConfig) ([]*models.Correction, error) {
	domain, err := c.getDomain(dc.Name)
	if err != nil {
		return nil, err
	}

	nss := []string{}
	for _, ns := range []string{domain.NameServer1, domain.NameServer2, domain.NameServer3, domain.NameServer4, domain.NameServer5, domain.NameServer6} {
		if ns != "" {
			nss = append(nss, ns)
		}
	}
	sort.Strings(nss)
	foundNameservers := strings.Join(nss, ",")

	expected := []string{}
	for _, ns := range dc.Nameservers {
		expected = append(expected, ns.Name)
	}
	sort.Strings(expected)
	expectedNameservers := strings.Join(expected, ",")

	if foundNameservers != expectedNameservers {
		return []*models.Correction{
			{
				Msg: fmt.Sprintf("Update nameservers %s -> %s", foundNameservers, expectedNameservers),
				F: func() error {
					return c.updateNameservers(expected, domain.ID)
				},
			},
		}, nil
	}
	return nil, nil
}
