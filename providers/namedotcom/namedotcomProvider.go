// Package namedotcom implements a registrar that uses the name.com api to set name servers. It will self register it's providers when imported.
package namedotcom

import (
	"encoding/json"
	"errors"
	"time"

	"github.com/DNSControl/dnscontrol/v4/pkg/providers"
	"github.com/namedotcom/go/namecom"
)

const defaultAPIBase = "api.name.com"

// namedotcomProvider describes a connection to the NDC API.
type namedotcomProvider struct {
	APIUrl  string `json:"apiurl"`
	APIUser string `json:"apiuser"`
	APIKey  string `json:"apikey"`
	client  *namecom.NameCom
}

var features = providers.DocumentationNotes{
	// The default for unlisted capabilities is 'Cannot'.
	// See providers/capabilities.go for the entire list of capabilities.
	providers.CanConcur:              providers.Unimplemented(),
	providers.CanGetZones:            providers.Can(),
	providers.CanOnlyDiff1Features:   providers.Can(),
	providers.CanUseAlias:            providers.Can(),
	providers.CanUseLOC:              providers.Cannot(),
	providers.CanUsePTR:              providers.Cannot("PTR records are not supported (See Link)", "https://www.name.com/support/articles/205188508-Reverse-DNS-records"),
	providers.CanUseSRV:              providers.Can("SRV records with empty targets are not supported"),
	providers.DocCreateDomains:       providers.Cannot("New domains require registration"),
	providers.DocDualHost:            providers.Can(),
	providers.DocOfficiallySupported: providers.Cannot(),
}

func newReg(conf map[string]string) (providers.Registrar, error) {
	return newProvider(conf)
}

func newDsp(conf map[string]string, meta json.RawMessage) (providers.DNSServiceProvider, error) {
	return newProvider(conf)
}

func newProvider(conf map[string]string) (*namedotcomProvider, error) {
	api := &namedotcomProvider{
		client: namecom.New(conf["apiuser"], conf["apikey"]),
	}
	api.client.Server = conf["apiurl"]
	api.APIUser, api.APIKey, api.APIUrl = conf["apiuser"], conf["apikey"], conf["apiurl"]
	if api.APIKey == "" || api.APIUser == "" {
		return nil, errors.New("missing Name.com apikey or apiuser")
	}
	if api.APIUrl == "" {
		api.APIUrl = defaultAPIBase
	}

	// Set the timeout to a high value.  Currently we get timeouts and
	// the namecom library doesn't make it easy to do a clean
	// retry-on-timeout or retry-on-429.  As a work-around we just give
	// it more time to finish.
	api.client.Client.Timeout = 60 * time.Second

	return api, nil
}

func init() {
	const providerName = "NAMEDOTCOM"
	const providerMaintainer = "NEEDS VOLUNTEER"
	providers.RegisterRegistrarType(providerName, newReg)
	fns := providers.DspFuncs{
		Initializer:   newDsp,
		RecordAuditor: AuditRecords,
	}
	providers.RegisterDomainServiceProviderType("NAMEDOTCOM", fns, features)
	providers.RegisterMaintainer(providerName, providerMaintainer)
}
