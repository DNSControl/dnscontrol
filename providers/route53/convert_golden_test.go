package route53

import (
	"testing"

	r53Types "github.com/aws/aws-sdk-go-v2/service/route53/types"

	"github.com/DNSControl/dnscontrol/v5/models"
	"github.com/DNSControl/dnscontrol/v5/pkg/providergolden"
)

func TestNativeToRecordsGolden(t *testing.T) {
	providergolden.CheckToRC(t, "nativeToRecords",
		func(dc *models.DomainConfig, native r53Types.ResourceRecordSet) (models.Records, error) {
			return nativeToRecords(dc, native, dc.Name)
		})
}
