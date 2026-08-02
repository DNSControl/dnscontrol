package azuredns

import (
	"testing"

	adns "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/dns/armdns"
	"github.com/DNSControl/dnscontrol/v5/models"
	"github.com/DNSControl/dnscontrol/v5/pkg/providergolden"
)

const testDomain = "example.com"

func TestNativeToRecordsGolden(t *testing.T) {
	providergolden.CheckToRC(t, "azuredns_nativetorecords", testDomain,
		func(dc *models.DomainConfig, native adns.RecordSet) ([]*models.RecordConfig, error) {
			return nativeToRecords(&native, dc), nil
		})
}
