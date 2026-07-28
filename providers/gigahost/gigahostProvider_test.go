package gigahost

import (
	"testing"

	"github.com/DNSControl/dnscontrol/v5/models"
)

func TestNativeToRecordConfig(t *testing.T) {
	dc, err := models.NewDomainConfig("example.com")
	if err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name       string
		record     record
		wantTarget string
	}{
		{"A", record{RecordName: "www", RecordType: "A", RecordValue: "192.0.2.1", RecordTTL: flexUint{Value: 300}}, "192.0.2.1"},
		{"MX", record{RecordName: "@", RecordType: "MX", RecordValue: "mail.example.net", RecordTTL: flexUint{Value: 300}, RecordPrio: flexUint{Value: 10}}, "10 mail.example.net."},
		{"TXT", record{RecordName: "@", RecordType: "TXT", RecordValue: "raw text", RecordTTL: flexUint{Value: 300}}, `"raw text"`},
		{"SRV", record{RecordName: "_sip._tcp", RecordType: "SRV", RecordValue: "1 2 5060 sip.example.net.", RecordTTL: flexUint{Value: 300}}, "1 2 5060 sip.example.net."},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			record := tc.record
			rc, err := nativeToRecordConfig(dc, &record)
			if err != nil {
				t.Fatal(err)
			}
			if got := rc.GetRDATA().String(); got != tc.wantTarget {
				t.Errorf("target = %q, want %q", got, tc.wantTarget)
			}
		})
	}
}
