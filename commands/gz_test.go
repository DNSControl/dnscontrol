package commands

import (
	"fmt"
	"log"
	"os"
	"testing"

	_ "github.com/DNSControl/dnscontrol/v4/pkg/providers/_all"
	_ "github.com/DNSControl/dnscontrol/v4/pkg/rtype"
	"github.com/andreyvit/diff"
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
		log.Fatal(fmt.Errorf("gz can't TempFile %q: %w", outputFiletmpl, err))
	}
	defer os.Remove(outfile.Name())

	// Convert test data to the experiment output.
	gzargs := GetZoneArgs{
		ZoneNames:    []string{domain},
		OutputFormat: format,
		OutputFile:   outfile.Name(),
		CredName:     "bind",
		ProviderName: "BIND",
	}
	gzargs.CredsFile = "test_data/bind-creds.json"

	// Read the zonefile and convert
	err = GetZone(gzargs)
	if err != nil {
		log.Fatal(fmt.Errorf("can't GetZone: %w", err))
	}

	// Read the actual result:
	got, err := os.ReadFile(outfile.Name())
	if err != nil {
		log.Fatal(fmt.Errorf("can't read actuals %q: %w", outfile.Name(), err))
	}

	// Read the expected result
	want, err := os.ReadFile(expectedFilename)
	if err != nil {
		log.Fatal(fmt.Errorf("can't read expected %q: %w", expectedFilename, err))
	}

	if w, g := string(want), string(got); w != g {
		// If the test fails, output a file showing "got"
		err = os.WriteFile(expectedFilename+".ACTUAL", got, 0o644)
		if err != nil {
			log.Fatal(err)
		}
		t.Errorf("testFormat mismatch (-got +want):\n%s", diff.LineDiff(g, w))
	}
}
