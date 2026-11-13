package models

import (
	"encoding/json"
	"testing"

	dnsv2 "codeberg.org/miekg/dns"
	"github.com/DNSControl/dnscontrol/v5/pkg/privatetypes"
)

// TestRecordConfigJSONRoundTrip verifies that a RecordConfig survives being
// marshaled to JSON (print-ir) and unmarshaled (--ir).
// See https://github.com/DNSControl/dnscontrol/issues/4854
func TestRecordConfigJSONRoundTrip(t *testing.T) {
	for _, tc := range jsonRoundTripTests() {
		t.Run(tc.name, func(t *testing.T) {
			want, err := json.Marshal(tc.rc)
			if err != nil {
				t.Fatalf("Marshal: %v", err)
			}

			var rc RecordConfig
			if err := json.Unmarshal(want, &rc); err != nil {
				t.Fatalf("Unmarshal(%s): %v", want, err)
			}
			if rc.GetRDATA() == nil {
				t.Fatalf("RDATA is nil after round-trip of %s", want)
			}

			got, err := json.Marshal(&rc)
			if err != nil {
				t.Fatalf("re-Marshal: %v", err)
			}
			if string(got) != string(want) {
				t.Errorf("JSON does not round-trip:\n got=%s\nwant=%s", got, want)
			}
		})
	}
}

type jsonRoundTripTest struct {
	name string
	rc   *RecordConfig
}

func jsonRoundTripTests() []jsonRoundTripTest {
	dc := MustNewDomainConfig("example.com")
	return []jsonRoundTripTest{
		{"A", dc.MustNewRecordConfig("@", 300, dnsv2.TypeA, "1.2.3.4")},
		{"MX", dc.MustNewRecordConfig("@", 300, dnsv2.TypeMX, 10, "mail.example.com.")},
		{"MX short", dc.MustNewRecordConfig("@", 300, dnsv2.TypeMX, 10, "mail")},
		{"TXT", dc.MustNewRecordConfig("@", 300, dnsv2.TypeTXT, "hello world")},
		{"TXT quotes", dc.MustNewRecordConfig("@", 300, dnsv2.TypeTXT, `a "quoted" back\slash; semi`)},
		{"TXT empty", dc.MustNewRecordConfig("@", 300, dnsv2.TypeTXT, "")},
		{"CAA", dc.MustNewRecordConfig("@", 300, dnsv2.TypeCAA, 0, "issue", "letsencrypt.org")},
		{"SRV", dc.MustNewRecordConfig("@", 300, dnsv2.TypeSRV, 10, 20, 5060, "sip.example.com.")},
		{"SVCB", dc.MustNewRecordConfig("@", 300, dnsv2.TypeSVCB, 1, ".", "alpn=h2,h3 port=443")},
		{"SOA", dc.MustNewRecordConfig("@", 300, dnsv2.TypeSOA, "ns1.example.com.", "admin.example.com.", 3600, 900, 604800, 86400)},
		{"NAPTR", dc.MustNewRecordConfig("@", 300, dnsv2.TypeNAPTR, 100, 10, "U", "E2U+sip", "!^.*$!sip:info@example.com!", "example")},
		{"NAPTR root", dc.MustNewRecordConfig("@", 300, dnsv2.TypeNAPTR, 100, 10, "U", "E2U+sip", "!^.*$!sip:info@example.com!", "")},
		// Private types. TestRecordConfigJSONRoundTripCoversPrivateTypes
		// verifies that every private type is listed here.
		{"ADGUARDHOME_A_PASSTHROUGH", dc.MustNewRecordConfig("@", 300, privatetypes.TypeADGUARDHOMEAPASSTHROUGH, "")},
		{"ADGUARDHOME_AAAA_PASSTHROUGH", dc.MustNewRecordConfig("@", 300, privatetypes.TypeADGUARDHOMEAAAAPASSTHROUGH, "")},
		{"AKAMAICDN", dc.MustNewRecordConfig("@", 300, privatetypes.TypeAKAMAICDN, "example.com.edgekey.net.")},
		{"AKAMAITLC", dc.MustNewRecordConfig("@", 300, privatetypes.TypeAKAMAITLC, "DUAL", "example.com.akamaiedge.net.")},
		{"ALIAS", dc.MustNewRecordConfig("@", 300, privatetypes.TypeALIAS, "target")},
		{"AZURE_ALIAS", dc.MustNewRecordConfig("@", 300, privatetypes.TypeAZUREALIAS, "A",
			"/subscriptions/726f8cd6/resourceGroups/test/providers/Microsoft.Network/trafficManagerProfiles/testpp2")},
		{"BUNNY_DNS_PZ", dc.MustNewRecordConfig("@", 300, privatetypes.TypeBUNNYDNSPZ, int64(123456))},
		{"BUNNY_DNS_RDR", dc.MustNewRecordConfig("@", 300, privatetypes.TypeBUNNYDNSRDR, "https://example.net/a b?c=d&e=\"f\"")},
		{"CF_WORKER_ROUTE", dc.MustNewRecordConfig("@", 300, privatetypes.TypeCFWORKERROUTE, "api.example.com/*", "my-worker")},
		{"CLOUDFLAREAPI_SINGLE_REDIRECT", dc.MustNewRecordConfig("@", 1, privatetypes.TypeCLOUDFLAREAPISINGLEREDIRECT,
			"301,test1.example.com,https://example.net/$1", 301,
			`http.host eq "test1.example.com" and http.request.uri.path eq "/"`,
			`concat("https://example.net", http.request.uri.path)`)},
		{"CLOUDNS_WR", dc.MustNewRecordConfig("@", 300, privatetypes.TypeCLOUDNSWR, "https://example.net/path?q=1")},
		{"FRAME", dc.MustNewRecordConfig("@", 300, privatetypes.TypeFRAME, "https://example.net/frame")},
		{"IMPORT_TRANSFORM", dc.MustNewRecordConfig("@", 300, privatetypes.TypeIMPORTTRANSFORM,
			"0.0.5.0 ~ 0.0.5.0 ~ ~ 1.2.3.4 ; 0.0.7.0 ~ 0.0.7.0 ~ ~ 5.6.7.8", 60, ".com", "foo.com")},
		{"LUA", dc.MustNewRecordConfig("@", 300, privatetypes.TypeLUA, "TXT", `return "a \"b\" c\\d"`)},
		{"MIKROTIK_FORWARDER", dc.MustNewRecordConfig("@", 300, privatetypes.TypeMIKROTIKFORWARDER, "8.8.8.8,8.8.4.4")},
		{"MIKROTIK_FWD", dc.MustNewRecordConfig("@", 300, privatetypes.TypeMIKROTIKFWD, "corp-dns")},
		{"MIKROTIK_NXDOMAIN", dc.MustNewRecordConfig("blocked", 300, privatetypes.TypeMIKROTIKNXDOMAIN)},
		{"NETLIFY", dc.MustNewRecordConfig("@", 300, privatetypes.TypeNETLIFY)},
		{"NETLIFYV6", dc.MustNewRecordConfig("@", 300, privatetypes.TypeNETLIFYV6)},
		{"PORKBUN_URLFWD", dc.MustNewRecordConfig("@", 300, privatetypes.TypePORKBUNURLFWD, "https://example.net/a b")},
		{"R53_ALIAS", dc.MustNewRecordConfig("@", 300, privatetypes.TypeR53ALIAS, "A", "foo.example.net.", "true", "Z2FTEDLFRXXXXX")},
		{"R53_ALIAS defaults", dc.MustNewRecordConfig("@", 300, privatetypes.TypeR53ALIAS, "AAAA", "foo.example.net.")},
		{"URL", dc.MustNewRecordConfig("@", 300, privatetypes.TypeURL, "https://example.net/?a=1;b=2")},
		{"URL301", dc.MustNewRecordConfig("@", 300, privatetypes.TypeURL301, "https://example.net/")},
	}
}

