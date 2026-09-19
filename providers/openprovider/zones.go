package openprovider

import (
	"fmt"
	"strings"

	"github.com/DNSControl/dnscontrol/v5/models"
	"golang.org/x/net/publicsuffix"
)

// EnsureZoneExists creates a standard master DNS zone when it is absent.
func (p *openproviderProvider) EnsureZoneExists(dc *models.DomainConfig) error {
	_, err := p.client.getZone(dc.Name)
	if err == nil {
		return nil
	}
	if !isNotFound(err) {
		return err
	}

	domain, err := splitZoneName(dc.Name)
	if err != nil {
		return err
	}
	desired := providerDomainConfig(dc)
	desired.Records = managedZoneRecords(desired.Records)
	if len(desired.Records) == 0 {
		return fmt.Errorf("OPENPROVIDER: cannot create DNS zone %q without at least one record", dc.Name)
	}
	normalizeTTLs(desired.Records)
	records := make([]apiRecord, 0, len(desired.Records))
	for _, record := range desired.Records {
		records = append(records, fromRecordConfig(record))
	}
	return p.client.createZone(createZoneRequest{
		Domain:               domain,
		Type:                 "master",
		Records:              records,
		IsSpamExpertsEnabled: "off",
		Secured:              false,
	})
}

func managedZoneRecords(records models.Records) models.Records {
	managed := make(models.Records, 0, len(records))
	for _, record := range records {
		if record.Type == "SOA" || (record.Type == "NS" && record.GetLabel() == apexLabel) {
			continue
		}
		managed = append(managed, record)
	}
	return managed
}

func splitZoneName(name string) (zoneDomain, error) {
	name = strings.TrimSuffix(strings.ToLower(name), ".")
	extension, _ := publicsuffix.PublicSuffix(name)
	base := strings.TrimSuffix(name, "."+extension)
	if extension == "" || base == "" || base == name {
		return zoneDomain{}, fmt.Errorf("OPENPROVIDER: cannot split DNS zone name %q into name and extension", name)
	}
	return zoneDomain{Name: base, Extension: extension}, nil
}
