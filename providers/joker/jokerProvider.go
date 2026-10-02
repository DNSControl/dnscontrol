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
		SupportedTypes: []string{
			"Default",
			"IMPORT_TRANSFORM",
			"NAPTR",
		},
		CanConcur:              providers.Cannot("Joker API has session-based authentication"),
		CanUseDSForChildren:    providers.Cannot(),
		DocDualHost:            providers.Cannot(),
		DocOfficiallySupported: providers.Cannot(),
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
