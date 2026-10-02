package normalize

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/DNSControl/dnscontrol/v5/models"
	"github.com/DNSControl/dnscontrol/v5/pkg/privatetypes"
	"github.com/DNSControl/dnscontrol/v5/pkg/providers"
)

type supportedTypesAuditor struct{ validationProvider }

func (*supportedTypesAuditor) AuditRecords(models.Records) []error {
	return []error{errors.New("stage 5 audit ran")}
}

type importedRecordsAuditor struct{ validationProvider }

func (*importedRecordsAuditor) AuditRecords(records models.Records) []error {
	for _, record := range records {
		if record.Type != "A" {
			return []error{fmt.Errorf("auditor received %s instead of a copied A record", record.Type)}
		}
	}
	return nil
}

func init() {
	for name, def := range map[string]providers.Definition{
		"S5_DEFAULT": {},
		"S5_DEFAULT_EXCEPT": {SupportedTypes: []string{
			"Default",
			"NS:Cannot",
			"CAA:Cannot",
		}},
		"S5_EMPTY": {SupportedTypes: []string{}},
		"S5_RFC": {SupportedTypes: []string{
			"RFC",
		}},
		"S5_ALL": {SupportedTypes: []string{
			"*",
		}},
		"S5_CHILD": {SupportedTypes: []string{
			"DS:Cannot",
		}, CanUseDSForChildren: providers.Can()},
		"S5_FULL_DS": {SupportedTypes: []string{
			"DS",
		}, CanUseDSForChildren: providers.Cannot()},
		"S5_NO_DS": {SupportedTypes: []string{
			"DS:Cannot",
		}, CanUseDSForChildren: providers.Cannot()},
	} {
		def.FriendlyName = name
		providers.Register[*validationProvider](name, def)
	}
	providers.Register[*supportedTypesAuditor]("S5_AUDIT", providers.Definition{FriendlyName: "Auditor", SupportedTypes: []string{
		"*",
	}})
	providers.Register[*importedRecordsAuditor]("S6_IMPORT_AUDIT", providers.Definition{FriendlyName: "Import auditor", SupportedTypes: []string{
		"A",
	}})
}

func TestExhaustiveRecordValidation(t *testing.T) {
	for _, tc := range []struct {
		provider, label, rtype, data string
		allowed                      bool
	}{
		{"S5_DEFAULT", "@", "A", "192.0.2.1", true},
		{"S5_EMPTY", "@", "A", "192.0.2.1", false},
		{"S5_DEFAULT", "@", "TXT", `"text"`, true},
		{"S5_DEFAULT", "child", "NS", "ns.example.net.", true},
		{"S5_DEFAULT", "@", "NS", "ns.example.net.", false},
		{"S5_DEFAULT", "@", "CAA", `0 issue "ca.example.net"`, true},
		{"S5_DEFAULT", "_sip._tcp", "SRV", "0 5 5060 sip.example.net.", true},
		{"S5_DEFAULT_EXCEPT", "child", "NS", "ns.example.net.", false},
		{"S5_DEFAULT_EXCEPT", "@", "CAA", `0 issue "ca.example.net"`, false},
		{"S5_DEFAULT_EXCEPT", "@", "TXT", `"text"`, true},
		{ProviderNoDS, "@", "TXT", `"text"`, true},
		{ProviderNoDS, "child", "NS", "ns.example.net.", true},
		{"S5_RFC", "@", "TXT", `"text"`, true},
		{"S5_RFC", "@", "HINFO", `"CPU" "OS"`, true},
		{"S5_RFC", "@", "ALIAS", "target.example.net.", false},
		{"S5_ALL", "@", "ALIAS", "target.example.net.", true},
		{"S5_ALL", "edge", "AKAMAITLC", "A target.example.net.", true},
		{"S5_ALL", "edge", "AKAMAITLC", "AAAA target.example.net.", true},
		{"S5_ALL", "edge", "AKAMAITLC", "DUAL target.example.net.", true},
		{"S5_DEFAULT", "edge", "AKAMAITLC", "A target.example.net.", false},
		{"S5_CHILD", "child", "DS", "12345 8 2 ABCD", true},
		{"S5_CHILD", "@", "DS", "12345 8 2 ABCD", false},
		{"S5_FULL_DS", "@", "DS", "12345 8 2 ABCD", true},
		{"S5_FULL_DS", "child", "DS", "12345 8 2 ABCD", true},
		{"S5_NO_DS", "child", "DS", "12345 8 2 ABCD", false},
	} {
		t.Run(tc.provider+"/"+tc.rtype+"/"+tc.label+"/"+tc.data, func(t *testing.T) {
			dc := lineDomain(tc.provider)
			r := dc.MustNewRecordConfigParse(tc.label, 300, tc.rtype, tc.data)
			dc.AddRecordConfig(r)
			errs := validateDomain(t, dc)
			if (len(errs) == 0) != tc.allowed {
				t.Fatalf("allowed=%v, errors=%v", tc.allowed, errs)
			}
		})
	}
}

