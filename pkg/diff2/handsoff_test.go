package diff2

import (
	"fmt"
	"strings"
	"testing"

	"github.com/DNSControl/dnscontrol/v4/models"
	"github.com/DNSControl/dnscontrol/v4/pkg/dnsrr"
	"github.com/DNSControl/dnscontrol/v4/pkg/js"
	dnsv1 "github.com/miekg/dns"
	testifyrequire "github.com/stretchr/testify/require"
)

// parseZoneContents is copied verbatim from providers/bind/bindProvider.go
// because import cycles and... tests shouldn't depend on huge modules.
func parseZoneContents(content string, zoneName string, zonefileName string) (models.Records, error) {
	zp := dnsv1.NewZoneParser(strings.NewReader(content), zoneName, zonefileName)

	foundRecords := models.Records{}
	for rr, ok := zp.Next(); ok; rr, ok = zp.Next() {
		rec, err := dnsrr.RRtoRCTxtBug(rr, zoneName)
		if err != nil {
			return nil, err
		}
		foundRecords = append(foundRecords, &rec)
	}

	if err := zp.Err(); err != nil {
		return nil, fmt.Errorf("error while parsing '%v': %w", zonefileName, err)
	}
	return foundRecords, nil
}

func showRecs(recs models.Records) string {
	var result strings.Builder
	for _, rec := range recs {
		result.WriteString(rec.GetLabel())
		result.WriteString(" ")
		result.WriteString(rec.Type)
		result.WriteString(" ")
		result.WriteString(rec.GetTargetCombined())
		result.WriteString("\n")
	}
	return result.String()
}

func handsoffHelper(t *testing.T, existingZone, desiredJs string, noPurge bool, resultWanted string) {
	t.Helper()

	existing, err := parseZoneContents(existingZone, "f.com", "no_file_name")
	if err != nil {
		t.Fatal(err)
	}

	dnsconfig, err := js.ExecuteJavascriptString([]byte(desiredJs), false, nil)
	if err != nil {
		panic(err)
	}
	dc := dnsconfig.FindDomain("f.com")
	desired := dc.Records
	absences := dc.EnsureAbsent
	unmanagedConfigs := dc.Unmanaged
	// BUG(tlim): For some reason ExecuteJavascriptString() isn't setting the NameFQDN on records.
	//            This fixes up the records. It is a crass workaround. We should find the real
	//            cause and fix it.
	for i, j := range desired {
		desired[i].SetLabel(j.GetLabel(), "f.com")
	}
	for i, j := range absences {
		absences[i].SetLabel(j.GetLabel(), "f.com")
	}

	ignored, purged, err := processIgnoreAndNoPurge(
		"f.com",
		existing, desired,
		absences,
		unmanagedConfigs,
		noPurge,
	)
	if err != nil {
		t.Fatal(err)
	}

	ignoredRecs := showRecs(ignored)
	purgedRecs := showRecs(purged)
	resultActual := "IGNORED:\n" + ignoredRecs + "FOREIGN:\n" + purgedRecs
	resultWanted = strings.TrimSpace(resultWanted) + "\n"
	resultActual = strings.TrimSpace(resultActual) + "\n"

	existingTxt := showRecs(existing)
	desiredTxt := showRecs(desired)
	debugTxt := "EXISTING:\n" + existingTxt + "DESIRED:\n" + desiredTxt

	if resultWanted != resultActual {
		testifyrequire.Equal(t,
			resultWanted,
			resultActual,
			"GOT =\n```\n%s```\nWANT=\n```%s```\nINPUTS=\n```\n%s\n```\n",
			resultActual,
			resultWanted,
			debugTxt)
	}
}

func Test_purge_empty(t *testing.T) {
	existingZone := `
foo1 IN A 1.1.1.1
foo2 IN A 2.2.2.2
`
	desiredJs := `
D("f.com", "none",
	A("foo1", "1.1.1.1"),
	A("foo2", "2.2.2.2"),
{})
`
	handsoffHelper(t, existingZone, desiredJs, false, `
IGNORED:
FOREIGN:
	`)
}

func Test_purge_1(t *testing.T) {
	existingZone := `
foo1 IN A 1.1.1.1
foo2 IN A 2.2.2.2
foo3 IN A 2.2.2.2
`
	desiredJs := `
D("f.com", "none",
	A("foo1", "1.1.1.1"),
	A("foo2", "2.2.2.2"),
{})
`
	handsoffHelper(t, existingZone, desiredJs, false, `
IGNORED:
FOREIGN:
	`)
}

