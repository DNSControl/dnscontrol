package providergolden

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/DNSControl/dnscontrol/v5/models"
)

func TestDomainReadsTheProvidersDomainVariable(t *testing.T) {
	t.Setenv("VERCEL_DOMAIN", "recorded.example.net")
	if got := Domain("VERCEL"); got != "recorded.example.net" {
		t.Errorf("Domain() = %q, want %q", got, "recorded.example.net")
	}
}

func TestDomainFallsBackToExampleCom(t *testing.T) {
	t.Setenv("VERCEL_DOMAIN", "")
	if got := Domain("VERCEL"); got != "example.com" {
		t.Errorf("Domain() = %q, want %q", got, "example.com")
	}
}

func TestFormatRecord(t *testing.T) {
	dc := models.MustNewDomainConfig("example.com")

	tests := []struct {
		name     string
		rc       *models.RecordConfig
		metadata map[string]string
		want     string
	}{
		{
			name: "A",
			rc:   dc.MustNewRecordConfig("www", 300, "A", "192.0.2.1"),
			want: "www 300 IN A 192.0.2.1",
		},
		{
			name: "zero TTL is included",
			rc:   dc.MustNewRecordConfig("www", 0, "A", "192.0.2.1"),
			want: "www 0 IN A 192.0.2.1",
		},
		{
			name: "MX at the apex",
			rc:   dc.MustNewRecordConfig("@", 3600, "MX", 10, "mail.example.com."),
			want: "@ 3600 IN MX 10 mail.example.com.",
		},
		{
			name:     "metadata is sorted and quoted",
			rc:       dc.MustNewRecordConfig("fwd", 300, "A", "192.0.2.1"),
			metadata: map[string]string{"wildcard": "no", "includePath": "yes"},
			want:     `fwd 300 IN A 192.0.2.1 ; includePath="yes" wildcard="no"`,
		},
		{
			name:     "metadata value with a space and a quote",
			rc:       dc.MustNewRecordConfig("fwd", 300, "A", "192.0.2.1"),
			metadata: map[string]string{"note": `a "b" c`},
			want:     `fwd 300 IN A 192.0.2.1 ; note="a \"b\" c"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.metadata != nil {
				tt.rc.Metadata = tt.metadata
			}
			if got := formatRecord(tt.rc); got != tt.want {
				t.Errorf("formatRecord() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestParseRecordsRoundTrip(t *testing.T) {
	input := strings.Join([]string{
		"@ 3600 IN A 192.0.2.1",
		"www 0 IN A 192.0.2.2",
		"@ 3600 IN MX 10 mail.example.com.",
		`@ 600 IN TXT "v=spf1 include:_spf.example.net -all"`,
		`@ 600 IN CAA 0 issue "letsencrypt.org"`,
		"_sip._tcp 300 IN SRV 10 20 5060 sip.example.com.",
		`fwd 300 IN A 192.0.2.3 ; includePath="yes" wildcard="no"`,
	}, "\n") + "\n"

	recs, err := parseRecords(models.MustNewDomainConfig("example.com"), input)
	if err != nil {
		t.Fatalf("parseRecords() error: %v", err)
	}

	var got strings.Builder
	for _, rc := range recs {
		got.WriteString(formatRecord(rc))
		got.WriteByte('\n')
	}
	if got.String() != input {
		t.Errorf("round trip produced:\n%s\nwant:\n%s", got.String(), input)
	}
}

func TestParseRecordsMetadata(t *testing.T) {
	recs, err := parseRecords(models.MustNewDomainConfig("example.com"),
		`fwd 300 IN A 192.0.2.1 ; includePath="yes" note="a; b" wildcard="no"`)
	if err != nil {
		t.Fatalf("parseRecords() error: %v", err)
	}
	if len(recs) != 1 {
		t.Fatalf("parseRecords() returned %d records, want 1", len(recs))
	}

	want := map[string]string{"includePath": "yes", "note": "a; b", "wildcard": "no"}
	for k, v := range want {
		if got := recs[0].Metadata[k]; got != v {
			t.Errorf("Metadata[%q] = %q, want %q", k, got, v)
		}
	}
}

func TestParseRecordsRejectsMalformedInput(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{name: "no ttl", input: "www IN A 192.0.2.1"},
		{name: "non-numeric ttl", input: "www abc IN A 192.0.2.1"},
		{name: "wrong class", input: "www 300 CH A 192.0.2.1"},
		{name: "too few fields", input: "www 300 A 192.0.2.1"},
		{name: "no rdata", input: "www 300 IN A"},
		{name: "unknown type", input: "www 300 IN NOSUCHTYPE 192.0.2.1"},
		{name: "unterminated metadata", input: `www 300 IN A 192.0.2.1 ; wildcard="no`},
		{name: "unquoted metadata", input: "www 300 IN A 192.0.2.1 ; wildcard=no"},
	}

	dc := models.MustNewDomainConfig("example.com")
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := parseRecords(dc, tt.input); err == nil {
				t.Errorf("parseRecords(%q) succeeded, want an error", tt.input)
			}
		})
	}
}

