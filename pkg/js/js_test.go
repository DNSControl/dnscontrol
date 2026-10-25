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

	"github.com/DNSControl/dnscontrol/v5/models"
	"github.com/DNSControl/dnscontrol/v5/pkg/normalize"
	"github.com/DNSControl/dnscontrol/v5/pkg/prettyzone"
	"github.com/DNSControl/dnscontrol/v5/pkg/providers"
	_ "github.com/DNSControl/dnscontrol/v5/pkg/providers/_all"
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
			testifyrequire.JSONEqf(t, es, as, "ACTUAL %q = \n```\n%s\n```", expectedFile+".ACTUAL", as)

			// For each domain, if there is a zone file, test against it:

			var dCount int
			for _, dc := range conf.Domains {
				var zoneFile string
				if dc.Tag != "" {
					zoneFile = filepath.Join(testDir, testName, dc.GetUniqueName()+".zone")
				} else {
					zoneFile = filepath.Join(testDir, testName, dc.Name+".zone")
				}
				expectedZone, err := os.ReadFile(zoneFile)
				if err != nil {
					//fmt.Printf("DEBUG: No zone file for domain %q (looking for %q)\n", dc.Name, zoneFile)
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
				testifyrequire.Equal(t, es, as, "2EXPECTING %q =\n```\n%s```", zoneFile, as)
			}
			if dCount > 0 && (len(conf.Domains) != dCount) {
				t.Fatal(fmt.Errorf("only %d of %d domains in %q have zonefiles", dCount, len(conf.Domains), name))
			}
		})
	}
}

func TestProviderSyntaxDomainsEquivalent(t *testing.T) {
	conf, err := ExecuteJavaScript(filepath.Join(testDir, "069-provider-syntax.js"), true, nil)
	testifyrequire.NoError(t, err)
	testifyrequire.Empty(t, normalize.ValidateAndNormalizeConfig(conf))
	testifyrequire.Len(t, conf.Domains, 2)

	// The split-horizon tags and source positions identify the two declarations;
	// all other fields in their generated JSON must be identical.
	for _, domain := range conf.Domains {
		domain.Name = "example.com"
		domain.Tag = ""
		domain.UniqueName = "example.com"
		for _, record := range domain.Records {
			record.FilePos = ""
		}
	}
	legacyJSON, err := json.Marshal(conf.Domains[0])
	testifyrequire.NoError(t, err)
	modernJSON, err := json.Marshal(conf.Domains[1])
	testifyrequire.NoError(t, err)
	testifyrequire.JSONEq(t, string(legacyJSON), string(modernJSON))
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
		{"M365_BUILDER without initialDomain", `D("foo.com","reg",M365_BUILDER("foo.com"))`},
		{"M365_BUILDER dash without domainGUID", `D("foo-bar.com","reg",M365_BUILDER("foo-bar.com",{initialDomain:"contoso.onmicrosoft.com"}))`},
		{"M365_BUILDER unknown option", `D("foo.com","reg",M365_BUILDER("foo.com",{initialDomain:"contoso.onmicrosoft.com",zzz:1}))`},
		{"M365_BUILDER name does not match the zone", `D("foo.com","reg",M365_BUILDER("other.com",{initialDomain:"contoso.onmicrosoft.com"}))`},
		{"M365_BUILDER with a record modifier", `D("foo.com","reg",M365_BUILDER("foo.com",{initialDomain:"contoso.onmicrosoft.com"},CF_PROXY_OFF))`},
		{"FORSALE_BUILDER without a label", `D("foo.com","reg",FORSALE_BUILDER({furi:"https://foo.com/buy"}))`},
		{"FORSALE_BUILDER unknown option", `D("foo.com","reg",FORSALE_BUILDER("@",{uri:"https://foo.com/buy"}))`},
		{"FORSALE_BUILDER bad price", `D("foo.com","reg",FORSALE_BUILDER("@",{fval:"999 EUR"}))`},
		{"FORSALE_BUILDER with a record modifier", `D("foo.com","reg",FORSALE_BUILDER("@",{fval:"EUR999"},CF_PROXY_OFF))`},
	}
	for _, tst := range tests {
		t.Run(tst.desc, func(t *testing.T) {
			if _, err := ExecuteJavaScript(tst.text, true, nil); err == nil {
				t.Fatal("Expected error but found none")
			}
		})
	}
}

