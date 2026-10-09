package cloudpress

import (
	"github.com/DNSControl/dnscontrol/v5/models"
	"github.com/DNSControl/dnscontrol/v5/pkg/printer"
)

// ListZones returns the list of zone names in the account.
func (c *cloudpressProvider) ListZones() ([]string, error) {
	zones, err := c.getAllZones()
	if err != nil {
		return nil, err
	}

	zoneNames := make([]string, 0, len(zones))
	for _, z := range zones {
		zoneNames = append(zoneNames, z.Name)
	}

	return zoneNames, nil
}

// EnsureZoneExists creates the zone if it does not already exist.
func (c *cloudpressProvider) EnsureZoneExists(dc *models.DomainConfig) error {
	domain := dc.Name
	if _, err := c.findZoneByDomain(domain); err == nil {
		return nil
	}

	z, err := c.createZone(domain)
	if err != nil {
		return err
	}

	printer.Warnf("CLOUDPRESS: Added zone %s with ID %s\n", domain, z.ID)
	return nil
}
