package cnr

// EnsureZoneExists returns an error
// * if access to dnszone is not allowed (not authorized) or
// * if it doesn't exist and creating it fails.
func (n *Client) EnsureZoneExists(domain string, metadata map[string]string) error {
	command := map[string]any{
		"COMMAND": "AddDNSZone",
		"DNSZONE": domain,
	}
	if n.APIEntity == "OTE" {
		command["SOATTL"] = "33200"
		command["SOASERIAL"] = "0000000000"
	}
	// Create the zone
	r := n.client.Request(command)
	if r.GetCode() == 549 || r.IsSuccess() {
		return nil
	}
	return n.GetAPIError("Failed to create not existing zone ", domain, r)
}

// ListZones lists all the.
func (n *Client) ListZones() ([]string, error) {
	var zones []string

	// Basic

	rs := n.client.RequestAllResponsePages(map[string]string{
		"COMMAND": "QueryDNSZoneList",
	})
	for _, r := range rs {
		if r.IsError() {
			return nil, n.GetAPIError("Error while QueryDNSZoneList", "Basic", &r)
		}
		zoneColumn := r.GetColumn("DNSZONE")
		if zoneColumn != nil {
			zones = append(zones, zoneColumn.GetData()...)
		}
	}

	return zones, nil
}
