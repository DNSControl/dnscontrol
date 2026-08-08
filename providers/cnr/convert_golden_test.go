package cnr

import (
	"testing"

	"github.com/DNSControl/dnscontrol/v5/models"
	"github.com/DNSControl/dnscontrol/v5/pkg/providergolden"
)

func TestCreateRecordStringGolden(t *testing.T) {
	providergolden.CheckToNative(t, "createRecordString",
		func(dc *models.DomainConfig, records models.Records) (string, error) {
			return (&Client{}).createRecordString(records[0], dc.Name)
		})
}
