package normalize

import (
	"fmt"
	"testing"

	dnsv2 "codeberg.org/miekg/dns"
	"github.com/DNSControl/dnscontrol/v5/models"
	"github.com/DNSControl/dnscontrol/v5/pkg/providers"
)

func TestSoaLabelAndTarget(t *testing.T) {
	tests := []struct {
		isError bool
		label   string
		target  string
	}{
		{false, "@", "ns1.foo.com."},

		// Invalid target
		// {true, "@", "ns1.foo.com"},
		// Commented out because MustNewRecordConfig fixes this.

		// Invalid label, only '@' is allowed for SOA records
		{true, "foo.com", "ns1.foo.com."},
	}
	dc := models.MustNewDomainConfig("foo.com")
	for _, test := range tests {
		experiment := fmt.Sprintf("%s %s", test.label, test.target)
		rc := dc.MustNewRecordConfig(test.label, 0, dnsv2.TypeSOA, test.target, "bar.foo.com", 1, 1, 1, 1, 1)
		err := checkTargets(rc, "foo.com")
		if err != nil && !test.isError {
			t.Errorf("%v: Error (%v)\n", experiment, err)
		}
		if err == nil && test.isError {
			t.Errorf("%v: Expected error but got none \n", experiment)
		}
	}
}

func TestCheckSoa(t *testing.T) {
	tests := []struct {
		isError bool
		expire  uint32
		minttl  uint32
		refresh uint32
		retry   uint32
		mbox    string
	}{
		// Expire
		{false, 123, 123, 123, 123, "foo.bar.com."},
		{true, 0, 123, 123, 123, "foo.bar.com."},
		// MinTTL
		{false, 123, 123, 123, 123, "foo.bar.com."},
		{true, 123, 0, 123, 123, "foo.bar.com."},
		// Refresh
		{false, 123, 123, 123, 123, "foo.bar.com."},
		{true, 123, 123, 0, 123, "foo.bar.com."},
		// Retry
		{false, 123, 123, 123, 123, "foo.bar.com."},
		{true, 123, 123, 123, 0, "foo.bar.com."},
		// Serial
		{false, 123, 123, 123, 123, "foo.bar.com."},
		{false, 123, 123, 123, 123, "foo.bar.com."},
		// MBox
		{true, 123, 123, 123, 123, ""},
		{true, 123, 123, 123, 123, "foo@bar.com."},
		{false, 123, 123, 123, 123, "foo.bar.com."},
	}

	for _, test := range tests {
		experiment := fmt.Sprintf("%d %d %d %d %s", test.expire, test.minttl, test.refresh,
			test.retry, test.mbox)
		t.Run(experiment, func(t *testing.T) {
			err := checkSoa(test.expire, test.minttl, test.refresh, test.retry, test.mbox)
			checkError(t, err, test.isError, experiment)
		})
	}
}

func TestCheckMultipleSOAs(t *testing.T) {
	dcSingleSoa := models.MustNewDomainConfig("foo.com")
	dcSingleSoa.RegistrarName = "BIND"
	dcSingleSoa.AddTestRC(t, "@", 0, dnsv2.TypeSOA, "ns1.foo.com.", "bar.foo.com", 1, 1, 1, 1, 1)

	t.Run("single_SOA", func(t *testing.T) {
		errs := checkMultipleSOAs(dcSingleSoa)
		if errs != nil {
			t.Error("checkMultipleSOAs function failed with single SOA record")
		}
	})

	dcTwoSoas := dcSingleSoa
	// Add another SOA record to DC
	dcTwoSoas.AddTestRC(t, "@", 0, dnsv2.TypeSOA, "ns2.foo.com.", "bar.foo.com", 1, 1, 1, 1, 1)

	t.Run("two_SOAs", func(t *testing.T) {
		errs := checkMultipleSOAs(dcTwoSoas)
		if len(errs) < 1 {
			t.Error("checkMultipleSOAs function failed to catch two SOAs")
		}
	})
}

