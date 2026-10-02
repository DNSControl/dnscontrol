package ns1

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/DNSControl/dnscontrol/v5/pkg/providers"
	"gopkg.in/ns1/ns1-go.v2/rest"
)

// clientRetries is the number of retries for API backend requests in case of StatusTooManyRequests responses.
const clientRetries = 10

func init() {
	providers.Register[*nsone]("NS1", providers.Definition{
		FriendlyName: "NS1",
		PortalURL:    "https://my.nsone.net/#/account/settings/keys",
		CredFields: []providers.CredsField{
			{
				Key:      "api_token",
				Label:    "API token",
				Help:     "Your NS1 API token.",
				Secret:   true,
				Required: true,
			},
		},
		Maintainer: "@costasd",
		SupportedTypes: []string{
			"Default",
			"ALIAS",
			"DHCID",
			"DNAME",
			"DS",
			"HTTPS",
			"NAPTR",
			"PTR",
			"SVCB",
			"TLSA",
		},
		CanAutoDNSSEC:          providers.Can(),
		CanConcur:              providers.Can(),
		CanUseDSForChildren:    providers.Can(),
		DocDualHost:            providers.Can(),
		DocOfficiallySupported: providers.Cannot(),
	})
}

type nsone struct {
	*rest.Client
	observer providers.ConversionObserver
}

func (n *nsone) SetConversionObserver(observer providers.ConversionObserver) {
	n.observer = observer
}

// Initialize initializes a fresh provider instance.
func (n *nsone) Initialize(creds map[string]string, meta json.RawMessage, options *providers.CreateOptions) error {
	if creds["api_token"] == "" {
		return errors.New("api_token required for ns1")
	}

	// Enable Sleep API Rate limit strategy - it will sleep until new tokens are available
	// see https://help.ns1.com/hc/en-us/articles/360020250573-About-API-rate-limiting
	// this strategy would imply the least sleep time for non-parallel client requests
	*n = nsone{Client: rest.NewClient(
		http.DefaultClient,
		rest.SetAPIKey(creds["api_token"]),
		func(c *rest.Client) {
			c.RateLimitStrategySleep()
		},
	)}
	n.SetConversionObserver(options.WithDefaults().ConversionObserver)
	return nil
}
