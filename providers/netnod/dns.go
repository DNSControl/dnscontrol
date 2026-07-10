package netnod

import (
	"strings"

	"github.com/DNSControl/dnscontrol/v4/models"

	netnodPrimaryDNS "github.com/netnod/netnod-primary-dns-client"
)

// GetNameservers returns the nameservers for a domain.
func (dsp *netnodProvider) GetNameservers(string) ([]*models.Nameserver, error) {
	var r []string
	for _, j := range dsp.nameservers {
		r = append(r, j.Name)
	}
	return models.ToNameservers(r)
}

// GetZoneRecords gets the records of a zone and returns them in RecordConfig format.
func (dsp *netnodProvider) GetZoneRecords(dc *models.DomainConfig) (models.Records, error) {
	domain := dc.Name
	curRecords := models.Records{}
	domainVariant := domain + "."
	zone, err := dsp.client.GetZone(domainVariant)
	if err != nil {
		return nil, err
	}
	if zone == nil {
		return curRecords, nil
	}

	// loop over grouped records by type, called RRSet
	for _, rrset := range zone.RRsets {
		// Skip SOA records - they are managed by the provider
		if rrset.Type == "SOA" {
			continue
		}
		ttl := 0
		if rrset.TTL != nil {
			ttl = int(*rrset.TTL)
		}
		// loop over single records of this group and create records
		for _, record := range rrset.Records {
			r, err := toRecordConfig(domain, record, ttl, rrset.Name, rrset.Type)
			if err != nil {
				return nil, err
			}
			curRecords = append(curRecords, r)
		}
	}

	return curRecords, nil
}

// GetZoneRecordsCorrections returns a list of corrections that will turn existing records into dc.Records.
func (dsp *netnodProvider) GetZoneRecordsCorrections(dc *models.DomainConfig, existing models.Records) ([]*models.Correction, int, error) {
	corrections, actualChangeCount, err := dsp.getDiff2DomainCorrections(dc, existing)
	if err != nil {
		return nil, 0, err
	}

	return corrections, actualChangeCount, nil
}

// EnsureZoneExists creates a zone if it does not exist.
func (dsp *netnodProvider) EnsureZoneExists(dc *models.DomainConfig) error {
	domain := dc.Name
	metadata := dc.Metadata
	domainVariant := domain + "."
	zone, err := dsp.client.GetZone(domainVariant)
	if err != nil {
		return err
	}
	if zone != nil {
		return nil
	}

	// Per-zone overrides take precedence over provider-level defaults.
	alsoNotify := dsp.AlsoNotify
	if v, ok := metadata["also_notify"]; ok {
		alsoNotify = strings.Split(v, ",")
	}

	allowTransferKeys := dsp.AllowTransferKeys
	if v, ok := metadata["allow_transfer_keys"]; ok {
		allowTransferKeys = strings.Split(v, ",")
	}

	_, err = dsp.client.CreateZone(&netnodPrimaryDNS.Zone{
		Name:              domainVariant,
		AllowTransferKeys: allowTransferKeys,
		AlsoNotify:        alsoNotify,
	})
	return err
}
