package autodns

import (
	"encoding/json"
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
		{"MX", &ResourceRecord{Name: "www", Type: "MX", Value: "mail.example.net.", Pref: ptrTo(int32(10)), TTL: 300}, "MX", "10 mail.example.net."},
		{"SRV", &ResourceRecord{Name: "_sip._tcp", Type: "SRV", Value: "2 443 service.example.net.", Pref: ptrTo(int32(1)), TTL: 300}, "SRV", "1 2 443 service.example.net."},
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
		wantPref  *int32
	}{
		{"A", dnsv2.TypeA, []any{"192.0.2.1"}, "192.0.2.1", nil},
		{"MX", dnsv2.TypeMX, []any{uint16(10), "mail.example.net."}, "mail.example.net.", ptrTo(int32(10))},
		{"MXZeroPreference", dnsv2.TypeMX, []any{uint16(0), "mail.example.net."}, "mail.example.net.", ptrTo(int32(0))},
		{"CNAME", dnsv2.TypeCNAME, []any{"target.example.net."}, "target.example.net.", nil},
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

			switch {
			case tc.wantPref == nil && native[0].Pref != nil:
				t.Errorf("Pref = %d, want unset", *native[0].Pref)
			case tc.wantPref != nil && native[0].Pref == nil:
				t.Errorf("Pref unset, want %d", *tc.wantPref)
			case tc.wantPref != nil && *native[0].Pref != *tc.wantPref:
				t.Errorf("Pref = %d, want %d", *native[0].Pref, *tc.wantPref)
			}
		})
	}
}

// TestRecordsToNativeMarshalsZeroPreference guards the wire format. A
// preference of 0 is valid for MX, but on a plain int32 field "omitempty"
// drops it, and AutoDNS then substitutes its own default preference -- so the
// pushed record silently differs from the config. Assert on the marshalled
// JSON rather than the struct, because that is where the field disappears.
func TestRecordsToNativeMarshalsZeroPreference(t *testing.T) {
	t.Parallel()

	dc := models.MustNewDomainConfig("example.com")

	rc, err := dc.NewRecordConfig("@", 300, dnsv2.TypeMX, uint16(0), "mail.example.net.")
	if err != nil {
		t.Fatalf("NewRecordConfig() error = %v", err)
	}

	_, _, native := recordsToNative(models.Records{rc})
	if len(native) != 1 {
		t.Fatalf("recordsToNative() returned %d records, want 1", len(native))
	}

	encoded, err := json.Marshal(native[0])
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}

	var sent map[string]any
	if err := json.Unmarshal(encoded, &sent); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}

	pref, ok := sent["pref"]
	if !ok {
		t.Fatalf("marshalled MX record has no pref field: %s", encoded)
	}
	if pref != float64(0) {
		t.Errorf("pref = %v, want 0", pref)
	}
}

func ptrTo[T any](v T) *T { return &v }
