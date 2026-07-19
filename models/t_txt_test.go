package models

import (
	"strings"
	"testing"

	dnsv2 "codeberg.org/miekg/dns"
	dnsrdatav2 "codeberg.org/miekg/dns/rdata"
)

func testTXTRecord(t *testing.T, value string) *RecordConfig {
	t.Helper()
	dc := MustNewDomainConfig("example.com")
	rc := dc.MustNewRecordConfig("foo", 0, dnsv2.TypeTXT, value)
	return rc
}

func TestTXTJoined(t *testing.T) {
	tests := []struct {
		name string
		txt  []string
		want string
	}{
		{name: "no segments", txt: []string{}, want: ""},
		{name: "one segment", txt: []string{"one"}, want: "one"},
		{name: "one empty segment", txt: []string{""}, want: ""},
		{name: "multiple segments", txt: []string{"one", "", "two"}, want: "onetwo"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rd := dnsrdatav2.TXT{Txt: tt.txt}
			if got := TXTJoined(rd); got != tt.want {
				t.Fatalf("TXTJoined() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestTXTSegmented(t *testing.T) {
	payload := strings.Repeat("a", 254) + "bc"
	tests := []struct {
		name string
		txt  []string
		want []string
	}{
		{name: "no segments", txt: []string{}, want: []string{""}},
		{name: "one segment", txt: []string{"one"}, want: []string{"one"}},
		{
			name: "removes empty segments and resegments",
			txt:  []string{"", payload[:100], "", payload[100:]},
			want: []string{payload[:255], payload[255:]},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := TXTSegmented(dnsrdatav2.TXT{Txt: tt.txt})
			if !equalStrings(got, tt.want) {
				t.Fatalf("TXTSegmented() = %#v, want %#v", got, tt.want)
			}
		})
	}
}

func TestRecordConfigSetRDATANormalizesTXT(t *testing.T) {
	tests := []struct {
		name string
		txt  []string
		want []string
	}{
		{
			name: "removes empty segments and resegments",
			txt:  []string{"", strings.Repeat("x", 200), "", strings.Repeat("y", 56)},
			want: []string{strings.Repeat("x", 200) + strings.Repeat("y", 55), "y"},
		},
		{
			name: "adds an empty segment to an empty record",
			txt:  []string{},
			want: []string{""},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rc := &RecordConfig{Type: "TXT", TypeNum: dnsv2.TypeTXT}
			rc.SetRDATA(dnsrdatav2.TXT{Txt: tt.txt})
			got := rc.GetRDATA().(dnsrdatav2.TXT).Txt
			if !equalStrings(got, tt.want) {
				t.Fatalf("SetRDATA stored %#v, want %#v", got, tt.want)
			}
		})
	}
}

func TestRecordConfigGetTargetTXTJoined(t *testing.T) {
	tests := []struct {
		name  string
		value string
	}{
		{name: "empty", value: ""},
		{name: "one segment", value: "one"},
		{name: "multiple segments", value: strings.Repeat("a", 255) + "b"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rc := testTXTRecord(t, tt.value)
			if got := rc.GetTargetTXTJoined(); got != tt.value {
				t.Fatalf("GetTargetTXTJoined() = %q, want %q", got, tt.value)
			}
		})
	}
}

func TestRecordConfigGetTargetTXTSegmented(t *testing.T) {
	tests := []struct {
		name  string
		value string
		want  []string
	}{
		{name: "empty", value: "", want: []string{""}},
		{name: "one segment", value: "one", want: []string{"one"}},
		{name: "multiple segments", value: strings.Repeat("a", 255) + "b", want: []string{strings.Repeat("a", 255), "b"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rc := testTXTRecord(t, tt.value)
			got := rc.GetTargetTXTSegmented()
			if !equalStrings(got, tt.want) {
				t.Fatalf("GetTargetTXTSegmented() = %#v, want %#v", got, tt.want)
			}
		})
	}
}

func TestRecordConfigSetTargetTXT(t *testing.T) {
	tests := []struct {
		name  string
		value string
		want  []string
	}{
		{name: "empty", value: "", want: []string{""}},
		{name: "one segment", value: "one", want: []string{"one"}},
		{name: "multiple segments", value: strings.Repeat("x", 256), want: []string{strings.Repeat("x", 255), "x"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rc := testTXTRecord(t, tt.value)
			rd := rc.GetRDATA().(dnsrdatav2.TXT)
			if !equalStrings(rd.Txt, tt.want) {
				t.Fatalf("SetTargetTXT stored %#v, want %#v", rd.Txt, tt.want)
			}
		})
	}
}

func TestRecordConfigSetTargetTXTs(t *testing.T) {
	tests := []struct {
		name string
		txt  []string
		want []string
	}{
		{name: "no segments", txt: []string{}, want: []string{""}},
		{name: "one segment", txt: []string{"one"}, want: []string{"one"}},
		{
			name: "removes empty segments and resegments",
			txt:  []string{"", strings.Repeat("x", 200), "", strings.Repeat("y", 56), ""},
			want: []string{strings.Repeat("x", 200) + strings.Repeat("y", 55), "y"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rc := &RecordConfig{Type: "TXT", TypeNum: dnsv2.TypeTXT}
			if err := rc.SetTargetTXTs(tt.txt); err != nil {
				t.Fatalf("SetTargetTXTs: %v", err)
			}
			rd := rc.GetRDATA().(dnsrdatav2.TXT)
			if !equalStrings(rd.Txt, tt.want) {
				t.Fatalf("SetTargetTXTs stored %#v, want %#v", rd.Txt, tt.want)
			}
		})
	}
}

func TestRecordConfigGetTargetFieldTXT(t *testing.T) {
	tests := []struct {
		name  string
		value string
	}{
		{name: "empty", value: ""},
		{name: "one segment", value: "joined target"},
		{name: "multiple segments", value: strings.Repeat("a", 256)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rc := testTXTRecord(t, tt.value)
			if got, want := rc.GetTargetField(), rc.GetTargetTXTJoined(); got != want {
				t.Fatalf("GetTargetField() = %q, want %q", got, want)
			}
		})
	}
}

