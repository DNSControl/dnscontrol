package hostingde

import (
	"strings"
	"testing"

	"github.com/DNSControl/dnscontrol/v5/models"
)

// hosting.de stores and returns TXT content in presentation format, so
// nativeToRecord has to parse it rather than take it literally.
func TestNativeToRecordTXT(t *testing.T) {
	long := strings.Repeat("a", 255)

	tests := []struct {
		name     string
		content  string
		want     string
		wantSegs int
	}{
		{"simple", `"v=spf1 -all"`, "v=spf1 -all", 1},
		{"spaces", `"one two three"`, "one two three", 1},
		{"escaped quotes", `"say \"hi\""`, `say "hi"`, 1},
		{"escaped backslash", `"a\\b"`, `a\b`, 1},
		{"empty", `""`, "", 1},
		{"multiple strings", `"` + long + `" "bcd"`, long + "bcd", 2},
	}

	for _, tst := range tests {
		t.Run(tst.name, func(t *testing.T) {
			dc := models.MustNewDomainConfig("example.com")
			r := record{Name: "example.com", Type: "TXT", Content: tst.content, TTL: 300}

			rc, err := r.nativeToRecord(dc)
			if err != nil {
				t.Fatalf("nativeToRecord returned error: %v", err)
			}
			if got := rc.GetTargetTXTJoined(); got != tst.want {
				t.Errorf("joined target = %q, want %q", got, tst.want)
			}
			if got := rc.GetTargetTXTSegmentCount(); got != tst.wantSegs {
				t.Errorf("segment count = %d, want %d", got, tst.wantSegs)
			}
		})
	}
}

// The old write path escaped quotes but not backslashes, so DNSControl
// itself could have stored content like this. Reading it back must not fail
// the whole zone.
func TestNativeToRecordTXTNotPresentationFormat(t *testing.T) {
	const content = `"a\"`

	dc := models.MustNewDomainConfig("example.com")
	r := record{Name: "example.com", Type: "TXT", Content: content, TTL: 300}

	rc, err := r.nativeToRecord(dc)
	if err != nil {
		t.Fatalf("nativeToRecord returned error: %v", err)
	}
	if got := rc.GetTargetTXTJoined(); got != content {
		t.Errorf("joined target = %q, want %q", got, content)
	}
}

// A record written by recordToNative must read back unchanged, otherwise
// every push reports the same TXT records as modified again.
func TestTXTRoundTrip(t *testing.T) {
	tests := []struct {
		name  string
		value string
	}{
		{"simple", "v=spf1 -all"},
		{"spaces", "one two three"},
		{"escaped quotes", `say "hi"`},
		{"backslash", `a\b`},
		{"empty", ""},
		{"multiple strings", strings.Repeat("a", 255) + "bcd"},
	}

	for _, tst := range tests {
		t.Run(tst.name, func(t *testing.T) {
			dc := models.MustNewDomainConfig("example.com")
			rc := dc.MustNewRecordConfig("@", 300, "TXT", tst.value)

			back, err := recordToNative(rc).nativeToRecord(dc)
			if err != nil {
				t.Fatalf("nativeToRecord returned error: %v", err)
			}
			if got := back.GetTargetTXTJoined(); got != tst.value {
				t.Errorf("round trip = %q, want %q", got, tst.value)
			}
		})
	}
}
