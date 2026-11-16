package normalize

import (
	"testing"

	dnsv2 "codeberg.org/miekg/dns"
	"github.com/DNSControl/dnscontrol/v5/models"
)

func TestTransforms(t *testing.T) {
	tests := []struct {
		givenIP         string
		expectedRecords []string
	}{
		{"0.0.5.5", []string{"2.0.5.5"}},
		{"3.0.5.5", []string{"5.5.5.5"}},
		{"7.0.5.5", []string{"9.9.9.9", "10.10.10.10"}},
	}
	const transform = "0.0.0.0~1.0.0.0~2.0.0.0~;   3.0.0.0~4.0.0.0~~5.5.5.5; 7.0.0.0~8.0.0.0~~9.9.9.9,10.10.10.10"
	for i, test := range tests {
		dc := models.MustNewDomainConfig("example.tld")
		rc1 := dc.MustNewRecordConfig("f", 0, dnsv2.TypeA, test.givenIP)
		rc1.Metadata = map[string]string{"transform": transform}
		dc.AddRecordConfig(rc1)

		err := applyRecordTransforms(dc)
		if err != nil {
			t.Errorf("error on test %d: %s", i, err)
			continue
		}
		if len(dc.Records) != len(test.expectedRecords) {
			t.Errorf("test %d: expect %d records but found %d", i, len(test.expectedRecords), len(dc.Records))
			continue
		}
		for r, rec := range dc.Records {
			got := rec.AsA().Addr.String()
			if got != test.expectedRecords[r] {
				t.Errorf("test %d at index %d: records don't match. Expect %s but found %s.", i, r, test.expectedRecords[r], got)
				continue
			}
		}
	}
}
