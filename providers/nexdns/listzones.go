package nexdns

import (
	"fmt"

	"github.com/DNSControl/dnscontrol/v4/models"
	"github.com/DNSControl/dnscontrol/v4/pkg/printer"
)

// ListZones returns every zone the API key can see.
func (n *nexdnsProvider) ListZones() ([]string, error) {
	zones, err := n.client.listZones()
	if err != nil {
		return nil, err
	}

	names := make([]string, 0, len(zones))
	for _, z := range zones {
		names = append(names, z.Name)
	}

	return names, nil
}

// EnsureZoneExists creates the zone if the account does not have it yet.
func (n *nexdnsProvider) EnsureZoneExists(dc *models.DomainConfig) error {
	_, err := n.getZone(dc.Name)
	if err == nil {
		return nil
	}
	if !isNotFound(err) {
		return err
	}

	if err := n.client.createZone(dc.Name); err != nil {
		return fmt.Errorf("NEXDNS: creating zone %s: %w", dc.Name, err)
	}

	printer.Warnf("NEXDNS: Added zone %s\n", dc.Name)
	return nil
}
