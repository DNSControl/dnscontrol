package normalize

import (
	"fmt"

	"github.com/DNSControl/dnscontrol/v5/models"
)

func checkMultipleSOAs(dc *models.DomainConfig) (errs []error) {
	soas := map[string]bool{}
	for _, r := range dc.Records {
		if r.Type == "SOA" {
			if soas[r.GetLabel()] {
				errs = append(errs, fmt.Errorf("%s: cannot have multiple SOAs with same name: %s", r.FilePos, r.GetLabelFQDN()))
			}
			soas[r.GetLabel()] = true
		}
	}
	return
}
