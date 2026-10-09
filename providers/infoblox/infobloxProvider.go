package infoblox

import (
	"encoding/json"
	"fmt"

	"github.com/DNSControl/dnscontrol/v5/models"
	"github.com/DNSControl/dnscontrol/v5/pkg/providers"
)

func init() {
	providers.Register[*infobloxProvider]("INFOBLOX", providers.Definition{
		FriendlyName: "Infoblox",
		Maintainer:   "@matthewmgamble",
		SupportedTypes: []string{
			"Basic8",
			"PTR",
			"NS:Cannot",
		},
		CanConcur:              providers.Can(),
		DocDualHost:            providers.Cannot(),
		DocOfficiallySupported: providers.Cannot(),
		// Features retains annotations for record types and interface-derived facts.
		Features: providers.DocumentationNotes{
			providers.DocCreateDomains: providers.Cannot("zones must be pre-created"),
		},
	})
}

type infobloxProvider struct {
	api *infobloxAPI
}

// Initialize initializes a fresh provider instance.
func (p *infobloxProvider) Initialize(conf map[string]string, _ json.RawMessage, _ *providers.CreateOptions) error {
	host := conf["host"]
	if host == "" {
		return fmt.Errorf("infoblox: host is required in creds.json")
	}

	username := conf["username"]
	if username == "" {
		return fmt.Errorf("infoblox: username is required in creds.json")
	}

	password := conf["password"]
	if password == "" {
		return fmt.Errorf("infoblox: password is required in creds.json")
	}

	wapiVersion := conf["wapi_version"]
	if wapiVersion == "" {
		wapiVersion = "2.12"
	}

	view := conf["view"]
	if view == "" {
		view = "default"
	}

	tlsSkipVerify := conf["tls_skip_verify"] == "true" || conf["tls_skip_verify"] == "1"
	caCert := conf["ca_cert"]

	api, err := newInfobloxAPI(host, username, password, wapiVersion, view, tlsSkipVerify, caCert)
	if err != nil {
		return err
	}

	*p = infobloxProvider{
		api: api,
	}
	return nil
}

// GetNameservers returns an empty list. Infoblox manages NS records internally;
// DNSControl does not need to manage parent delegation for this provider.
func (p *infobloxProvider) GetNameservers(domain string) ([]*models.Nameserver, error) {
	return nil, nil
}
