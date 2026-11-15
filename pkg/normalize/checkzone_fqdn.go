package normalize

import (
	"fmt"
	"strings"

	"github.com/DNSControl/dnscontrol/v5/models"
)

// checkFQDNs checks that every record's FQDN is in the domain.
func checkFQDNs(d *models.DomainConfig) (errs []error) {
	for _, r := range d.Records {
		if r.NameFQDN == "" || !strings.HasSuffix(r.NameFQDN, d.Name) {
			errs = append(errs, fmt.Errorf("record named '%s' does not have correct FQDN for domain '%s'. FQDN: %s", r.Name, d.Name, r.NameFQDN))
		}
	}
	return errs
}
