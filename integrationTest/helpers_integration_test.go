package main

// Functions for integration_test.go

import (
	"errors"
	"flag"
	"fmt"
	"maps"
	"os"
	"slices"
	"strings"
	"testing"
	"time"

	dnsv2 "codeberg.org/miekg/dns"
	"github.com/DNSControl/dnscontrol/v5/models"
	"github.com/DNSControl/dnscontrol/v5/pkg/domaintags"
	"github.com/DNSControl/dnscontrol/v5/pkg/nameservers"
	"github.com/DNSControl/dnscontrol/v5/pkg/nameutil"
	"github.com/DNSControl/dnscontrol/v5/pkg/privatetypes"
	"github.com/DNSControl/dnscontrol/v5/pkg/providers"
	"github.com/DNSControl/dnscontrol/v5/pkg/zonerecs"
)

var (
	startIdx     = flag.Int("start", -1, "Test number to begin with")
	endIdx       = flag.Int("end", -1, "Test index to stop after")
	verbose      = flag.Bool("verbose", false, "Print corrections as you run them")
	printElapsed = flag.Bool("elapsed", false, "Print elapsed time for each testgroup")
)

// Global variable to hold the current DomainConfig for use in NewRecordConfig calls.
// This is an ugly, ugly, hack. We have to find something better.
var globalDC *models.DomainConfig

// Global variable to hold the current DomainConfig     for use in FromRaw calls.
var globalDCN *domaintags.DomainNameVarieties

// Default TTL used in integration tests.
var defaultTTL = uint32(300)

// Helper functions to perform substitutions

func fillTemplate(s string) string {
	return strings.Replace(s, "**current-domain**", globalDC.Name, 1)
}

// Helper constants/funcs for the HEDNS Dynamic DNS testing:

func hednsDynamicA(name, target, status string) *models.RecordConfig {
	r := a(name, target)
	r.Metadata = make(map[string]string)
	r.Metadata["hedns_dynamic"] = status
	return r
}

func hednsDdnsKeyA(name, target, key string) *models.RecordConfig {
	r := a(name, target)
	r.Metadata = make(map[string]string)
	r.Metadata["hedns_dynamic"] = "on"
	r.Metadata["hedns_ddns_key"] = key
	return r
}

func hednsDynamicAAAA(name, target, status string) *models.RecordConfig {
	r := aaaa(name, target)
	r.Metadata = make(map[string]string)
	r.Metadata["hedns_dynamic"] = status
	return r
}

func hednsDdnsKeyAAAA(name, target, key string) *models.RecordConfig {
	r := aaaa(name, target)
	r.Metadata = make(map[string]string)
	r.Metadata["hedns_dynamic"] = "on"
	r.Metadata["hedns_ddns_key"] = key
	return r
}

func hednsDynamicTXT(name, target, status string) *models.RecordConfig {
	r := txt(name, target)
	r.Metadata = make(map[string]string)
	r.Metadata["hedns_dynamic"] = status
	return r
}

// Helper constants/funcs for the CLOUDFLARE proxy testing:

// A-record proxy off/on.
func CfProxyOff() *TestCase { return tc("proxyoff", cfProxyA("prxy", "174.136.107.111", "off")) }
func CfProxyOn() *TestCase  { return tc("proxyon", cfProxyA("prxy", "174.136.107.111", "on")) }

// CNAME-record proxy off/on.
func CfCProxyOff() *TestCase { return tc("cproxyoff", cfProxyCNAME("cproxy", "example.com.", "off")) }
func CfCProxyOn() *TestCase  { return tc("cproxyon", cfProxyCNAME("cproxy", "example.com.", "on")) }

// Helper constants/funcs for the CLOUDFLARE CNAME flattening testing:

// CNAME flattening off/on (requires paid plan).
func CfFlattenOff() *TestCase {
	return tc("flattenoff", cfFlattenCNAME("cflatten", "example.com.", "off"))
}
func CfFlattenOn() *TestCase {
	return tc("flattenon", cfFlattenCNAME("cflatten", "example.com.", "on"))
}

