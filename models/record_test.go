package models

import (
	"reflect"
	"testing"

	privatetypesrdata "github.com/DNSControl/dnscontrol/v5/pkg/privatetypes/rdata"
)

// TestR53AliasTargetSurvivesRDATAUpdate verifies the provider-side mutation
// pattern: copy the typed RDATA, update it, and store it with SetRDATA.
func TestR53AliasTargetSurvivesRDATAUpdate(t *testing.T) {
	const origin = "example.com"
	const wantTarget = "kyle.example.com."

	dc := MustNewDomainConfig(origin)
	rc, err := dc.NewRecordConfig("kenny", 300, "R53_ALIAS", "A", wantTarget, "false")
	if err != nil {
		t.Fatalf("NewRecordConfig: %v", err)
	}

	if got := rc.AsR53ALIAS().Target; got != wantTarget {
		t.Fatalf("target after construction = %q, want %q", got, wantTarget)
	}

	// Simulate Route53 filling in the zone ID.
	rd := rc.AsR53ALIAS()
	rd.ZoneID = "Z0389923"
	rc.SetRDATA(rd)

	if got := rc.AsR53ALIAS().Target; got != wantTarget {
		t.Errorf("target after SetRDATA = %q, want %q", got, wantTarget)
	}
	if got := rc.AsR53ALIAS().ZoneID; got != "Z0389923" {
		t.Errorf("zone ID after SetRDATA = %q, want %q", got, "Z0389923")
	}
	if got := rc.GetTargetField(); got != wantTarget {
		t.Errorf("GetTargetField after SetRDATA = %q, want %q", got, wantTarget)
	}
}

// TestAzureAliasTargetComesFromRDATA verifies that AZURE_ALIAS no longer
// depends on the legacy AzureAlias map or target field.
func TestAzureAliasTargetComesFromRDATA(t *testing.T) {
	const origin = "example.com"
	const wantTarget = "/subscriptions/sub/resourceGroups/rg/providers/Microsoft.Network/dnszones/example.com/A/kyle"

	dc := MustNewDomainConfig(origin)
	rc, err := dc.NewRecordConfig("kenny", 300, "AZURE_ALIAS", "A", wantTarget)
	if err != nil {
		t.Fatalf("NewRecordConfig: %v", err)
	}

	if got := rc.AsAZUREALIAS().Target; got != wantTarget {
		t.Fatalf("target after construction = %q, want %q", got, wantTarget)
	}

	if got := rc.GetTargetField(); got != wantTarget {
		t.Errorf("GetTargetField = %q, want %q", got, wantTarget)
	}
}

// TestAliasToCnameChangeType reproduces the bug where converting an ALIAS to a
// CNAME via ChangeType() (as CLOUDFLAREAPI and other flattening providers do)
// panicked ("FixUp: .RDATA is nil for type CNAME") and/or lost the target.
func TestAliasToCnameChangeType(t *testing.T) {
	const origin = "example.com"
	const wantTarget = "foo.example.com."

	dc := MustNewDomainConfig(origin)
	rc, err := dc.NewRecordConfig("@", 300, "ALIAS", wantTarget)
	if err != nil {
		t.Fatalf("NewRecordConfig: %v", err)
	}

	// A provider converts the apex ALIAS into a CNAME (CNAME flattening).
	rc.ChangeType("CNAME", origin)

	// ChangeType installs native CNAME RDATA, so FixRD is now a no-op.
	rc.FixRD(origin)

	if rc.GetRDATA() == nil {
		t.Fatal("RDATA is nil after FixRD")
	}
	if got := rc.AsCNAME().Target; got != wantTarget {
		t.Errorf("CNAME target = %q, want %q", got, wantTarget)
	}
	if got := rc.GetTargetField(); got != wantTarget {
		t.Errorf("GetTargetField = %q, want %q", got, wantTarget)
	}
}

func TestHasRecordTypeName(t *testing.T) {
	x := &RecordConfig{
		Type: "A",
		Name: "@",
	}
	recs := Records{}
	if recs.HasRecordTypeName("A", "@") {
		t.Errorf("%v: expected (%v) got (%v)\n", recs, false, true)
	}
	recs = append(recs, x)
	if !recs.HasRecordTypeName("A", "@") {
		t.Errorf("%v: expected (%v) got (%v)\n", recs, true, false)
	}
	if recs.HasRecordTypeName("AAAA", "@") {
		t.Errorf("%v: expected (%v) got (%v)\n", recs, false, true)
	}
}

func TestKey(t *testing.T) {
	tests := []struct {
		rc       RecordConfig
		expected RecordKey
	}{
		{
			RecordConfig{Type: "A", NameFQDN: "example.com"},
			RecordKey{Type: "A", NameFQDN: "example.com"},
		},
		{
			RecordConfig{Type: "R53_ALIAS", NameFQDN: "example.com", rdata: privatetypesrdata.R53ALIAS{AliasType: "AAAA"}},
			RecordKey{Type: "R53_ALIAS_AAAA", NameFQDN: "example.com"},
		},
		{
			RecordConfig{Type: "AZURE_ALIAS", NameFQDN: "example.com", rdata: privatetypesrdata.AZUREALIAS{AliasType: "CNAME"}},
			RecordKey{Type: "AZURE_ALIAS_CNAME", NameFQDN: "example.com"},
		},
	}
	for i, test := range tests {
		actual := test.rc.Key()
		if test.expected != actual {
			t.Errorf("%d: Expected %s, got %s", i, test.expected, actual)
		}
	}
}

