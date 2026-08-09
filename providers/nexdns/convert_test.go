package nexdns

import (
	"testing"

	"github.com/DNSControl/dnscontrol/v5/models"
)

const testOrigin = "example.com"

// TestRecordRoundTrip reads a record the way the API returns it, converts it
// back into a request and checks that nothing was lost. A response carries the
// assembled rdata while a request carries the primary value plus separate
// fields, so a mistake in either direction shows up as a change DNSControl
// would keep proposing for a record that already matches.
func TestRecordRoundTrip(t *testing.T) {
	tests := []struct {
		name    string
		stored  apiRecord
		content string
		verify  func(t *testing.T, req recordRequest)
	}{
		{
			name:    "A",
			stored:  apiRecord{ID: "r1", Name: "www", Type: "A", Content: "203.0.113.10", TTL: 300},
			content: "203.0.113.10",
		},
		{
			name:    "CNAME keeps its trailing dot",
			stored:  apiRecord{ID: "r2", Name: "alias", Type: "CNAME", Content: "www.example.net.", TTL: 300},
			content: "www.example.net.",
		},
		{
			name:    "MX splits off the preference",
			stored:  apiRecord{ID: "r3", Name: "@", Type: "MX", Content: "10 mail.example.net.", TTL: 300},
			content: "mail.example.net.",
			verify: func(t *testing.T, req recordRequest) {
				if req.Priority == nil || *req.Priority != 10 {
					t.Errorf("priority = %v, want 10", req.Priority)
				}
			},
		},
		{
			name:    "null MX keeps its bare dot",
			stored:  apiRecord{ID: "r4", Name: "@", Type: "MX", Content: "0 .", TTL: 300},
			content: ".",
			verify: func(t *testing.T, req recordRequest) {
				if req.Priority == nil || *req.Priority != 0 {
					t.Errorf("priority = %v, want 0", req.Priority)
				}
			},
		},
		{
			name:    "SRV splits off priority, weight and port",
			stored:  apiRecord{ID: "r5", Name: "_sip._tcp", Type: "SRV", Content: "10 60 5060 sip.example.net.", TTL: 300},
			content: "sip.example.net.",
			verify: func(t *testing.T, req recordRequest) {
				if req.Priority == nil || req.Weight == nil || req.Port == nil {
					t.Fatalf("srv fields = %v %v %v, want all three", req.Priority, req.Weight, req.Port)
				}
				if *req.Priority != 10 || *req.Weight != 60 || *req.Port != 5060 {
					t.Errorf("srv fields = %d %d %d, want 10 60 5060", *req.Priority, *req.Weight, *req.Port)
				}
			},
		},
		{
			name:    "CAA splits off the flags and the tag",
			stored:  apiRecord{ID: "r6", Name: "@", Type: "CAA", Content: `0 issue "letsencrypt.org"`, TTL: 300},
			content: "letsencrypt.org",
			verify: func(t *testing.T, req recordRequest) {
				if req.Tag != "issue" {
					t.Errorf("tag = %q, want issue", req.Tag)
				}
				if req.Flags == nil || *req.Flags != 0 {
					t.Errorf("flags = %v, want 0", req.Flags)
				}
			},
		},
		{
			name:    "DS sends the bare digest",
			stored:  apiRecord{ID: "r7", Name: "child", Type: "DS", Content: "12345 13 2 2bb183af5f22588179a53b0a98631fad1a292118bd8e9ba3d3a1e01f2e1bd9c1", TTL: 300},
			content: "2BB183AF5F22588179A53B0A98631FAD1A292118BD8E9BA3D3A1E01F2E1BD9C1",
			verify: func(t *testing.T, req recordRequest) {
				if req.KeyTag == nil || *req.KeyTag != 12345 {
					t.Errorf("keytag = %v, want 12345", req.KeyTag)
				}
				if req.Algorithm == nil || *req.Algorithm != 13 {
					t.Errorf("algorithm = %v, want 13", req.Algorithm)
				}
				if req.DigestType == nil || *req.DigestType != 2 {
					t.Errorf("digest_type = %v, want 2", req.DigestType)
				}
			},
		},
		{
			name:    "TLSA sends the bare certificate data",
			stored:  apiRecord{ID: "r8", Name: "_443._tcp", Type: "TLSA", Content: "3 1 1 abcdef0123456789", TTL: 300},
			content: "ABCDEF0123456789",
			verify: func(t *testing.T, req recordRequest) {
				if req.Usage == nil || req.Selector == nil || req.MatchingType == nil {
					t.Fatalf("tlsa fields = %v %v %v, want all three", req.Usage, req.Selector, req.MatchingType)
				}
				if *req.Usage != 3 || *req.Selector != 1 || *req.MatchingType != 1 {
					t.Errorf("tlsa fields = %d %d %d, want 3 1 1", *req.Usage, *req.Selector, *req.MatchingType)
				}
			},
		},
		{
			name:    "TXT is unquoted",
			stored:  apiRecord{ID: "r9", Name: "@", Type: "TXT", Content: `"v=spf1 -all"`, TTL: 300},
			content: "v=spf1 -all",
		},
		// The API returns TXT in presentation form but accepts it verbatim, so
		// a value whose escapes are not decoded on the way in is re-escaped on
		// the way out. Each run then doubles them, and the record drifts away
		// from what its owner wrote without anyone editing it.
		{
			name:    "TXT unescapes a backslash",
			stored:  apiRecord{ID: "r10", Name: "bs", Type: "TXT", Content: `"1back\\slash"`, TTL: 300},
			content: `1back\slash`,
		},
		{
			name:    "TXT unescapes an interior quote",
			stored:  apiRecord{ID: "r11", Name: "dq", Type: "TXT", Content: `"in\"side"`, TTL: 300},
			content: `in"side`,
		},
		{
			name:    "TXT joins the chunks of a value over 255 octets",
			stored:  apiRecord{ID: "r12", Name: "long", Type: "TXT", Content: `"first" "second"`, TTL: 300},
			content: "firstsecond",
		},
		{
			name:    "TXT accepts a value that carries no quotes at all",
			stored:  apiRecord{ID: "r13", Name: "bare", Type: "TXT", Content: "plain", TTL: 300},
			content: "plain",
		},
	}

	dc := models.MustNewDomainConfig(testOrigin)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rc, err := toRecordConfig(dc, tt.stored)
			if err != nil {
				t.Fatalf("toRecordConfig() error = %v", err)
			}
			if rc.TTL != uint32(tt.stored.TTL) {
				t.Errorf("TTL = %d, want %d", rc.TTL, tt.stored.TTL)
			}

			req := fromRecordConfig(rc)
			if req.Name != tt.stored.Name {
				t.Errorf("name = %q, want %q", req.Name, tt.stored.Name)
			}
			if req.Type != tt.stored.Type {
				t.Errorf("type = %q, want %q", req.Type, tt.stored.Type)
			}
			if req.Content != tt.content {
				t.Errorf("content = %q, want %q", req.Content, tt.content)
			}
			if tt.verify != nil {
				tt.verify(t, req)
			}
		})
	}
}

