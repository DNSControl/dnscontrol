package models

import "testing"

func TestRR(t *testing.T) {
	experiment := RecordConfig{
		Type:         "A",
		Name:         "foo",
		NameFQDN:     "foo.example.com",
		target:       "1.2.3.4",
		TTL:          0,
		MxPreference: 0,
	}
	expected := "foo.example.com.\t300\tIN\tA\t1.2.3.4"
	found := experiment.ToRR().String()
	if found != expected {
		t.Errorf("RR expected (%#v) got (%#v)\n", expected, found)
	}

	experiment = RecordConfig{
		Type:     "CAA",
		Name:     "@",
		NameFQDN: "example.com",
		target:   "mailto:test@example.com",
		TTL:      300,
		CaaTag:   "iodef",
		CaaFlag:  1,
	}
	expected = "example.com.\t300\tIN\tCAA\t1 iodef \"mailto:test@example.com\""
	found = experiment.ToRR().String()
	if found != expected {
		t.Errorf("RR expected (%#v) got (%#v)\n", expected, found)
	}

	experiment = RecordConfig{
		Type:             "TLSA",
		Name:             "@",
		NameFQDN:         "_443._tcp.example.com",
		target:           "abcdef0123456789",
		TTL:              300,
		TlsaUsage:        0,
		TlsaSelector:     0,
		TlsaMatchingType: 1,
	}
	expected = "_443._tcp.example.com.\t300\tIN\tTLSA\t0 0 1 abcdef0123456789"
	found = experiment.ToRR().String()
	if found != expected {
		t.Errorf("RR expected (%#v) got (%#v)\n", expected, found)
	}
}

func TestRRInvalidIPNoPanic(t *testing.T) {
	// A malformed AAAA target (e.g. an incomplete IPv6 address) leaves the
	// stored target unparseable. ToRR must not panic while building the RR so
	// that normalize validation can report the error to the user.
	experiment := RecordConfig{
		Type:     "AAAA",
		Name:     "@",
		NameFQDN: "example.com",
		target:   "2a00:1450:4003:809:1",
		TTL:      300,
	}
	experiment.ToRR()
}

func TestSvcbAutoHintsTargetCombined(t *testing.T) {
	experiment := RecordConfig{
		Type:        "HTTPS",
		Name:        "foo",
		NameFQDN:    "foo.example.com",
		SvcPriority: 1,
		SvcParams:   "alpn=h3,h2 ipv4hint=auto ipv6hint=auto",
		TTL:         300,
	}
	experiment.MustSetTarget(".")

	expected := "1 . alpn=h3,h2 ipv4hint=auto ipv6hint=auto"
	if found := experiment.ToComparableNoTTL(); found != expected {
		t.Errorf("ToComparableNoTTL expected (%#v) got (%#v)\n", expected, found)
	}
}

func TestDowncase(t *testing.T) {
	dc := DomainConfig{Records: Records{
		&RecordConfig{Type: "MX", Name: "lower", target: "targetmx"},
		&RecordConfig{Type: "MX", Name: "UPPER", target: "TARGETMX"},
	}}
	Downcase(dc.Records)
	if !dc.Records.HasRecordTypeName("MX", "lower") {
		t.Errorf("%v: expected (%v) got (%v)\n", dc.Records, false, true)
	}
	if !dc.Records.HasRecordTypeName("MX", "upper") {
		t.Errorf("%v: expected (%v) got (%v)\n", dc.Records, false, true)
	}
	if dc.Records[0].GetTargetField() != "targetmx" {
		t.Errorf("%v: target0 expected (%v) got (%v)\n", dc.Records, "targetmx", dc.Records[0].GetTargetField())
	}
	if dc.Records[1].GetTargetField() != "targetmx" {
		t.Errorf("%v: target1 expected (%v) got (%v)\n", dc.Records, "targetmx", dc.Records[1].GetTargetField())
	}
}
