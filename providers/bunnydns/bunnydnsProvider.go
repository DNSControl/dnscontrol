package bunnydns

import (
	"encoding/json"
	"errors"

	"github.com/DNSControl/dnscontrol/v4/models"
	"github.com/DNSControl/dnscontrol/v4/pkg/providers"
)

var features = providers.DocumentationNotes{
	// The default for unlisted capabilities is 'Cannot'.
	// See providers/capabilities.go for the entire list of capabilities.
	providers.CanAutoDNSSEC:          providers.Can(),
	providers.CanGetZones:            providers.Can(),
	providers.CanConcur:              providers.Unimplemented(),
	providers.CanUseAlias:            providers.Can("Bunny flattens CNAME records into A/AAAA records dynamically"),
	providers.CanUseCAA:              providers.Can(),
	providers.CanUseDHCID:            providers.Cannot(),
	providers.CanUseDS:               providers.Cannot(),
	providers.CanUseDSForChildren:    providers.Cannot(),
	providers.CanUseHTTPS:            providers.Can(),
	providers.CanUseLOC:              providers.Cannot(),
	providers.CanUseNAPTR:            providers.Cannot(),
	providers.CanUsePTR:              providers.Can(),
	providers.CanUseSOA:              providers.Cannot(),
	providers.CanUseSRV:              providers.Can(),
	providers.CanUseSSHFP:            providers.Cannot(),
	providers.CanUseSVCB:             providers.Can(),
	providers.CanUseTLSA:             providers.Can(),
	providers.DocCreateDomains:       providers.Can(),
	providers.DocDualHost:            providers.Cannot(),
	providers.DocOfficiallySupported: providers.Cannot(),
}

type bunnydnsProvider struct {
	apiKey string
	zones  map[string]*zone
}

func init() {
	const providerName = "BUNNY_DNS"
	const providerMaintainer = "@ppmathis"
	fns := providers.DspFuncs{
		Initializer:   newBunnydns,
		RecordAuditor: AuditRecords,
	}
	providers.RegisterDomainServiceProviderType(providerName, fns, features)
	providers.RegisterMaintainer(providerName, providerMaintainer)

	providers.RegisterCustomRecordType("BUNNY_DNS_RDR", providerName, "")
	providers.RegisterCustomRecordType("BUNNY_DNS_PZ", providerName, "")

	providers.RegisterCredsMetadata(providerName, providers.CredsMetadata{
		DisplayName: "Bunny DNS",
		Kind:        providers.KindDNS,
		DocsURL:     "https://docs.dnscontrol.org/provider/bunnydns",
		PortalURL:   "https://dash.bunny.net/account/api-key",
		Fields: []providers.CredsField{
			{
				Key:      "api_key",
				Label:    "API key",
				Help:     "Bunny.net account API key.",
				Secret:   true,
				Required: true,
			},
		},
	})
}

func newBunnydns(settings map[string]string, _ json.RawMessage) (providers.DNSServiceProvider, error) {
	apiKey := settings["api_key"]
	if apiKey == "" {
		return nil, errors.New("missing BUNNY_DNS api_key")
	}

	return &bunnydnsProvider{
		apiKey: apiKey,
	}, nil
}

func (b *bunnydnsProvider) GetNameservers(domain string) ([]*models.Nameserver, error) {
	zone, err := b.findZoneByDomain(domain)
	if err != nil {
		return nil, err
	}

	return models.ToNameservers(zone.Nameservers())
}
