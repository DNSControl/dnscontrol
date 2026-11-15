package normalize

import (
	"strings"

	"github.com/DNSControl/dnscontrol/v5/models"
	"github.com/DNSControl/dnscontrol/v5/pkg/nameutil"
)

// normalizeNameservers canonicalizes the names of a domain's nameservers.
// It returns an error for each name that is not a valid target.
func normalizeNameservers(domain *models.DomainConfig) (errs []error) {
	for _, ns := range domain.Nameservers {
		// NB(tlim): Like any target, NAMESERVER() is input by the user
		// as a shortname or a FQDN+dot.
		if err := checkTarget(ns.Name); err != nil {
			errs = append(errs, err)
		}
		// Unlike any other FQDN in this system, it is stored as a FQDN without the trailing dot.
		n := nameutil.ToFqdnWithDot(ns.Name, domain.Name)
		ns.Name = strings.TrimSuffix(n, ".")
	}
	return errs
}
