package normalize

import (
	"fmt"
	"testing"

	"github.com/DNSControl/dnscontrol/v4/models"
	"github.com/DNSControl/dnscontrol/v4/pkg/providers"
)

func TestSoaLabelAndTarget(t *testing.T) {
	tests := []struct {
		isError bool
		label   string
		target  string
	}{
		{false, "@", "ns1.foo.com."},
		// Invalid target
		{true, "@", "ns1.foo.com"},
		// Invalid label, only '@' is allowed for SOA records
		{true, "foo.com", "ns1.foo.com."},
	}
	for _, test := range tests {
		experiment := fmt.Sprintf("%s %s", test.label, test.target)
		rc := makeRC(test.label, "foo.com", test.target, models.RecordConfig{
			Type:      "SOA",
			SoaExpire: 1, SoaMinttl: 1, SoaRefresh: 1, SoaRetry: 1, SoaSerial: 1, SoaMbox: "bar.foo.com",
		})
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
	dcSingleSoa := &models.DomainConfig{
		Name:          "foo.com",
		RegistrarName: "BIND",
		Records: []*models.RecordConfig{
			makeRC("@", "foo.com", "ns1.foo.com.", models.RecordConfig{
				Type:      "SOA",
				SoaExpire: 1, SoaMinttl: 1, SoaRefresh: 1, SoaRetry: 1, SoaSerial: 1, SoaMbox: "bar.foo.com",
			}),
		},
	}

	t.Run("single_SOA", func(t *testing.T) {
		errs := checkMultipleSOAs(dcSingleSoa)
		if errs != nil {
			t.Error("checkMultipleSOAs function failed with single SOA record")
		}
	})

	dcTwoSoas := dcSingleSoa
	// Add another SOA record to DC
	dcTwoSoas.Records = append(dcTwoSoas.Records,
		makeRC("@", "foo.com", "ns2.foo.com.", models.RecordConfig{
			Type:      "SOA",
			SoaExpire: 1, SoaMinttl: 1, SoaRefresh: 1, SoaRetry: 1, SoaSerial: 1, SoaMbox: "bar.foo.com",
		}),
	)

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
		dc := &models.DomainConfig{
			Records: []*models.RecordConfig{
				makeRC("f", "example.tld", test.givenIP, models.RecordConfig{Type: "A", Metadata: map[string]string{"transform": transform}}),
			},
		}
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
			dc := &models.DomainConfig{
				Name:    "example.com",
				Records: []*models.RecordConfig{recA, recB},
			}
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
	dc := &models.DomainConfig{
		Name:    "example.com",
		Records: []*models.RecordConfig{recCNAME, recMX},
	}
	errs := checkCNAMEs(dc)
	if len(errs) != 0 {
		t.Errorf("Expected no errors for proxied CNAME + MX, got: %v", errs)
	}

	// A non-proxied CNAME should still fail.
	recCNAME2 := &models.RecordConfig{Type: "CNAME"}
	recCNAME2.SetLabel("mail", "mail.example.com")
	recCNAME2.MustSetTarget("example.com.")
	dc2 := &models.DomainConfig{
		Name:    "example.com",
		Records: []*models.RecordConfig{recCNAME2, recMX},
	}
	errs2 := checkCNAMEs(dc2)
	if len(errs2) == 0 {
		t.Error("Expected error for non-proxied CNAME + MX, got none")
	}
}

func TestCAAValidation(t *testing.T) {
	config := &models.DNSConfig{
		Domains: []*models.DomainConfig{
			{
				Name:          "example.com",
				RegistrarName: "BIND",
				Records: []*models.RecordConfig{
					makeRC("@", "example.com", "example.com", models.RecordConfig{Type: "CAA", CaaTag: "invalid"}),
				},
			},
		},
	}
	errs := ValidateAndNormalizeConfig(config)
	if len(errs) != 1 {
		t.Error("Expect error on invalid CAA but got none")
	}
}

func TestCheckDuplicates(t *testing.T) {
	records := []*models.RecordConfig{
		// The only difference is the target:
		makeRC("www", "example.com", "4.4.4.4", models.RecordConfig{Type: "A"}),
		makeRC("www", "example.com", "5.5.5.5", models.RecordConfig{Type: "A"}),
		// The only difference is the rType:
		makeRC("aaa", "example.com", "uniquestring.com.", models.RecordConfig{Type: "NS"}),
		makeRC("aaa", "example.com", "uniquestring.com.", models.RecordConfig{Type: "PTR"}),
		// Three records each with a different target.
		makeRC("@", "example.com", "ns1.foo.com.", models.RecordConfig{Type: "NS"}),
		makeRC("@", "example.com", "ns2.foo.com.", models.RecordConfig{Type: "NS"}),
		makeRC("@", "example.com", "ns3.foo.com.", models.RecordConfig{Type: "NS"}),
		// NOTE: The comparison ignores ttl. Therefore we don't test that.
	}
	errs := checkDuplicates(records)
	if len(errs) != 0 {
		t.Errorf("Expected duplicate NOT found but found %q", errs)
	}
}

func TestCheckDuplicates_dup_a(t *testing.T) {
	records := []*models.RecordConfig{
		// A records that are exact dupliates.
		makeRC("@", "example.com", "1.1.1.1", models.RecordConfig{Type: "A"}),
		makeRC("@", "example.com", "1.1.1.1", models.RecordConfig{Type: "A"}),
	}
	errs := checkDuplicates(records)
	if len(errs) == 0 {
		t.Error("Expect duplicate found but found none")
	}
}

func TestCheckDuplicates_dup_ns(t *testing.T) {
	records := []*models.RecordConfig{
		// Three records, the last 2 are duplicates.
		// NB: This is a common issue.
		makeRC("@", "example.com", "ns1.foo.com.", models.RecordConfig{Type: "NS"}),
		makeRC("@", "example.com", "ns2.foo.com.", models.RecordConfig{Type: "NS"}),
		makeRC("@", "example.com", "ns2.foo.com.", models.RecordConfig{Type: "NS"}),
	}
	errs := checkDuplicates(records)
	if len(errs) == 0 {
		t.Error("Expect duplicate found but found none")
	}
}

func TestCheckRecordSetHasMultipleTTLs_err_1type_2ttl(t *testing.T) {
	records := []*models.RecordConfig{
		// different ttl per record
		makeRC("zzz", "example.com", "4.4.4.4", models.RecordConfig{Type: "A", TTL: 111}),
		makeRC("zzz", "example.com", "4.4.4.5", models.RecordConfig{Type: "A", TTL: 222}),
	}
	errs := checkRecordSetHasMultipleTTLs(records)
	if len(errs) == 0 {
		t.Error("Expected error on multiple TTLs under the same label, but got none")
	}
}

func TestCheckRecordSetHasMultipleTTLs_noerr_1type_1ttl(t *testing.T) {
	records := []*models.RecordConfig{
		// different ttl per record
		makeRC("zzz", "example.com", "4.4.4.4", models.RecordConfig{Type: "A", TTL: 111}),
		makeRC("zzz", "example.com", "4.4.4.5", models.RecordConfig{Type: "A", TTL: 111}),
	}
	errs := checkRecordSetHasMultipleTTLs(records)
	if len(errs) != 0 {
		t.Errorf("Expected 0 errors (same type, same TTL), but got %d", len(errs))
	}
}

func TestCheckRecordSetHasMultipleTTLs_noerr_2type_2ttl(t *testing.T) {
	records := []*models.RecordConfig{
		// different record types, different TTLs
		makeRC("zzz", "example.com", "4.4.4.4", models.RecordConfig{Type: "A", TTL: 333}),
		makeRC("zzz", "example.com", "4.4.4.5", models.RecordConfig{Type: "NS", TTL: 444}),
	}
	errs := checkRecordSetHasMultipleTTLs(records)
	if len(errs) != 0 {
		t.Errorf("Expected 0 errors (different types, different TTLs), but got %d: %v", len(errs), errs)
	}
}

func TestCheckRecordSetHasMultipleTTLs_noerr_2type_1ttl(t *testing.T) {
	records := []*models.RecordConfig{
		// different record types, different TTLs
		makeRC("zzz", "example.com", "4.4.4.4", models.RecordConfig{Type: "A", TTL: 333}),
		makeRC("zzz", "example.com", "4.4.4.5", models.RecordConfig{Type: "NS", TTL: 333}),
	}
	errs := checkRecordSetHasMultipleTTLs(records)
	if len(errs) != 0 {
		t.Errorf("Expected 0 errors (different types, same TTLs) but got %d: %v", len(errs), errs)
	}
}

func TestCheckRecordSetHasMultipleTTLs_err_3type_2ttl(t *testing.T) {
	records := []*models.RecordConfig{
		// different record types, different TTLs
		makeRC("zzz", "example.com", "4.4.4.4", models.RecordConfig{Type: "A", TTL: 555}),
		makeRC("zzz", "example.com", "4.4.4.4", models.RecordConfig{Type: "A", TTL: 555}),
		makeRC("zzz", "example.com", "4.4.4.5", models.RecordConfig{Type: "NS", TTL: 666}),
	}
	errs := checkRecordSetHasMultipleTTLs(records)
	if len(errs) != 0 {
		t.Errorf("Expected 0 errors (different types, no errors), but got %d: %v", len(errs), errs)
	}
}

func TestCheckRecordSetHasMultipleTTLs_err_3type_3ttl(t *testing.T) {
	records := []*models.RecordConfig{
		// different record types, different TTLs
		makeRC("zzz", "example.com", "4.4.4.4", models.RecordConfig{Type: "A", TTL: 777}),
		makeRC("zzz", "example.com", "4.4.4.4", models.RecordConfig{Type: "A", TTL: 888}),
		makeRC("zzz", "example.com", "4.4.4.5", models.RecordConfig{Type: "NS", TTL: 999}),
	}
	errs := checkRecordSetHasMultipleTTLs(records)
	if len(errs) != 1 {
		t.Errorf("Expected 0 errors (different types, 1 error), but got %d: %v", len(errs), errs)
	}
}

func TestTLSAValidation(t *testing.T) {
	config := &models.DNSConfig{
		Domains: []*models.DomainConfig{
			{
				Name:          "_443._tcp.example.com",
				RegistrarName: "BIND",
				Records: []*models.RecordConfig{
					makeRC("_443._tcp", "_443._tcp.example.com", "abcdef0", models.RecordConfig{
						Type: "TLSA", TlsaUsage: 4, TlsaSelector: 1, TlsaMatchingType: 1,
					}),
				},
			},
		},
	}
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
	records := []*models.RecordConfig{
		makeRC("@", "example.com", "1.2.3.4", models.RecordConfig{
			Type:     "A",
			Metadata: map[string]string{"r53_weight": "70", "r53_set_identifier": "primary", "r53_health_check_id": "hc-1"},
		}),
		makeRC("@", "example.com", "2.3.4.5", models.RecordConfig{
			Type:     "A",
			Metadata: map[string]string{"r53_weight": "70", "r53_set_identifier": "primary", "r53_health_check_id": "hc-1"},
		}),
	}
	errs := checkR53WeightedGroupConsistency(records)
	if len(errs) != 0 {
		t.Errorf("Expected 0 errors but got %d: %v", len(errs), errs)
	}
}

func TestCheckR53WeightedGroupConsistency_noerr_different_set_ids(t *testing.T) {
	records := []*models.RecordConfig{
		makeRC("@", "example.com", "1.2.3.4", models.RecordConfig{
			Type:     "A",
			Metadata: map[string]string{"r53_weight": "70", "r53_set_identifier": "primary"},
		}),
		makeRC("@", "example.com", "5.6.7.8", models.RecordConfig{
			Type:     "A",
			Metadata: map[string]string{"r53_weight": "30", "r53_set_identifier": "secondary"},
		}),
	}
	errs := checkR53WeightedGroupConsistency(records)
	if len(errs) != 0 {
		t.Errorf("Expected 0 errors but got %d: %v", len(errs), errs)
	}
}

func TestCheckR53WeightedGroupConsistency_noerr_no_metadata(t *testing.T) {
	records := []*models.RecordConfig{
		makeRC("@", "example.com", "1.2.3.4", models.RecordConfig{Type: "A"}),
		makeRC("@", "example.com", "5.6.7.8", models.RecordConfig{Type: "A"}),
	}
	errs := checkR53WeightedGroupConsistency(records)
	if len(errs) != 0 {
		t.Errorf("Expected 0 errors but got %d: %v", len(errs), errs)
	}
}

func TestCheckR53WeightedGroupConsistency_err_different_weights(t *testing.T) {
	records := []*models.RecordConfig{
		makeRC("@", "example.com", "1.2.3.4", models.RecordConfig{
			Type:     "A",
			Metadata: map[string]string{"r53_weight": "70", "r53_set_identifier": "primary"},
		}),
		makeRC("@", "example.com", "2.3.4.5", models.RecordConfig{
			Type:     "A",
			Metadata: map[string]string{"r53_weight": "50", "r53_set_identifier": "primary"},
		}),
	}
	errs := checkR53WeightedGroupConsistency(records)
	if len(errs) != 1 {
		t.Errorf("Expected 1 error for inconsistent weights but got %d: %v", len(errs), errs)
	}
}

func TestCheckR53WeightedGroupConsistency_err_different_health_checks(t *testing.T) {
	records := []*models.RecordConfig{
		makeRC("@", "example.com", "1.2.3.4", models.RecordConfig{
			Type:     "A",
			Metadata: map[string]string{"r53_weight": "70", "r53_set_identifier": "primary", "r53_health_check_id": "hc-1"},
		}),
		makeRC("@", "example.com", "2.3.4.5", models.RecordConfig{
			Type:     "A",
			Metadata: map[string]string{"r53_weight": "70", "r53_set_identifier": "primary", "r53_health_check_id": "hc-2"},
		}),
	}
	errs := checkR53WeightedGroupConsistency(records)
	if len(errs) != 1 {
		t.Errorf("Expected 1 error for inconsistent health checks but got %d: %v", len(errs), errs)
	}
}

func TestCheckR53WeightedGroupConsistency_err_both_inconsistent(t *testing.T) {
	records := []*models.RecordConfig{
		makeRC("@", "example.com", "1.2.3.4", models.RecordConfig{
			Type:     "A",
			Metadata: map[string]string{"r53_weight": "70", "r53_set_identifier": "primary", "r53_health_check_id": "hc-1"},
		}),
		makeRC("@", "example.com", "2.3.4.5", models.RecordConfig{
			Type:     "A",
			Metadata: map[string]string{"r53_weight": "50", "r53_set_identifier": "primary", "r53_health_check_id": "hc-2"},
		}),
	}
	errs := checkR53WeightedGroupConsistency(records)
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
