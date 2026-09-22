package commands

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/DNSControl/dnscontrol/v5/models"
	"github.com/DNSControl/dnscontrol/v5/pkg/js"
	_ "github.com/DNSControl/dnscontrol/v5/pkg/providers/_all"
	"github.com/google/go-cmp/cmp"
)

func TestFormatTypes(t *testing.T) {
	/*
	  Input:                   Converted to:   Should match contents of:
	  test_data/$DOMAIN.zone   js              test_data/$DOMAIN.zone.js
	  test_data/$DOMAIN.zone   tsv             test_data/$DOMAIN.zone.tsv
	  test_data/$DOMAIN.zone   zone            test_data/$DOMAIN.zone.zone
	*/

	for _, domain := range []string{"simple.com", "example.org", "apex.com", "ds.com"} {
		t.Run(domain+"/js", func(t *testing.T) { testFormat(t, domain, "js") })
		t.Run(domain+"/djs", func(t *testing.T) { testFormat(t, domain, "djs") })
		t.Run(domain+"/tsv", func(t *testing.T) { testFormat(t, domain, "tsv") })
		t.Run(domain+"/zone", func(t *testing.T) { testFormat(t, domain, "zone") })
	}
}

func testFormat(t *testing.T, domain, format string) {
	t.Helper()

	expectedFilename := fmt.Sprintf("test_data/%s.zone.%s", domain, format)
	outputFiletmpl := fmt.Sprintf("%s.zone.%s.*.txt", domain, format)

	outfile, err := os.CreateTemp(t.TempDir(), outputFiletmpl)
	if err != nil {
		t.Fatalf("gz can't TempFile %q: %v", outputFiletmpl, err)
	}
	if err := outfile.Close(); err != nil {
		t.Fatalf("gz can't close TempFile %q: %v", outfile.Name(), err)
	}

	// Convert test data to the experiment output.
	gzargs := GetZoneArgs{
		ZoneNames:    []string{domain},
		OutputFormat: format,
		OutputFile:   outfile.Name(),
		CredName:     "bind",
		ProviderName: "BIND",
		CredsFile:    "test_data/bind-creds.json",
	}

	// Read the zonefile and convert
	err = GetZone(gzargs)
	if err != nil {
		t.Fatalf("can't GetZone: %v", err)
	}

	// Read the actual result:
	got, err := os.ReadFile(outfile.Name())
	if err != nil {
		t.Fatalf("can't read actuals %q: %v", outfile.Name(), err)
	}

	// Read the expected result
	want, err := os.ReadFile(expectedFilename)
	if err != nil {
		t.Fatalf("can't read expected %q: %v", expectedFilename, err)
	}

	if diff := cmp.Diff(string(got), string(want)); diff != "" {
		// If the test fails, output a file showing "got"
		err = os.WriteFile(expectedFilename+".ACTUAL", got, 0o644)
		if err != nil {
			t.Fatalf("can't write actual output: %v", err)
		}
		t.Errorf("testFormat mismatch (-got +want):\n%s", diff)
	}
}

func TestFormatDslEscaping(t *testing.T) {
	dc := &models.DomainConfig{Name: "example.com"}
	hostile := `x"), A("injected", "192.0.2.66`

	tests := []struct {
		name  string
		label string
		rtype string
		args  []any
	}{
		{"label", hostile, "A", []any{"192.0.2.1"}},
		{"ns", "sub", "NS", []any{hostile + ".example.net."}},
		{"apex ns", "@", "NS", []any{"ns1.example.net.\n" + hostile}},
		{"caa", "@", "CAA", []any{0, "issue", hostile}},
		{"caa critical", "@", "CAA", []any{128, "issue", hostile}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rc, err := dc.NewRecordConfig(tt.label, 300, tt.rtype, tt.args...)
			if err != nil {
				t.Fatalf("NewRecordConfig: %v", err)
			}
			line := formatDsl(rc, 300)
			script := fmt.Sprintf("D(\"example.com\", NewRegistrar(\"none\"), DnsProvider(NewDnsProvider(\"none\")),\n%s\n);\n", line)
			conf, err := js.ExecuteJavascriptString([]byte(script), false, nil)
			if err != nil {
				t.Fatalf("generated line does not parse: %v\n%s", err, line)
			}
			records := conf.Domains[0].Records
			if strings.HasPrefix(line, "//") {
				if len(records) != 0 {
					t.Fatalf("commented-out line produced %d records:\n%s", len(records), line)
				}
				return
			}
			if len(records) != 1 {
				t.Fatalf("got %d records, want 1:\n%s", len(records), line)
			}
			if got, want := records[0].GetRDATA().String(), rc.GetRDATA().String(); got != want {
				t.Errorf("RDATA = %q, want %q", got, want)
			}
		})
	}
}
