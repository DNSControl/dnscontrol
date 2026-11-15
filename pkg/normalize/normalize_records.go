package normalize

import (
	"github.com/DNSControl/dnscontrol/v5/models"
)

// normalizeRecords puts a domain's records into canonical form.
func normalizeRecords(domain *models.DomainConfig) {
	for _, rec := range domain.Records {
		if rec.TTL == 0 {
			rec.TTL = models.DefaultTTL
		}

		// Populate FQDN:
		rec.SetLabel(rec.GetLabel(), domain.Name)
	}
}
