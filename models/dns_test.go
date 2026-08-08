package models

import (
	"testing"

	dnsv2 "codeberg.org/miekg/dns"
)

func TestDowncase(t *testing.T) {
	dc, err := NewDomainConfig("example.com")
	if err != nil {
		panic("Should not happen")
	}
	dc.AddRecordConfig(dc.MustNewRecordConfig(dc.LabelFromShort("lower"), 0, dnsv2.TypeMX, 10, "targetmx"))
	dc.AddRecordConfig(dc.MustNewRecordConfig(dc.LabelFromShort("UPPER"), 0, dnsv2.TypeMX, 10, "TARGETMX"))
	Downcase(dc.Records)
	if !dc.Records.HasRecordTypeName("MX", "lower") {
		t.Errorf("%v: expected (%v) got (%v)\n", dc.Records, false, true)
	}
	if !dc.Records.HasRecordTypeName("MX", "upper") {
		t.Errorf("%v: expected (%v) got (%v)\n", dc.Records, false, true)
	}
	if dc.Records[0].GetTargetField() != "targetmx.example.com." {
		t.Errorf("%v: target0 expected (%v) got (%v)\n", dc.Records, "targetmx.example.com.", dc.Records[0].GetTargetField())
	}
	if dc.Records[1].GetTargetField() != "targetmx.example.com." {
		t.Errorf("%v: target1 expected (%v) got (%v)\n", dc.Records, "targetmx.example.com.", dc.Records[1].GetTargetField())
	}
}
