package infoblox

import (
	"encoding/json"
	"fmt"

	"github.com/DNSControl/dnscontrol/v4/models"
	"github.com/DNSControl/dnscontrol/v4/pkg/providers"
)

var features = providers.DocumentationNotes{
	// The default for unlisted capabilities is 'Cannot'.
	// See providers/capabilities.go for the entire list of capabilities.
	providers.CanConcur:              providers.Cannot(),
	providers.CanGetZones:            providers.Cannot(), // MVP: no ListZones
	providers.CanUseCAA:              providers.Can(),
	providers.CanUsePTR:              providers.Can(),
	providers.CanUseSRV:              providers.Can(),
	providers.DocCreateDomains:       providers.Cannot("Infoblox zones must be pre-created"),
	providers.DocDualHost:            providers.Cannot(),
	providers.DocOfficiallySupported: providers.Cannot(),
}

func init() {
	const providerName = "INFOBLOX"
	const providerMaintainer = "@mgamble"
	fns := providers.DspFuncs{
		Initializer:   newInfobloxDsp,
		RecordAuditor: AuditRecords,
	}
	providers.RegisterDomainServiceProviderType(providerName, fns, features)
	providers.RegisterMaintainer(providerName, providerMaintainer)
}

type infobloxProvider struct {
	api *infobloxAPI
}

func newInfobloxDsp(conf map[string]string, metadata json.RawMessage) (providers.DNSServiceProvider, error) {
	return newInfoblox(conf)
}

func newInfoblox(conf map[string]string) (*infobloxProvider, error) {
	host := conf["host"]
	if host == "" {
		return nil, fmt.Errorf("infoblox: host is required in creds.json")
	}

	username := conf["username"]
	if username == "" {
		return nil, fmt.Errorf("infoblox: username is required in creds.json")
	}

	password := conf["password"]
	if password == "" {
		return nil, fmt.Errorf("infoblox: password is required in creds.json")
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
		return nil, err
	}

	return &infobloxProvider{
		api: api,
	}, nil
}

// GetNameservers returns an empty list. Infoblox manages NS records internally;
// DNSControl does not need to manage parent delegation for this provider.
func (p *infobloxProvider) GetNameservers(domain string) ([]*models.Nameserver, error) {
	return nil, nil
}
