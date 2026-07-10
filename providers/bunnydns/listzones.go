package bunnydns

import (
	"github.com/DNSControl/dnscontrol/v4/models"
	"github.com/DNSControl/dnscontrol/v4/pkg/printer"
)

func (b *bunnydnsProvider) ListZones() ([]string, error) {
	zones, err := b.getAllZones()
	if err != nil {
		return nil, err
	}

	zoneNames := make([]string, 0, len(zones))
	for _, zone := range zones {
		zoneNames = append(zoneNames, zone.Domain)
	}

	return zoneNames, nil
}

func (b *bunnydnsProvider) EnsureZoneExists(dc *models.DomainConfig) error {
	domain := dc.Name
	_, err := b.findZoneByDomain(domain)
	if err == nil {
		return nil
	}

	zone, err := b.createZone(domain)
	if err != nil {
		return err
	}

	printer.Warnf("BUNNY_DNS: Added zone %s with ID %d", domain, zone.ID)
	return nil
}
