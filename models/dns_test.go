package models

import (
	"testing"
)

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
