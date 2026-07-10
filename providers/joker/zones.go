package joker

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/DNSControl/dnscontrol/v4/models"
)

// EnsureZoneExists creates a zone if it doesn't exist.
func (api *jokerProvider) EnsureZoneExists(dc *models.DomainConfig) error {
	domain := dc.Name
	// For Joker, all domains you manage automatically have DNS zones available
	// We just need to verify we can access the zone
	_, body, err := api.makeRequest("dns-zone-get", url.Values{"domain": {domain}})
	if err == nil {
		// Zone exists and is accessible - check if it has content
		if strings.TrimSpace(body) != "" {
			// Zone exists with content, no action needed
			return nil
		}
	}

	// Zone doesn't exist or is empty, initialize it
	params := url.Values{}
	params.Set("domain", domain)
	params.Set("zone", "# Zone initialized by dnscontrol")

	_, _, err = api.makeRequest("dns-zone-put", params)
	return err
}

// ListZones returns a list of zones managed by this provider.
func (api *jokerProvider) ListZones() ([]string, error) {
	_, body, err := api.makeRequest("dns-zone-list", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to list zones: %w", err)
	}

	var zones []string
	lines := strings.SplitSeq(strings.TrimSpace(body), "\n")
	for line := range lines {
		line = strings.TrimSpace(line)
		if line != "" && !strings.HasPrefix(line, "#") {
			// dns-zone-list returns domains with expiration dates: "domain.com 2025-12-31"
			// We only need the domain name (first part)
			parts := strings.Fields(line)
			if len(parts) >= 1 {
				zones = append(zones, parts[0])
			}
		}
	}

	return zones, nil
}
