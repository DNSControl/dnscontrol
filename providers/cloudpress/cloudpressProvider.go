package cloudpress

import (
	"encoding/json"
	"errors"
	"strings"

	"github.com/DNSControl/dnscontrol/v5/models"
	"github.com/DNSControl/dnscontrol/v5/pkg/providers"
)

type cloudpressProvider struct {
	apiToken  string
	baseURL   string
	accountID string
	zones     map[string]*zone
	observer  providers.ConversionObserver
}

// SetConversionObserver lets the integration tests record the conversions
// between native and DNSControl records (golden files).
func (c *cloudpressProvider) SetConversionObserver(observer providers.ConversionObserver) {
	c.observer = observer
}

func init() {
	providers.Register[*cloudpressProvider]("CLOUDPRESS", providers.Definition{
		FriendlyName: "CloudPress",
		PortalURL:    "https://docs.cloudpress.com/API/api-keys/",
		CredFields: []providers.CredsField{
			{
				Key:      "base_url",
				Label:    "API base URL",
				Help:     "Base URL of your CloudPress instance, e.g. https://app.cloudpress.com (CloudPress is brand/instance specific, so there is no single default host).",
				Required: true,
			},
			{
				Key:      "api_token",
				Label:    "API token",
				Help:     "CloudPress API key, sent as a Bearer token. Create one via POST /api/api_keys; the value is only returned once.",
				Secret:   true,
				Required: true,
			},
			{
				Key:   "account_id",
				Label: "Account ID",
				Help:  "Account ID sent as the X-Auth-Account header. Required when creating new zones with a user API key.",
			},
		},
		Maintainer: "@arnoschoon",
		SupportedTypes: []string{
			"Basic8",
			"PTR",
		},
		CanAutoDNSSEC:          providers.Can(),
		CanConcur:              providers.Unimplemented(),
		DocDualHost:            providers.Cannot(),
		DocOfficiallySupported: providers.Cannot(),
	})
}

// Initialize initializes a fresh provider instance.
func (c *cloudpressProvider) Initialize(settings map[string]string, _ json.RawMessage, options *providers.CreateOptions) error {
	baseURL := strings.TrimRight(settings["base_url"], "/")
	if baseURL == "" {
		return errors.New("missing CLOUDPRESS base_url")
	}
	// The credential's api-url may already include the "/api" path; the request
	// helper adds "/api/..." itself, so normalize to the bare scheme+host.
	baseURL = strings.TrimSuffix(baseURL, "/api")

	apiToken := settings["api_token"]
	if apiToken == "" {
		return errors.New("missing CLOUDPRESS api_token")
	}

	*c = cloudpressProvider{
		baseURL:   baseURL,
		apiToken:  apiToken,
		accountID: settings["account_id"],
	}
	c.SetConversionObserver(options.WithDefaults().ConversionObserver)
	return nil
}

// GetNameservers returns the nameservers for a domain.
func (c *cloudpressProvider) GetNameservers(domain string) ([]*models.Nameserver, error) {
	z, err := c.findZoneByDomain(domain)
	if err != nil {
		return nil, err
	}

	// The list-zones endpoint does not include nameservers; fetch the full zone.
	full, err := c.getZone(z.ID)
	if err != nil {
		return nil, err
	}

	return models.ToNameservers(full.Nameservers)
}
