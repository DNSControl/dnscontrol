package normalize

import (
	"encoding/json"

	"github.com/DNSControl/dnscontrol/v5/models"
	"github.com/DNSControl/dnscontrol/v5/pkg/providers"
)

// validationProvider supplies metadata without constructing an API client.
type validationProvider struct{}

func (*validationProvider) Initialize(map[string]string, json.RawMessage, *providers.CreateOptions) error {
	panic("validation must not initialize providers")
}
func (*validationProvider) AuditRecords(models.Records) []error { return nil }
func (*validationProvider) GetNameservers(string) ([]*models.Nameserver, error) {
	return nil, nil
}
func (*validationProvider) GetZoneRecords(*models.DomainConfig) (models.Records, error) {
	return nil, nil
}
func (*validationProvider) GetZoneRecordsCorrections(*models.DomainConfig, models.Records) ([]*models.Correction, int, error) {
	return nil, 0, nil
}
