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
		FriendlyName:           "RWTH Aachen",
		Maintainer:             "@mistererwin",
		SupportedTypes:         []string{"Default", "IMPORT_TRANSFORM", "PTR", "SSHFP", "DS:Unimplemented"},
		CanAutoDNSSEC:          providers.Unimplemented("Supported by RWTH but not implemented yet."),
		CanConcur:              providers.Unimplemented(),
		DocDualHost:            providers.Cannot(),
		DocOfficiallySupported: providers.Cannot(),
		// Features retains annotations for record types and interface-derived facts.
		Features: providers.DocumentationNotes{
			providers.CanUseDS:  providers.Unimplemented("DS records are only supported at the apex and require a different API call that hasn't been implemented yet."),
			providers.CanUsePTR: providers.Can("PTR records with empty targets are not supported"),
			providers.CanUseSRV: providers.Can("SRV records with empty targets are not supported."),
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
