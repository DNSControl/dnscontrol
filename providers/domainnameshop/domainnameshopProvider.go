package domainnameshop

import (
	"encoding/json"
	"errors"

	"github.com/DNSControl/dnscontrol/v5/pkg/providers"
)

/**

Domainnameshop Provider

Info required in 'creds.json':
	- token		API Token
	- secret	API Secret

*/

type domainNameShopProvider struct {
	Token  string // The API token
	Secret string // The API secret
}

func init() {
	providers.Register[*domainNameShopProvider]("DOMAINNAMESHOP", providers.Definition{
		FriendlyName: "Domainnameshop",
		DocsURL:      "https://docs.dnscontrol.org/provider/domainnameshop",
		Maintainer:   "@SimenBai",
		Features: providers.DocumentationNotes{
			// The default for unlisted capabilities is 'Cannot'.
			// See providers/capabilities.go for the entire list of capabilities.
			providers.CanAutoDNSSEC:          providers.Cannot(), // Maybe there is support for it
			providers.CanConcur:              providers.Unimplemented(),
			providers.CanGetZones:            providers.Unimplemented(), //
			providers.CanOnlyDiff1Features:   providers.Can(),
			providers.CanUseAlias:            providers.Unimplemented("Needs custom implementation"), // Can possibly be implemented, needs further research
			providers.CanUseCAA:              providers.Can(),
			providers.CanUseDS:               providers.Unimplemented(), // Seems to support but needs to be implemented
			providers.CanUseDSForChildren:    providers.Unimplemented(), // Seems to support but needs to be implemented
			providers.CanUseLOC:              providers.Cannot(),
			providers.CanUseNAPTR:            providers.Cannot("According to Domainnameshop this will probably never be supported"), // Does not seem to support it
			providers.CanUsePTR:              providers.Cannot("According to Domainnameshop this will probably never be supported"), // Seems to support but needs to be implemented
			providers.CanUseSOA:              providers.Cannot(),                                                                    // Does not seem to support it
			providers.CanUseSRV:              providers.Can(),
			providers.CanUseSSHFP:            providers.Cannot("Might be supported in the future"),                                   // Does not seem to support it
			providers.CanUseTLSA:             providers.Unimplemented("Has support but no documentation. Needs to be investigated."), // Seems to support but needs to be implemented
			providers.DocCreateDomains:       providers.Unimplemented(),                                                              // Not tested
			providers.DocDualHost:            providers.Unimplemented(),                                                              // Not tested
			providers.DocOfficiallySupported: providers.Cannot(),
		},
	})
}

// Initialize initializes a fresh provider instance.
func (api *domainNameShopProvider) Initialize(conf map[string]string, _ json.RawMessage, _ *providers.CreateOptions) error {
	if conf["token"] == "" {
		return errors.New("no Domainnameshop token provided")
	} else if conf["secret"] == "" {
		return errors.New("no Domainnameshop secret provided")
	}

	*api = domainNameShopProvider{
		Token:  conf["token"],
		Secret: conf["secret"],
	}

	// Consider testing if creds work
	return nil
}

type domainResponse struct {
	ID          int      `json:"id"`
	Domain      string   `json:"domain"`
	Nameservers []string `json:"nameservers"`
}

// The Actual fields are the values in the right format according to what is needed for RecordConfig.
//
//	While the values without Actual are the values directly as received from the Domainnameshop API.
//	This is done to make it easier to use the values at later points.
type domainNameShopRecord struct {
	ID             int    `json:"id"`
	Host           string `json:"host"`
	TTL            uint16 `json:"ttl,omitempty"`
	Type           string `json:"type"`
	Data           string `json:"data"`
	Priority       string `json:"priority,omitempty"`
	ActualPriority uint16
	Weight         string `json:"weight,omitempty"`
	ActualWeight   uint16
	Port           string `json:"port,omitempty"`
	ActualPort     uint16
	CAATag         string `json:"tag,omitempty"`
	ActualCAAFlag  string `json:"flags,omitempty"`
	CAAFlag        uint64
	DomainID       string
}
