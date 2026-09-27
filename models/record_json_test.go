package models

import (
	"encoding/json"
	"testing"

	dnsv2 "codeberg.org/miekg/dns"
	"github.com/DNSControl/dnscontrol/v5/pkg/privatetypes"
)

// TestRecordConfigJSONRoundTrip verifies that marshaling a RecordConfig to
// JSON (the IR format used by `print-ir`) and unmarshaling it back produces
// an equivalent RDATA. This guards against
// https://github.com/DNSControl/dnscontrol/issues/4854, where .rdata was
// silently dropped on unmarshal because it is an unexported field typed as
// the dnsv2.RDATA interface. RDATA.String()/MyNewData() are used to read
// it back, which also covers types like SVCB whose RDATA embeds further
// interfaces (svcb.Pair) with no generic JSON-object representation.
func TestRecordConfigJSONRoundTrip(t *testing.T) {
	dc := MustNewDomainConfig("example.com")

	tests := []struct {
		name string
		rc   *RecordConfig
	}{
		{"A", dc.MustNewRecordConfig("@", 300, dnsv2.TypeA, "1.2.3.4")},
		{"MX", dc.MustNewRecordConfig("@", 300, dnsv2.TypeMX, 10, "mail.example.com.")},
		{"TXT", dc.MustNewRecordConfig("@", 300, dnsv2.TypeTXT, "hello world")},
		{"CAA", dc.MustNewRecordConfig("@", 300, dnsv2.TypeCAA, 0, "issue", "letsencrypt.org")},
		{"SRV", dc.MustNewRecordConfig("@", 300, dnsv2.TypeSRV, 10, 20, 5060, "sip.example.com.")},
		{"SVCB", dc.MustNewRecordConfig("@", 300, dnsv2.TypeSVCB, 1, ".", "alpn=h2,h3")},
		{"CLOUDFLAREAPI_SINGLE_REDIRECT", dc.MustNewRecordConfig("@", 300, privatetypes.TypeCLOUDFLAREAPISINGLEREDIRECT, "name", 301, "when", "then")},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			original := tc.rc

			data, err := json.Marshal(original)
			if err != nil {
				t.Fatalf("Marshal: %v", err)
			}

			var got RecordConfig
			if err := json.Unmarshal(data, &got); err != nil {
				t.Fatalf("Unmarshal: %v", err)
			}

			if got.GetRDATA() == nil {
				t.Fatalf("RDATA is nil after round-trip; want %+v", original.GetRDATA())
			}

			wantStr := original.GetRDATA().String()
			gotStr := got.GetRDATA().String()
			if gotStr != wantStr {
				t.Errorf("RDATA.String() after round-trip = %q, want %q", gotStr, wantStr)
			}

			if got.TypeNum != original.TypeNum {
				t.Errorf("TypeNum after round-trip = %d, want %d", got.TypeNum, original.TypeNum)
			}
			if got.Type != original.Type {
				t.Errorf("Type after round-trip = %q, want %q", got.Type, original.Type)
			}

			// Marshaling the round-tripped record should reproduce the same JSON.
			data2, err := json.Marshal(&got)
			if err != nil {
				t.Fatalf("re-Marshal: %v", err)
			}
			if string(data2) != string(data) {
				t.Errorf("re-marshaled JSON differs:\n got=%s\nwant=%s", data2, data)
			}
		})
	}
}

// TestRecordConfigJSONUnmarshalMissingRDATA verifies that a record missing
// its "rdata" object fails to unmarshal with a clear error instead of
// silently producing a record with nil RDATA that panics later.
func TestRecordConfigJSONUnmarshalMissingRDATA(t *testing.T) {
	data := []byte(`{"typenum":1,"type":"A","name":"@","ttl":300}`)

	var rc RecordConfig
	err := json.Unmarshal(data, &rc)
	if err == nil {
		t.Fatalf("expected an error for a record missing rdata, got nil (RDATA=%+v)", rc.GetRDATA())
	}
}
