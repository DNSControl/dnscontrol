package joker

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/DNSControl/dnscontrol/v5/pkg/providers"
)

/*

Joker DMAPI provider:

Info required in `creds.json`:
   - username
   - password
   OR
   - api-key

*/

func init() {
	providers.Register[*jokerProvider]("JOKER", providers.Definition{
		FriendlyName: "Joker.com",
		DocsURL:      "https://docs.dnscontrol.org/provider/joker",
		PortalURL:    "https://joker.com/", // TODO: Verify
		CredFields: []providers.CredsField{
			{
				Key:      "username",
				Label:    "Username",
				Help:     "Your Joker.com DMAPI username.",
				Required: true,
			},
			{
				Key:      "password",
				Label:    "Password",
				Help:     "Your Joker.com DMAPI password.",
				Secret:   true,
				Required: true,
			},
		},
		Maintainer: "@atrull",
		Features: providers.DocumentationNotes{
			// The default for unlisted capabilities is 'Cannot'.
			// See providers/capabilities.go for the entire list of capabilities.
			providers.CanGetZones:            providers.Can(),
			providers.CanConcur:              providers.Cannot("Joker API has session-based authentication"),
			providers.CanUseAlias:            providers.Cannot(),
			providers.CanUseCAA:              providers.Can(),
			providers.CanUseDNSKEY:           providers.Cannot(),
			providers.CanUseDS:               providers.Cannot(),
			providers.CanUseDSForChildren:    providers.Cannot(),
			providers.CanUseHTTPS:            providers.Cannot(),
			providers.CanUseLOC:              providers.Cannot(),
			providers.CanUseNAPTR:            providers.Can(),
			providers.CanUsePTR:              providers.Cannot(),
			providers.CanUseSOA:              providers.Cannot(),
			providers.CanUseSRV:              providers.Can(),
			providers.CanUseSSHFP:            providers.Cannot(),
			providers.CanUseSVCB:             providers.Cannot(),
			providers.CanUseTLSA:             providers.Cannot(),
			providers.DocCreateDomains:       providers.Can(),
			providers.DocDualHost:            providers.Cannot(),
			providers.DocOfficiallySupported: providers.Cannot(),
		},
	})
}

// jokerProvider is the handle for API calls.
type jokerProvider struct {
	apiURL     string
	username   string
	password   string
	apiKey     string
	authSID    string
	httpClient *http.Client
}

// Initialize initializes a fresh provider instance.
func (api *jokerProvider) Initialize(m map[string]string, _ json.RawMessage, _ *providers.CreateOptions) error {
	*api = jokerProvider{
		apiURL:     "https://dmapi.joker.com/request/",
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}

	// Check for authentication methods
	api.username = m["username"]
	api.password = m["password"]
	api.apiKey = m["api-key"]

	if api.apiKey == "" && (api.username == "" || api.password == "") {
		return errors.New("missing Joker credentials: either 'api-key' or both 'username' and 'password' required")
	}

	// Authenticate to get session ID
	if err := api.authenticate(); err != nil {
		return fmt.Errorf("authentication failed: %w", err)
	}

	return nil
}
