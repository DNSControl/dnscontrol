package mikrotik

import (
	"testing"

	"github.com/DNSControl/dnscontrol/v4/models"
)

// --- Duration parsing/formatting ---

func TestParseMikrotikDuration(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    uint32
		wantErr bool
	}{
		{"empty", "", 0, false},
		{"seconds_only", "30s", 30, false},
		{"minutes_only", "5m", 300, false},
		{"hours_only", "2h", 7200, false},
		{"days_only", "1d", 86400, false},
		{"weeks_only", "1w", 604800, false},
		{"composite_wdhms", "1w2d3h4m5s", 788645, false},
		{"composite_dh", "1d12h", 129600, false},
		{"hms_format", "01:30:00", 5400, false},
		{"hms_with_days", "2d05:30:15", 192615, false},
		{"zero_seconds", "0s", 0, false},
		{"invalid", "bogus", 0, true},
		{"negative_not_supported", "-1d", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseMikrotikDuration(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Errorf("parseMikrotikDuration(%q) expected error, got %d", tt.input, got)
				}
				return
			}
			if err != nil {
				t.Errorf("parseMikrotikDuration(%q) unexpected error: %v", tt.input, err)
				return
			}
			if got != tt.want {
				t.Errorf("parseMikrotikDuration(%q) = %d, want %d", tt.input, got, tt.want)
			}
		})
	}
}

