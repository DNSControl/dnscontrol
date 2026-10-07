package bunnydns

import (
	"encoding/json"
	"errors"

	"github.com/DNSControl/dnscontrol/v5/models"
	"github.com/DNSControl/dnscontrol/v5/pkg/providers"
)

type bunnydnsProvider struct {
	apiKey   string
	zones    map[string]*zone
	observer providers.ConversionObserver
}

func (b *bunnydnsProvider) SetConversionObserver(observer providers.ConversionObserver) {
	b.observer = observer
}

func init() {
	const providerName = "BUNNY_DNS"
	providers.RegisterCustomRecordType("BUNNY_DNS_RDR", providerName, "")
	providers.RegisterCustomRecordType("BUNNY_DNS_PZ", providerName, "")
	providers.Register[*bunnydnsProvider](providerName, providers.Definition{
		FriendlyName: "Bunny DNS",
		DocsURL:      "https://docs.dnscontrol.org/provider/bunnydns",
		PortalURL:    "https://dash.bunny.net/account/api-key",
		CredFields: []providers.CredsField{
			{
				Key:      "api_key",
				Label:    "API key",
				Help:     "Bunny.net account API key.",
				Secret:   true,
				Required: true,
			},
		},
		Maintainer: "@ppmathis",
		Features: providers.DocumentationNotes{
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
		},
	})
}

// Initialize initializes a fresh provider instance.
func (b *bunnydnsProvider) Initialize(settings map[string]string, _ json.RawMessage, options *providers.CreateOptions) error {
	apiKey := settings["api_key"]
	if apiKey == "" {
		return errors.New("missing BUNNY_DNS api_key")
	}

	*b = bunnydnsProvider{
		apiKey: apiKey,
	}
	b.SetConversionObserver(options.WithDefaults().ConversionObserver)
	return nil
}

// GetNameservers returns the nameservers Bunny DNS serves the zone from, so that a registrar can be pointed at them.
// DNSControl also turns them into apex NS records, which Bunny DNS does not permit; removeApexNS drops those again.
func (b *bunnydnsProvider) GetNameservers(domain string) ([]*models.Nameserver, error) {
	zone, err := b.findZoneByDomain(domain)
	if err != nil {
		return nil, err
	}

	return models.ToNameservers(zone.Nameservers())
}
