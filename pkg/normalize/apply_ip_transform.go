package normalize

import (
	"github.com/DNSControl/dnscontrol/v5/models"
	"github.com/DNSControl/dnscontrol/v5/pkg/transform"
)

func applyRecordTransforms(domain *models.DomainConfig) error {
	for _, rec := range domain.Records {
		if rec.Type != "A" {
			continue
		}
		tt, ok := rec.Metadata["transform"]
		if !ok {
			continue
		}
		table, err := transform.DecodeTransformTable(tt)
		if err != nil {
			return err
		}
		ip := rec.GetTargetIP()
		newIPs, err := transform.IPToList(rec.GetTargetIP(), table)
		if err != nil {
			return err
		}
		for i, newIP := range newIPs {
			if i == 0 && newIP.Compare(ip) != 0 {
				// replace target of first record if different
				if err := rec.SetTargetIP(newIP); err != nil {
					return err
				}
			} else if i > 0 {
				// any additional ips need identical records with the alternate ip added to the domain
				cpy, err := rec.Copy()
				if err != nil {
					return err
				}
				if err := cpy.SetTargetIP(newIP); err != nil {
					return err
				}
				domain.Records = append(domain.Records, cpy)
			}
		}
	}
	return nil
}
