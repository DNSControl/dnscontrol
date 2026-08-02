package models

import "testing"

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
	dc, err := NewDomainConfig("example.com")
	if err != nil {
		panic("Should not happen")
	}
	dc.AddRecordConfig(&RecordConfig{Type: "MX", Name: "lower", target: "targetmx"})
	dc.AddRecordConfig(&RecordConfig{Type: "MX", Name: "UPPER", target: "TARGETMX"})
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
