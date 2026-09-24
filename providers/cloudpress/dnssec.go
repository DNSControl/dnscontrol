package cloudpress

import "github.com/DNSControl/dnscontrol/v4/models"

// getDNSSECCorrections returns corrections to align the zone's DNSSEC state with
// the AutoDNSSEC setting from the domain config.
func (c *cloudpressProvider) getDNSSECCorrections(dc *models.DomainConfig, z *zone) []*models.Correction {
	if z.DNSSEC && dc.AutoDNSSEC == "off" {
		return []*models.Correction{
			{Msg: "Disable DNSSEC", F: func() error {
				return c.setDNSSEC(z.ID, false)
			}},
		}
	}

	if !z.DNSSEC && dc.AutoDNSSEC == "on" {
		return []*models.Correction{
			{Msg: "Enable DNSSEC", F: func() error {
				return c.setDNSSEC(z.ID, true)
			}},
		}
	}

	return nil
}
