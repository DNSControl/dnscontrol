package azuredns

import (
	"testing"

	adns "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/dns/armdns"
	"github.com/DNSControl/dnscontrol/v5/models"
	"github.com/DNSControl/dnscontrol/v5/pkg/providergolden"
)

func TestNativeToRecordsGolden(t *testing.T) {
	providergolden.CheckToRC(t, "nativeToRecords",
		func(dc *models.DomainConfig, native adns.RecordSet) (models.Records, error) {
			return nativeToRecords(&native, dc), nil
		})
}
