package ns1

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/DNSControl/dnscontrol/v4/pkg/providers"
	"gopkg.in/ns1/ns1-go.v2/rest"
)

var docNotes = providers.DocumentationNotes{
	// The default for unlisted capabilities is 'Cannot'.
	// See providers/capabilities.go for the entire list of capabilities.
	providers.CanAutoDNSSEC:          providers.Can(),
	providers.CanGetZones:            providers.Can(),
	providers.CanConcur:              providers.Can(),
	providers.CanUseAlias:            providers.Can(),
	providers.CanUseCAA:              providers.Can(),
	providers.CanUseDNAME:            providers.Can(),
	providers.CanUseDS:               providers.Can(),
	providers.CanUseDSForChildren:    providers.Can(),
	providers.CanUseDHCID:            providers.Can(),
	providers.CanUseHTTPS:            providers.Can(),
	providers.CanUseLOC:              providers.Cannot(),
	providers.CanUseNAPTR:            providers.Can(),
	providers.CanUsePTR:              providers.Can(),
	providers.CanUseSRV:              providers.Can(),
	providers.CanUseSVCB:             providers.Can(),
	providers.CanUseTLSA:             providers.Can(),
	providers.DocCreateDomains:       providers.Can(),
	providers.DocDualHost:            providers.Can(),
	providers.DocOfficiallySupported: providers.Cannot(),
}

// clientRetries is the number of retries for API backend requests in case of StatusTooManyRequests responses.
const clientRetries = 10

func init() {
	const providerName = "NS1"
	const providerMaintainer = "@costasd"
	fns := providers.DspFuncs{
		Initializer:   newProvider,
		RecordAuditor: AuditRecords,
	}
	providers.RegisterDomainServiceProviderType(providerName, fns, docNotes)
	providers.RegisterMaintainer(providerName, providerMaintainer)
}

type nsone struct {
	*rest.Client
}

func newProvider(creds map[string]string, meta json.RawMessage) (providers.DNSServiceProvider, error) {
	if creds["api_token"] == "" {
		return nil, errors.New("api_token required for ns1")
	}

	// Enable Sleep API Rate limit strategy - it will sleep until new tokens are available
	// see https://help.ns1.com/hc/en-us/articles/360020250573-About-API-rate-limiting
	// this strategy would imply the least sleep time for non-parallel client requests
	return &nsone{rest.NewClient(
		http.DefaultClient,
		rest.SetAPIKey(creds["api_token"]),
		func(c *rest.Client) {
			c.RateLimitStrategySleep()
		},
	)}, nil
}
