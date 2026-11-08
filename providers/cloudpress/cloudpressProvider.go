package cloudpress

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/DNSControl/dnscontrol/v5/models"
	"github.com/DNSControl/dnscontrol/v5/pkg/providers"
)

type cloudpressProvider struct {
	apiToken       string
	baseURL        string
	accountID      string
	retryBaseDelay time.Duration
	retryMaxDelay  time.Duration
	zones          map[string]*zone // by name, from the list endpoint
	fullZones      map[string]*zone // by ID, with records and nameservers
	observer       providers.ConversionObserver
}

// SetConversionObserver lets the integration tests record the conversions
// between native and DNSControl records (golden files).
func (c *cloudpressProvider) SetConversionObserver(observer providers.ConversionObserver) {
	c.observer = observer
}

func init() {
	providers.Register[*cloudpressProvider]("CLOUDPRESS", providers.Definition{
		FriendlyName: "CloudPress",
		Maintainer:   "@arnoschoon",
		SupportedTypes: []string{
			"Basic8",
			"PTR",
		},
		CanAutoDNSSEC: providers.Can(),
		CanDualHost:   providers.Cannot(),
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
			{
				Key:   "retry_base_delay",
				Label: "Retry base delay",
				Help:  "Initial wait before retrying a rate-limited (HTTP 429) or failed request, as a Go duration such as 10s. Doubles on each retry. Defaults to 1s.",
			},
			{
				Key:   "retry_max_delay",
				Label: "Retry max delay",
				Help:  "Upper bound for a single retry wait, as a Go duration such as 5m. Defaults to 30s.",
			},
		},
		PortalURL: "https://docs.cloudpress.com/API/api-keys/",
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

	retryBaseDelay, err := parseDurationSetting(settings, "retry_base_delay", defaultRetryBaseDelay)
	if err != nil {
		return err
	}
	retryMaxDelay, err := parseDurationSetting(settings, "retry_max_delay", defaultRetryMaxDelay)
	if err != nil {
		return err
	}

	*c = cloudpressProvider{
		baseURL:        baseURL,
		apiToken:       apiToken,
		accountID:      settings["account_id"],
		retryBaseDelay: retryBaseDelay,
		retryMaxDelay:  retryMaxDelay,
	}
	c.SetConversionObserver(options.WithDefaults().ConversionObserver)
	return nil
}

// parseDurationSetting reads an optional positive duration (e.g. "10s") from
// the provider settings, returning def when it is not set.
func parseDurationSetting(settings map[string]string, key string, def time.Duration) (time.Duration, error) {
	v := settings[key]
	if v == "" {
		return def, nil
	}
	d, err := time.ParseDuration(v)
	if err != nil || d <= 0 {
		return 0, fmt.Errorf("invalid CLOUDPRESS %s %q: must be a positive duration such as 10s", key, v)
	}
	return d, nil
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
