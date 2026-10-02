package rwth

import (
	"encoding/json"
	"errors"

	"github.com/DNSControl/dnscontrol/v5/pkg/providers"
)

type rwthProvider struct {
	apiToken string
	zones    map[string]zone
}

func init() {
	providers.Register[*rwthProvider]("RWTH", providers.Definition{
		FriendlyName: "RWTH Aachen",
		DocsURL:      "https://docs.dnscontrol.org/provider/rwth",
		Maintainer:   "@mistererwin",
		Features: providers.DocumentationNotes{
			// The default for unlisted capabilities is 'Cannot'.
			// See providers/capabilities.go for the entire list of capabilities.
			providers.CanAutoDNSSEC:          providers.Unimplemented("Supported by RWTH but not implemented yet."),
			providers.CanConcur:              providers.Unimplemented(),
			providers.CanGetZones:            providers.Can(),
			providers.CanUseAlias:            providers.Cannot(),
			providers.CanUseCAA:              providers.Can(),
			providers.CanUseDS:               providers.Unimplemented("DS records are only supported at the apex and require a different API call that hasn't been implemented yet."),
			providers.CanUseLOC:              providers.Cannot(),
			providers.CanUseNAPTR:            providers.Cannot(),
			providers.CanUsePTR:              providers.Can("PTR records with empty targets are not supported"),
			providers.CanUseSRV:              providers.Can("SRV records with empty targets are not supported."),
			providers.CanUseSSHFP:            providers.Can(),
			providers.CanUseTLSA:             providers.Cannot(),
			providers.DocCreateDomains:       providers.Cannot(),
			providers.DocDualHost:            providers.Cannot(),
			providers.DocOfficiallySupported: providers.Cannot(),
		},
	})
}

// Initialize initializes a fresh provider instance.
func (api *rwthProvider) Initialize(settings map[string]string, _ json.RawMessage, _ *providers.CreateOptions) error {
	if settings["api_token"] == "" {
		return errors.New("missing RWTH api_token")
	}

	*api = rwthProvider{apiToken: settings["api_token"]}

	return nil
}
