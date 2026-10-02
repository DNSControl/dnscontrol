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

type nexdnsProvider struct {
	client *apiClient
	zones  map[string]*apiZone
}

func init() {
	providers.Register[*nexdnsProvider]("NEXDNS", providers.Definition{
		FriendlyName: "NexDNS",
		PortalURL:    "https://nexdns.tech/settings/api-keys",
		Notes:        "The API is available on a plan that includes API access. See https://nexdns.tech/pricing.",
		CredFields: []providers.CredsField{
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
		Maintainer:             "@nexdns",
		DefaultTTL:             defaultTTL,
		SupportedTypes:         []string{"Default", "ALIAS", "DNAME", "IMPORT_TRANSFORM", "PTR", "TLSA"},
		CanAutoDNSSEC:          providers.Unimplemented("DNSSEC is switched on per zone outside of DNSControl"),
		CanConcur:              providers.Unimplemented(),
		CanUseDSForChildren:    providers.Can(),
		DocDualHost:            providers.Cannot("The NS records at the zone apex cannot be changed through the API"),
		DocOfficiallySupported: providers.Cannot(),
		// Features retains annotations for record types and interface-derived facts.
		Features: providers.DocumentationNotes{
			providers.CanUseDS:  providers.Cannot("DS at the zone apex belongs in the parent zone"),
			providers.CanUseSOA: providers.Cannot("The SOA record is maintained by the platform"),
		},
	})
}

// Initialize initializes a fresh provider instance.
func (n *nexdnsProvider) Initialize(settings map[string]string, _ json.RawMessage, _ *providers.CreateOptions) error {
	token := settings["api_token"]
	if token == "" {
		return errors.New("missing NEXDNS api_token")
	}

	apiURL := settings["api_url"]
	if apiURL == "" {
		apiURL = defaultAPIURL
	}

	*n = nexdnsProvider{
		client: newAPIClient(apiURL, token),
		zones:  map[string]*apiZone{},
	}
	return nil
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
