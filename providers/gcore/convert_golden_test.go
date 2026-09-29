package gcore

import (
	"testing"

	"github.com/DNSControl/dnscontrol/v5/models"
	"github.com/DNSControl/dnscontrol/v5/pkg/providergolden"
	dnssdk "github.com/G-Core/gcore-dns-sdk-go"
)

func TestNativeToRecordsGolden(t *testing.T) {
	providergolden.CheckToRC(t, "nativeToRecords",
		func(dc *models.DomainConfig, native gcoreRRSetExtended) (models.Records, error) {
			return nativeToRecords(native, dc)
		})
}

func TestRecordsToNativeGolden(t *testing.T) {
	providergolden.CheckToNative(t, "recordsToNative",
		func(_ *models.DomainConfig, records models.Records) (*dnssdk.RRSet, error) {
			return recordsToNative(records, records[0].Key())
		})
}