func Test_nopurge_1(t *testing.T) {
	existingZone := `
foo1 IN A 1.1.1.1
foo2 IN A 2.2.2.2
foo3 IN A 3.3.3.3
`
	desiredJs := `
D("f.com", "none",
	A("foo1", "1.1.1.1"),
	A("foo2", "2.2.2.2"),
{})
`
	handsoffHelper(t, existingZone, desiredJs, true, `
IGNORED:
FOREIGN:
foo3 A 3.3.3.3
	`)
}

func Test_absent_1(t *testing.T) {
	existingZone := `
foo1 IN A 1.1.1.1
foo2 IN A 2.2.2.2
foo3 IN A 3.3.3.3
`
	desiredJs := `
D("f.com", "none",
	A("foo1", "1.1.1.1"),
	A("foo2", "2.2.2.2"),
	A("foo3", "3.3.3.3", ENSURE_ABSENT_REC()),
{})
`
	handsoffHelper(t, existingZone, desiredJs, false, `
IGNORED:
FOREIGN:
	`)
}

func Test_ignore_lab(t *testing.T) {
	existingZone := `
foo1 IN A 1.1.1.1
foo2 IN A 2.2.2.2
foo3 IN A 3.3.3.3
foo3 IN MX 10 mymx.example.com.
`
	desiredJs := `
D("f.com", "none",
	A("foo1", "1.1.1.1"),
	A("foo2", "2.2.2.2"),
	IGNORE_NAME("foo3"),
{})
`
	handsoffHelper(t, existingZone, desiredJs, false, `
IGNORED:
foo3 A 3.3.3.3
foo3 MX 10 mymx.example.com.
FOREIGN:
	`)
}

func Test_ignore_labAndType(t *testing.T) {
	existingZone := `
foo1 IN A 1.1.1.1
foo2 IN A 2.2.2.2
foo3 IN A 3.3.3.3
foo3 IN MX 10 mymx.example.com.
`
	desiredJs := `
D("f.com", "none",
	A("foo1", "1.1.1.1"),
	A("foo2", "2.2.2.2"),
	A("foo3", "3.3.3.3"),
	IGNORE_NAME("foo3", "MX"),
{})
`
	handsoffHelper(t, existingZone, desiredJs, false, `
IGNORED:
foo3 MX 10 mymx.example.com.
FOREIGN:
	`)
}

func Test_ignore_target(t *testing.T) {
	existingZone := `
foo1 IN A 1.1.1.1
foo2 IN A 2.2.2.2
_2222222222222222.cr IN CNAME _333333.nnn.acm-validations.aws.
`
	desiredJs := `
D("f.com", "none",
	A("foo1", "1.1.1.1"),
	A("foo2", "2.2.2.2"),
	MX("foo3", 10, "mymx.example.com."),
	IGNORE_TARGET('**.acm-validations.aws.', 'CNAME'),
{})
`
	handsoffHelper(t, existingZone, desiredJs, false, `
IGNORED:
_2222222222222222.cr CNAME _333333.nnn.acm-validations.aws.
FOREIGN:
	`)
}

// Test_ignore_external_dns tests the IGNORE_EXTERNAL_DNS feature
// using the full handsoff() function.
func Test_ignore_external_dns(t *testing.T) {
	domain := "f.com"

	// Existing zone has external-dns managed records
	existing := models.Records{
		// External-dns TXT ownership record
		makeTestRecord("a-myapp", "TXT", "heritage=external-dns,external-dns/owner=k8s-cluster", domain),
		// The A record managed by external-dns
		makeTestRecord("myapp", "A", "10.0.0.1", domain),
		// Static record not managed by external-dns
		makeTestRecord("static", "A", "1.2.3.4", domain),
		// Another external-dns managed record
		makeTestRecord("cname-api", "TXT", "heritage=external-dns,external-dns/owner=k8s-cluster", domain),
		makeTestRecord("api", "CNAME", "myapp.f.com.", domain),
	}

	// Desired only has the static record
	desired := models.Records{
		makeTestRecord("static", "A", "1.2.3.4", domain),
	}

	// Call handsoff with IGNORE_EXTERNAL_DNS enabled
	result, msgs, err := handsoff(
		domain,
		existing,
		desired,
		nil,   // absences
		nil,   // unmanagedConfigs
		false, // unmanagedSafely
		false, // noPurge
		true,  // ignoreExternalDNS
		"",    // externalDNSPrefix (empty = default)
	)
	if err != nil {
		t.Fatal(err)
	}

	// Check that external-dns records are in the result (so they won't be deleted)
	foundMyappA := false
	foundMyappTXT := false
	foundAPICNAME := false
	foundAPITXT := false
	foundStatic := false

	for _, rec := range result {
		switch {
		case rec.GetLabel() == "myapp" && rec.Type == "A":
			foundMyappA = true
		case rec.GetLabel() == "a-myapp" && rec.Type == "TXT":
			foundMyappTXT = true
		case rec.GetLabel() == "api" && rec.Type == "CNAME":
			foundAPICNAME = true
		case rec.GetLabel() == "cname-api" && rec.Type == "TXT":
			foundAPITXT = true
		case rec.GetLabel() == "static" && rec.Type == "A":
			foundStatic = true
		}
	}

	if !foundMyappA {
		t.Error("Expected myapp A record to be preserved")
	}
	if !foundMyappTXT {
		t.Error("Expected a-myapp TXT record to be preserved")
	}
	if !foundAPICNAME {
		t.Error("Expected api CNAME record to be preserved")
	}
	if !foundAPITXT {
		t.Error("Expected cname-api TXT record to be preserved")
	}
	if !foundStatic {
		t.Error("Expected static A record to be preserved")
	}

	// Check that we got a message about external-dns records
	foundMsg := false
	for _, msg := range msgs {
		if strings.Contains(msg, "IGNORE_EXTERNAL_DNS") {
			foundMsg = true
			break
		}
	}
	if !foundMsg {
		t.Error("Expected message about IGNORE_EXTERNAL_DNS records")
	}
}