func TestCheckLabel(t *testing.T) {
	tests := []struct {
		label       string
		rType       string
		target      string
		isError     bool
		hasSkipMeta bool
	}{
		{"@", "A", "zap", false, false},
		{"foo.bar", "A", "zap", false, false},
		{"_foo", "A", "zap", false, false},
		{"_foo", "SRV", "zap", false, false},
		{"_foo", "TLSA", "zap", false, false},
		{"_foo", "TXT", "zap", false, false},
		{"_y2", "CNAME", "foo", false, false},
		{"s1._domainkey", "CNAME", "foo", false, false},
		{"_y3", "CNAME", "asfljds.acm-validations.aws.", false, false},
		{"test.foo.tld", "A", "zap", true, false},
		{"test.foo.tld", "A", "zap", false, true},
	}

	for i, test := range tests {
		t.Run(fmt.Sprintf("%s %s", test.label, test.rType), func(t *testing.T) {
			meta := map[string]string{}
			if test.hasSkipMeta {
				meta["skip_fqdn_check"] = "true"
			}
			err := checkLabel(test.label, test.rType, "foo.tld", meta)
			if err != nil && !test.isError {
				t.Errorf("%02d: Expected no error but got %s", i, err)
			}
			if err == nil && test.isError {
				t.Errorf("%02d: Expected error but got none", i)
			}
		})
	}
}

func checkError(t *testing.T, err error, shouldError bool, experiment string) {
	if err != nil && !shouldError {
		t.Errorf("%v: Error (%v)\n", experiment, err)
	}
	if err == nil && shouldError {
		t.Errorf("%v: Expected error but got none \n", experiment)
	}
}

func Test_assert_valid_ipv4(t *testing.T) {
	tests := []struct {
		experiment string
		isError    bool
	}{
		{"1.2.3.4", false},
		{"1.2.3.4/10", true},
		{"1.2.3", true},
		{"foo", true},
	}

	for _, test := range tests {
		err := checkIPv4(test.experiment)
		checkError(t, err, test.isError, test.experiment)
	}
}

