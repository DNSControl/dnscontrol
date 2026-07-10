package huaweicloud

import (
	"github.com/DNSControl/dnscontrol/v4/models"
	"github.com/huaweicloud/huaweicloud-sdk-go-v3/services/dns/v2/model"
)

func (c *huaweicloudProvider) getDNSSECCorrections(dc *models.DomainConfig) ([]*models.Correction, int, error) {
	if dc.AutoDNSSEC == "" {
		return nil, 0, nil
	}

	zoneID := c.zoneIDByDomain[dc.Name]

	enabled, err := c.isDNSSECEnabled(zoneID)
	if err != nil {
		return nil, 0, err
	}

	if enabled && dc.AutoDNSSEC == "off" {
		return []*models.Correction{{
			Msg: "Disable DNSSEC",
			F:   func() error { return c.disableDNSSEC(zoneID) },
		}}, 1, nil
	}

	if !enabled && dc.AutoDNSSEC == "on" {
		return []*models.Correction{{
			Msg: "Enable DNSSEC",
			F:   func() error { return c.enableDNSSEC(zoneID) },
		}}, 1, nil
	}

	return nil, 0, nil
}

func (c *huaweicloudProvider) isDNSSECEnabled(zoneID string) (bool, error) {
	req := &model.ShowDnssecConfigRequest{ZoneId: zoneID}
	var resp *model.ShowDnssecConfigResponse
	var err error
	withRetry(func() error {
		resp, err = c.client.ShowDnssecConfig(req)
		return err
	})
	if err != nil {
		return false, err
	}
	return resp.Status != nil && *resp.Status == "ENABLE", nil
}

func (c *huaweicloudProvider) enableDNSSEC(zoneID string) error {
	req := &model.EnableDnssecConfigRequest{ZoneId: zoneID}
	var err error
	withRetry(func() error {
		_, err = c.client.EnableDnssecConfig(req)
		return err
	})
	return err
}

func (c *huaweicloudProvider) disableDNSSEC(zoneID string) error {
	req := &model.DisableDnssecConfigRequest{ZoneId: zoneID}
	var err error
	withRetry(func() error {
		_, err = c.client.DisableDnssecConfig(req)
		return err
	})
	return err
}
