package normalize

import (
	"fmt"

	"github.com/DNSControl/dnscontrol/v5/models"
	"github.com/DNSControl/dnscontrol/v5/pkg/providers"
)

// auditRecords asks each of a domain's DNS providers whether there are any
// records it can't handle.
func auditRecords(domain *models.DomainConfig) (errs []error) {
	for _, provider := range domain.DNSProviderInstances {
		if provider.ProviderType == "-" {
			// "-" indicates that we don't yet know who the provider type
			// is.  This is probably due to the fact that `dnscontrol
			// check` doesn't read creds.json, which is where the TYPE is
			// set.  We will skip this test in this instance.  Later if
			// `dnscontrol preview` or `push` is used, the full check will
			// be performed.
			continue
		}
		if es := providers.AuditRecords(provider.ProviderType, domain.Records); len(es) != 0 {
			for _, e := range es {
				errs = append(errs, fmt.Errorf("%s rejects domain %s: %w", provider.ProviderType, domain.Name, e))
			}
		}
	}
	return errs
}
