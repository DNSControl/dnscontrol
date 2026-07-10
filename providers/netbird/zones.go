package netbird

import (
	"fmt"
)

// listZones returns all zones from the NetBird API.
func (api *netbirdProvider) listZones() ([]Zone, error) {
	var zones []Zone
	err := api.doRequest("GET", "/dns/zones", nil, &zones)
	return zones, err
}

// findZoneByDomain finds a zone by its domain name.
func (api *netbirdProvider) findZoneByDomain(domain string) (*Zone, error) {
	zones, err := api.listZones()
	if err != nil {
		return nil, err
	}

	for _, zone := range zones {
		if zone.Domain == domain {
			return &zone, nil
		}
	}

	return nil, fmt.Errorf("zone not found for domain: %s", domain)
}

// createZone creates a new zone.
func (api *netbirdProvider) createZone(zone *Zone) error {
	var result Zone
	return api.doRequest("POST", "/dns/zones", zone, &result)
}

// updateZone updates an existing zone.
func (api *netbirdProvider) updateZone(zoneID string, zone *Zone) error {
	var result Zone
	return api.doRequest("PUT", fmt.Sprintf("/dns/zones/%s", zoneID), zone, &result)
}