func getDomainConfigWithNameservers(t *testing.T, prv providers.DNSServiceProvider, domainName string) *models.DomainConfig {
	dc := models.MustNewDomainConfig(domainName)
	dc.PostProcess()
	dc.FixLegacyDC()

	// fix up nameservers
	ns, err := prv.GetNameservers(domainName)
	if err != nil {
		t.Fatal("Failed getting nameservers", err)
	}
	dc.Nameservers = ns
	nameservers.AddNSRecords(dc)
	return dc
}

// testPermitted returns nil if the test is permitted, otherwise an
// error explaining why it is not.
func testPermitted(p string, f TestGroup) error {
	// not() and only() can't be mixed.
	if len(f.only) != 0 && len(f.not) != 0 {
		return errors.New("invalid filter: can't mix not() and only()")
	}
	// TODO(tlim): Have a separate validation pass so that such mistakes
	// are more visible?

	// If there are any trueflags, make sure they are all true.
	for _, c := range f.trueflags {
		if !c {
			return fmt.Errorf("excluded by alltrue(%v)", f.trueflags)
		}
	}

	// If there are any required capabilities, make sure they all exist.
	if len(f.required) != 0 {
		for _, c := range f.required {
			if !providers.ProviderHasCapability(*providerFlag, c) {
				return fmt.Errorf("%s not supported", c)
			}
		}
	}

	// If there are any "only" items, you must be one of them.
	if len(f.only) != 0 {
		if slices.Contains(f.only, p) {
			return nil
		}
		return errors.New("disabled by only")
	}

	// If there are any "not" items, you must NOT be one of them.
	if len(f.not) != 0 {
		for _, provider := range f.not {
			if p == provider {
				return fmt.Errorf("excluded by not(\"%s\")", provider)
			}
		}
		return nil
	}

	return nil
}

// makeChanges runs one set of DNS record tests. Returns true on success.
func makeChanges(t *testing.T, prv providers.DNSServiceProvider, dc *models.DomainConfig, tst *TestCase, desc string, expectChanges bool, origConfig map[string]string, domainMeta map[string]string) bool {

	return t.Run(desc+":"+tst.Desc, func(t *testing.T) {
		dom, _ := dc.Copy()

		// Apply domain-level metadata if provided (e.g., for Cloudflare comments/tags management)
		if domainMeta != nil {
			if dom.Metadata == nil {
				dom.Metadata = make(map[string]string)
			}
			maps.Copy(dom.Metadata, domainMeta)
		}

		for _, r := range tst.Records {
			rc := models.RecordConfig(*r)

			replaceIntegrationTargetTokens(&rc, origConfig["SubscriptionID"], origConfig["ResourceGroup"])

			dom.Records = append(dom.Records, &rc)
		}
		dom.Unmanaged = tst.Unmanaged
		dom.UnmanagedUnsafe = tst.UnmanagedUnsafe
		// Bind will refuse a DDNS update when the resulting zone
		// contains a NS record without an associated address
		// records (A or AAAA). In order to run the integration tests
		// against bind, the initial zone contains the following records:
		// - `@ NS dummy-ns.example.com`
		// - `dummy-ns A 9.8.7.6`
		// We 'hardcode' an ignore rule for the `A` record.
		dom.Unmanaged = append(dom.Unmanaged, &models.UnmanagedConfig{
			LabelPattern:  "dummy-ns",
			RTypePattern:  "A",
			TargetPattern: "",
		})
		models.PostProcessRecords(dom.Records)
		dom.FixLegacyDC()
		dom2, _ := dom.Copy()

		if err := providers.AuditRecords(*providerFlag, dom.Records); err != nil {
			t.Skipf("***SKIPPED(PROVIDER DOES NOT SUPPORT '%s' ::%q)", err, desc)
			return
		}

		//fmt.Printf("DEBUG: Running test %q: Names %q %q %q\n", desc, dom.Name, dom.NameRaw, dom.NameUnicode)

		// get and run corrections for first time

		_, corrections, actualChangeCount, err := zonerecs.CorrectZoneRecords(prv, dom)
		if err != nil {
			t.Fatal(fmt.Errorf("runTests: %w", err))
		}
		if tst.Changeless {
			if actualChangeCount != 0 {
				t.Logf("Expected 0 corrections on FIRST run, but found %d.", actualChangeCount)
				for i, c := range corrections {
					t.Logf("UNEXPECTED #%d: %s", i, c.Msg)
				}
				t.FailNow()
			}
		} else if (len(corrections) == 0 && expectChanges) && (tst.Desc != "Empty") && !tst.ChangesOptional {
			t.Fatalf("Expected changes, but got none")
		}
		for _, c := range corrections {
			if *verbose {
				t.Log("\n" + c.Msg)
			}
			if c.F != nil { // F == nil if there is just a msg, no action.
				err = c.F()
				if err != nil {
					t.Fatal(err)
				}
			}
		}

		// If we just emptied out the zone, no need for a second pass.
		if len(tst.Records) == 0 {
			return
		}

		// run a second time and expect zero corrections

		_, corrections, actualChangeCount, err = zonerecs.CorrectZoneRecords(prv, dom2)
		if err != nil {
			t.Fatal(err)
		}
		if actualChangeCount != 0 {
			t.Logf("Expected 0 corrections on second run, but found %d.", actualChangeCount)
			for i, c := range corrections {
				t.Logf("UNEXPECTED #%d: %s", i, c.Msg)
			}
			t.FailNow()
		}
	})
}

