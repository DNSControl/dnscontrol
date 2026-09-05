package spflib

import (
	"bytes"
	"fmt"
	"io"
	"strings"
	"testing"
)

func dump(rec *SPFRecord, indent string, w io.Writer) {
	fmt.Fprintf(w, "%sTotal Lookups: %d\n", indent, rec.Lookups())
	fmt.Fprint(w, indent+"v=spf1")
	for _, p := range rec.Parts {
		fmt.Fprint(w, " "+p.Text)
	}
	fmt.Fprintln(w)
	indent += "\t"
	for _, p := range rec.Parts {
		if p.IsLookup {
			fmt.Fprintln(w, indent+p.Text)
		}
		if p.IncludeRecord != nil {
			dump(p.IncludeRecord, indent+"\t", w)
		}
	}
}

// Print prints an SPFRecord.
func (s *SPFRecord) Print() string {
	w := &bytes.Buffer{}
	dump(s, "", w)
	return w.String()
}

func TestParse(t *testing.T) {
	dnsres, err := NewCache("testdata-dns1.json")
	if err != nil {
		t.Fatal(err)
	}
	rec, err := Parse(strings.Join([]string{
		"v=spf1",
		"ip4:198.252.206.0/24",
		"ip4:192.111.0.0/24",
		"include:_spf.google.com",
		"include:mailgun.org",
		// "include:spf-basic.fogcreek.com",
		"include:mail.zendesk.com",
		"include:servers.mcsv.net",
		"include:sendgrid.net",
		"include:spf.mtasv.net",
		"exists:%{i}._spf.sparkpostmail.com",
		"ptr:sparkpostmail.com",
		"~all",
	}, " "), dnsres)
	if err != nil {
		t.Fatal(err)
	}
	t.Log(rec.Print())
}

func TestParseWithDoubleSpaces(t *testing.T) {
	dnsres, err := NewCache("testdata-dns1.json")
	if err != nil {
		t.Fatal(err)
	}
	rec, err := Parse("v=spf1 ip4:192.111.0.0/24  ip4:192.111.1.0/24 -all", dnsres)
	if err != nil {
		t.Fatal(err)
	}
	t.Log(rec.Print())
}

func TestParseRedirectNotLast(t *testing.T) {
	// Make sure redirect=foo fails if it isn't the last item.
	dnsres, err := NewCache("testdata-dns1.json")
	if err != nil {
		t.Fatal(err)
	}
	_, err = Parse(strings.Join([]string{
		"v=spf1",
		"redirect=servers.mcsv.net",
		"~all",
	}, " "), dnsres)
	if err == nil {
		t.Fatal("should fail")
	}
}

func TestParseRedirectColon(t *testing.T) {
	// Make sure redirect:foo fails.
	dnsres, err := NewCache("testdata-dns1.json")
	if err != nil {
		t.Fatal(err)
	}
	_, err = Parse(strings.Join([]string{
		"v=spf1",
		"redirect:servers.mcsv.net",
	}, " "), dnsres)
	if err == nil {
		t.Fatal("should fail")
	}
}

func TestParseRedirectOnly(t *testing.T) {
	dnsres, err := NewCache("testdata-dns1.json")
	if err != nil {
		t.Fatal(err)
	}
	rec, err := Parse(strings.Join([]string{
		"v=spf1",
		"redirect=servers.mcsv.net",
	}, " "), dnsres)
	if err != nil {
		t.Fatal(err)
	}
	t.Log(rec.Print())
}

func TestParseQualifiedMechanisms(t *testing.T) {
	// Regression test: SPF mechanisms with qualifiers (+mx, -all, ~all, ?mx)
	// should be parsed correctly. See https://github.com/DNSControl/dnscontrol/issues/4042
	dnsres, err := NewCache("testdata-dns1.json")
	if err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		input   string
		wantErr bool
		lookups int
	}{
		{"v=spf1 +mx -all", false, 1},
		{"v=spf1 ~mx -all", false, 1},
		{"v=spf1 ?mx -all", false, 1},
		{"v=spf1 +a -all", false, 1},
		{"v=spf1 ~a -all", false, 1},
		{"v=spf1 +mx +a -all", false, 2},
		{"v=spf1 +ip4:192.168.0.0/24 -all", false, 0},
		{"v=spf1 +ip6:2001:db8::/32 -all", false, 0},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			rec, err := Parse(tt.input, dnsres)
			if (err != nil) != tt.wantErr {
				t.Fatalf("Parse(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			}
			if err == nil && rec.Lookups() != tt.lookups {
				t.Errorf("Parse(%q) lookups = %d, want %d", tt.input, rec.Lookups(), tt.lookups)
			}
		})
	}
}

