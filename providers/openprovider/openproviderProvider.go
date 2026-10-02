package openprovider

import (
	"encoding/json"
	"errors"

	"github.com/DNSControl/dnscontrol/v5/pkg/providers"
)

const minimumTTL = uint32(900)

type openproviderProvider struct {
	client *apiClient
}

func init() {
	providers.Register[*openproviderProvider]("OPENPROVIDER", providers.Definition{
		FriendlyName: "Openprovider",
		PortalURL:    "https://cp.openprovider.eu/",
		CredFields: []providers.CredsField{
			{Key: "username", Label: "Username", Required: true},
			{Key: "password", Label: "Password", Required: true, Secret: true},
			{
				Key:     "api_url",
				Label:   "API base URL",
				Help:    "Only needed for a non-production Openprovider environment.",
				Default: defaultAPIURL,
			},
		},
		Maintainer: "@nvanlaerebeke",
		DefaultTTL: minimumTTL,
		SupportedTypes: []string{
			"Default",
			"TLSA",
			"NS:Cannot",
		},
		CanAutoDNSSEC:          providers.Unimplemented("DNSSEC can be enabled outside DNSControl"),
		CanConcur:              providers.Unimplemented(),
		DocDualHost:            providers.Cannot("Openprovider manages the authoritative NS records"),
		DocOfficiallySupported: providers.Cannot(),
		// Features retains annotations for record types and interface-derived facts.
		Features: providers.DocumentationNotes{
			providers.CanUseDNSKEY: providers.Cannot("DNSSEC keys are managed by Openprovider"),
			providers.CanUseSOA:    providers.Cannot("The SOA record is managed by Openprovider"),
		},
	})
}

// Initialize initializes a fresh provider instance.
func (p *openproviderProvider) Initialize(settings map[string]string, _ json.RawMessage, _ *providers.CreateOptions) error {
	username := settings["username"]
	password := settings["password"]
	if username == "" {
		return errors.New("missing OPENPROVIDER username")
	}
	if password == "" {
		return errors.New("missing OPENPROVIDER password")
	}

	apiURL := settings["api_url"]
	if apiURL == "" {
		apiURL = defaultAPIURL
	}
	client, err := newAPIClient(apiURL, username, password)
	if err != nil {
		return err
	}
	*p = openproviderProvider{client: client}
	return nil
}