func replaceIntegrationTargetTokens(rc *models.RecordConfig, subscriptionID, resourceGroup string) {
	originalTarget := rc.GetTargetField()
	target := strings.NewReplacer(
		"**subscription-id**", subscriptionID,
		"**resource-group**", strings.ToLower(resourceGroup),
	).Replace(originalTarget)
	if target == originalTarget {
		return
	}

	if rc.Type == "AZURE_ALIAS" {
		rd := rc.AsAZUREALIAS()
		rd.Target = target
		rc.SetRDATA(rd)
		return
	}

	_ = rc.SetTarget(target)
	rc.ClearRDATA()
}

func runTests(t *testing.T, prv providers.DNSServiceProvider, domainName string, origConfig map[string]string) {
	dc := getDomainConfigWithNameservers(t, prv, domainName)
	globalDC = dc

	testGroups := makeTests()

	firstGroup := *startIdx
	if firstGroup == -1 {
		firstGroup = 0
	}
	lastGroup := *endIdx
	if lastGroup == -1 {
		lastGroup = len(testGroups)
	}

	curGroup := -1
	for gIdx, group := range testGroups {
		// Abide by -start -end flags
		curGroup++
		if curGroup < firstGroup || curGroup > lastGroup {
			continue
		}

		// Abide by filter
		// fmt.Printf("DEBUG testPermitted: prov=%q profile=%q\n", *providerFlag, *profileFlag)
		if err := testPermitted(*profileFlag, *group); err != nil {
			t.Run(fmt.Sprintf("%02d:%s ***SKIPPED(%v)***:Empty", gIdx, group.Desc, err), func(t *testing.T) {
				t.SkipNow()
			})
			continue
		}

		// Start the testgroup with a clean slate.
		makeChanges(t, prv, dc, tc("Empty"), "Clean Slate", false, nil, nil)

		// Run the tests.
		start := time.Now()

		for _, tst := range group.tests {
			// TODO(tlim): This is the old version. It skipped the remaining tc() statements if one failed.
			// The new code continues to test the remaining tc() statements.  Keeping this as a comment
			// in case we ever want to do something similar.
			// https://github.com/DNSControl/dnscontrol/pull/2252#issuecomment-1492204409
			//      makeChanges(t, prv, dc, tst, fmt.Sprintf("%02d:%s", gIdx, group.Desc), true, origConfig)
			//      if t.Failed() {
			//        break
			//      }
			if ok := makeChanges(t, prv, dc, tst, fmt.Sprintf("%02d:%s", gIdx, group.Desc), true, origConfig, group.domainMeta); !ok {
				break
			}
		}

		elapsed := time.Since(start)
		if *printElapsed {
			fmt.Printf("ELAPSED %02d %7.2f %q\n", gIdx, elapsed.Seconds(), group.Desc)
		}
	}
}

type TestGroup struct {
	Desc       string
	required   []providers.Capability
	only       []string
	not        []string
	trueflags  []bool
	domainMeta map[string]string
	tests      []*TestCase
}

type TestCase struct {
	Desc            string
	Records         []*models.RecordConfig
	Unmanaged       []*models.UnmanagedConfig
	UnmanagedUnsafe bool // DISABLE_IGNORE_SAFETY_CHECK
	Changeless      bool // set to true if any changes would be an error
	ChangesOptional bool // set to true if either changes or no changes are acceptable
}

