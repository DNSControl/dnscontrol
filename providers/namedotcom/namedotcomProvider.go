// Package namedotcom implements a registrar that uses the name.com api to set name servers. It will self register it's providers when imported.
package namedotcom

import (
	"encoding/json"
	"errors"
	"time"

	"github.com/DNSControl/dnscontrol/v5/pkg/providers"
	"github.com/namedotcom/go/namecom"
)

const defaultAPIBase = "api.name.com"

// namedotcomProvider describes a connection to the NDC API.
type namedotcomProvider struct {
	observer providers.ConversionObserver
	APIUrl   string `json:"apiurl"`
	APIUser  string `json:"apiuser"`
	APIKey   string `json:"apikey"`
	client   *namecom.NameCom
}

func (n *namedotcomProvider) SetConversionObserver(observer providers.ConversionObserver) {
	n.observer = observer
}

// Initialize initializes a fresh provider instance.
func (n *namedotcomProvider) Initialize(conf map[string]string, _ json.RawMessage, options *providers.CreateOptions) error {
	*n = namedotcomProvider{
		client: namecom.New(conf["apiuser"], conf["apikey"]),
	}
	n.client.Server = conf["apiurl"]
	n.APIUser, n.APIKey, n.APIUrl = conf["apiuser"], conf["apikey"], conf["apiurl"]
	if n.APIKey == "" || n.APIUser == "" {
		return errors.New("missing Name.com apikey or apiuser")
	}
	if n.APIUrl == "" {
		n.APIUrl = defaultAPIBase
	}

	// Set the timeout to a high value.  Currently we get timeouts and
	// the namecom library doesn't make it easy to do a clean
	// retry-on-timeout or retry-on-429.  As a work-around we just give
	// it more time to finish.
	n.client.Client.Timeout = 60 * time.Second

	n.SetConversionObserver(options.WithDefaults().ConversionObserver)
	return nil
}

func init() {
	providers.Register[*namedotcomProvider]("NAMEDOTCOM", providers.Definition{
		FriendlyName: "Name.com",
		Maintainer:   "NEEDS VOLUNTEER",
		SupportedTypes: []string{
			"Basic8",
			"ALIAS",
			"CAA:Cannot",
		},
		CanConcur:              providers.Unimplemented(),
		DocDualHost:            providers.Can(),
		DocOfficiallySupported: providers.Cannot(),
		// Features retains annotations for record types and interface-derived facts.
		Features: providers.DocumentationNotes{
			providers.CanUsePTR:        providers.Cannot("PTR records are not supported (See Link)", "https://www.name.com/support/articles/205188508-Reverse-DNS-records"),
			providers.CanUseSRV:        providers.Can("SRV records with empty targets are not supported"),
			providers.DocCreateDomains: providers.Cannot("New domains require registration"),
		},
	})
}
