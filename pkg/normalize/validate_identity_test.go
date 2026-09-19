package normalize

import (
	"strings"
	"testing"

	dnsv2 "codeberg.org/miekg/dns"

	"github.com/DNSControl/dnscontrol/v5/models"
	"github.com/DNSControl/dnscontrol/v5/pkg/providers"
	_ "github.com/DNSControl/dnscontrol/v5/providers/tencentdns"
)

const lineZone = "example.com"

// A provider that stores one record per line and declares no identity function
// keeps the duplicate rules that existed before the hook.
const plainProviderType = "TEST_NO_IDENTITY"

func init() {
	providers.RegisterDomainServiceProviderType(plainProviderType, providers.DspFuncs{}, providers.DocumentationNotes{})
}

// Four lines answer one name: two share a target, two point elsewhere.
var lineRoutes = []struct {
	line   string
	target string
}{
	{"0", "origin.example.net."},
	{"10=1", "origin.example.net."},
	{"10=3", "backup.example.net."},
	{"10=2", "backup.example.net."},
}

func lineDomain(providerType string) *models.DomainConfig {
	dc := models.MustNewDomainConfig(lineZone)
	dc.RegistrarName = "NONE"
	dc.DNSProviderNames = map[string]int{"lines": 1}
	dc.DNSProviderInstances = []*models.DNSProviderInstance{{
		Name:         "lines",
		ProviderType: providerType,
	}}
	return dc
}

func addLineRecords(dc *models.DomainConfig, rType uint16, target string) {
	for _, route := range lineRoutes {
		r := dc.MustNewRecordConfig("edge", 60, rType, target)
		r.Metadata["tencentdns_line_id"] = route.line
		dc.AddRecordConfig(r)
	}
}

func validateDomain(t *testing.T, dc *models.DomainConfig) []error {
	t.Helper()
	return ValidateAndNormalizeConfig(&models.DNSConfig{
		Domains: []*models.DomainConfig{dc},
	})
}

func TestPerLineCNAMEsShareOneName(t *testing.T) {
	dc := lineDomain("TENCENTDNS")
	for _, route := range lineRoutes {
		r := dc.MustNewRecordConfig("edge", 60, dnsv2.TypeCNAME, route.target)
		r.Metadata["tencentdns_line_id"] = route.line
		dc.AddRecordConfig(r)
	}

	if errs := validateDomain(t, dc); len(errs) != 0 {
		t.Fatalf("expected no validation errors, got %v", errs)
	}
}

func TestPerLineAddressesMayShareOneTarget(t *testing.T) {
	dc := lineDomain("TENCENTDNS")
	addLineRecords(dc, dnsv2.TypeA, "192.0.2.10")

	if errs := validateDomain(t, dc); len(errs) != 0 {
		t.Fatalf("expected no validation errors, got %v", errs)
	}
}

func TestPerLineRecordsStillFailWithoutProviderIdentity(t *testing.T) {
	if _, ok := providers.DNSProviderTypes[plainProviderType]; !ok {
		t.Fatalf("test setup: %s is not registered", plainProviderType)
	}
	if providers.GetRecordIdentity(plainProviderType) != nil {
		t.Fatalf("test setup: %s declares an identity function", plainProviderType)
	}
	dc := lineDomain(plainProviderType)
	addLineRecords(dc, dnsv2.TypeA, "192.0.2.10")

	errs := validateDomain(t, dc)
	if len(errs) == 0 {
		t.Fatal("expected duplicate errors for a provider that declares no record identity")
	}
	if !strings.Contains(errs[0].Error(), "exact duplicate record found") {
		t.Fatalf("unexpected first error: %v", errs[0])
	}
}

func TestTwoCNAMEsOnOneLineRemainAnError(t *testing.T) {
	dc := lineDomain("TENCENTDNS")
	first := dc.MustNewRecordConfig("edge", 60, dnsv2.TypeCNAME, lineRoutes[0].target)
	first.Metadata["tencentdns_line_id"] = lineRoutes[0].line
	dc.AddRecordConfig(first)
	second := dc.MustNewRecordConfig("edge", 60, dnsv2.TypeCNAME, "other.example.net.")
	second.Metadata["tencentdns_line_id"] = lineRoutes[0].line
	dc.AddRecordConfig(second)

	errs := validateDomain(t, dc)
	if len(errs) == 0 {
		t.Fatal("expected an error for two CNAMEs on the same line")
	}
	if !strings.Contains(errs[0].Error(), "cannot have multiple CNAMEs with same name") {
		t.Fatalf("unexpected first error: %v", errs[0])
	}
}