// ExpectNoChanges indicates that no changes is not an error, it is a requirement.
func (tc *TestCase) ExpectNoChanges() *TestCase {
	tc.Changeless = true
	return tc
}

// AllowNoChanges indicates that an already-converged first run is acceptable.
func (tc *TestCase) AllowNoChanges() *TestCase {
	tc.ChangesOptional = true
	return tc
}

// UnsafeIgnore is the equivalent of DISABLE_IGNORE_SAFETY_CHECK.
func (tc *TestCase) UnsafeIgnore() *TestCase {
	tc.UnmanagedUnsafe = true
	return tc
}

func SetLabel(r *models.RecordConfig, label, domain string) {
	r.Name = label
	r.NameFQDN = nameutil.ToFqdnWithDot(label, "**current-domain**.")
}

func withMeta(record *models.RecordConfig, metadata map[string]string) *models.RecordConfig {
	record.Metadata = metadata
	return record
}

func a(name, target string) *models.RecordConfig {
	r, err := globalDC.NewRecordConfig(name, defaultTTL, dnsv2.TypeA, target)
	panicOnErr(err)
	return r
}

func aaaa(name, target string) *models.RecordConfig {
	r, err := globalDC.NewRecordConfig(name, defaultTTL, dnsv2.TypeAAAA, target)
	panicOnErr(err)
	return r
}

func alias(name, target string) *models.RecordConfig {
	target = fillTemplate(target)
	r, err := globalDC.NewRecordConfig(name, defaultTTL, privatetypes.TypeALIAS, target)
	panicOnErr(err)
	return r
}

func azureAlias(name, aliasType, target string) *models.RecordConfig {
	target = fillTemplate(target)
	r, err := globalDC.NewRecordConfig(name, defaultTTL, privatetypes.TypeAZUREALIAS, aliasType, target)
	panicOnErr(err)
	return r
}

func caa(name string, flag uint8, tag string, target string) *models.RecordConfig {
	r, err := globalDC.NewRecordConfig(name, defaultTTL, dnsv2.TypeCAA, flag, tag, target)
	panicOnErr(err)
	return r
}

func cfProxyA(name, target, status string) *models.RecordConfig {
	r := a(name, target)
	r.Metadata = make(map[string]string)
	r.Metadata["cloudflare_proxy"] = status
	return r
}

func cfProxyCNAME(name, target, status string) *models.RecordConfig {
	r := cname(name, target)
	r.Metadata = make(map[string]string)
	r.Metadata["cloudflare_proxy"] = status
	return r
}

func cfFlattenCNAME(name, target, status string) *models.RecordConfig {
	r := cname(name, target)
	r.Metadata = make(map[string]string)
	r.Metadata["cloudflare_cname_flatten"] = status
	return r
}

func cfCommentA(name, target, comment string) *models.RecordConfig {
	r := a(name, target)
	r.Metadata = make(map[string]string)
	r.Metadata["cloudflare_comment"] = comment
	return r
}

func cfTagsA(name, target, tags string) *models.RecordConfig {
	r := a(name, target)
	r.Metadata = make(map[string]string)
	r.Metadata["cloudflare_tags"] = tags
	return r
}

func cfSingleRedirectEnabled() bool {
	return (*enableCFRedirectMode)
}

func cfSingleRedirect(name string, code any, when, then string) *models.RecordConfig {
	r, err := globalDC.NewRecordConfig("@", 1, privatetypes.TypeCLOUDFLAREAPISINGLEREDIRECT, name, code, when, then)
	panicOnErr(err)
	return r
}

func cfWorkerRoute(pattern, target string) *models.RecordConfig {
	pattern = fillTemplate(pattern)
	r, err := globalDC.NewRecordConfig("@", 1, privatetypes.TypeCFWORKERROUTE, pattern, target)
	panicOnErr(err)
	return r
}

func bunnyPullZone(name, pullZoneID string) *models.RecordConfig {
	r, err := globalDC.NewRecordConfig(name, 1, privatetypes.TypeBUNNYDNSPZ, pullZoneID)
	panicOnErr(err)
	return r
}

func aghAPassthrough(pattern, target string) *models.RecordConfig {
	r, err := globalDC.NewRecordConfig(pattern, defaultTTL, privatetypes.TypeADGUARDHOMEAPASSTHROUGH, target)
	panicOnErr(err)
	return r
}

