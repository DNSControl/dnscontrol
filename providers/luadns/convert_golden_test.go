package luadns

import (
	"testing"

	api "github.com/luadns/luadns-go"

	"github.com/DNSControl/dnscontrol/v5/models"
	"github.com/DNSControl/dnscontrol/v5/pkg/providergolden"
)

func TestNativeToRecordGolden(t *testing.T) {
	providergolden.CheckToRC(t, "luadns_nativetorecord",
		func(dc *models.DomainConfig, native api.Record) ([]*models.RecordConfig, error) {
			rc, err := nativeToRecord(dc, &native)
			return []*models.RecordConfig{rc}, err
		})
}

func TestRecordsToNativeGolden(t *testing.T) {
	providergolden.CheckToNative(t, "luadns_recordstonative",
		func(rc *models.RecordConfig) (*api.RR, error) {
			return recordsToNative([]*models.RecordConfig{rc})[0], nil
		})
}
