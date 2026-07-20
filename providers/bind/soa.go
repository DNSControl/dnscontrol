package bind

import (
	"strings"

	"github.com/DNSControl/dnscontrol/v4/models"
	"github.com/DNSControl/dnscontrol/v4/pkg/soautil"
)

func makeSoa(origin string, defSoa *SoaDefaults, existing, desired *models.RecordConfig) (*models.RecordConfig, uint32) {
	// Create a SOA record.  Take data from desired, existing, default,
	// or hardcoded defaults.
	soaRec := models.RecordConfig{}
	soaRec.SetLabel("@", origin)

	if defSoa == nil {
		defSoa = &SoaDefaults{}
	}
	if existing == nil {
		existing = &models.RecordConfig{}
	}

	if desired == nil {
		desired = &models.RecordConfig{}
	}

	soaMail := firstNonNull(desired.SoaMbox, existing.SoaMbox, defSoa.Mbox, "DEFAULT_NOT_SET.")
	if strings.Contains(soaMail, "@") {
		soaMail = soautil.RFC5322MailToBind(soaMail)
	}

	soaRec.TTL = firstNonZero(desired.TTL, defSoa.TTL, existing.TTL, models.DefaultTTL)
	err := soaRec.SetTargetSOA(
		firstNonNull(desired.GetTargetField(), existing.GetTargetField(), defSoa.Ns, "DEFAULT_NOT_SET."),
		soaMail,
		firstNonZero(desired.SoaSerial, existing.SoaSerial, defSoa.Serial, 1),
		firstNonZero(desired.SoaRefresh, existing.SoaRefresh, defSoa.Refresh, 3600),
		firstNonZero(desired.SoaRetry, existing.SoaRetry, defSoa.Retry, 600),
		firstNonZero(desired.SoaExpire, existing.SoaExpire, defSoa.Expire, 604800),
		firstNonZero(desired.SoaMinttl, existing.SoaMinttl, defSoa.Minttl, 1440),
	)
	if err != nil {
		panic(err) // Should never happen.
	}

	return &soaRec, generateSerial(soaRec.SoaSerial)
}

func firstNonNull(items ...string) string {
	for _, item := range items {
		if item != "" {
			return item
		}
	}
	return "FAIL"
}

func firstNonZero(items ...uint32) uint32 {
	for _, item := range items {
		if item != 0 {
			return item
		}
	}
	return 999
}
