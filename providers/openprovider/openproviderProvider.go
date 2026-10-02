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
		Features: providers.DocumentationNotes{
			providers.CanAutoDNSSEC:          providers.Unimplemented("DNSSEC can be enabled outside DNSControl"),
			providers.CanConcur:              providers.Unimplemented(),
			providers.CanGetZones:            providers.Can(),
			providers.CanUseAlias:            providers.Cannot(),
			providers.CanUseCAA:              providers.Can(),
			providers.CanUseDHCID:            providers.Cannot(),
			providers.CanUseDNAME:            providers.Cannot(),
			providers.CanUseDNSKEY:           providers.Cannot("DNSSEC keys are managed by Openprovider"),
			providers.CanUseDS:               providers.Cannot(),
			providers.CanUseHTTPS:            providers.Cannot(),
			providers.CanUseLOC:              providers.Cannot(),
			providers.CanUseNAPTR:            providers.Cannot(),
			providers.CanUseOPENPGPKEY:       providers.Cannot(),
			providers.CanUsePTR:              providers.Cannot(),
			providers.CanUseRP:               providers.Cannot(),
			providers.CanUseSMIMEA:           providers.Cannot(),
			providers.CanUseSOA:              providers.Cannot("The SOA record is managed by Openprovider"),
			providers.CanUseSRV:              providers.Can(),
			providers.CanUseSSHFP:            providers.Cannot(),
			providers.CanUseSVCB:             providers.Cannot(),
			providers.CanUseTLSA:             providers.Can(),
			providers.DocCreateDomains:       providers.Can(),
			providers.DocDualHost:            providers.Cannot("Openprovider manages the authoritative NS records"),
			providers.DocOfficiallySupported: providers.Cannot(),
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