func TestFormatMikrotikDuration(t *testing.T) {
	tests := []struct {
		name  string
		input uint32
		want  string
	}{
		{"zero", 0, "0s"},
		{"seconds", 45, "45s"},
		{"minutes", 300, "5m"},
		{"hours", 7200, "2h"},
		{"days", 86400, "1d"},
		{"weeks", 604800, "1w"},
		{"composite", 788645, "1w2d3h4m5s"},
		{"day_plus_half", 129600, "1d12h"},
		{"minutes_and_seconds", 90, "1m30s"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatMikrotikDuration(tt.input)
			if got != tt.want {
				t.Errorf("formatMikrotikDuration(%d) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestDurationRoundTrip(t *testing.T) {
	values := []uint32{0, 1, 60, 3600, 86400, 604800, 788645, 90, 129600}
	for _, v := range values {
		s := formatMikrotikDuration(v)
		got, err := parseMikrotikDuration(s)
		if err != nil {
			t.Errorf("round-trip %d -> %q -> parse error: %v", v, s, err)
			continue
		}
		if got != v {
			t.Errorf("round-trip %d -> %q -> %d, want %d", v, s, got, v)
		}
	}
}

// --- Helper functions ---

func TestEnsureTrailingDot(t *testing.T) {
	tests := []struct{ input, want string }{
		{"", ""},
		{"example.com", "example.com."},
		{"example.com.", "example.com."},
		{"host", "host."},
	}
	for _, tt := range tests {
		if got := ensureTrailingDot(tt.input); got != tt.want {
			t.Errorf("ensureTrailingDot(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestStripTrailingDot(t *testing.T) {
	tests := []struct{ input, want string }{
		{"", ""},
		{"example.com.", "example.com"},
		{"example.com", "example.com"},
		{"host.", "host"},
	}
	for _, tt := range tests {
		if got := stripTrailingDot(tt.input); got != tt.want {
			t.Errorf("stripTrailingDot(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

// --- nativeToRecords ---

func TestNativeToRecords_A(t *testing.T) {
	nr := dnsStaticRecord{
		ID:      "*1",
		Name:    "host.example.com",
		Type:    "A",
		Address: "192.168.1.1",
		TTL:     "1d",
	}
	rcs, err := nativeToRecords(nr, "example.com")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(rcs) != 1 {
		t.Fatalf("expected 1 record, got %d", len(rcs))
	}
	rc := rcs[0]
	assertStr(t, "Type", rc.Type, "A")
	assertStr(t, "Label", rc.GetLabel(), "host")
	assertStr(t, "Target", rc.GetTargetIP().String(), "192.168.1.1")
	assertUint32(t, "TTL", rc.TTL, 86400)
}

func TestNativeToRecords_AAAA(t *testing.T) {
	nr := dnsStaticRecord{
		Name: "v6.example.com", Type: "AAAA", Address: "2001:db8::1", TTL: "1h",
	}
	rcs, err := nativeToRecords(nr, "example.com")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	assertStr(t, "Type", rcs[0].Type, "AAAA")
	assertStr(t, "Target", rcs[0].GetTargetIP().String(), "2001:db8::1")
}

func TestNativeToRecords_CNAME(t *testing.T) {
	nr := dnsStaticRecord{
		Name: "alias.example.com", Type: "CNAME", CName: "target.example.com", TTL: "5m",
	}
	rcs, err := nativeToRecords(nr, "example.com")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	rc := rcs[0]
	assertStr(t, "Type", rc.Type, "CNAME")
	assertStr(t, "Target", rc.GetTargetField(), "target.example.com.")
	assertUint32(t, "TTL", rc.TTL, 300)
}

func TestNativeToRecords_MX(t *testing.T) {
	nr := dnsStaticRecord{
		Name: "example.com", Type: "MX", MxExchange: "mail.example.com", MxPreference: "10", TTL: "1d",
	}
	rcs, err := nativeToRecords(nr, "example.com")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	rc := rcs[0]
	assertStr(t, "Type", rc.Type, "MX")
	if rc.MxPreference != 10 {
		t.Errorf("MxPreference = %d, want 10", rc.MxPreference)
	}
	assertStr(t, "Target", rc.GetTargetField(), "mail.example.com.")
}

func TestNativeToRecords_NS(t *testing.T) {
	nr := dnsStaticRecord{
		Name: "sub.example.com", Type: "NS", NS: "ns1.example.com", TTL: "1d",
	}
	rcs, err := nativeToRecords(nr, "example.com")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	rc := rcs[0]
	assertStr(t, "Type", rc.Type, "NS")
	assertStr(t, "Target", rc.GetTargetField(), "ns1.example.com.")
}

func TestNativeToRecords_SRV(t *testing.T) {
	nr := dnsStaticRecord{
		Name: "_sip._tcp.example.com", Type: "SRV",
		SrvTarget: "sipserver.example.com", SrvPort: "5060", SrvPriority: "10", SrvWeight: "20",
		TTL: "1h",
	}
	rcs, err := nativeToRecords(nr, "example.com")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	rc := rcs[0]
	assertStr(t, "Type", rc.Type, "SRV")
	if rc.SrvPriority != 10 {
		t.Errorf("SrvPriority = %d, want 10", rc.SrvPriority)
	}
	if rc.SrvWeight != 20 {
		t.Errorf("SrvWeight = %d, want 20", rc.SrvWeight)
	}
	if rc.SrvPort != 5060 {
		t.Errorf("SrvPort = %d, want 5060", rc.SrvPort)
	}
	assertStr(t, "Target", rc.GetTargetField(), "sipserver.example.com.")
}

func TestNativeToRecords_TXT(t *testing.T) {
	nr := dnsStaticRecord{
		Name: "example.com", Type: "TXT", Text: "v=spf1 include:example.com ~all", TTL: "1d",
	}
	rcs, err := nativeToRecords(nr, "example.com")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	rc := rcs[0]
	assertStr(t, "Type", rc.Type, "TXT")
	assertStr(t, "Target", rc.GetTargetTXTJoined(), "v=spf1 include:example.com ~all")
}

func TestNativeToRecords_FWD(t *testing.T) {
	nr := dnsStaticRecord{
		Name: "example.com", Type: "FWD", ForwardTo: "8.8.8.8",
		MatchSubdomain: "true", AddressList: "vpn-list", TTL: "1d",
	}
	rcs, err := nativeToRecords(nr, "example.com")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	rc := rcs[0]
	assertStr(t, "Type", rc.Type, "MIKROTIK_FWD")
	assertStr(t, "Target", rc.GetTargetField(), "8.8.8.8")
	assertMeta(t, rc, "match_subdomain", "true")
	assertMeta(t, rc, "address_list", "vpn-list")
}

func TestNativeToRecords_FWD_WithRegexp(t *testing.T) {
	nr := dnsStaticRecord{
		Name: "example.com", Type: "FWD", ForwardTo: "8.8.8.8",
		Regexp: `.*\.test$`, TTL: "1h",
	}
	rcs, err := nativeToRecords(nr, "example.com")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	assertMeta(t, rcs[0], "regexp", `.*\.test$`)
}

func TestNativeToRecords_NXDOMAIN(t *testing.T) {
	nr := dnsStaticRecord{
		Name: "blocked.example.com", Type: "NXDOMAIN",
		MatchSubdomain: "true", Comment: "blocked domain", TTL: "1d",
	}
	rcs, err := nativeToRecords(nr, "example.com")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	rc := rcs[0]
	assertStr(t, "Type", rc.Type, "MIKROTIK_NXDOMAIN")
	assertStr(t, "Target", rc.GetTargetField(), "NXDOMAIN")
	assertMeta(t, rc, "match_subdomain", "true")
	assertMeta(t, rc, "comment", "blocked domain")
}

func TestNativeToRecords_MetadataOnStandardTypes(t *testing.T) {
	nr := dnsStaticRecord{
		Name: "host.example.com", Type: "A", Address: "10.0.0.1", TTL: "1d",
		MatchSubdomain: "true", AddressList: "my-list", Comment: "test comment",
	}
	rcs, err := nativeToRecords(nr, "example.com")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	rc := rcs[0]
	if rc.Metadata == nil {
		t.Fatal("Metadata is nil, expected map")
	}
	assertMeta(t, rc, "match_subdomain", "true")
	assertMeta(t, rc, "address_list", "my-list")
	assertMeta(t, rc, "comment", "test comment")
}

func TestNativeToRecords_NoMetadata(t *testing.T) {
	nr := dnsStaticRecord{
		Name: "host.example.com", Type: "A", Address: "10.0.0.1", TTL: "1d",
	}
	rcs, err := nativeToRecords(nr, "example.com")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rcs[0].Metadata != nil {
		t.Errorf("Metadata = %v, want nil", rcs[0].Metadata)
	}
}

func TestNativeToRecords_UnsupportedType(t *testing.T) {
	nr := dnsStaticRecord{Name: "host.example.com", Type: "BOGUS", TTL: "1d"}
	_, err := nativeToRecords(nr, "example.com")
	if err == nil {
		t.Error("expected error for unsupported type")
	}
}

func TestNativeToRecords_InvalidAddress(t *testing.T) {
	nr := dnsStaticRecord{Name: "host.example.com", Type: "A", Address: "not-an-ip", TTL: "1d"}
	_, err := nativeToRecords(nr, "example.com")
	if err == nil {
		t.Error("expected error for invalid IP address")
	}
}

func TestNativeToRecords_InvalidTTL(t *testing.T) {
	nr := dnsStaticRecord{Name: "host.example.com", Type: "A", Address: "10.0.0.1", TTL: "bogus"}
	_, err := nativeToRecords(nr, "example.com")
	if err == nil {
		t.Error("expected error for invalid TTL")
	}
}

func TestNativeToRecords_ApexRecord(t *testing.T) {
	nr := dnsStaticRecord{Name: "example.com", Type: "A", Address: "10.0.0.1", TTL: "1d"}
	rcs, err := nativeToRecords(nr, "example.com")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	assertStr(t, "Label", rcs[0].GetLabel(), "@")
}

// --- recordToNative ---

func TestRecordToNative_A(t *testing.T) {
	rc := makeRC("A", "host", "example.com", "10.0.0.1")
	rc.TTL = 3600
	nr, err := recordToNative(rc)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	assertStr(t, "Type", nr.Type, "A")
	assertStr(t, "Address", nr.Address, "10.0.0.1")
	assertStr(t, "Name", nr.Name, "host.example.com")
	assertStr(t, "TTL", nr.TTL, "1h")
}

func TestRecordToNative_AAAA(t *testing.T) {
	rc := makeRC("AAAA", "v6", "example.com", "2001:db8::1")
	nr, err := recordToNative(rc)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	assertStr(t, "Type", nr.Type, "AAAA")
	assertStr(t, "Address", nr.Address, "2001:db8::1")
}

func TestRecordToNative_CNAME(t *testing.T) {
	rc := makeRC("CNAME", "alias", "example.com", "target.example.com.")
	nr, err := recordToNative(rc)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	assertStr(t, "Type", nr.Type, "CNAME")
	assertStr(t, "CName", nr.CName, "target.example.com")
}

func TestRecordToNative_FWD(t *testing.T) {
	rc := &models.RecordConfig{}
	rc.SetLabel("@", "example.com")
	rc.Type = "MIKROTIK_FWD"
	_ = rc.SetTarget("8.8.8.8")
	rc.TTL = 86400
	rc.Metadata = map[string]string{
		"match_subdomain": "true",
		"address_list":    "vpn-list",
		"regexp":          `.*\.test`,
	}

	nr, err := recordToNative(rc)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	assertStr(t, "Type", nr.Type, "FWD")
	assertStr(t, "ForwardTo", nr.ForwardTo, "8.8.8.8")
	assertStr(t, "MatchSubdomain", nr.MatchSubdomain, "yes")
	assertStr(t, "AddressList", nr.AddressList, "vpn-list")
	assertStr(t, "Regexp", nr.Regexp, `.*\.test`)
}

func TestRecordToNative_NXDOMAIN(t *testing.T) {
	rc := &models.RecordConfig{}
	rc.SetLabel("blocked", "example.com")
	rc.Type = "MIKROTIK_NXDOMAIN"
	_ = rc.SetTarget("NXDOMAIN")
	rc.TTL = 86400
	rc.Metadata = map[string]string{
		"match_subdomain": "true",
		"comment":         "blocked",
	}

	nr, err := recordToNative(rc)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	assertStr(t, "Type", nr.Type, "NXDOMAIN")
	assertStr(t, "MatchSubdomain", nr.MatchSubdomain, "yes")
	assertStr(t, "Comment", nr.Comment, "blocked")
}

func TestRecordToNative_MX(t *testing.T) {
	rc := &models.RecordConfig{}
	rc.SetLabel("@", "example.com")
	rc.Type = "MX"
	_ = rc.SetTargetMX(10, "mail.example.com.")
	rc.TTL = 86400

	nr, err := recordToNative(rc)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	assertStr(t, "Type", nr.Type, "MX")
	assertStr(t, "MxExchange", nr.MxExchange, "mail.example.com")
	assertStr(t, "MxPreference", nr.MxPreference, "10")
}

func TestRecordToNative_SRV(t *testing.T) {
	rc := &models.RecordConfig{}
	rc.SetLabel("_sip._tcp", "example.com")
	rc.Type = "SRV"
	_ = rc.SetTargetSRV(10, 20, 5060, "sipserver.example.com.")
	rc.TTL = 3600

	nr, err := recordToNative(rc)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	assertStr(t, "Type", nr.Type, "SRV")
	assertStr(t, "SrvTarget", nr.SrvTarget, "sipserver.example.com")
	assertStr(t, "SrvPort", nr.SrvPort, "5060")
	assertStr(t, "SrvPriority", nr.SrvPriority, "10")
	assertStr(t, "SrvWeight", nr.SrvWeight, "20")
}

func TestRecordToNative_TXT(t *testing.T) {
	rc := &models.RecordConfig{}
	rc.SetLabel("@", "example.com")
	rc.Type = "TXT"
	_ = rc.SetTargetTXT("v=spf1 ~all")
	rc.TTL = 86400

	nr, err := recordToNative(rc)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	assertStr(t, "Type", nr.Type, "TXT")
	assertStr(t, "Text", nr.Text, "v=spf1 ~all")
}

func TestRecordToNative_MetadataOnStandardTypes(t *testing.T) {
	rc := makeRC("A", "host", "example.com", "10.0.0.1")
	rc.Metadata = map[string]string{
		"match_subdomain": "true",
		"address_list":    "my-list",
		"comment":         "my comment",
	}

	nr, err := recordToNative(rc)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	assertStr(t, "MatchSubdomain", nr.MatchSubdomain, "yes")
	assertStr(t, "AddressList", nr.AddressList, "my-list")
	assertStr(t, "Comment", nr.Comment, "my comment")
}

func TestRecordToNative_UnsupportedType(t *testing.T) {
	rc := &models.RecordConfig{}
	rc.SetLabel("@", "example.com")
	rc.Type = "BOGUS"
	_ = rc.SetTarget("whatever")

	_, err := recordToNative(rc)
	if err == nil {
		t.Error("expected error for unsupported type")
	}
}

// --- Round-trip conversion ---

func TestNativeRoundTrip_A(t *testing.T) {
	original := dnsStaticRecord{
		ID: "*1", Name: "host.example.com", Type: "A", Address: "192.168.1.1", TTL: "1d",
		MatchSubdomain: "true", AddressList: "my-list", Comment: "round trip",
	}

	rcs, err := nativeToRecords(original, "example.com")
	if err != nil {
		t.Fatalf("nativeToRecords: %v", err)
	}
	nr, err := recordToNative(rcs[0])
	if err != nil {
		t.Fatalf("recordToNative: %v", err)
	}

	assertStr(t, "Type", nr.Type, "A")
	assertStr(t, "Address", nr.Address, "192.168.1.1")
	assertStr(t, "Name", nr.Name, "host.example.com")
	assertStr(t, "TTL", nr.TTL, "1d")
	assertStr(t, "MatchSubdomain", nr.MatchSubdomain, "yes")
	assertStr(t, "AddressList", nr.AddressList, "my-list")
	assertStr(t, "Comment", nr.Comment, "round trip")
}

func TestNativeRoundTrip_FWD(t *testing.T) {
	original := dnsStaticRecord{
		Name: "example.com", Type: "FWD", ForwardTo: "8.8.8.8", TTL: "1d",
		MatchSubdomain: "true", AddressList: "vpn-list", Regexp: `.*\.test`,
	}

	rcs, err := nativeToRecords(original, "example.com")
	if err != nil {
		t.Fatalf("nativeToRecords: %v", err)
	}
	nr, err := recordToNative(rcs[0])
	if err != nil {
		t.Fatalf("recordToNative: %v", err)
	}

	assertStr(t, "Type", nr.Type, "FWD")
	assertStr(t, "ForwardTo", nr.ForwardTo, "8.8.8.8")
	assertStr(t, "MatchSubdomain", nr.MatchSubdomain, "yes")
	assertStr(t, "AddressList", nr.AddressList, "vpn-list")
	assertStr(t, "Regexp", nr.Regexp, `.*\.test`)
}

func TestNativeRoundTrip_NXDOMAIN(t *testing.T) {
	original := dnsStaticRecord{
		Name: "blocked.example.com", Type: "NXDOMAIN", TTL: "1h",
		MatchSubdomain: "true", Comment: "blocked",
	}

	rcs, err := nativeToRecords(original, "example.com")
	if err != nil {
		t.Fatalf("nativeToRecords: %v", err)
	}
	nr, err := recordToNative(rcs[0])
	if err != nil {
		t.Fatalf("recordToNative: %v", err)
	}

	assertStr(t, "Type", nr.Type, "NXDOMAIN")
	assertStr(t, "MatchSubdomain", nr.MatchSubdomain, "yes")
	assertStr(t, "Comment", nr.Comment, "blocked")
}

// --- Forwarder conversion ---

func TestForwarderToRecord(t *testing.T) {
	fwd := dnsForwarder{
		ID: "*1", Name: "my-forwarder", DNSServers: "1.1.1.1,8.8.8.8",
		DohServers: "https://dns.google/dns-query", VerifyDohCert: "true",
	}

	rc := forwarderToRecord(fwd)
	assertStr(t, "Type", rc.Type, "MIKROTIK_FORWARDER")
	assertStr(t, "Label", rc.GetLabel(), "my-forwarder")
	assertStr(t, "Target", rc.GetTargetField(), "1.1.1.1,8.8.8.8")
	assertUint32(t, "TTL", rc.TTL, 300)
	assertMeta(t, rc, "doh_servers", "https://dns.google/dns-query")
	assertMeta(t, rc, "verify_doh_cert", "true")
}

func TestForwarderToRecord_Minimal(t *testing.T) {
	fwd := dnsForwarder{Name: "simple", DNSServers: "1.1.1.1"}
	rc := forwarderToRecord(fwd)
	assertStr(t, "Target", rc.GetTargetField(), "1.1.1.1")
	if v, ok := rc.Metadata["doh_servers"]; ok {
		t.Errorf("doh_servers should not be present, got %q", v)
	}
	if v, ok := rc.Metadata["verify_doh_cert"]; ok {
		t.Errorf("verify_doh_cert should not be present, got %q", v)
	}
}

func TestRecordToForwarder(t *testing.T) {
	rc := &models.RecordConfig{}
	rc.SetLabel("my-fwd", ForwarderZone)
	rc.Type = "MIKROTIK_FORWARDER"
	_ = rc.SetTarget("1.1.1.1,8.8.8.8")
	rc.Metadata = map[string]string{
		"doh_servers":     "https://dns.google/dns-query",
		"verify_doh_cert": "true",
	}

	f := recordToForwarder(rc)
	assertStr(t, "Name", f.Name, "my-fwd")
	assertStr(t, "DnsServers", f.DNSServers, "1.1.1.1,8.8.8.8")
	assertStr(t, "DohServers", f.DohServers, "https://dns.google/dns-query")
	assertStr(t, "VerifyDohCert", f.VerifyDohCert, "true")
}

func TestForwarderRoundTrip(t *testing.T) {
	original := dnsForwarder{
		ID: "*1", Name: "fwd1", DNSServers: "1.1.1.1",
		DohServers: "https://dns.google/dns-query", VerifyDohCert: "true",
	}

	rc := forwarderToRecord(original)
	f := recordToForwarder(rc)

	assertStr(t, "Name", f.Name, "fwd1")
	assertStr(t, "DnsServers", f.DNSServers, "1.1.1.1")
	assertStr(t, "DohServers", f.DohServers, "https://dns.google/dns-query")
	assertStr(t, "VerifyDohCert", f.VerifyDohCert, "true")
}

// --- Test helpers ---

func makeRC(rtype, label, origin, target string) *models.RecordConfig {
	rc := &models.RecordConfig{}
	rc.SetLabel(label, origin)
	rc.Type = rtype
	switch rtype {
	case "A", "AAAA":
		rc.MustSetTarget(target)
	case "TXT":
		_ = rc.SetTargetTXT(target)
	default:
		_ = rc.SetTarget(target)
	}
	return rc
}

func assertStr(t *testing.T, field, got, want string) {
	t.Helper()
	if got != want {
		t.Errorf("%s = %q, want %q", field, got, want)
	}
}

func assertUint32(t *testing.T, field string, got, want uint32) {
	t.Helper()
	if got != want {
		t.Errorf("%s = %d, want %d", field, got, want)
	}
}

func assertMeta(t *testing.T, rc *models.RecordConfig, key, want string) {
	t.Helper()
	if rc.Metadata == nil {
		t.Errorf("Metadata is nil, want %s=%q", key, want)
		return
	}
	if got := rc.Metadata[key]; got != want {
		t.Errorf("Metadata[%q] = %q, want %q", key, got, want)
	}
}
