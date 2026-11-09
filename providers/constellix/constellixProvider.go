package constellix

import (
	"encoding/json"
	"errors"
	"strings"

	"github.com/DNSControl/dnscontrol/v5/models"
	"github.com/DNSControl/dnscontrol/v5/pkg/printer"
	"github.com/DNSControl/dnscontrol/v5/pkg/providers"
	"github.com/DNSControl/dnscontrol/v5/pkg/zonecache"
)

/*
Constellix DNS provider:

Info required in `creds.json`:
   - api_key
   - secret_key

The v4 API stores one record object per name + type, with a list of values,
and a PUT replaces that list. This provider therefore uses diff2.ByRecordSet()
and sends one request per changed record set, addressed by the record's ID.

Constellix serves its own nameservers at the apex without listing them as
records; configured apex NS records that repeat them are dropped (see
filterApexNS). Records that use features DNSControl does not model (failover,
pools, GTD regions, IP filters, GeoProximity, disabled records or values,
types such as HTTP redirects) make the zone read fail instead of being
overwritten. See records.go.

The rate limit is shared by the whole account; see the gate in api.go.
*/

type constellixProvider struct {
	client   *apiClient
	zones    zonecache.ZoneCache[zone]
	observer providers.ConversionObserver
}

func init() {
	providers.Register[*constellixProvider]("CONSTELLIX", providers.Definition{
		FriendlyName:    "Constellix",
		VendorAPIDocURL: "https://api.dns.constellix.com/v4/docs",
		PortalURL:       "https://dns.constellix.com/",
		CredFields: []providers.CredsField{
			{
				Key:      "api_key",
				Label:    "API key",
				Help:     "Your Constellix API key.",
				Secret:   true,
				Required: true,
			},
			{
				Key:      "secret_key",
				Label:    "Secret key",
				Help:     "Your Constellix secret key.",
				Secret:   true,
				Required: true,
			},
		},
		Maintainer: "@ZeroClover",
		SupportedTypes: []string{
			"Basic8",
			"ALIAS",
			"PTR:Can:Only in reverse zones (in-addr.arpa, ip6.arpa)",
			"SOA:Unimplemented:Constellix keeps the SOA on the zone object, not as a record",
		},
		CanAutoDNSSEC: providers.Cannot("The Constellix v4 API has no DNSSEC endpoints"),
		CanConcur:     providers.Can(),
		CanDualHost:   providers.Can(),
	})
}

// Initialize initializes a fresh provider instance.
func (p *constellixProvider) Initialize(settings map[string]string, _ json.RawMessage, options *providers.CreateOptions) error {
	apiKey, secretKey := settings["api_key"], settings["secret_key"]
	if apiKey == "" {
		return errors.New("missing CONSTELLIX api_key")
	}
	if secretKey == "" {
		return errors.New("missing CONSTELLIX secret_key")
	}

	*p = constellixProvider{client: newAPIClient(defaultAPIURL, apiKey, secretKey)}
	p.zones = zonecache.New(p.fetchZones)
	p.SetConversionObserver(options.WithDefaults().ConversionObserver)
	return nil
}

// SetConversionObserver lets the integration tests record conversions for the golden files.
func (p *constellixProvider) SetConversionObserver(observer providers.ConversionObserver) {
	p.observer = observer
}

// GetNameservers returns the nameservers Constellix lists on the zone.
//
// preview and push ask for the nameservers before they check whether a zone
// still has to be created, so a missing zone is a warning, not an error. A
// zone this run tried to create but cannot read yet is an error instead, so
// that no delegation without Constellix is computed.
func (p *constellixProvider) GetNameservers(domain string) ([]*models.Nameserver, error) {
	z, err := p.getZone(domain)
	if errors.Is(err, zonecache.ErrZoneNotFound) {
		printer.Warnf("CONSTELLIX: zone %s not found (reporting no nameservers)\n", domain)
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	names := make([]string, 0, len(z.nameservers))
	for _, ns := range z.nameservers {
		names = append(names, strings.TrimSuffix(ns, "."))
	}
	return models.ToNameservers(names)
}
