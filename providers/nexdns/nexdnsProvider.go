package nexdns

import (
	"encoding/json"
	"errors"

	"github.com/DNSControl/dnscontrol/v5/models"
	"github.com/DNSControl/dnscontrol/v5/pkg/providers"
)

/*
NexDNS DNS provider:

Info required in `creds.json`:
   - api_token

Optional in `creds.json`:
   - api_url    Base URL of the API. Defaults to https://api.nexdns.tech/v1.

The API addresses one record value at a time rather than a whole rrset, so this
provider uses diff2.ByRecord(). A record's id is derived from its name, type and
content, which means an id stops resolving as soon as the record it names is
changed. Each correction therefore uses only the id it read from the zone it is
about to modify.

The SOA record and the NS records at the zone apex are maintained by the platform
and are rejected by the API, so they are left out of both the zone contents and
the desired state. See records.go.
*/

var features = providers.DocumentationNotes{
	// The default for unlisted capabilities is 'Cannot'.
	// See providers/capabilities.go for the entire list of capabilities.
	providers.CanAutoDNSSEC:          providers.Unimplemented("DNSSEC is switched on per zone outside of DNSControl"),
	providers.CanConcur:              providers.Unimplemented(),
	providers.CanGetZones:            providers.Can(),
	providers.CanUseAlias:            providers.Can(),
	providers.CanUseCAA:              providers.Can(),
	providers.CanUseDHCID:            providers.Cannot(),
	providers.CanUseDNAME:            providers.Can(),
	providers.CanUseDNSKEY:           providers.Cannot(),
	providers.CanUseDS:               providers.Cannot("DS at the zone apex belongs in the parent zone"),
	providers.CanUseDSForChildren:    providers.Can(),
	providers.CanUseHTTPS:            providers.Cannot(),
	providers.CanUseLOC:              providers.Cannot(),
	providers.CanUseNAPTR:            providers.Cannot(),
	providers.CanUseOPENPGPKEY:       providers.Cannot(),
	providers.CanUsePTR:              providers.Can(),
	providers.CanUseRP:               providers.Cannot(),
	providers.CanUseSMIMEA:           providers.Cannot(),
	providers.CanUseSOA:              providers.Cannot("The SOA record is maintained by the platform"),
	providers.CanUseSRV:              providers.Can(),
	providers.CanUseSSHFP:            providers.Cannot(),
	providers.CanUseSVCB:             providers.Cannot(),
	providers.CanUseTLSA:             providers.Can(),
	providers.DocCreateDomains:       providers.Can(),
	providers.DocDualHost:            providers.Cannot("The NS records at the zone apex cannot be changed through the API"),
	providers.DocOfficiallySupported: providers.Cannot(),
}

type nexdnsProvider struct {
	client *apiClient
	zones  map[string]*apiZone
}

func init() {
	const providerName = "NEXDNS"
	const providerMaintainer = "@nexdns"
	fns := providers.DspFuncs{
		Initializer:   newNexdns,
		RecordAuditor: AuditRecords,
	}
	providers.RegisterDomainServiceProviderType(providerName, fns, features)
	providers.RegisterMaintainer(providerName, providerMaintainer)
	providers.RegisterDefaultTTL(providerName, defaultTTL)

	providers.RegisterCredsMetadata(providerName, providers.CredsMetadata{
		DisplayName: "NexDNS",
		Kind:        providers.KindDNS,
		DocsURL:     "https://docs.dnscontrol.org/provider/nexdns",
		PortalURL:   "https://nexdns.tech/settings/api-keys",
		Notes:       "The API is available on a plan that includes API access. See https://nexdns.tech/pricing.",
		Fields: []providers.CredsField{
			{
				Key:      "api_token",
				Label:    "API token",
				Help:     "An API key carrying the zones.read, zones.write, records.read and records.write scopes.",
				Secret:   true,
				Required: true,
			},
			{
				Key:     "api_url",
				Label:   "API base URL",
				Help:    "Only needed to point the provider at a different endpoint.",
				Default: defaultAPIURL,
			},
		},
	})
}

// newNexdns builds a provider from the entry in creds.json.
func newNexdns(settings map[string]string, _ json.RawMessage) (providers.DNSServiceProvider, error) {
	token := settings["api_token"]
	if token == "" {
		return nil, errors.New("missing NEXDNS api_token")
	}

	apiURL := settings["api_url"]
	if apiURL == "" {
		apiURL = defaultAPIURL
	}

	return &nexdnsProvider{
		client: newAPIClient(apiURL, token),
		zones:  map[string]*apiZone{},
	}, nil
}

// GetNameservers returns the nameservers the zone is served from.
func (n *nexdnsProvider) GetNameservers(domain string) ([]*models.Nameserver, error) {
	zone, err := n.getZone(domain)
	if err != nil {
		return nil, err
	}

	return models.ToNameservers(zone.Nameservers)
}

// getZone looks a zone up by name and remembers it. Every entry point needs the
// zone's id before it can address anything inside the zone, and the lookup costs
// two requests, so it is worth doing once per run.
func (n *nexdnsProvider) getZone(domain string) (*apiZone, error) {
	if zone, ok := n.zones[domain]; ok {
		return zone, nil
	}

	zone, err := n.client.getZone(domain)
	if err != nil {
		return nil, err
	}

	n.zones[domain] = zone
	return zone, nil
}
