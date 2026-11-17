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
		Maintainer:   "@nvanlaerebeke",
		SupportedTypes: []string{
			"Basic8",
			"TLSA",
			"NS:Cannot:Openprovider manages the apex nameservers and does not support delegations",
			"DNSKEY:Cannot:DNSSEC keys are managed by Openprovider",
			"SOA:Cannot:The SOA record is managed by Openprovider",
		},
		DefaultTTL:    minimumTTL,
		CanAutoDNSSEC: providers.Unimplemented("DNSSEC can be enabled outside DNSControl"),
		CanDualHost:   providers.Cannot("Openprovider manages the authoritative NS records"),
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
		PortalURL: "https://cp.openprovider.eu/",
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
