package bind

import (
	"testing"
	"time"

	dnsv2 "codeberg.org/miekg/dns"
	"github.com/DNSControl/dnscontrol/v5/models"
)

func makeSOAForTest(t *testing.T, serial uint32) *models.RecordConfig {
	t.Helper()
	dc := &models.DomainConfig{Name: "example.com"}
	rc, err := dc.NewRecordConfig(
		"@", 3600, dnsv2.TypeSOA,
		"ns.example.com.", "hostmaster.example.com.",
		serial, uint32(3600), uint32(600), uint32(604800), uint32(1440),
	)
	if err != nil {
		t.Fatal(err)
	}
	return rc
}

// Test_updateSerialNumber verifies that the new SOA serial is based on the
// serial already published in the existing zone, not on the (default) desired
// SOA. See https://github.com/DNSControl/dnscontrol/issues/4840.
func Test_updateSerialNumber(t *testing.T) {
	day, _ := time.Parse("20060102", "20260901")
	nowFunc = func() time.Time { return day }
	defer func() { nowFunc = time.Now }()

	tests := []struct {
		name       string
		desired    uint32 // serial on the desired SOA (AddSoaIfMissing produces 1)
		existing   uint32 // serial on the existing/on-disk SOA (0 == no SOA on disk)
		forced     uint32
		wantSerial uint32
	}{
		// First push, nothing on disk: use today's default.
		{name: "fresh zone", desired: 1, existing: 0, forced: 0, wantSerial: 2026090100},
		// #4840: same-day re-push must advance past the on-disk serial rather
		// than regenerate YYYYMMDD00 from the default desired serial.
		{name: "same-day increment", desired: 1, existing: 2026090100, forced: 0, wantSerial: 2026090101},
		{name: "same-day increment again", desired: 1, existing: 2026090101, forced: 0, wantSerial: 2026090102},
		// On-disk serial from an earlier day: jump to today's default.
		{name: "advance to today", desired: 1, existing: 2026083100, forced: 0, wantSerial: 2026090100},
		// A forced serial (--bindserial) wins regardless of on-disk value.
		{name: "forced serial", desired: 1, existing: 2026090100, forced: 2026, wantSerial: 2026},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			desiredSOA := makeSOAForTest(t, tc.desired)
			desired := models.Records{desiredSOA}
			var existing models.Records
			if tc.existing != 0 {
				existing = models.Records{makeSOAForTest(t, tc.existing)}
			}

			updateSerialNumber(desired, existing, tc.forced)

			if got := desiredSOA.AsSOA().Serial; got != tc.wantSerial {
				t.Errorf("serial = %d; want %d", got, tc.wantSerial)
			}
		})
	}
}
