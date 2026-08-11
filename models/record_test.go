package models

import (
	"reflect"
	"testing"
)

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
			RecordConfig{Type: "R53_ALIAS", NameFQDN: "example.com"},
			RecordKey{Type: "R53_ALIAS", NameFQDN: "example.com"},
		},
		{
			RecordConfig{Type: "R53_ALIAS", NameFQDN: "example.com", R53Alias: map[string]string{"foo": "bar"}},
			RecordKey{Type: "R53_ALIAS", NameFQDN: "example.com"},
		},
		{
			RecordConfig{Type: "R53_ALIAS", NameFQDN: "example.com", R53Alias: map[string]string{"type": "AAAA"}},
			RecordKey{Type: "R53_ALIAS_AAAA", NameFQDN: "example.com"},
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
		R53Alias         map[string]string
		AzureAlias       map[string]string
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
				R53Alias:         map[string]string{"a": "eh", "b": "bee"},
				AzureAlias:       map[string]string{"az": "az", "ure": "your"},
				// Original         any,
			},
			want: &RecordConfig{
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
				R53Alias:         map[string]string{"a": "eh", "b": "bee"},
				AzureAlias:       map[string]string{"az": "az", "ure": "your"},
				// Original         any,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rc := &RecordConfig{
				Type:             tt.fields.Type,
				Name:             tt.fields.Name,
				SubDomain:        tt.fields.SubDomain,
				NameFQDN:         tt.fields.NameFQDN,
				target:           tt.fields.target,
				TTL:              tt.fields.TTL,
				Metadata:         tt.fields.Metadata,
				MxPreference:     tt.fields.MxPreference,
				SrvPriority:      tt.fields.SrvPriority,
				SrvWeight:        tt.fields.SrvWeight,
				SrvPort:          tt.fields.SrvPort,
				CaaTag:           tt.fields.CaaTag,
				CaaFlag:          tt.fields.CaaFlag,
				DsKeyTag:         tt.fields.DsKeyTag,
				DsAlgorithm:      tt.fields.DsAlgorithm,
				DsDigestType:     tt.fields.DsDigestType,
				DsDigest:         tt.fields.DsDigest,
				NaptrOrder:       tt.fields.NaptrOrder,
				NaptrPreference:  tt.fields.NaptrPreference,
				NaptrFlags:       tt.fields.NaptrFlags,
				NaptrService:     tt.fields.NaptrService,
				NaptrRegexp:      tt.fields.NaptrRegexp,
				SshfpAlgorithm:   tt.fields.SshfpAlgorithm,
				SshfpFingerprint: tt.fields.SshfpFingerprint,
				SoaMbox:          tt.fields.SoaMbox,
				SoaSerial:        tt.fields.SoaSerial,
				SoaRefresh:       tt.fields.SoaRefresh,
				SoaRetry:         tt.fields.SoaRetry,
				SoaExpire:        tt.fields.SoaExpire,
				SoaMinttl:        tt.fields.SoaMinttl,
				TlsaUsage:        tt.fields.TlsaUsage,
				TlsaSelector:     tt.fields.TlsaSelector,
				TlsaMatchingType: tt.fields.TlsaMatchingType,
				R53Alias:         tt.fields.R53Alias,
				AzureAlias:       tt.fields.AzureAlias,
				Original:         tt.fields.Original,
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
