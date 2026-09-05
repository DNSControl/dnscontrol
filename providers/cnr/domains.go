package cnr

import "github.com/DNSControl/dnscontrol/v5/models"

// EnsureZoneExists returns an error
// * if access to dnszone is not allowed (not authorized) or
// * if it doesn't exist and creating it fails.
func (client *Client) EnsureZoneExists(dc *models.DomainConfig) error {
	domain := dc.Name
	command := map[string]any{
		"COMMAND": "AddDNSZone",
		"DNSZONE": domain,
	}
	if client.APIEntity == "OTE" {
		command["SOATTL"] = "33200"
		command["SOASERIAL"] = "0000000000"
	}
	// Create the zone
	r := client.client.Request(command)
	if r.GetCode() == 549 || r.IsSuccess() {
		return nil
	}
	return client.GetAPIError("Failed to create not existing zone ", domain, r)
}

// ListZones lists all the.
func (client *Client) ListZones() ([]string, error) {
	var zones []string

	// Basic

	rs := client.client.RequestAllResponsePages(map[string]string{
		"COMMAND": "QueryDNSZoneList",
	})
	for _, r := range rs {
		if r.IsError() {
			return nil, client.GetAPIError("Error while QueryDNSZoneList", "Basic", &r)
		}
		zoneColumn := r.GetColumn("DNSZONE")
		if zoneColumn != nil {
			zones = append(zones, zoneColumn.GetData()...)
		}
	}

	return zones, nil
}
