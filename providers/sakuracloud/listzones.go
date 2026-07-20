package sakuracloud

import (
	"github.com/DNSControl/dnscontrol/v4/models"
	"github.com/DNSControl/dnscontrol/v4/pkg/printer"
)

// ListZones return all the zones in the account.
func (s *sakuracloudProvider) ListZones() ([]string, error) {
	itemMap, err := s.api.GetCommonServiceItemMap()
	if err != nil {
		return nil, err
	}

	var zones []string
	for _, item := range itemMap {
		zones = append(zones, item.Status.Zone)
	}
	return zones, nil
}

// EnsureZoneExists creates a zone if it does not exist.
func (s *sakuracloudProvider) EnsureZoneExists(dc *models.DomainConfig) error {
	domain := dc.Name
	itemMap, err := s.api.GetCommonServiceItemMap()
	if err != nil {
		return err
	}

	if _, ok := itemMap[domain]; ok {
		return nil
	}

	printer.Printf("Adding zone for %s to Sakura Cloud account\n", domain)
	return s.api.CreateZone(domain)
}