func TestRecordConfig_Copy(t *testing.T) {
	type fields struct {
		Type             string
		Name             string
		SubDomain        string
		NameFQDN         string
		target           string
		TTL              uint32
		Metadata         map[string]string
		MxPreference     uint16
		SrvPriority      uint16
		SrvWeight        uint16
		SrvPort          uint16
		CaaTag           string
		CaaFlag          uint8
		DsKeyTag         uint16
		DsAlgorithm      uint8
		DsDigestType     uint8
		DsDigest         string
		NaptrOrder       uint16
		NaptrPreference  uint16
		NaptrFlags       string
		NaptrService     string
		NaptrRegexp      string
		SshfpAlgorithm   uint8
		SshfpFingerprint uint8
		SoaMbox          string
		SoaSerial        uint32
		SoaRefresh       uint32
		SoaRetry         uint32
		SoaExpire        uint32
		SoaMinttl        uint32
		TlsaUsage        uint8
		TlsaSelector     uint8
		TlsaMatchingType uint8
		Original         any
	}
	tests := []struct {
		name    string
		fields  fields
		want    *RecordConfig
		wantErr bool
	}{
		{
			name: "only",
			fields: fields{
				Type:             "type",
				Name:             "name",
				SubDomain:        "sub",
				NameFQDN:         "namef",
				target:           "targette",
				TTL:              12345,
				Metadata:         map[string]string{"me": "ah", "da": "ta"},
				MxPreference:     123,
				SrvPriority:      223,
				SrvWeight:        345,
				SrvPort:          456,
				CaaTag:           "caata",
				CaaFlag:          100,
				DsKeyTag:         12341,
				DsAlgorithm:      99,
				DsDigestType:     98,
				DsDigest:         "dsdig",
				NaptrOrder:       10000,
				NaptrPreference:  12220,
				NaptrFlags:       "naptrfl",
				NaptrService:     "naptrser",
				NaptrRegexp:      "naptrreg",
				SshfpAlgorithm:   4,
				SshfpFingerprint: 5,
				SoaMbox:          "soambox",
				SoaSerial:        456789,
				SoaRefresh:       192000,
				SoaRetry:         293293,
				SoaExpire:        3434343,
				SoaMinttl:        34234324,
				TlsaUsage:        1,
				TlsaSelector:     2,
				TlsaMatchingType: 3,
				// Original         any,
			},
			want: &RecordConfig{
				Type:      "type",
				Name:      "name",
				SubDomain: "sub",
				NameFQDN:  "namef",
				target:    "targette",
				TTL:       12345,
				Metadata:  map[string]string{"me": "ah", "da": "ta"},
				// MxPreference:     123,
				// SrvPriority:  223,
				// SrvWeight:    345,
				// SrvPort:      456,
				// CaaTag:       "caata",
				// CaaFlag:      100,
				// DsKeyTag:     12341,
				// DsAlgorithm:  99,
				// DsDigestType: 98,
				// DsDigest:     "dsdig",
				// NaptrOrder:   10000,
				// NaptrPreference:  12220,
				// NaptrFlags:       "naptrfl",
				// NaptrService:     "naptrser",
				// NaptrRegexp:      "naptrreg",
				// SshfpAlgorithm:   4,
				// SshfpFingerprint: 5,
				// TlsaUsage:        1,
				// TlsaSelector:     2,
				// TlsaMatchingType: 3,
				// Original         any,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rc := &RecordConfig{
				Type:      tt.fields.Type,
				Name:      tt.fields.Name,
				SubDomain: tt.fields.SubDomain,
				NameFQDN:  tt.fields.NameFQDN,
				target:    tt.fields.target,
				TTL:       tt.fields.TTL,
				Metadata:  tt.fields.Metadata,
				Original:  tt.fields.Original,
			}
			got, err := rc.Copy()
			if (err != nil) != tt.wantErr {
				t.Errorf("RecordConfig.Copy() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("RecordConfig.Copy() = %v, want %v", got, tt.want)
			}
		})
	}
}
func TestFixPosition(t *testing.T) {
	tests := []struct {
		name string
		pos  any
		want string
	}{
		{
			name: "empty string",
			pos:  "",
			want: "",
		},
		{
			name: "anonymous position",
			pos:  "at <anonymous>:2904:5",
			want: "[line:2904:5]",
		},
		{
			name: "random string",
			pos:  "alsdjfsljd",
			want: "[alsdjfsljd]",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FixPosition(tt.pos.(string))
			if got != tt.want {
				t.Errorf("fixPosition() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_makeLabelNameFQDN(t *testing.T) {
	tests := []struct {
		tname string // description of this test case
		// Named input parameters for target function.
		origin string
		name   string
		want   string
	}{
		{"a", "bosun.org", "@", "bosun.org"},
		{"b", "bosun.org", "foo", "foo.bosun.org"},
		{"c", "bosun.org", "bosun.org.", "bosun.org"},
		{"d", "bosun.org", "foo.bosun.org.", "foo.bosun.org"},
	}
	for _, tt := range tests {
		t.Run(tt.tname, func(t *testing.T) {
			got := makeLabelNameFQDN(tt.origin, tt.name)
			// TODO: update the condition below to compare got with tt.want.
			if got != tt.want {
				t.Errorf("makeNameFQDN(%q) = %v, want %v", tt.name, got, tt.want)
			}
		})
	}
}

func Test_makeLabelNameUnicode(t *testing.T) {
	tests := []struct {
		tname string // description of this test case
		// Named input parameters for target function.
		name string
		want string
	}{
		{"a", "foo.com", "foo.com"},
		{"b", "xn--mnchen-3ya.com", "münchen.com"},
	}
	for _, tt := range tests {
		t.Run(tt.tname, func(t *testing.T) {
			got, _ := makeLabelNameUnicode(tt.name)
			// TODO: update the condition below to compare got with tt.want.
			if got != tt.want {
				t.Errorf("makeNameUnicode(%q) = %v, want %v", tt.name, got, tt.want)
			}
		})
	}
}