func Test_assert_valid_target(t *testing.T) {
	tests := []struct {
		experiment string
		isError    bool
	}{
		{"@", false},
		{"foo", false},
		{"foo.bar.", false},
		{"foo.", false},
		{"foo.bar", true},
		{"foo&bar", true},
		{"foo bar", true},
		{"elb21.freshdesk.com/", true},
		{"elb21.freshdesk.com/.", true},
	}

	for _, test := range tests {
		err := checkTarget(test.experiment)
		checkError(t, err, test.isError, test.experiment)
	}
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

func TestNSAtRoot(t *testing.T) {
	// do not allow ns records for @
	rec := &models.RecordConfig{Type: "NS"}
	rec.SetLabel("test", "foo.com")
	rec.MustSetTarget("ns1.name.com.")
	errs := checkTargets(rec, "foo.com")
	if len(errs) > 0 {
		t.Error("Expect no error with ns record on subdomain")
	}
	rec.SetLabel("@", "foo.com")
	errs = checkTargets(rec, "foo.com")
	if len(errs) != 1 {
		t.Error("Expect error with ns record on @")
	}
}

func TestTransforms(t *testing.T) {
	tests := []struct {
		givenIP         string
		expectedRecords []string
	}{
		{"0.0.5.5", []string{"2.0.5.5"}},
		{"3.0.5.5", []string{"5.5.5.5"}},
		{"7.0.5.5", []string{"9.9.9.9", "10.10.10.10"}},
	}
	const transform = "0.0.0.0~1.0.0.0~2.0.0.0~;   3.0.0.0~4.0.0.0~~5.5.5.5; 7.0.0.0~8.0.0.0~~9.9.9.9,10.10.10.10"
	for i, test := range tests {
		dc := models.MustNewDomainConfig("example.tld")
		rc1 := dc.MustNewRecordConfig("f", 0, dnsv2.TypeA, test.givenIP)
		rc1.Metadata = map[string]string{"transform": transform}
		dc.Records = []*models.RecordConfig{rc1}

		err := applyRecordTransforms(dc)
		if err != nil {
			t.Errorf("error on test %d: %s", i, err)
			continue
		}
		if len(dc.Records) != len(test.expectedRecords) {
			t.Errorf("test %d: expect %d records but found %d", i, len(test.expectedRecords), len(dc.Records))
			continue
		}
		for r, rec := range dc.Records {
			if rec.GetTargetField() != test.expectedRecords[r] {
				t.Errorf("test %d at index %d: records don't match. Expect %s but found %s.", i, r, test.expectedRecords[r], rec.GetTargetField())
				continue
			}
		}
	}
}

func TestCNAMEMutex(t *testing.T) {
	recA := &models.RecordConfig{Type: "CNAME"}
	recA.SetLabel("foo", "foo.example.com")
	recA.MustSetTarget("example.com.")
	tests := []struct {
		rType string
		name  string
		fail  bool
	}{
		{"A", "foo", true},
		{"A", "foo2", false},
		{"CNAME", "foo", true},
		{"CNAME", "foo2", false},
	}
	for _, tst := range tests {
		t.Run(fmt.Sprintf("%s %s", tst.rType, tst.name), func(t *testing.T) {
			recB := &models.RecordConfig{Type: tst.rType}
			recB.SetLabel(tst.name, "example.com")
			recB.MustSetTarget("example2.com.")
			dc := models.MustNewDomainConfig("example.com")
			dc.Records = []*models.RecordConfig{recA, recB}
			errs := checkCNAMEs(dc)
			if errs != nil && !tst.fail {
				t.Error("Got error but expected none")
			}
			if errs == nil && tst.fail {
				t.Error("Expected error but got none")
			}
		})
	}
}

func TestCNAMECloudflareProxied(t *testing.T) {
	// A proxied (flattened) CNAME should be allowed alongside other record types.
	recCNAME := &models.RecordConfig{
		Type:     "CNAME",
		Metadata: map[string]string{"cloudflare_proxy": "on"},
	}
	recCNAME.SetLabel("mail", "mail.example.com")
	recCNAME.MustSetTarget("example.com.")
	recMX := &models.RecordConfig{Type: "MX"}
	recMX.SetLabel("mail", "mail.example.com")
	recMX.MustSetTarget("smtp.example.com.")
	dc := models.MustNewDomainConfig("example.com")
	dc.Records = []*models.RecordConfig{recCNAME, recMX}
	errs := checkCNAMEs(dc)
	if len(errs) != 0 {
		t.Errorf("Expected no errors for proxied CNAME + MX, got: %v", errs)
	}

	// A non-proxied CNAME should still fail.
	recCNAME2 := &models.RecordConfig{Type: "CNAME"}
	recCNAME2.SetLabel("mail", "mail.example.com")
	recCNAME2.MustSetTarget("example.com.")
	dc2 := models.MustNewDomainConfig("example.com")
	dc2.Records = []*models.RecordConfig{recCNAME2, recMX}
	errs2 := checkCNAMEs(dc2)
	if len(errs2) == 0 {
		t.Error("Expected error for non-proxied CNAME + MX, got none")
	}
}

func TestCheckDuplicates(t *testing.T) {
	dc := models.MustNewDomainConfig("example.com")

	// The only difference is the target:
	dc.AddTestRC(t, "www", 0, dnsv2.TypeA, "4.4.4.4")
	dc.AddTestRC(t, "www", 0, dnsv2.TypeA, "5.5.5.5")
	// The only difference is the rType:
	dc.AddTestRC(t, "aaa", 0, dnsv2.TypeNS, "uniquestring.com.")
	dc.AddTestRC(t, "aaa", 0, dnsv2.TypePTR, "uniquestring.com.")
	// Three records each with a different target.
	dc.AddTestRC(t, "@", 0, dnsv2.TypeNS, "ns1.foo.com.")
	dc.AddTestRC(t, "@", 0, dnsv2.TypeNS, "ns2.foo.com.")
	dc.AddTestRC(t, "@", 0, dnsv2.TypeNS, "ns3.foo.com.")

	// NOTE: The comparison ignores ttl. Therefore we don't test that.
	errs := checkDuplicates(dc.Records)
	if len(errs) != 0 {
		t.Errorf("Expected duplicate NOT found but found %q", errs)
	}
}

func TestCheckDuplicates_dup_a(t *testing.T) {
	dc := models.MustNewDomainConfig("example.com")

	// A records that are exact dupliates.
	dc.AddTestRC(t, "@", 0, dnsv2.TypeA, "1.1.1.1")
	dc.AddTestRC(t, "@", 0, dnsv2.TypeA, "1.1.1.1")

	errs := checkDuplicates(dc.Records)
	if len(errs) == 0 {
		t.Error("Expect duplicate found but found none")
	}
}

func TestCheckDuplicates_dup_ns(t *testing.T) {
	dc := models.MustNewDomainConfig("example.com")

	// Three records, the last 2 are duplicates.
	// NB: This is a common issue.
	dc.AddTestRC(t, "@", 0, dnsv2.TypeNS, "ns1.foo.com.")
	dc.AddTestRC(t, "@", 0, dnsv2.TypeNS, "ns2.foo.com.")
	dc.AddTestRC(t, "@", 0, dnsv2.TypeNS, "ns2.foo.com.")

	errs := checkDuplicates(dc.Records)
	if len(errs) == 0 {
		t.Error("Expect duplicate found but found none")
	}
}

func TestCheckRecordSetHasMultipleTTLs_err_1type_2ttl(t *testing.T) {
	dc := models.MustNewDomainConfig("example.com")

	// different ttl per record
	dc.AddTestRC(t, "zzz", 111, dnsv2.TypeA, "4.4.4.4")
	dc.AddTestRC(t, "zzz", 222, dnsv2.TypeA, "4.4.4.5")

	errs := checkRecordSetHasMultipleTTLs(dc.Records)
	if len(errs) == 0 {
		t.Error("Expected error on multiple TTLs under the same label, but got none")
	}
}

func TestCheckRecordSetHasMultipleTTLs_noerr_1type_1ttl(t *testing.T) {
	dc := models.MustNewDomainConfig("example.com")

	// different ttl per record
	dc.AddTestRC(t, "zzz", 111, dnsv2.TypeA, "4.4.4.4")
	dc.AddTestRC(t, "zzz", 111, dnsv2.TypeA, "4.4.4.5")

	errs := checkRecordSetHasMultipleTTLs(dc.Records)
	if len(errs) != 0 {
		t.Errorf("Expected 0 errors (same type, same TTL), but got %d", len(errs))
	}
}

func TestCheckRecordSetHasMultipleTTLs_noerr_2type_2ttl(t *testing.T) {
	dc := models.MustNewDomainConfig("example.com")

	// different record types, different TTLs
	dc.AddTestRC(t, "zzz", 333, dnsv2.TypeA, "4.4.4.4")
	dc.AddTestRC(t, "zzz", 444, dnsv2.TypeNS, "4.4.4.5")

	errs := checkRecordSetHasMultipleTTLs(dc.Records)
	if len(errs) != 0 {
		t.Errorf("Expected 0 errors (different types, different TTLs), but got %d: %v", len(errs), errs)
	}
}

func TestCheckRecordSetHasMultipleTTLs_noerr_2type_1ttl(t *testing.T) {
	dc := models.MustNewDomainConfig("example.com")

	// different record types, different TTLs
	dc.AddTestRC(t, "zzz", 333, dnsv2.TypeA, "4.4.4.4")
	dc.AddTestRC(t, "zzz", 333, dnsv2.TypeNS, "4.4.4.5")

	errs := checkRecordSetHasMultipleTTLs(dc.Records)
	if len(errs) != 0 {
		t.Errorf("Expected 0 errors (different types, same TTLs) but got %d: %v", len(errs), errs)
	}
}

func TestCheckRecordSetHasMultipleTTLs_err_3type_2ttl(t *testing.T) {
	dc := models.MustNewDomainConfig("example.com")

	// different record types, different TTLs
	dc.AddTestRC(t, "zzz", 555, dnsv2.TypeA, "4.4.4.4")
	dc.AddTestRC(t, "zzz", 555, dnsv2.TypeA, "4.4.4.4")
	dc.AddTestRC(t, "zzz", 666, dnsv2.TypeNS, "4.4.4.5")

	errs := checkRecordSetHasMultipleTTLs(dc.Records)
	if len(errs) != 0 {
		t.Errorf("Expected 0 errors (different types, no errors), but got %d: %v", len(errs), errs)
	}
}

func TestCheckRecordSetHasMultipleTTLs_err_3type_3ttl(t *testing.T) {
	dc := models.MustNewDomainConfig("example.com")

	// different record types, different TTLs
	dc.AddTestRC(t, "zzz", 777, dnsv2.TypeA, "4.4.4.4")
	dc.AddTestRC(t, "zzz", 888, dnsv2.TypeA, "4.4.4.4")
	dc.AddTestRC(t, "zzz", 999, dnsv2.TypeNS, "4.4.4.5")

	errs := checkRecordSetHasMultipleTTLs(dc.Records)
	if len(errs) != 1 {
		t.Errorf("Expected 0 errors (different types, 1 error), but got %d: %v", len(errs), errs)
	}
}

func TestTLSAValidation(t *testing.T) {
	dc := models.MustNewDomainConfig("_443._tcp.example.com")
	dc.RegistrarName = "BIND"
	dc.AddTestRC(t, "_443._tcp", 0, dnsv2.TypeTLSA, 4, 1, 1, "abcdef0")

	config := &models.DNSConfig{}
	config.Domains = []*models.DomainConfig{dc}

	errs := ValidateAndNormalizeConfig(config)
	if len(errs) != 1 {
		t.Error("Expect error on invalid TLSA but got none")
	}
}

const (
	ProviderNoDS        = "NO_DS_SUPPORT"
	ProviderFullDS      = "FULL_DS_SUPPORT"
	ProviderChildDSOnly = "CHILD_DS_SUPPORT"
	ProviderBothDSCaps  = "BOTH_DS_CAPABILITIES"
)

func init() {
	providers.RegisterDomainServiceProviderType(ProviderNoDS, providers.DspFuncs{}, providers.DocumentationNotes{})
	providers.RegisterDomainServiceProviderType(ProviderFullDS, providers.DspFuncs{}, providers.DocumentationNotes{
		providers.CanUseDS: providers.Can(),
	})
	providers.RegisterDomainServiceProviderType(ProviderChildDSOnly, providers.DspFuncs{}, providers.DocumentationNotes{
		providers.CanUseDSForChildren: providers.Can(),
	})
	providers.RegisterDomainServiceProviderType(ProviderBothDSCaps, providers.DspFuncs{}, providers.DocumentationNotes{
		providers.CanUseDS:            providers.Can(),
		providers.CanUseDSForChildren: providers.Can(),
	})
}

func Test_DSChecks(t *testing.T) {
	t.Run("no DS support", func(t *testing.T) {
		err := checkProviderDS(ProviderNoDS, nil)
		if err == nil {
			t.Errorf("Provider %s implements no DS capabilities, so should have failed the check", ProviderNoDS)
		}
	})

	t.Run("full DS support", func(t *testing.T) {
		apexDS := models.RecordConfig{Type: "DS"}
		apexDS.SetLabel("@", "example.com")

		childDS := models.RecordConfig{Type: "DS"}
		childDS.SetLabel("child", "example.com")

		records := models.Records{&apexDS, &childDS}

		// check permutations of ProviderCanDS and having both DS caps
		for _, pType := range []string{ProviderFullDS, ProviderBothDSCaps} {
			err := checkProviderDS(pType, records)
			if err != nil {
				t.Errorf("Provider %s implements full DS capabilities and should process the provided records", ProviderFullDS)
			}
		}
	})

	t.Run("child DS support only", func(t *testing.T) {
		apexDS := models.RecordConfig{Type: "DS"}
		apexDS.SetLabel("@", "example.com")

		childDS := models.RecordConfig{Type: "DS"}
		childDS.SetLabel("child", "example.com")

		// this record is included at the apex to check the Type of the
		// recordset is verified to only inspect records with type == DS
		apexA := models.RecordConfig{Type: "A"}
		apexA.SetLabel("@", "example.com")

		t.Run("accepts when child DS records only", func(t *testing.T) {
			records := models.Records{&childDS, &apexA}
			err := checkProviderDS(ProviderChildDSOnly, records)
			if err != nil {
				t.Errorf("Provider %s implements child DS support so the provided records should be accepted",
					ProviderChildDSOnly,
				)
			}
		})

		t.Run("fails with apex and child DS records", func(t *testing.T) {
			records := models.Records{&apexDS, &childDS, &apexA}
			err := checkProviderDS(ProviderChildDSOnly, records)
			if err == nil {
				t.Errorf("Provider %s does not implement DS support at the zone apex, so should reject provided records",
					ProviderChildDSOnly,
				)
			}
		})
	})
}

func TestCheckR53WeightedGroupConsistency_noerr_consistent(t *testing.T) {
	dc := models.MustNewDomainConfig("example.com")

	rc1 := dc.MustNewRecordConfig("@", 0, dnsv2.TypeA, "1.2.3.4")
	rc1.Metadata = map[string]string{"r53_weight": "70", "r53_set_identifier": "primary", "r53_health_check_id": "hc-1"}

	rc2 := dc.MustNewRecordConfig("@", 0, dnsv2.TypeA, "2.3.4.5")
	rc2.Metadata = map[string]string{"r53_weight": "70", "r53_set_identifier": "primary", "r53_health_check_id": "hc-1"}

	dc.Records = []*models.RecordConfig{rc1, rc2}

	errs := checkR53WeightedGroupConsistency(dc.Records)
	if len(errs) != 0 {
		t.Errorf("Expected 0 errors but got %d: %v", len(errs), errs)
	}
}

func TestCheckR53WeightedGroupConsistency_noerr_different_set_ids(t *testing.T) {
	dc := models.MustNewDomainConfig("example.com")

	rc1 := dc.MustNewRecordConfig("@", 0, dnsv2.TypeA, "1.2.3.4")
	rc1.Metadata = map[string]string{"r53_weight": "70", "r53_set_identifier": "primary"}

	rc2 := dc.MustNewRecordConfig("@", 0, dnsv2.TypeA, "5.6.7.8")
	rc2.Metadata = map[string]string{"r53_weight": "30", "r53_set_identifier": "secondary"}

	dc.Records = []*models.RecordConfig{rc1, rc2}

	errs := checkR53WeightedGroupConsistency(dc.Records)
	if len(errs) != 0 {
		t.Errorf("Expected 0 errors but got %d: %v", len(errs), errs)
	}
}

func TestCheckR53WeightedGroupConsistency_noerr_no_metadata(t *testing.T) {
	dc := models.MustNewDomainConfig("example.com")
	dc.AddTestRC(t, "@", 0, dnsv2.TypeA, "1.2.3.4")
	dc.AddTestRC(t, "@", 0, dnsv2.TypeA, "5.6.7.8")
	errs := checkR53WeightedGroupConsistency(dc.Records)
	if len(errs) != 0 {
		t.Errorf("Expected 0 errors but got %d: %v", len(errs), errs)
	}
}

func TestCheckR53WeightedGroupConsistency_err_different_weights(t *testing.T) {
	dc := models.MustNewDomainConfig("example.com")

	rc1 := dc.MustNewRecordConfig("@", 0, dnsv2.TypeA, "1.2.3.4")
	rc1.Metadata = map[string]string{"r53_weight": "70", "r53_set_identifier": "primary"}

	rc2 := dc.MustNewRecordConfig("@", 0, dnsv2.TypeA, "2.3.4.5")
	rc2.Metadata = map[string]string{"r53_weight": "50", "r53_set_identifier": "primary"}

	dc.Records = []*models.RecordConfig{rc1, rc2}

	errs := checkR53WeightedGroupConsistency(dc.Records)
	if len(errs) != 1 {
		t.Errorf("Expected 1 error for inconsistent weights but got %d: %v", len(errs), errs)
	}
}

func TestCheckR53WeightedGroupConsistency_err_different_health_checks(t *testing.T) {
	dc := models.MustNewDomainConfig("example.com")

	rc1 := dc.MustNewRecordConfig("@", 0, dnsv2.TypeA, "1.2.3.4")
	rc1.Metadata = map[string]string{"r53_weight": "70", "r53_set_identifier": "primary", "r53_health_check_id": "hc-1"}

	rc2 := dc.MustNewRecordConfig("@", 0, dnsv2.TypeA, "2.3.4.5")
	rc2.Metadata = map[string]string{"r53_weight": "70", "r53_set_identifier": "primary", "r53_health_check_id": "hc-2"}

	dc.Records = []*models.RecordConfig{rc1, rc2}

	errs := checkR53WeightedGroupConsistency(dc.Records)
	if len(errs) != 1 {
		t.Errorf("Expected 1 error for inconsistent health checks but got %d: %v", len(errs), errs)
	}
}

func TestCheckR53WeightedGroupConsistency_err_both_inconsistent(t *testing.T) {
	dc := models.MustNewDomainConfig("example.com")

	rc1 := dc.MustNewRecordConfig("@", 0, dnsv2.TypeA, "1.2.3.4")
	rc1.Metadata = map[string]string{"r53_weight": "70", "r53_set_identifier": "primary", "r53_health_check_id": "hc-1"}

	rc2 := dc.MustNewRecordConfig("@", 0, dnsv2.TypeA, "2.3.4.5")
	rc2.Metadata = map[string]string{"r53_weight": "50", "r53_set_identifier": "primary", "r53_health_check_id": "hc-2"}

	dc.Records = []*models.RecordConfig{rc1, rc2}

	errs := checkR53WeightedGroupConsistency(dc.Records)
	if len(errs) != 2 {
		t.Errorf("Expected 2 errors (weight + health check) but got %d: %v", len(errs), errs)
	}
}

func Test_errorRepeat(t *testing.T) {
	type args struct {
		label  string
		domain string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "1",
			args: args{label: "foo.bar.com", domain: "bar.com"},
			want: `The name "foo.bar.com.bar.com." is an error (repeats the domain).` +
				` Maybe instead of "foo.bar.com" you intended "foo"?` +
				` If not add DISABLE_REPEATED_DOMAIN_CHECK to this record to permit this as-is.`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := errorRepeat(tt.args.label, tt.args.domain); got != tt.want {
				t.Errorf("errorRepeat() = \n'%s', want\n'%s'", got, tt.want)
			}
		})
	}
}
