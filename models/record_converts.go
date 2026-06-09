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

// rc := models.MakeTestRCParse(label, ttl, type, args)                        record_helpers_test.go
// If you need many...
// dc, err := models.NewDomainConfig(zone)                                     domain.go
// dc.AddRecordConfig(models.MakeTestRC(label, ttl, type, args))               domain.go
// dc.AddRecordConfig(models.MakeTestRCParse(label, ttl, type, args))          domain.go
