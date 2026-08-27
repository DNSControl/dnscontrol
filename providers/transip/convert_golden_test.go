package transip

import (
	"testing"

	"github.com/DNSControl/dnscontrol/v5/models"
	"github.com/DNSControl/dnscontrol/v5/pkg/providergolden"
	"github.com/transip/gotransip/v6/domain"
)

func TestRecordToNativeGolden(t *testing.T) {
	providergolden.CheckToNative(t, "recordToNative", func(_ *models.DomainConfig, records models.Records) (domain.DNSEntry, error) {
		return recordToNative(records[0])
	})
}