func TestFilterApexNS(t *testing.T) {
	nameservers, err := models.ToNameservers([]string{"ns1.example.net", "ns2.example.net"})
	if err != nil {
		t.Fatalf("ToNameservers() error = %v", err)
	}

	dc := &models.DomainConfig{
		Name:        testOrigin,
		Nameservers: nameservers,
		Records: models.Records{
			makeRecord(t, "NS", "@", "ns1.example.net."),
			makeRecord(t, "NS", "@", "ns9.example.org."),
			makeRecord(t, "NS", "sub", "ns1.example.org."),
			makeRecord(t, "A", "www", "203.0.113.10"),
		},
	}

	filterApexNS(dc)

	if len(dc.Records) != 2 {
		t.Fatalf("filterApexNS() left %d records, want 2: %v", len(dc.Records), dc.Records)
	}
	for _, rec := range dc.Records {
		if rec.Type == "NS" && rec.GetLabel() == apexLabel {
			t.Errorf("an apex NS record survived: %v", rec)
		}
	}
}

func makeRecord(t *testing.T, rtype, label, target string) *models.RecordConfig {
	t.Helper()

	dc := models.MustNewDomainConfig(testOrigin)
	rc := dc.MustNewRecordConfigParse(dc.LabelFromFQDNNoDot(label), 300, rtype, target)
	return rc
}