func aghAAAAPassthrough(pattern, target string) *models.RecordConfig {
	r, err := globalDC.NewRecordConfig(pattern, defaultTTL, privatetypes.TypeADGUARDHOMEAAAAPASSTHROUGH, target)
	panicOnErr(err)
	return r
}

func mikrotikFwd(name, target string) *models.RecordConfig {
	r, err := globalDC.NewRecordConfig(name, defaultTTL, privatetypes.TypeMIKROTIKFWD, target)
	panicOnErr(err)
	return r
}

func mikrotikNxdomain(name string) *models.RecordConfig {
	r, err := globalDC.NewRecordConfig(name, defaultTTL, privatetypes.TypeMIKROTIKNXDOMAIN)
	panicOnErr(err)
	return r
}

func cname(name, target string) *models.RecordConfig {
	target = fillTemplate(target)
	r, err := globalDC.NewRecordConfig(name, defaultTTL, dnsv2.TypeCNAME, target)
	panicOnErr(err)
	return r
}

func dhcid(name, target string) *models.RecordConfig {
	r, err := globalDC.NewRecordConfig(name, defaultTTL, dnsv2.TypeDHCID, target)
	panicOnErr(err)
	return r
}

func dname(name, target string) *models.RecordConfig {
	r, err := globalDC.NewRecordConfig(name, defaultTTL, dnsv2.TypeDNAME, target)
	panicOnErr(err)
	return r
}

func ds(name string, keyTag uint16, algorithm, digestType uint8, digest string) *models.RecordConfig {
	r, err := globalDC.NewRecordConfig(name, defaultTTL, dnsv2.TypeDS, keyTag, algorithm, digestType, digest)
	panicOnErr(err)
	return r
}

func dnskey(name string, flags uint16, protocol, algorithm uint8, publicKey string) *models.RecordConfig {
	r, err := globalDC.NewRecordConfig(name, defaultTTL, dnsv2.TypeDNSKEY, flags, protocol, algorithm, publicKey)
	panicOnErr(err)
	return r
}

func https(name string, priority uint16, target string, params string) *models.RecordConfig {
	r, err := globalDC.NewRecordConfig(name, defaultTTL, dnsv2.TypeHTTPS, priority, target, params)
	panicOnErr(err)
	return r
}

func ignoreName(labelSpec string) *models.RecordConfig {
	return ignore(labelSpec, "*", "*")
}

func ignoreTarget(targetSpec string, typeSpec string) *models.RecordConfig {
	return ignore("*", typeSpec, targetSpec)
}

func ignore(labelSpec string, typeSpec string, targetSpec string) *models.RecordConfig {
	r := &models.RecordConfig{
		Type:     "IGNORE",
		Metadata: map[string]string{},
	}

	r.Metadata["ignore_LabelPattern"] = labelSpec
	r.Metadata["ignore_RTypePattern"] = typeSpec
	r.Metadata["ignore_TargetPattern"] = targetSpec
	r.FixRD(globalDC.Name) // Hack. Populates .RDATA and .TypeNum if needed.
	return r
}

func loc(name string, d1 uint8, m1 uint8, s1 float32, ns string,
	d2 uint8, m2 uint8, s2 float32, ew string, al float32, sz float32, hp float32, vp float32,
) *models.RecordConfig {
	r, err := globalDC.NewRecordConfig(name, defaultTTL, dnsv2.TypeLOC, d1, m1, s1, ns, d2, m2, s2, ew, al, sz, hp, vp)
	panicOnErr(err)
	return r
}

func manyA(namePattern, target string, n int) []*models.RecordConfig {
	recs := models.Records{}
	for i := range n {
		r, err := globalDC.NewRecordConfig(fmt.Sprintf(namePattern, i), defaultTTL, dnsv2.TypeA, target)
		panicOnErr(err)
		recs = append(recs, r)
	}
	return recs
}

func mx(name string, prio uint16, target string) *models.RecordConfig {
	target = fillTemplate(target)
	r, err := globalDC.NewRecordConfig(name, defaultTTL, dnsv2.TypeMX, prio, target)
	panicOnErr(err)
	return r
}

