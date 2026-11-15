package normalize

import (
	"fmt"

	"github.com/DNSControl/dnscontrol/v5/models"
)

func checkDuplicates(records models.Records, extra func(*models.RecordConfig) string) (errs []error) {
	seen := make(map[string]*models.RecordConfig)
	for _, r := range records {
		diffable := recordIdentityString(r, extra)

		if seen[diffable] != nil {
			errs = append(errs, fmt.Errorf("exact duplicate record found: %s", diffable))
		}
		seen[diffable] = r
	}
	return errs
}