// Test_ignore_external_dns_custom_prefix tests IGNORE_EXTERNAL_DNS with custom prefix.
func Test_ignore_external_dns_custom_prefix(t *testing.T) {
	domain := "f.com"

	// Existing zone has external-dns managed records with custom prefix "extdns-"
	existing := models.Records{
		// External-dns TXT ownership record with custom prefix
		makeTestRecord("extdns-www", "TXT", "heritage=external-dns,external-dns/owner=k3s-cluster", domain),
		// The A record managed by external-dns
		makeTestRecord("www", "A", "10.0.0.1", domain),
		// Static record
		makeTestRecord("static", "A", "1.2.3.4", domain),
	}

	// Desired only has the static record
	desired := models.Records{
		makeTestRecord("static", "A", "1.2.3.4", domain),
	}

	// Call handsoff with custom prefix
	result, _, err := handsoff(
		domain,
		existing,
		desired,
		nil,       // absences
		nil,       // unmanagedConfigs
		false,     // unmanagedSafely
		false,     // noPurge
		true,      // ignoreExternalDNS
		"extdns-", // externalDNSPrefix
	)
	if err != nil {
		t.Fatal(err)
	}

	// Check that external-dns records with custom prefix are preserved
	foundWwwA := false
	foundWwwTXT := false

	for _, rec := range result {
		switch {
		case rec.GetLabel() == "www" && rec.Type == "A":
			foundWwwA = true
		case rec.GetLabel() == "extdns-www" && rec.Type == "TXT":
			foundWwwTXT = true
		}
	}

	if !foundWwwA {
		t.Error("Expected www A record to be preserved with custom prefix")
	}
	if !foundWwwTXT {
		t.Error("Expected extdns-www TXT record to be preserved with custom prefix")
	}
}

// Test_ignore_external_dns_conflict tests conflict detection.
func Test_ignore_external_dns_conflict(t *testing.T) {
	domain := "f.com"

	// Existing zone has external-dns managed record
	existing := models.Records{
		makeTestRecord("a-myapp", "TXT", "heritage=external-dns,external-dns/owner=k8s-cluster", domain),
		makeTestRecord("myapp", "A", "10.0.0.1", domain),
	}

	// Desired ALSO has myapp - this is a conflict!
	desired := models.Records{
		makeTestRecord("myapp", "A", "192.168.1.1", domain), // Different IP
	}

	// Call handsoff with IGNORE_EXTERNAL_DNS enabled
	result, msgs, err := handsoff(
		domain,
		existing,
		desired,
		nil,   // absences
		nil,   // unmanagedConfigs
		false, // unmanagedSafely
		false, // noPurge
		true,  // ignoreExternalDNS
		"",    // externalDNSPrefix
	)
	if err != nil {
		t.Fatal(err)
	}

	// Should get a warning about the conflict
	foundConflictWarning := false
	for _, msg := range msgs {
		if strings.Contains(msg, "WARNING") && strings.Contains(msg, "external-dns") {
			foundConflictWarning = true
			break
		}
	}
	if !foundConflictWarning {
		t.Error("Expected warning about conflict between desired and external-dns records")
	}

	// The desired record should be in result (not duplicated)
	myappCount := 0
	for _, rec := range result {
		if rec.GetLabel() == "myapp" && rec.Type == "A" {
			myappCount++
		}
	}
	if myappCount != 1 {
		t.Errorf("Expected exactly 1 myapp A record in result, got %d", myappCount)
	}
}
