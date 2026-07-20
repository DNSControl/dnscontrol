package js

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"unicode"

	"github.com/DNSControl/dnscontrol/v4/models"
	"github.com/DNSControl/dnscontrol/v4/pkg/normalize"
	"github.com/DNSControl/dnscontrol/v4/pkg/prettyzone"
	"github.com/DNSControl/dnscontrol/v4/pkg/providers"
	_ "github.com/DNSControl/dnscontrol/v4/pkg/providers/_all"
	_ "github.com/DNSControl/dnscontrol/v4/pkg/rtype"
	testifyrequire "github.com/stretchr/testify/require"
)

const (
	testDir = "pkg/js/parse_tests"
)

func init() {
	// go up a directory so we helpers.js is in a consistent place.
	if err := os.Chdir("../.."); err != nil {
		panic(err)
	}
}

func TestParsedFiles(t *testing.T) {
	files, err := os.ReadDir(testDir)
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range files {
		name := f.Name()

		// run all js files that start with a number. Skip others.
		if filepath.Ext(name) != ".js" || !unicode.IsNumber(rune(name[0])) {
			continue
		}
		t.Run(name, func(t *testing.T) {
			var err error

			// Compile the .js file:
			conf, err := ExecuteJavaScript(string(filepath.Join(testDir, name)), true, nil)
			if err != nil {
				t.Fatal(err)
			}

			errs := normalize.ValidateAndNormalizeConfig(conf)
			if len(errs) != 0 {
				t.Fatal(errs[0])
			}

			for _, dc := range conf.Domains {
				// fmt.Printf("DEBUG: PrettySort: domain=%q #rec=%d\n", dc.Name, len(dc.Records))
				// fmt.Printf("DEBUG: records = %d %v\n", len(dc.Records), dc.Records)
				ps := prettyzone.PrettySort(dc.Records, dc.Name, 0, nil)
				dc.Records = ps.Records
				if len(dc.Records) == 0 {
					dc.Records = models.Records{}
				}
			}

			// Initialize any DNS providers mentioned.
			for _, dProv := range conf.DNSProviders {
				pcfg := map[string]string{}

				if dProv.Type == "-" {
					// Pretend any "look up provider type in creds.json" results
					// in a provider type that actually exists.
					dProv.Type = "CLOUDFLAREAPI"
				}

				// Fake out any provider's validation tests.
				switch dProv.Type {
				case "CLOUDFLAREAPI":
					pcfg["apitoken"] = "fake"
				default:
				}
				_, err := providers.CreateDNSProvider(dProv.Type, pcfg, nil)
				if err != nil {
					t.Fatal(err)
				}
			}

			// Test the JS compiled as expected (compare to the .json file)
			actualJSON, err := json.MarshalIndent(conf, "", "  ")
			if err != nil {
				t.Fatal(err)
			}
			testName := name[:len(name)-3]
			expectedFile := filepath.Join(testDir, testName+".json")
			expectedJSON, err := os.ReadFile(expectedFile)
			if err != nil {
				t.Fatal(err)
			}
			es := string(expectedJSON)
			as := string(actualJSON)
			_, _ = es, as
			// Leave behind the actual result:
			if err := os.WriteFile(expectedFile+".ACTUAL", []byte(as), 0o644); err != nil {
				t.Fatal(err)
			}
			testifyrequire.JSONEqf(t, es, as, "EXPECTING %q = \n```\n%s\n```", expectedFile, as)

			// For each domain, if there is a zone file, test against it:

			var dCount int
			for _, dc := range conf.Domains {
				var zoneFile string
				if dc.Tag != "" {
					zoneFile = filepath.Join(testDir, testName, dc.GetUniqueName()+".zone")
				} else {
					zoneFile = filepath.Join(testDir, testName, dc.Name+".zone")
				}
				//fmt.Printf("DEBUG: zonefile = %q\n", zoneFile)
				expectedZone, err := os.ReadFile(zoneFile)
				if err != nil {
					continue
				}
				dCount++

				// Generate the zonefile
				var buf bytes.Buffer
				err = prettyzone.WriteZoneFileRC(&buf, dc.Records, dc.Name, 300, nil)
				if err != nil {
					t.Fatal(err)
				}
				actualZone := buf.String()

				es := strings.TrimSpace(string(expectedZone))
				as := strings.TrimSpace(actualZone)
				if es != as {
					// On failure, leave behind the .ACTUAL file.
					if err := os.WriteFile(zoneFile+".ACTUAL", []byte(actualZone), 0o644); err != nil {
						t.Fatal(err)
					}
				}
				testifyrequire.Equal(t, es, as, "EXPECTING %q =\n```\n%s```", zoneFile, as)
			}
			if dCount > 0 && (len(conf.Domains) != dCount) {
				t.Fatal(fmt.Errorf("only %d of %d domains in %q have zonefiles", dCount, len(conf.Domains), name))
			}
		})
	}
}

func TestErrors(t *testing.T) {
	tests := []struct{ desc, text string }{
		{"old dsp style", `D("foo.com","reg","dsp")`},
		{"MX no priority", `D("foo.com","reg",MX("@","test."))`},
		{"MX reversed", `D("foo.com","reg",MX("@","test.", 5))`},
		{"CF_REDIRECT With comma", `D("foo.com","reg",CF_REDIRECT("foo.com,","baaa"))`},
		{"CF_TEMP_REDIRECT With comma", `D("foo.com","reg",CF_TEMP_REDIRECT("foo.com","baa,a"))`},
		{"CF_WORKER_ROUTE With comma", `D("foo.com","reg",CF_WORKER_ROUTE("foo.com","baa,a"))`},
		{"ADGUARDHOME_A_PASSTHROUGH With non-empty value", `D("foo.com","reg",ADGUARDHOME_A_PASSTHROUGH("foo","baaa"))`},
		{"ADGUARDHOME_AAAA_PASSTHROUGH With non-empty value", `D("foo.com","reg",ADGUARDHOME_AAAA_PASSTHROUGH("foo,","baaa"))`},
		{"Bad cidr", `D(reverse("foo.com"), "reg")`},
		{"Dup domains", `D("example.org", "reg"); D("example.org", "reg")`},
		{"Bad NAMESERVER", `D("example.com","reg", NAMESERVER("@","ns1.foo.com."))`},
		{"Bad Hash function", `D(HASH("123", "abc"),"reg")`},
	}
	for _, tst := range tests {
		t.Run(tst.desc, func(t *testing.T) {
			if _, err := ExecuteJavaScript(tst.text, true, nil); err == nil {
				t.Fatal("Expected error but found none")
			}
		})
	}
}