// TestDMARCBuilderValues runs the script text itself with
// ExecuteJavascriptString, because TestErrors passes text to
// ExecuteJavaScript, which expects a file name.
func TestDMARCBuilderValues(t *testing.T) {
	tests := []struct{ desc, opts, wantErr string }{
		{"policy only", `{policy:"reject"}`, ""},
		{"every option valid", `{policy:"quarantine",subdomainPolicy:"none",nonexistentSubdomainPolicy:"reject",publicSuffixDomain:"u",testMode:"y"}`, ""},
		{"bad policy", `{policy:"rejct"}`, "Invalid DMARC policy"},
		{"bad subdomainPolicy", `{policy:"reject",subdomainPolicy:"quarentine"}`, "Invalid DMARC subdomain policy"},
		{"bad nonexistentSubdomainPolicy", `{policy:"reject",nonexistentSubdomainPolicy:"quarentine"}`, "Invalid DMARC nonexistent-subdomain policy"},
		{"bad publicSuffixDomain", `{policy:"reject",publicSuffixDomain:"yes"}`, "Invalid public-suffix-domain tag"},
		{"bad testMode", `{policy:"reject",testMode:"yes"}`, "Invalid test-mode tag"},
	}
	for _, tst := range tests {
		t.Run(tst.desc, func(t *testing.T) {
			script := `D("foo.com","none",DMARC_BUILDER(` + tst.opts + `))`
			_, err := ExecuteJavascriptString([]byte(script), true, nil)
			switch {
			case tst.wantErr == "" && err != nil:
				t.Fatalf("Expected no error but found: %v", err)
			case tst.wantErr != "" && (err == nil || !strings.Contains(err.Error(), tst.wantErr)):
				t.Fatalf("Expected error %q but found: %v", tst.wantErr, err)
			}
		})
	}
}

func TestLOCBuilderDDCoordinates(t *testing.T) {
	tests := []struct{ desc, opts, want, wantErr string }{
		{"zero coordinates", `{label:"origin",x:0,y:0,alt:0}`, "00 00 0.000 S 00 00 0.000 W 0m 0.00m 0.00m 0.00m", ""},
		{"missing x", `{label:"site",y:1,alt:0}`, "", "LOC_BUILDER_DD requires x and y elements"},
		{"missing y", `{label:"site",x:1,alt:0}`, "", "LOC_BUILDER_DD requires x and y elements"},
	}
	for _, tst := range tests {
		t.Run(tst.desc, func(t *testing.T) {
			script := `D("example.com","none",LOC_BUILDER_DD(` + tst.opts + `))`
			conf, err := ExecuteJavascriptString([]byte(script), true, nil)
			switch {
			case tst.wantErr == "" && err != nil:
				t.Fatalf("Expected no error but found: %v", err)
			case tst.wantErr != "" && (err == nil || !strings.Contains(err.Error(), tst.wantErr)):
				t.Fatalf("Expected error %q but found: %v", tst.wantErr, err)
			case tst.wantErr == "":
				if got := conf.Domains[0].Records[0].ComparableV3; got != tst.want {
					t.Fatalf("Expected %q but found %q", tst.want, got)
				}
			}
		})
	}
}

// TestDKIMBuilderNote checks that a non-ASCII note is written as UTF-8 octets.
func TestDKIMBuilderNote(t *testing.T) {
	tests := []struct{ desc, note, want string }{
		{"ascii", "a note", "v=DKIM1; n=a=20note; p=abc"},
		{"two-octet character", "café", "v=DKIM1; n=caf=C3=A9; p=abc"},
		{"three-octet character", "€5", "v=DKIM1; n==E2=82=AC5; p=abc"},
		{"four-octet character", "\U0001F600", "v=DKIM1; n==F0=9F=98=80; p=abc"},
	}
	for _, tst := range tests {
		t.Run(tst.desc, func(t *testing.T) {
			script := `D("foo.com","none",DKIM_BUILDER({selector:"s1",pubkey:"abc",note:"` + tst.note + `"}))`
			conf, err := ExecuteJavascriptString([]byte(script), true, nil)
			if err != nil {
				t.Fatal(err)
			}
			if got := conf.Domains[0].Records[0].GetTargetTXTJoined(); got != tst.want {
				t.Fatalf("Expected %q but found %q", tst.want, got)
			}
		})
	}
}