// TestRecordConfigJSONRoundTripCoversPrivateTypes fails if a private type is
// added without adding it to jsonRoundTripTests.
func TestRecordConfigJSONRoundTripCoversPrivateTypes(t *testing.T) {
	covered := map[uint16]bool{}
	for _, tc := range jsonRoundTripTests() {
		covered[tc.rc.TypeNum] = true
	}
	// Private types are the ones in the private-use range that have a maker.
	// (dnsv2 has its own types in that range, such as DELEG.)
	for typeNum := range privatetypes.TypeToMakeRDATA {
		if privatetypes.IsPrivateType(typeNum) && !covered[typeNum] {
			t.Errorf("private type %s has no case in jsonRoundTripTests", dnsv2.TypeToString[typeNum])
		}
	}
}

func TestRecordConfigJSONUnmarshal(t *testing.T) {
	tests := []struct {
		name      string
		data      string
		wantRDATA string
		wantErr   bool
	}{
		{"typenum and type", `{"typenum":1,"type":"A","name":"@","rdata":"1.2.3.4"}`, "1.2.3.4", false},
		{"type only", `{"type":"MX","name":"@","rdata":"10 mx.example.com."}`, "10 mx.example.com.", false},
		{"typenum only", `{"typenum":16,"name":"@","rdata":"\"foo\""}`, `"foo"`, false},
		{"missing rdata", `{"typenum":1,"type":"A","name":"@"}`, "", true},
		{"old-style rdata", `{"typenum":1,"type":"A","name":"@","rdata":{"Addr":"1.2.3.4"}}`, "", true},
		{"invalid rdata", `{"typenum":1,"type":"A","name":"@","rdata":"not-an-ip"}`, "", true},
		{"empty rdata", `{"typenum":1,"type":"A","name":"@","rdata":""}`, "", true},
		{"no type", `{"name":"@","rdata":"1.2.3.4"}`, "", true},
		{"unknown type", `{"type":"BOGUS","name":"@","rdata":"1.2.3.4"}`, "", true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var rc RecordConfig
			err := json.Unmarshal([]byte(tc.data), &rc)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected an error, got RDATA=%+v", rc.GetRDATA())
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if got := rc.GetRDATA().String(); got != tc.wantRDATA {
				t.Errorf("RDATA = %q, want %q", got, tc.wantRDATA)
			}
			if rc.TypeNum == 0 || rc.Type == "" {
				t.Errorf("TypeNum=%d Type=%q; both should be set", rc.TypeNum, rc.Type)
			}
		})
	}
}