func TestSupportedTypesRetainsOtherValidation(t *testing.T) {
	t.Run("auditor", func(t *testing.T) {
		dc := lineDomain("S5_AUDIT")
		dc.AddRecordConfig(dc.MustNewRecordConfig("www", 300, "A", "192.0.2.1"))
		if errs := validateDomain(t, dc); !strings.Contains(fmt.Sprint(errs), "stage 5 audit ran") {
			t.Fatalf("errors = %v", errs)
		}
	})
	t.Run("placement", func(t *testing.T) {
		dc := lineDomain("S5_ALL")
		dc.AddRecordConfig(dc.MustNewRecordConfig("@", 300, "CNAME", "target.example.net."))
		if errs := validateDomain(t, dc); !strings.Contains(fmt.Sprint(errs), "cannot create CNAME record for bare domain") {
			t.Fatalf("errors = %v", errs)
		}
	})
	t.Run("unknown", func(t *testing.T) {
		dc := lineDomain("S5_ALL")
		r := dc.MustNewRecordConfig("www", 300, "A", "192.0.2.1")
		r.Type = "UNRECOGNIZED"
		dc.AddRecordConfig(r)
		if errs := validateDomain(t, dc); !strings.Contains(fmt.Sprint(errs), "unknown record type UNRECOGNIZED") {
			t.Fatalf("errors = %v", errs)
		}
	})
	t.Run("every provider", func(t *testing.T) {
		dc := lineDomain("S5_ALL")
		dc.DNSProviderInstances = append(dc.DNSProviderInstances, &models.DNSProviderInstance{Name: "second", ProviderType: "S5_DEFAULT"})
		dc.AddRecordConfig(dc.MustNewRecordConfigParse("@", 300, "HINFO", `"CPU" "OS"`))
		if errs := validateDomain(t, dc); !strings.Contains(fmt.Sprint(errs), "S5_DEFAULT does not support") {
			t.Fatalf("errors = %v", errs)
		}
	})
	t.Run("operational", func(t *testing.T) {
		dc := lineDomain("S5_ALL")
		dc.AutoDNSSEC = "on"
		if err := checkProviderCapabilities(dc); err == nil || !strings.Contains(err.Error(), "AUTODNSSEC") {
			t.Fatalf("error = %v", err)
		}
	})
}

func TestImportTransformChecksCopiedRecords(t *testing.T) {
	for _, tc := range []struct {
		name, provider string
		copyRecord     bool
		wantError      string
	}{
		{"default", "S5_DEFAULT", true, ""},
		{"RFC", "S5_RFC", true, ""},
		{"wildcard", "S5_ALL", true, ""},
		{"auditor sees only copied records", "S6_IMPORT_AUDIT", true, ""},
		{"empty declaration accepts command", "S5_EMPTY", false, ""},
		{"empty declaration rejects copied records", "S5_EMPTY", true, "uses A records"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			source := models.MustNewDomainConfig("source.example")
			if tc.copyRecord {
				source.AddRecordConfig(source.MustNewRecordConfig("www", 300, "A", "192.0.2.1"))
			}
			dest := lineDomain(tc.provider)
			transform := "0.0.0.0~255.255.255.255~~0.0.0.0"
			r := dest.MustNewRecordConfig("@", 300, privatetypes.TypeIMPORTTRANSFORM, transform, 300, "", source.Name)
			r.Metadata["transform_table"] = transform
			dest.AddRecordConfig(r)
			config := &models.DNSConfig{Domains: []*models.DomainConfig{source, dest}}
			if err := config.PostProcess(); err != nil {
				t.Fatal(err)
			}
			errs := ValidateAndNormalizeConfig(config)
			if tc.wantError == "" && len(errs) != 0 || tc.wantError != "" && (len(errs) != 1 || !strings.Contains(errs[0].Error(), tc.wantError)) {
				t.Fatalf("want error %q, got %v", tc.wantError, errs)
			}
			if tc.copyRecord {
				if len(dest.Records) != 1 || dest.Records[0].Type != "A" {
					t.Fatalf("command did not produce an A record: %v", dest.Records)
				}
			} else if len(dest.Records) != 0 {
				t.Fatalf("command was not consumed: %v", dest.Records)
			}
		})
	}
}

func TestCatalogValidationWithoutResolvedProvider(t *testing.T) {
	for _, pTypes := range [][]string{nil, {"-"}, {plainProviderType}} {
		r := &models.RecordConfig{Type: "UNKNOWN_TYPE"}
		if err := validateSupportedRecordTypes(r, "example.com", pTypes); err == nil || !strings.Contains(err.Error(), "unknown record type") {
			t.Fatalf("providers=%v, error=%v", pTypes, err)
		}
		r.Type = "URL"
		if err := validateSupportedRecordTypes(r, "example.com", pTypes); err != nil {
			t.Fatalf("providers=%v, error=%v", pTypes, err)
		}
	}
}
