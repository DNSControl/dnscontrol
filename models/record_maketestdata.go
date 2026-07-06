package models

import (
	"fmt"
	"testing"

	"github.com/DNSControl/dnscontrol/v4/pkg/mustbe"
)

func (dc *DomainConfig) AddTestRC(t *testing.T, label string, ttl uint32, typeNum uint16, args ...any) *RecordConfig {
	mustbe.ValidArgs(args)
	rc, err := dc.NewRecordConfig(label, ttl, typeNum, args...)
	if err != nil {
		fmt.Printf("dc.NewRecordConfig() returned %v", err)
		t.FailNow()
	}
	dc.AddRecordConfig(rc)
	return rc
}

func (dc *DomainConfig) AddTestRCParse(label string, ttl uint32, typeNum uint16, contents string) *RecordConfig {
	rc, err := dc.NewRecordConfigParse(label, ttl, typeNum, contents)
	if err != nil {
		panic(fmt.Sprintf("dc.NewRecordConfigParse() returned %v", err))
	}
	dc.AddRecordConfig(rc)
	return rc
}
