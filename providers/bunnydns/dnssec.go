package bunnydns

import "github.com/DNSControl/dnscontrol/v4/models"

func (b *bunnydnsProvider) getDNSSECCorrections(dc *models.DomainConfig, zone *zone) ([]*models.Correction, error) {
	if zone.HasDNSSEC && dc.AutoDNSSEC == "off" {
		return []*models.Correction{
			{Msg: "Disable DNSSEC", F: func() error {
				return b.disableDNSSEC(zone.ID)
			}},
		}, nil
	}

	if !zone.HasDNSSEC && dc.AutoDNSSEC == "on" {
		return []*models.Correction{
			{Msg: "Enable DNSSEC", F: func() error {
				return b.enableDNSSEC(zone.ID)
			}},
		}, nil
	}

	return []*models.Correction{}, nil
}