func TestInputFileIsNamedAfterTheProvidersDirectory(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "cloudns")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	t.Chdir(dir)

	for ext, want := range map[string]string{nativesExt: "cloudns.json", recordsExt: "cloudns.records"} {
		got, err := inputFile(ext)
		if err != nil {
			t.Fatalf("inputFile(%q) error: %v", ext, err)
		}
		if got != want {
			t.Errorf("inputFile(%q) = %q, want %q", ext, got, want)
		}
	}
}

func TestRecordedInputsAreNamedAfterTheirProvider(t *testing.T) {
	root, err := moduleRoot()
	if err != nil {
		t.Fatal(err)
	}

	dirs, err := filepath.Glob(filepath.Join(root, "providers", "*", testdataDir))
	if err != nil {
		t.Fatal(err)
	}
	if len(dirs) == 0 {
		t.Fatal("no provider has a testdata directory")
	}

	for _, dir := range dirs {
		provider := filepath.Base(filepath.Dir(dir))
		entries, err := os.ReadDir(dir)
		if err != nil {
			t.Fatal(err)
		}
		for _, entry := range entries {
			ext := filepath.Ext(entry.Name())
			if ext != nativesExt && ext != recordsExt {
				continue
			}
			if want := provider + ext; entry.Name() != want {
				t.Errorf("providers/%s/%s holds %s, want %s: a recorded input is named after its provider, not after a test",
					provider, testdataDir, entry.Name(), want)
			}
		}
	}
}

func TestLoadInputReportsMissingFileAsNotEnrolled(t *testing.T) {
	_, ok, err := loadInput(t.TempDir(), "absent.json")
	if err != nil {
		t.Fatalf("loadInput() error: %v", err)
	}
	if ok {
		t.Error("loadInput() ok = true for a file that does not exist")
	}
}

func TestCompareGoldenSkipsWhenTheGoldenFileIsMissing(t *testing.T) {
	skip, diff, err := compareGolden(t.TempDir(), "absent", []byte("anything\n"), false)
	if err != nil {
		t.Fatalf("compareGolden() error: %v", err)
	}
	if skip == "" {
		t.Error("compareGolden() returned no skip reason for a missing golden file")
	}
	if diff != "" {
		t.Errorf("compareGolden() diff = %q, want empty", diff)
	}
}

func TestCompareGoldenDetectsADifference(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "x.golden"), []byte("www 300 IN A 192.0.2.1\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	skip, diff, err := compareGolden(dir, "x", []byte("www 300 IN A 192.0.2.99\n"), false)
	if err != nil {
		t.Fatalf("compareGolden() error: %v", err)
	}
	if skip != "" {
		t.Errorf("compareGolden() skip = %q, want empty", skip)
	}
	if diff == "" {
		t.Error("compareGolden() reported no difference between 192.0.2.1 and 192.0.2.99")
	}

	if _, diff, err = compareGolden(dir, "x", []byte("www 300 IN A 192.0.2.1\n"), false); err != nil {
		t.Fatalf("compareGolden() error: %v", err)
	} else if diff != "" {
		t.Errorf("compareGolden() diff = %q for identical content, want empty", diff)
	}
}

func TestCompareGoldenUpdateWritesTheFile(t *testing.T) {
	dir := t.TempDir()
	want := "www 300 IN A 192.0.2.1\n"

	if _, _, err := compareGolden(dir, "x", []byte(want), true); err != nil {
		t.Fatalf("compareGolden() error: %v", err)
	}

	got, err := os.ReadFile(filepath.Join(dir, "x.golden"))
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != want {
		t.Errorf("golden file = %q, want %q", got, want)
	}
}