func TestWeightDoesNotSplitTheIdentity(t *testing.T) {
	dc := lineDomain("TENCENTDNS")
	for _, weight := range []string{"10", "20"} {
		r := dc.MustNewRecordConfig("weighted", 60, dnsv2.TypeA, "203.0.113.10")
		r.Metadata["tencentdns_line_id"] = "10=1"
		r.Metadata["tencentdns_weight"] = weight
		dc.AddRecordConfig(r)
	}

	errs := validateDomain(t, dc)
	if len(errs) == 0 {
		t.Fatal("expected duplicate detection: the service keys records without weight")
	}
	if !strings.Contains(errs[0].Error(), "exact duplicate record found") {
		t.Fatalf("unexpected first error: %v", errs[0])
	}
}

func TestLineNamesAloneAlsoCarryIdentity(t *testing.T) {
	dc := lineDomain("TENCENTDNS")
	for _, line := range []string{"电信", "联通"} {
		r := dc.MustNewRecordConfig("named", 60, dnsv2.TypeA, "203.0.113.10")
		r.Metadata["tencentdns_line"] = line
		dc.AddRecordConfig(r)
	}

	if errs := validateDomain(t, dc); len(errs) != 0 {
		t.Fatalf("expected no validation errors, got %v", errs)
	}
}

// A record without line metadata answers on the default line, so it is the same
// record as one that names the default line explicitly.
func TestDefaultLineMatchesAnExplicitDefaultLine(t *testing.T) {
	dc := lineDomain("TENCENTDNS")
	dc.AddRecordConfig(dc.MustNewRecordConfig("default-host", 60, dnsv2.TypeA, "192.0.2.10"))
	explicit := dc.MustNewRecordConfig("default-host", 60, dnsv2.TypeA, "192.0.2.10")
	explicit.Metadata["tencentdns_line_id"] = "0"
	dc.AddRecordConfig(explicit)

	errs := validateDomain(t, dc)
	if len(errs) == 0 {
		t.Fatal("expected duplicate detection: no line means the default line")
	}
	if !strings.Contains(errs[0].Error(), "exact duplicate record found") {
		t.Fatalf("unexpected first error: %v", errs[0])
	}
}

// The default line has both a name and an ID, and both describe one record.
func TestDefaultLineNameMatchesTheDefaultLineID(t *testing.T) {
	dc := lineDomain("TENCENTDNS")
	dc.AddRecordConfig(dc.MustNewRecordConfig("default-name", 60, dnsv2.TypeA, "192.0.2.10"))
	named := dc.MustNewRecordConfig("default-name", 60, dnsv2.TypeA, "192.0.2.10")
	named.Metadata["tencentdns_line"] = "默认"
	dc.AddRecordConfig(named)

	errs := validateDomain(t, dc)
	if len(errs) == 0 {
		t.Fatal("expected duplicate detection: the default line has one name and one ID")
	}
}

// Validation runs before the provider reads the zone, so it cannot resolve a
// line name into a line ID. A configuration that describes one line in both
// styles passes here, and the service rejects the duplicate at push time.
func TestMixedLineStylesPassValidation(t *testing.T) {
	dc := lineDomain("TENCENTDNS")
	byName := dc.MustNewRecordConfig("mixed", 60, dnsv2.TypeA, "192.0.2.10")
	byName.Metadata["tencentdns_line"] = "电信"
	dc.AddRecordConfig(byName)
	byID := dc.MustNewRecordConfig("mixed", 60, dnsv2.TypeA, "192.0.2.10")
	byID.Metadata["tencentdns_line_id"] = "10=1"
	dc.AddRecordConfig(byID)

	if errs := validateDomain(t, dc); len(errs) != 0 {
		t.Fatalf("expected no validation errors, got %v", errs)
	}
}
