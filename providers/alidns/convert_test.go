package alidns

import (
	"testing"

	"github.com/DNSControl/dnscontrol/v5/models"
	alinative "github.com/aliyun/alibaba-cloud-sdk-go/services/alidns"
)

func TestNativeToRecord(t *testing.T) {
	t.Parallel()

	dc := models.MustNewDomainConfig("example.com")
	tests := []struct {
		name         string
		native       *alinative.Record
		wantType     string
		wantContent  string
		wantPriority int64
	}{
		{"A", &alinative.Record{RR: "www", Type: "A", Value: "192.0.2.1", TTL: 300}, "A", "192.0.2.1", 0},
		{"MX", &alinative.Record{RR: "www", Type: "MX", Value: "mail.example.net", TTL: 300, Priority: 10}, "MX", "mail.example.net.", 10},
		{"SRV", &alinative.Record{RR: "_sip._tcp", Type: "SRV", Value: "1 2 443 service.example.net", TTL: 300}, "SRV", "1 2 443 service.example.net.", 1},
		{"CAA", &alinative.Record{RR: "www", Type: "CAA", Value: `0 issue "letsencrypt.org"`, TTL: 300}, "CAA", `0 issue "letsencrypt.org"`, 0},
		{"TXT", &alinative.Record{RR: "www", Type: "TXT", Value: "hello world", TTL: 300}, "TXT", "hello world", 0},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			rc, err := nativeToRecord(tc.native, dc)
			if err != nil {
				t.Fatalf("nativeToRecord() error = %v", err)
			}
			if rc.Type != tc.wantType {
				t.Errorf("nativeToRecord() type = %q, want %q", rc.Type, tc.wantType)
			}
			if got := recordToNativeContent(rc); got != tc.wantContent {
				t.Errorf("recordToNativeContent() = %q, want %q", got, tc.wantContent)
			}
			if got := recordToNativePriority(rc); got != tc.wantPriority {
				t.Errorf("recordToNativePriority() = %d, want %d", got, tc.wantPriority)
			}
		})
	}
}
