package bind

import (
	"strings"

	"github.com/DNSControl/dnscontrol/v4/models"
	"github.com/DNSControl/dnscontrol/v4/pkg/soautil"
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

	soaRec := models.RecordConfig{
		Type: "SOA",
		TTL:  firstNonZero(defaultSoaValues.TTL, models.DefaultTTL),
	}
	err := soaRec.SetTargetSOA(
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
	soaRec.SetLabel("@", dc.Name)
	soaRec.FixRD(dc.Name)

	dc.Records = append(dc.Records, &soaRec)
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
