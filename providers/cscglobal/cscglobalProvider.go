package cscglobal

import (
	"encoding/json"
	"errors"
	"strings"

	"github.com/DNSControl/dnscontrol/v5/pkg/providers"
)

/*

CSC Global Registrar:

Info required in `creds.json`:
   - api-key             Api Key
   - user-token          User Token
   - notification_emails (optional) Comma separated list of email addresses to send notifications to
*/

type providerClient struct {
	key          string
	token        string
	notifyEmails []string
}

// Set cscDebug to true if you want to see the JSON of important API requests and responses.
var cscDebug = false

// Initialize initializes a fresh provider instance.
func (client *providerClient) Initialize(m map[string]string, _ json.RawMessage, _ *providers.CreateOptions) error {
	client.key, client.token = m["api-key"], m["user-token"]
	if client.key == "" || client.token == "" {
		return errors.New("missing CSC Global api-key and/or user-token")
	}

	if m["notification_emails"] != "" {
		client.notifyEmails = strings.Split(m["notification_emails"], ",")
	}

	return nil
}

func init() {
	providers.Register[*providerClient]("CSCGLOBAL", providers.Definition{
		FriendlyName: "CSC Global",
		Maintainer:   "@mikenz",
		SupportedTypes: []string{
			"Default",
		},
		CanConcur:              providers.Can(),
		DocOfficiallySupported: providers.Can(),
	})
}
