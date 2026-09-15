package autodns

import (
	"testing"

	dnsv2 "codeberg.org/miekg/dns"

	"github.com/DNSControl/dnscontrol/v5/models"
)

func TestToRecordConfig(t *testing.T) {
	t.Parallel()

	dc := models.MustNewDomainConfig("example.com")
	tests := []struct {
		name     string
		native   *ResourceRecord
		wantType string
		wantData string
	}{
		{"A", &ResourceRecord{Name: "www", Type: "A", Value: "192.0.2.1", TTL: 300}, "A", "192.0.2.1"},
		{"MX", &ResourceRecord{Name: "www", Type: "MX", Value: "mail.example.net.", Pref: 10, TTL: 300}, "MX", "10 mail.example.net."},
		{"SRV", &ResourceRecord{Name: "_sip._tcp", Type: "SRV", Value: "2 443 service.example.net.", Pref: 1, TTL: 300}, "SRV", "1 2 443 service.example.net."},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			rc, err := toRecordConfig(dc, tc.native)
			if err != nil {
				t.Fatalf("toRecordConfig() error = %v", err)
			}
			if rc.Type != tc.wantType {
				t.Errorf("toRecordConfig() type = %q, want %q", rc.Type, tc.wantType)
			}
			if got := rc.GetRDATA().String(); got != tc.wantData {
				t.Errorf("toRecordConfig() data = %q, want %q", got, tc.wantData)
			}
		})
	}
}

// TestRecordsToNative covers the outbound direction. AutoDNS carries the MX
// preference in a dedicated "pref" field, so "value" must hold the bare target
// FQDN; emitting the full RDATA makes the gateway reject the whole zone update
// with EF020541 "The MX resource record value is invalid.".
func TestRecordsToNative(t *testing.T) {
	t.Parallel()

	dc := models.MustNewDomainConfig("example.com")
	tests := []struct {
		name      string
		rtype     uint16
		args      []any
		wantValue string
		wantPref  int32
	}{
		{"A", dnsv2.TypeA, []any{"192.0.2.1"}, "192.0.2.1", 0},
		{"MX", dnsv2.TypeMX, []any{uint16(10), "mail.example.net."}, "mail.example.net.", 10},
		{"CNAME", dnsv2.TypeCNAME, []any{"target.example.net."}, "target.example.net.", 0},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			rc, err := dc.NewRecordConfig("@", 300, tc.rtype, tc.args...)
			if err != nil {
				t.Fatalf("NewRecordConfig() error = %v", err)
			}

			_, _, native := recordsToNative(models.Records{rc})
			if len(native) != 1 {
				t.Fatalf("recordsToNative() returned %d records, want 1", len(native))
			}

			if got := native[0].Value; got != tc.wantValue {
				t.Errorf("Value = %q, want %q", got, tc.wantValue)
			}
			if got := native[0].Pref; got != tc.wantPref {
				t.Errorf("Pref = %d, want %d", got, tc.wantPref)
			}
		})
	}
}