func ns(name, target string) *models.RecordConfig {
	target = fillTemplate(target)
	r, err := globalDC.NewRecordConfig(name, defaultTTL, dnsv2.TypeNS, target)
	panicOnErr(err)
	return r
}

func naptr(name string, order uint16, preference uint16, flags string, service string, regexp string, target string) *models.RecordConfig {
	r, err := globalDC.NewRecordConfig(name, defaultTTL, dnsv2.TypeNAPTR, order, preference, flags, service, regexp, target)
	panicOnErr(err)
	return r
}

func openpgpkey(name, target string) *models.RecordConfig {
	r, err := globalDC.NewRecordConfig(name, defaultTTL, dnsv2.TypeOPENPGPKEY, target)
	panicOnErr(err)
	return r
}

func ptr(name, target string) *models.RecordConfig {
	r, err := globalDC.NewRecordConfig(name, defaultTTL, dnsv2.TypePTR, target)
	panicOnErr(err)
	return r
}

func r53alias(name, aliasType, target, evalTargetHealth string) *models.RecordConfig {
	target = fillTemplate(target)
	r, err := globalDC.NewRecordConfig(name, defaultTTL, privatetypes.TypeR53ALIAS, aliasType, target, evalTargetHealth)
	panicOnErr(err)
	return r
}

func r53weighted(name, target, rtype string, weight int, setID string) *models.RecordConfig {
	r, err := globalDC.NewRecordConfig(name, defaultTTL, rtype, target)
	panicOnErr(err)
	r.Metadata = map[string]string{
		"r53_weight":         fmt.Sprintf("%d", weight),
		"r53_set_identifier": setID,
	}
	return r
}

func rp(name string, m, t string) *models.RecordConfig {
	r, err := globalDC.NewRecordConfig(name, defaultTTL, dnsv2.TypeRP, m, t)
	panicOnErr(err)
	return r
}

func smimea(name string, usage, selector, matchingtype uint8, target string) *models.RecordConfig {
	r, err := globalDC.NewRecordConfig(name, defaultTTL, dnsv2.TypeSMIMEA, usage, selector, matchingtype, target)
	panicOnErr(err)
	return r
}

func soa(name string, ns, mbox string, serial, refresh, retry, expire, minttl uint32) *models.RecordConfig {
	r, err := globalDC.NewRecordConfig(name, defaultTTL, dnsv2.TypeSOA, ns, mbox, serial, refresh, retry, expire, minttl)
	panicOnErr(err)
	return r
}

func srv(name string, priority, weight, port uint16, target string) *models.RecordConfig {
	r, err := globalDC.NewRecordConfig(name, defaultTTL, dnsv2.TypeSRV, priority, weight, port, target)
	panicOnErr(err)
	return r
}

func sshfp(name string, algorithm uint8, fingerprint uint8, target string) *models.RecordConfig {
	r, err := globalDC.NewRecordConfig(name, defaultTTL, dnsv2.TypeSSHFP, algorithm, fingerprint, target)
	panicOnErr(err)
	return r
}

func svcb(name string, priority uint16, target string, params string) *models.RecordConfig {
	r, err := globalDC.NewRecordConfig(name, defaultTTL, dnsv2.TypeSVCB, priority, target, params)
	panicOnErr(err)
	return r
}

func ovhdkim(name, target string) *models.RecordConfig {
	return makeOvhNativeRecord(name, target, "DKIM")
}

func ovhspf(name, target string) *models.RecordConfig {
	return makeOvhNativeRecord(name, target, "SPF")
}

func ovhdmarc(name, target string) *models.RecordConfig {
	return makeOvhNativeRecord(name, target, "DMARC")
}

func makeOvhNativeRecord(name, target, rType string) *models.RecordConfig {
	r, err := globalDC.NewRecordConfig(name, defaultTTL, dnsv2.TypeTXT, target)
	panicOnErr(err)
	r.Metadata = map[string]string{}
	r.Metadata["create_ovh_native_record"] = rType
	return r
}

