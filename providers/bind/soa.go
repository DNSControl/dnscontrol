package bind

import (
	"strings"

	dnsv2 "codeberg.org/miekg/dns"
	"github.com/DNSControl/dnscontrol/v5/models"
	"github.com/DNSControl/dnscontrol/v5/pkg/soautil"
)

func AddSoaIfMissing(dc *models.DomainConfig, defaultSoaValues SoaDefaults) {
	// Exit if SOA already exists.
	for _, rec := range dc.Records {
		if rec.Type == "SOA" {
			return
		}
	}

	soaMail := firstNonNull(defaultSoaValues.Mbox, "default_not_set.")
	if strings.Contains(soaMail, "@") {
		soaMail = soautil.RFC5322MailToBind(soaMail)
	}

	soaRec, err := dc.NewRecordConfig(
		"@",
		firstNonZero(defaultSoaValues.TTL, models.DefaultTTL),
		dnsv2.TypeSOA,
		firstNonNull(defaultSoaValues.Ns, "default_not_set."),
		soaMail,
		firstNonZero(defaultSoaValues.Serial, 1),
		firstNonZero(defaultSoaValues.Refresh, 3600),
		firstNonZero(defaultSoaValues.Retry, 600),
		firstNonZero(defaultSoaValues.Expire, 604800),
		firstNonZero(defaultSoaValues.Minttl, 1440),
	)
	if err != nil {
		panic(err) // Should never happen.
	}

	dc.AddRecordConfig(soaRec)
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
