package cloudpress

import (
	"encoding/json"
	"errors"
	"strings"

	"github.com/DNSControl/dnscontrol/v4/models"
	"github.com/DNSControl/dnscontrol/v4/pkg/providers"
)

var features = providers.DocumentationNotes{
	// The default for unlisted capabilities is 'Cannot'.
	// See providers/capabilities.go for the entire list of capabilities.
	providers.CanAutoDNSSEC:          providers.Can(),
	providers.CanConcur:              providers.Unimplemented(),
	providers.CanGetZones:            providers.Can(),
	providers.CanUseCAA:              providers.Can(),
	providers.CanUsePTR:              providers.Can(),
	providers.CanUseSRV:              providers.Can(),
	providers.DocCreateDomains:       providers.Can(),
	providers.DocDualHost:            providers.Cannot(),
	providers.DocOfficiallySupported: providers.Cannot(),
}

type cloudpressProvider struct {
	apiToken  string
	baseURL   string
	accountID string
	zones     map[string]*zone
}

func init() {
	const providerName = "CLOUDPRESS"
	const providerMaintainer = "@arnoschoon"
	fns := providers.DspFuncs{
		Initializer:   newCloudpress,
		RecordAuditor: AuditRecords,
	}
	providers.RegisterDomainServiceProviderType(providerName, fns, features)
	providers.RegisterMaintainer(providerName, providerMaintainer)

	providers.RegisterCredsMetadata(providerName, providers.CredsMetadata{
		DisplayName: "CloudPress",
		Kind:        providers.KindDNS,
		DocsURL:     "https://docs.dnscontrol.org/provider/cloudpress",
		PortalURL:   "https://docs.cloudpress.com/API/api-keys/",
		Fields: []providers.CredsField{
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
	})
}

func newCloudpress(settings map[string]string, _ json.RawMessage) (providers.DNSServiceProvider, error) {
	baseURL := strings.TrimRight(settings["base_url"], "/")
	if baseURL == "" {
		return nil, errors.New("missing CLOUDPRESS base_url")
	}
	// The credential's api-url may already include the "/api" path; the request
	// helper adds "/api/..." itself, so normalize to the bare scheme+host.
	baseURL = strings.TrimSuffix(baseURL, "/api")

	apiToken := settings["api_token"]
	if apiToken == "" {
		return nil, errors.New("missing CLOUDPRESS api_token")
	}

	return &cloudpressProvider{
		baseURL:   baseURL,
		apiToken:  apiToken,
		accountID: settings["account_id"],
	}, nil
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
