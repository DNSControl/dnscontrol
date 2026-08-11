package huaweicloud

import (
	"strings"

	"github.com/DNSControl/dnscontrol/v4/models"
	"github.com/DNSControl/dnscontrol/v4/pkg/printer"
	"github.com/huaweicloud/huaweicloud-sdk-go-v3/services/dns/v2/model"
)

// EnsureZoneExists creates a zone if it does not exist.
func (c *huaweicloudProvider) EnsureZoneExists(dc *models.DomainConfig) error {
	domain := dc.Name

	if err := c.getZones(); err != nil {
		return err
	}

	if _, ok := c.zoneIDByDomain[domain]; ok {
		return nil
	}

	printer.Printf("Adding zone for %s to huaweicloud account in region %s\n", domain, c.region.Id)
	createPayload := model.CreatePublicZoneRequest{
		Body: &model.CreatePublicZoneReq{
			Name: domain,
		},
	}
	res, err := c.client.CreatePublicZone(&createPayload)
	if err != nil {
		return err
	}
	if res.Id == nil {
		// clear cache
		c.zoneIDByDomain = nil
		c.domainByZoneID = nil
		return nil
	}
	c.zoneIDByDomain[domain] = *res.Id
	c.domainByZoneID[*res.Id] = domain
	return nil
}

// ListZones returns all DNS zones managed by this provider.
func (c *huaweicloudProvider) ListZones() ([]string, error) {
	if err := c.getZones(); err != nil {
		return nil, err
	}
	var zones []string
	for i := range c.zoneIDByDomain {
		zones = append(zones, i)
	}
	return zones, nil
}

func (c *huaweicloudProvider) getZones() error {
	if c.zoneIDByDomain != nil {
		return nil
	}

	var nextMarker *string
	c.zoneIDByDomain = make(map[string]string)
	c.domainByZoneID = make(map[string]string)

	for {
		listPayload := model.ListPublicZonesRequest{
			Marker: nextMarker,
		}
		zonesRes, err := c.client.ListPublicZones(&listPayload)
		if err != nil {
			return err
		}
		// empty zones
		if zonesRes.Zones == nil {
			return nil
		}
		for _, zone := range *zonesRes.Zones {
			// just a safety check
			if zone.Name == nil || zone.Id == nil {
				continue
			}
			domain := strings.TrimSuffix(*zone.Name, ".")
			c.zoneIDByDomain[domain] = *zone.Id
			c.domainByZoneID[*zone.Id] = domain
		}

		// if has next page, continue to get next page
		if zonesRes.Links.Next != nil {
			marker, err := parseMarkerFromURL(*zonesRes.Links.Next)
			if err != nil {
				return err
			}
			nextMarker = &marker
		} else {
			return nil
		}
	}
}
