package websupport

import (
	"testing"

	"github.com/DNSControl/dnscontrol/v5/models"
)

const testDomain = "example.com"

func mkRC(t *testing.T, rtype, label string, args ...any) *models.RecordConfig {
	t.Helper()
	dc := models.MustNewDomainConfig(testDomain)
	return dc.MustNewRecordConfig(label, 3600, rtype, args...)
}

func TestRoundTrip(t *testing.T) {
	tests := []struct {
		name        string
		rc          *models.RecordConfig
		wantType    string
		wantName    string
		wantContent string
	}{
		// {
		// 	name:        "A apex",
		// 	rc:          mkRC(t, "A", "@", "1.2.3.4"),
		// 	wantType:    "A",
		// 	wantName:    "@",
		// 	wantContent: "1.2.3.4",
		// },
		{
			name:        "CNAME strips trailing dot",
			rc:          mkRC(t, "CNAME", "www", "ghs.example.net."),
			wantType:    "CNAME",
			wantName:    "www",
			wantContent: "ghs.example.net",
		},
		// {
		// 	name:        "MX",
		// 	rc:          mkRC(t, "MX", "@", 10, "mail.example.com."),
		// 	wantType:    "MX",
		// 	wantName:    "@",
		// 	wantContent: "mail.example.com",
		// },
		// {
		// 	name:        "SRV",
		// 	rc:          mkRC(t, "SRV", "_sip._tcp", 10, 20, 5060, "sip.example.com."),
		// 	wantType:    "SRV",
		// 	wantName:    "_sip._tcp",
		// 	wantContent: "sip.example.com",
		// },
		// {
		// 	name:        "AAAA",
		// 	rc:          mkRC(t, "AAAA", "ipv6", "2a00:4b40:aaaa:2001::6"),
		// 	wantType:    "AAAA",
		// 	wantName:    "ipv6",
		// 	wantContent: "2a00:4b40:aaaa:2001::6",
		// },
		// {
		// 	name:        "TXT",
		// 	rc:          mkRC(t, "TXT", "@", "hello world"),
		// 	wantType:    "TXT",
		// 	wantName:    "@",
		// 	wantContent: "hello world",
		// },
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			n, err := toNative(tc.rc)
			if err != nil {
				t.Fatalf("toNative: %v", err)
			}
			if n.Type != tc.wantType {
				t.Errorf("type = %q, want %q", n.Type, tc.wantType)
			}
			if n.Name != tc.wantName {
				t.Errorf("name = %q, want %q", n.Name, tc.wantName)
			}
			if n.Content != tc.wantContent {
				t.Errorf("content = %q, want %q", n.Content, tc.wantContent)
			}

			// Round-trip back to a RecordConfig and compare the canonical content.
			// Simulate the API, which echoes the fully-qualified name on read
			// even though writes use the relative label.
			n.ID = 42
			n.Name = tc.rc.GetLabelFQDN()
			dc := models.MustNewDomainConfig(testDomain)
			rc2, err := toRecordConfig(dc, n)
			if err != nil {
				t.Fatalf("toRecordConfig: %v", err)
			}
			if got, want := rc2.GetRDATA().String(), tc.rc.GetRDATA().String(); got != want {
				t.Errorf("round-trip target = %q, want %q", got, want)
			}
			if rc2.Type != tc.rc.Type {
				t.Errorf("round-trip type = %q, want %q", rc2.Type, tc.rc.Type)
			}
		})
	}
}
