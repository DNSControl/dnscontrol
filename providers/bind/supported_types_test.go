package bind

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/DNSControl/dnscontrol/v5/models"
	"github.com/DNSControl/dnscontrol/v5/pkg/normalize"
	"github.com/DNSControl/dnscontrol/v5/pkg/providers"
)

func TestWildcardRecordsRoundTrip(t *testing.T) {
	directory := t.TempDir()
	p, err := providers.CreateDNSProvider("BIND", map[string]string{"directory": directory}, nil)
	if err != nil {
		t.Fatal(err)
	}
	dc := models.MustNewDomainConfig("example.com")
	dc.RegistrarName = "NONE"
	dc.DNSProviderInstances = []*models.DNSProviderInstance{{Name: "bind", ProviderType: "BIND"}}
	// Exercise real parsers, including a type that lacks a legacy capability,
	// pseudo-types used by other providers, and every AKAMAITLC answer mode.
	for _, record := range []struct{ label, rtype, data string }{
		{"info", "HINFO", `"CPU" "OS"`},
		{"alias", "ALIAS", "target.example.net."},
		{"route53", "R53_ALIAS", "A target.example.net. false Z123"},
		{"tlca", "AKAMAITLC", "A target.example.net."},
		{"tlcaaaa", "AKAMAITLC", "AAAA target.example.net."},
		{"tlcdual", "AKAMAITLC", "DUAL target.example.net."},
	} {
		dc.AddRecordConfig(dc.MustNewRecordConfigParse(record.label, 300, record.rtype, record.data))
	}
	if errs := normalize.ValidateAndNormalizeConfig(&models.DNSConfig{Domains: []*models.DomainConfig{dc}}); len(errs) != 0 {
		t.Fatal(errs)
	}
	corrections, count, err := p.GetZoneRecordsCorrections(dc, nil)
	if err != nil || len(corrections) != 1 || count == 0 {
		t.Fatalf("corrections=%v count=%d error=%v", corrections, count, err)
	}
	if err := corrections[0].F(); err != nil {
		t.Fatal(err)
	}
	content, err := os.ReadFile(filepath.Join(directory, "example.com.zone"))
	if err != nil {
		t.Fatal(err)
	}
	for _, rtype := range []string{"HINFO", "ALIAS", "R53_ALIAS", "AKAMAITLC"} {
		if !strings.Contains(string(content), "IN "+rtype) {
			t.Fatalf("%s missing from zonefile:\n%s", rtype, content)
		}
	}
	found, err := p.GetZoneRecords(dc)
	if err != nil || len(found) != len(dc.Records) {
		t.Fatalf("records=%v error=%v", found, err)
	}
	corrections, count, err = p.GetZoneRecordsCorrections(dc, found)
	if err != nil || len(corrections) != 0 || count != 0 {
		t.Fatalf("round trip changed records: corrections=%v count=%d error=%v", corrections, count, err)
	}
	if _, err := dc.NewRecordConfigParse("bad", 300, "AKAMAITLC", "A"); err == nil {
		t.Fatal("wildcard bypassed pseudo-type parsing errors")
	}
}
