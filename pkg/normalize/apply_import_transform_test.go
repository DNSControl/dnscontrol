package normalize

import (
	"strings"
	"testing"

	dnsv2 "codeberg.org/miekg/dns"
	"github.com/DNSControl/dnscontrol/v5/models"
	"github.com/DNSControl/dnscontrol/v5/pkg/privatetypes"
)

func TestImportTransform(t *testing.T) {
	const transformDouble = "0.0.0.0~1.1.1.1~~9.0.0.0,10.0.0.0"
	const transformSingle = "0.0.0.0~1.1.1.1~~8.0.0.0"

	dcSrc := models.MustNewDomainConfig("stackexchange.com")
	dcSrc.AddRecordConfig(dcSrc.MustNewRecordConfig("*", 0, dnsv2.TypeA, "0.0.2.2"))
	dcSrc.AddRecordConfig(dcSrc.MustNewRecordConfig("www", 0, dnsv2.TypeA, "0.0.1.1"))

	dcDst := models.MustNewDomainConfig("internal")
	d1 := dcDst.MustNewRecordConfig("*.stackexchange.com", 0, dnsv2.TypeA, "0.0.3.3")
	d1.Metadata = map[string]string{"transform_table": transformSingle}
	dcDst.AddRecordConfig(d1)

	d2 := dcSrc.MustNewRecordConfig("@", 0, privatetypes.TypeIMPORTTRANSFORM, transformDouble, 299, "com.internal", "internal")
	d2.Metadata["transform_table"] = transformDouble
	dcDst.AddRecordConfig(d2)

	cfg := &models.DNSConfig{}
	cfg.Domains = append(cfg.Domains, dcSrc, dcDst)
	err := cfg.PostProcess()
	if err != nil {
		t.Fatal(err)
	}

	if errs := ValidateAndNormalizeConfig(cfg); len(errs) != 0 {
		for _, err := range errs {
			t.Error(err)
		}
		t.FailNow()
	}
	d := cfg.FindDomain("internal")
	if len(d.Records) != 3 {
		for _, r := range d.Records {
			t.Error(r)
		}
		t.Fatalf("Expected 3 records in internal, but got %d", len(d.Records))
	}
}

// An IMPORT_TRANSFORM that names a domain that is not in the config must be
// reported as an error. It must not crash.
func TestImportTransformMissingDomain(t *testing.T) {
	const transformSingle = "0.0.0.0~1.1.1.1~~8.0.0.0"

	dc := models.MustNewDomainConfig("internal")
	dc.AddRecordConfig(dc.MustNewRecordConfig("www", 0, dnsv2.TypeA, "0.0.3.3"))
	dc.AddRecordConfig(dc.MustNewRecordConfig("@", 0, privatetypes.TypeIMPORTTRANSFORM, transformSingle, 299, "com.internal", "missing.example"))

	cfg := &models.DNSConfig{}
	cfg.Domains = append(cfg.Domains, dc)
	if err := cfg.PostProcess(); err != nil {
		t.Fatal(err)
	}

	var errs []error
	func() {
		defer func() {
			if r := recover(); r != nil {
				t.Fatalf("ValidateAndNormalizeConfig panicked: %v", r)
			}
		}()
		errs = ValidateAndNormalizeConfig(cfg)
	}()

	for _, err := range errs {
		if strings.Contains(err.Error(), `IMPORT_TRANSFORM mentions non-existent domain "missing.example"`) {
			return
		}
	}
	t.Errorf("expected an error about the non-existent domain, got %v", errs)
}

func Test_transform_cname(t *testing.T) {
	tests := []struct {
		experiment string
		expected   string
	}{
		{"@", "old.com.new.com."},
		{"foo", "foo.old.com.new.com."},
		{"foo.bar", "foo.bar.old.com.new.com."},
		{"foo.bar.", "foo.bar.new.com."},
		{"chat.stackexchange.com.", "chat.stackexchange.com.new.com."},
	}

	for _, test := range tests {
		actual := transformCNAME(test.experiment, "old.com", "new.com", "")
		if test.expected != actual {
			t.Errorf("%v: expected (%v) got (%v)\n", test.experiment, test.expected, actual)
		}
	}
}