func TestParsePtrMechanism(t *testing.T) {
	tests := []struct {
		input   string
		wantErr bool
		lookups int
	}{
		{"v=spf1 ptr -all", false, 1},
		{"v=spf1 a mx ptr -all", false, 3},
		{"v=spf1 mx ptr ip4:107.161.151.0/24 ~all", false, 2},
		{"v=spf1 +ptr -all", false, 1},
		{"v=spf1 ~ptr -all", false, 1},
		{"v=spf1 ?ptr -all", false, 1},
		{"v=spf1 ptr:example.com -all", false, 1},
		{"v=spf1 ptrfoo -all", true, 0},
		{"v=spf1 ptrfoo:example.com -all", true, 0},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			rec, err := Parse(tt.input, nil)
			if (err != nil) != tt.wantErr {
				t.Fatalf("Parse(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			}
			if err == nil && rec.Lookups() != tt.lookups {
				t.Errorf("Parse(%q) lookups = %d, want %d", tt.input, rec.Lookups(), tt.lookups)
			}
		})
	}
}

func TestParseIncludeLoop(t *testing.T) {
	tests := []struct {
		description string
		dnsres      fakeResolver
		input       string
		wantErr     string
	}{
		{
			description: "a domain that includes itself",
			dnsres: fakeResolver{
				"a.example.com": "v=spf1 include:a.example.com ~all",
			},
			input:   "v=spf1 include:a.example.com ~all",
			wantErr: "in included SPF: SPF include loop: a.example.com -> a.example.com",
		},
		{
			description: "two domains that include each other",
			dnsres: fakeResolver{
				"a.example.com": "v=spf1 include:b.example.com ~all",
				"b.example.com": "v=spf1 include:a.example.com ~all",
			},
			input:   "v=spf1 include:a.example.com ~all",
			wantErr: "in included SPF: in included SPF: SPF include loop: a.example.com -> b.example.com -> a.example.com",
		},
		{
			description: "a domain that redirects to itself",
			dnsres: fakeResolver{
				"a.example.com": "v=spf1 redirect=a.example.com",
			},
			input:   "v=spf1 redirect=a.example.com",
			wantErr: "in included SPF: SPF include loop: a.example.com -> a.example.com",
		},
		{
			description: "a qualified include that loops",
			dnsres: fakeResolver{
				"a.example.com": "v=spf1 +include:a.example.com ~all",
			},
			input:   "v=spf1 ?include:a.example.com ~all",
			wantErr: "in included SPF: SPF include loop: a.example.com -> a.example.com",
		},
		{
			description: "a loop that changes the case of the domain",
			dnsres: fakeResolver{
				"a.example.com": "v=spf1 include:A.EXAMPLE.COM ~all",
				"A.EXAMPLE.COM": "v=spf1 include:a.example.com ~all",
			},
			input:   "v=spf1 include:a.example.com ~all",
			wantErr: "in included SPF: SPF include loop: a.example.com -> A.EXAMPLE.COM",
		},
	}
	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			_, err := Parse(tt.input, tt.dnsres)
			if err == nil {
				t.Fatalf("Parse(%q) error = nil, want %q", tt.input, tt.wantErr)
			}
			if err.Error() != tt.wantErr {
				t.Errorf("Parse(%q) error = %q, want %q", tt.input, err, tt.wantErr)
			}
		})
	}
}

func TestParseSharedIncludeIsNotALoop(t *testing.T) {
	dnsres := fakeResolver{
		"a.example.com":      "v=spf1 include:shared.example.com ~all",
		"b.example.com":      "v=spf1 include:shared.example.com ~all",
		"shared.example.com": "v=spf1 ip4:192.0.2.0/24 ~all",
	}
	rec, err := Parse("v=spf1 include:a.example.com include:b.example.com ~all", dnsres)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := len(rec.Parts), 3; got != want {
		t.Errorf("len(Parts) = %d, want %d", got, want)
	}
	if got, want := rec.Lookups(), 4; got != want {
		t.Errorf("Lookups() = %d, want %d", got, want)
	}
}

func TestParseRedirectLast(t *testing.T) {
	dnsres, err := NewCache("testdata-dns1.json")
	if err != nil {
		t.Fatal(err)
	}
	rec, err := Parse(strings.Join([]string{
		"v=spf1",
		"ip4:198.252.206.0/24",
		"redirect=servers.mcsv.net",
	}, " "), dnsres)
	if err != nil {
		t.Fatal(err)
	}
	t.Log(rec.Print())
}
