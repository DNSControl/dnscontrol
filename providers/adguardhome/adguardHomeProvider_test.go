package adguardhome

import (
	"reflect"
	"testing"

	"github.com/DNSControl/dnscontrol/v5/models"
)

func TestRewriteEntryConversions(t *testing.T) {
	t.Parallel()

	dc := models.MustNewDomainConfig("example.com")
	tests := []struct {
		name        string
		entry       rewriteEntry
		wantType    string
		wantRewrite rewriteEntry
	}{
		{"A", rewriteEntry{"www.example.com", "192.0.2.1"}, "A", rewriteEntry{"www.example.com", "192.0.2.1"}},
		{"AAAA", rewriteEntry{"www.example.com", "2001:db8::1"}, "AAAA", rewriteEntry{"www.example.com", "2001:db8::1"}},
		{"A passthrough", rewriteEntry{"www.example.com", "A"}, "ADGUARDHOME_A_PASSTHROUGH", rewriteEntry{"www.example.com", "A"}},
		{"AAAA passthrough", rewriteEntry{"www.example.com", "AAAA"}, "ADGUARDHOME_AAAA_PASSTHROUGH", rewriteEntry{"www.example.com", "AAAA"}},
		{"CNAME", rewriteEntry{"www.example.com", "target.example.com"}, "CNAME", rewriteEntry{"www.example.com", "target"}},
		{"ALIAS", rewriteEntry{"example.com", "target.example.net"}, "ALIAS", rewriteEntry{"example.com", "target.example.net."}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			rc, err := toRc(dc, tc.entry)
			if err != nil {
				t.Fatalf("toRc() error = %v", err)
			}
			if rc.Type != tc.wantType {
				t.Errorf("toRc() type = %q, want %q", rc.Type, tc.wantType)
			}
			if rc.Original != tc.entry {
				t.Errorf("toRc() original = %#v, want %#v", rc.Original, tc.entry)
			}

			got, err := toRewriteEntry(dc, rc)
			if err != nil {
				t.Fatalf("toRewriteEntry() error = %v", err)
			}
			if !reflect.DeepEqual(got, tc.wantRewrite) {
				t.Errorf("toRewriteEntry() = %#v, want %#v", got, tc.wantRewrite)
			}
		})
	}
}
