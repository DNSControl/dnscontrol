package easyname

import (
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/DNSControl/dnscontrol/v5/models"
	"github.com/DNSControl/dnscontrol/v5/pkg/providers"
)

type easynameProvider struct {
	apikey   string
	apiauth  string
	signSalt string
	domains  map[string]easynameDomain
}

func init() {
	providers.Register[*easynameProvider]("EASYNAME", providers.Definition{
		FriendlyName:   "easyname",
		Maintainer:     "@tresni",
		SupportedTypes: []string{},
		CanConcur:      providers.Unimplemented(),
	})
}

// Initialize initializes a fresh provider instance.
func (c *easynameProvider) Initialize(m map[string]string, _ json.RawMessage, _ *providers.CreateOptions) error {
	if m["email"] == "" || m["userid"] == "" || m["apikey"] == "" || m["authsalt"] == "" || m["signsalt"] == "" {
		return errors.New("missing easyname email, userid, apikey, authsalt and/or signsalt")
	}

	c.apikey, c.signSalt = m["apikey"], m["signsalt"]
	composed := fmt.Sprintf(m["authsalt"], m["userid"], m["email"])
	c.apiauth = hashEncodeString(composed)

	return nil
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