func testgroup(desc string, items ...any) *TestGroup {
	group := &TestGroup{Desc: desc}
	for _, item := range items {
		switch v := item.(type) {
		case requiresFilter:
			if len(group.tests) != 0 {
				fmt.Printf("ERROR: requires() must be before all tc(): %v\n", desc)
				os.Exit(1)
			}
			group.required = append(group.required, v.caps...)
		case notFilter:
			if len(group.tests) != 0 {
				fmt.Printf("ERROR: not() must be before all tc(): %v\n", desc)
				os.Exit(1)
			}
			group.not = append(group.not, v.names...)
		case onlyFilter:
			if len(group.tests) != 0 {
				fmt.Printf("ERROR: only() must be before all tc(): %v\n", desc)
				os.Exit(1)
			}
			group.only = append(group.only, v.names...)
		case alltrueFilter:
			if len(group.tests) != 0 {
				fmt.Printf("ERROR: alltrue() must be before all tc(): %v\n", desc)
				os.Exit(1)
			}
			group.trueflags = append(group.trueflags, v.flags...)
		case domainMetaFilter:
			if len(group.tests) != 0 {
				fmt.Printf("ERROR: domainMeta() must be before all tc(): %v\n", desc)
				os.Exit(1)
			}
			if group.domainMeta == nil {
				group.domainMeta = make(map[string]string)
			}
			maps.Copy(group.domainMeta, v.meta)
		case *TestCase:
			group.tests = append(group.tests, v)
		default:
			fmt.Printf("I don't know about type %T (%v)\n", v, v)
		}
	}
	return group
}

func tc(desc string, recs ...*models.RecordConfig) *TestCase {
	var records models.Records
	var unmanagedItems []*models.UnmanagedConfig
	for _, r := range recs {
		if r == nil {
			continue
		}
		switch r.Type {
		case "IGNORE":
			unmanagedItems = append(unmanagedItems, &models.UnmanagedConfig{
				LabelPattern:  r.Metadata["ignore_LabelPattern"],
				RTypePattern:  r.Metadata["ignore_RTypePattern"],
				TargetPattern: r.Metadata["ignore_TargetPattern"],
			})
			continue
		default:
			records = append(records, r)
		}
	}
	return &TestCase{
		Desc:      desc,
		Records:   records,
		Unmanaged: unmanagedItems,
	}
}

func txt(name, target string) *models.RecordConfig {
	r, err := globalDC.NewRecordConfig(name, defaultTTL, dnsv2.TypeTXT, target)
	panicOnErr(err)
	return r
}

// func (r *models.RecordConfig) ttl(t uint32) *models.RecordConfig {.
func ttl(r *models.RecordConfig, t uint32) *models.RecordConfig {
	r.TTL = t
	return r
}

func tlsa(name string, usage, selector, matchingtype uint8, target string) *models.RecordConfig {
	r, err := globalDC.NewRecordConfig(name, defaultTTL, dnsv2.TypeTLSA, usage, selector, matchingtype, target)
	panicOnErr(err)
	return r
}

func porkbunUrlfwd(name, target string) *models.RecordConfig {
	rc, err := globalDC.NewRecordConfig(name, defaultTTL, privatetypes.TypePORKBUNURLFWD, target)
	panicOnErr(err)
	return rc
}

func url(name, target string) *models.RecordConfig {
	rc, err := globalDC.NewRecordConfig(name, defaultTTL, privatetypes.TypeURL, target)
	panicOnErr(err)
	return rc
}

func url301(name, target string) *models.RecordConfig {
	rc, err := globalDC.NewRecordConfig(name, defaultTTL, privatetypes.TypeURL301, target)
	panicOnErr(err)
	return rc
}

func frame(name, target string) *models.RecordConfig {
	rc, err := globalDC.NewRecordConfig(name, defaultTTL, privatetypes.TypeFRAME, target)
	panicOnErr(err)
	return rc
}

func tcEmptyZone() *TestCase {
	return tc("Empty")
}

type requiresFilter struct {
	caps []providers.Capability
}

func requires(c ...providers.Capability) requiresFilter {
	return requiresFilter{caps: c}
}

type notFilter struct {
	names []string
}

func not(n ...string) notFilter {
	return notFilter{names: n}
}

type onlyFilter struct {
	names []string
}

func only(n ...string) onlyFilter {
	return onlyFilter{names: n}
}

type alltrueFilter struct {
	flags []bool
}

func alltrue(f ...bool) alltrueFilter {
	return alltrueFilter{flags: f}
}

type domainMetaFilter struct {
	meta map[string]string
}

func domainMeta(m map[string]string) domainMetaFilter {
	return domainMetaFilter{meta: m}
}
