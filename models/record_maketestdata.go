package models

import (
	"fmt"
	"testing"

	"github.com/DNSControl/dnscontrol/v4/pkg/mustbe"
)

// AddTestRC is a convenience function that uses
// models.NewRecordConfig() to create a models.RecordConfig and adds it to
// a models.DomainConfig. It is for use in unit tests.
// It panics on error.
// It returns a pointer to the newly-created RecordConfig, and adds it to rc.Records.
// If this is not a test, consider dc.AddRecordConfig().
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

// AddTestRCParse is a convenience function that uses
// models.NewRecordConfigParse() to create a models.RecordConfig and adds it to
// a models.DomainConfig. It is for use in unit tests.
// It panics on error.
// It returns a pointer to the newly-created RecordConfig, and adds it to rc.Records.
// If this is not a test, consider dc.AddRecordConfig().
// It returns a pointer to the newly-created RecordConfig, and adds it to rc.Records.
func (dc *DomainConfig) AddTestRCParse(label string, ttl uint32, typeNum uint16, contents string) *RecordConfig {
	rc, err := dc.NewRecordConfigParse(label, ttl, typeNum, contents)
	if err != nil {
		panic(fmt.Sprintf("dc.NewRecordConfigParse() returned %v", err))
	}
	dc.AddRecordConfig(rc)
	return rc
}

// MustNewRecordConfig is like models.NewRecordConfig but panics if initialization fails.
// It is intended for use in variable initializations in unit tests.
// Before using this, consider using models.AddTestRC() first.
func (dc *DomainConfig) MustNewRecordConfig(name string, ttl uint32, typeAny any, args ...any) *RecordConfig {
	rc, err := dc.NewRecordConfig(name, ttl, typeAny, args...)
	if err != nil {
		panic(err)
	}
	return rc
}

// MustNewRecordConfigParse is like NewRecordConfigParse but panics if initialization fails.
// It is intended for use in variable initializations in unit tests.
// Before using this, consider using models.AddTestRCParse() first.
func (dc *DomainConfig) MustNewRecordConfigParse(name string, ttl uint32, typeAny any, data string) *RecordConfig {
	rc, err := dc.NewRecordConfigParse(name, ttl, typeAny, data)
	if err != nil {
		panic(err)
	}
	return rc
}
