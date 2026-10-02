// Package websupport implements a DNS provider for WebSupport (websupport.sk),
// using their REST API v2 (https://rest.websupport.sk/v2/docs).
package websupport

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/DNSControl/dnscontrol/v5/models"
	"github.com/DNSControl/dnscontrol/v5/pkg/providers"
)

type websupportProvider struct {
	observer   providers.ConversionObserver
	apiKey     string
	secret     string
	baseURL    string
	httpClient *http.Client
	// services caches the domain -> numeric service id mapping that the v2
	// DNS endpoints require as their {service} path segment.
	services map[string]int64
}

func (c *websupportProvider) SetConversionObserver(observer providers.ConversionObserver) {
	c.observer = observer
}

func init() {
	providers.Register[*websupportProvider]("WEBSUPPORT", providers.Definition{
		FriendlyName: "WebSupport",
		PortalURL:    "https://admin.websupport.sk/en/auth/security",
		CredFields: []providers.CredsField{
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
		Maintainer: "@mtmn",
		SupportedTypes: []string{
			"Default",
			"IMPORT_TRANSFORM",
			"CAA:Cannot",
			"NS:Cannot",
		},
		CanConcur:              providers.Unimplemented(),
		DocDualHost:            providers.Cannot(),
		DocOfficiallySupported: providers.Cannot(),
		// Features retains annotations for record types and interface-derived facts.
		Features: providers.DocumentationNotes{
			providers.CanGetZones:      providers.Cannot("WebSupport has no list-all-zones endpoint."),
			providers.CanUseAlias:      providers.Cannot("WebSupport's ANAME is apex-only and conflicts with other apex records."),
			providers.CanUseCAA:        providers.Cannot("The v2 API does not return CAA tag/flags on read, so records cannot be managed without churn."),
			providers.DocCreateDomains: providers.Cannot("Zones must be created via the WebSupport portal."),
		},
	})
}

// Initialize initializes a fresh provider instance.
func (c *websupportProvider) Initialize(settings map[string]string, _ json.RawMessage, options *providers.CreateOptions) error {
	apiKey := settings["api_key"]
	secret := settings["secret"]
	if apiKey == "" || secret == "" {
		return errors.New("WEBSUPPORT: missing api_key and/or secret")
	}

	baseURL := settings["base_url"]
	if baseURL == "" {
		baseURL = defaultBaseURL
	}

	*c = websupportProvider{
		apiKey:     apiKey,
		secret:     secret,
		baseURL:    baseURL,
		httpClient: &http.Client{Timeout: 30 * time.Second},
		services:   map[string]int64{},
	}
	c.SetConversionObserver(options.WithDefaults().ConversionObserver)
	return nil
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