func TestRecordConfigGetTargetIPTXT(t *testing.T) {
	rc := testTXTRecord(t, "not an IP")
	defer func() {
		if recover() == nil {
			t.Fatal("GetTargetIP() did not panic for TXT")
		}
	}()
	rc.GetTargetIP()
}

func TestRecordConfigGetTargetCombinedFuncTXT(t *testing.T) {
	tests := []struct {
		name     string
		value    string
		encodeFn func(string) string
		want     string
	}{
		{name: "nil encoder", value: "raw text", encodeFn: nil, want: "raw text"},
		{
			name:     "custom encoder",
			value:    "raw text",
			encodeFn: func(s string) string { return "encoded[" + s + "]" },
			want:     "encoded[raw text]",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rc := testTXTRecord(t, tt.value)
			if got := rc.GetTargetCombinedFunc(tt.encodeFn); got != tt.want {
				t.Fatalf("GetTargetCombinedFunc() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestRecordConfigGetTargetCombinedTXT(t *testing.T) {
	tests := []struct {
		name  string
		value string
		want  string
	}{
		{name: "empty", value: "", want: `""`},
		{name: "one segment", value: "one", want: `"one"`},
		{name: "multiple segments", value: strings.Repeat("a", 255) + "b", want: `"` + strings.Repeat("a", 255) + `" "b"`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rc := testTXTRecord(t, tt.value)
			if got := rc.GetTargetCombined(); got != tt.want {
				t.Fatalf("GetTargetCombined() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestRecordConfigGetTargetRFC1035QuotedTXT(t *testing.T) {
	rc := testTXTRecord(t, `quote" slash\\`)
	if got, want := rc.GetTargetRFC1035Quoted(), rc.GetRDATA().String(); got != want {
		t.Fatalf("GetTargetRFC1035Quoted() = %q, want RDATA.String() %q", got, want)
	}
}

func TestRecordConfigGetTargetDebugTXT(t *testing.T) {
	rc := testTXTRecord(t, "debug text")
	if got, want := rc.GetTargetDebug(), rc.GetRDATA().String(); got != want {
		t.Fatalf("GetTargetDebug() = %q, want RDATA.String() %q", got, want)
	}
}

func TestRecordConfigGetTargetJSTXT(t *testing.T) {
	tests := []struct {
		name  string
		value string
		want  string
	}{
		{name: "empty", value: "", want: `[""]`},
		{name: "plain", value: "one", want: `["one"]`},
		{name: "JSON escapes", value: "<txt>&\n", want: `["\u003ctxt\u003e\u0026\n"]`},
		{name: "long255", value: strings.Repeat("a", 255), want: `["` + strings.Repeat("a", 255) + `"]`},
		{name: "long256", value: strings.Repeat("a", 255) + "b", want: `["` + strings.Repeat("a", 255) + `","b"]`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rc := testTXTRecord(t, tt.value)
			if got := rc.GetTargetJS(); got != tt.want {
				t.Fatalf("GetTargetJS() = %s, want JSON string %s", got, tt.want)
			}
		})
	}
}

func equalStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