func Test_transform_cname_strip(t *testing.T) {
	tests := []struct {
		p        []string
		expected string
	}{
		{
			[]string{"ai.meta.stackexchange.com.", "stackexchange.com", "com.internal", "com"},
			"ai.meta.stackexchange.com.internal.",
		},
		{
			[]string{"askubuntu.com.", "askubuntu.com", "com.internal", "com"},
			"askubuntu.com.internal.",
		},
		{
			[]string{"blogoverflow.com.", "stackoverflow.com", "com.internal", "com"},
			"blogoverflow.com.internal.",
		},
		{
			[]string{"careers.stackoverflow.com.", "stackoverflow.com", "com.internal", "com"},
			"careers.stackoverflow.com.internal.",
		},
		{
			[]string{"chat.stackexchange.com.", "askubuntu.com", "com.internal", "com"},
			"chat.stackexchange.com.internal.",
		},
		{
			[]string{"chat.stackexchange.com.", "stackoverflow.com", "com.internal", "com"},
			"chat.stackexchange.com.internal.",
		},
		{
			[]string{"chat.stackexchange.com.", "superuser.com", "com.internal", "com"},
			"chat.stackexchange.com.internal.",
		},
		{
			[]string{"sstatic.net.", "sstatic.net", "net.internal", "net"},
			"sstatic.net.internal.",
		},
		{
			[]string{"stackapps.com.", "stackapps.com", "com.internal", "com"},
			"stackapps.com.internal.",
		},
		{
			[]string{"stackexchange.com.", "stackexchange.com", "com.internal", "com"},
			"stackexchange.com.internal.",
		},
		{
			[]string{"stackoverflow.com.", "stackoverflow.com", "com.internal", "com"},
			"stackoverflow.com.internal.",
		},
		{
			[]string{"superuser.com.", "superuser.com", "com.internal", "com"},
			"superuser.com.internal.",
		},
		{
			[]string{"teststackoverflow.com.", "teststackoverflow.com", "com.internal", "com"},
			"teststackoverflow.com.internal.",
		},
		{
			[]string{"webapps.stackexchange.com.", "stackexchange.com", "com.internal", "com"},
			"webapps.stackexchange.com.internal.",
		},
		//
		{
			[]string{"sstatic.net.", "sstatic.net", "com.internal", "com"},
			"sstatic.net.internal.",
		},
	}

	for _, test := range tests {
		actual := transformCNAME(test.p[0], test.p[1], test.p[2], test.p[3])
		if test.expected != actual {
			t.Errorf("%v: expected (%v) got (%v)\n", test.p, test.expected, actual)
		}
	}
}

// An IMPORT_TRANSFORM with no TTL keeps the TTL of the source record.
func TestImportTransformKeepsSourceTTL(t *testing.T) {
	const transform = "0.0.0.0~255.255.255.255~~0.0.0.0"

	src := models.MustNewDomainConfig("example.com")
	src.AddRecordConfig(src.MustNewRecordConfig("www", 600, dnsv2.TypeA, "192.0.2.1"))

	dst := models.MustNewDomainConfig("internal")
	dst.AddRecordConfig(dst.MustNewRecordConfig("@", 0, privatetypes.TypeIMPORTTRANSFORM, transform, 0, "", src.Name))

	cfg := &models.DNSConfig{Domains: []*models.DomainConfig{src, dst}}
	if err := cfg.PostProcess(); err != nil {
		t.Fatal(err)
	}

	if errs := ValidateAndNormalizeConfig(cfg); len(errs) != 0 {
		t.Fatalf("unexpected errors: %v", errs)
	}
	if len(dst.Records) != 1 {
		t.Fatalf("expected 1 record in %s, got %v", dst.Name, dst.Records)
	}
	rec := dst.Records[0]
	if rec.TTL != 600 {
		t.Errorf("imported record TTL = %d, want 600", rec.TTL)
	}
	if rec.GetLabelFQDN() != "www.example.com.internal" {
		t.Errorf("imported record FQDN = %q, want %q", rec.GetLabelFQDN(), "www.example.com.internal")
	}
}
