// Package websupport implements a DNS provider for WebSupport (websupport.sk),
// using their REST API v2 (https://rest.websupport.sk/v2/docs).
package websupport

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/DNSControl/dnscontrol/v4/models"
	"github.com/DNSControl/dnscontrol/v4/pkg/providers"
)

var features = providers.DocumentationNotes{
	// The default for unlisted capabilities is 'Cannot'.
	// See providers/capabilities.go for the entire list of capabilities.
	providers.CanGetZones:            providers.Cannot("WebSupport has no list-all-zones endpoint."),
	providers.CanConcur:              providers.Unimplemented(),
	providers.CanUseAlias:            providers.Cannot("WebSupport's ANAME is apex-only and conflicts with other apex records."),
	providers.CanUseCAA:              providers.Cannot("The v2 API does not return CAA tag/flags on read, so records cannot be managed without churn."),
	providers.CanUseLOC:              providers.Cannot(),
	providers.CanUseNAPTR:            providers.Cannot(),
	providers.CanUsePTR:              providers.Cannot(),
	providers.CanUseSRV:              providers.Can(),
	providers.CanUseSSHFP:            providers.Cannot(),
	providers.CanUseTLSA:             providers.Cannot(),
	providers.CanUseDS:               providers.Cannot(),
	providers.CanUseSOA:              providers.Cannot(),
	providers.DocCreateDomains:       providers.Cannot("Zones must be created via the WebSupport portal."),
	providers.DocDualHost:            providers.Cannot(),
	providers.DocOfficiallySupported: providers.Cannot(),
}

type websupportProvider struct {
	apiKey     string
	secret     string
	baseURL    string
	httpClient *http.Client
	// services caches the domain -> numeric service id mapping that the v2
	// DNS endpoints require as their {service} path segment.
	services map[string]int64
}

func init() {
	const providerName = "WEBSUPPORT"
	const providerMaintainer = "@mtmn"
	fns := providers.DspFuncs{
		Initializer:   newWebsupport,
		RecordAuditor: AuditRecords,
	}
	providers.RegisterDomainServiceProviderType(providerName, fns, features)
	providers.RegisterMaintainer(providerName, providerMaintainer)
	providers.RegisterCredsMetadata(providerName, providers.CredsMetadata{
		DisplayName: "WebSupport",
		Kind:        providers.KindDNS,
		DocsURL:     "https://docs.dnscontrol.org/provider/websupport",
		PortalURL:   "https://admin.websupport.sk/en/auth/security",
		Fields: []providers.CredsField{
			{
				Key:      "api_key",
				Label:    "API key",
				Help:     "WebSupport API key (generated in the Security section of the admin console).",
				Secret:   true,
				Required: true,
			},
			{
				Key:      "secret",
				Label:    "API secret",
				Help:     "WebSupport API secret used to sign requests.",
				Secret:   true,
				Required: true,
			},
		},
	})
}

func newWebsupport(settings map[string]string, _ json.RawMessage) (providers.DNSServiceProvider, error) {
	apiKey := settings["api_key"]
	secret := settings["secret"]
	if apiKey == "" || secret == "" {
		return nil, errors.New("WEBSUPPORT: missing api_key and/or secret")
	}

	baseURL := settings["base_url"]
	if baseURL == "" {
		baseURL = defaultBaseURL
	}

	return &websupportProvider{
		apiKey:     apiKey,
		secret:     secret,
		baseURL:    baseURL,
		httpClient: &http.Client{Timeout: 30 * time.Second},
		services:   map[string]int64{},
	}, nil
}

// GetNameservers returns the apex NS records of the zone. WebSupport's zone
// detail endpoint does not expose nameservers directly, so they are read from
// the zone's own NS records.
func (c *websupportProvider) GetNameservers(domain string) ([]*models.Nameserver, error) {
	svcID, err := c.serviceID(domain)
	if err != nil {
		return nil, err
	}

	records, err := c.getAllRecords(svcID)
	if err != nil {
		return nil, err
	}

	// The v2 API returns fully-qualified record names, so apex NS records
	// have the bare domain as their name.
	var ns []string
	for _, r := range records {
		if r.Type == "NS" && r.Name == domain {
			ns = append(ns, trimDot(r.Content))
		}
	}
	return models.ToNameservers(ns)
}
