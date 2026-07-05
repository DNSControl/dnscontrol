package powerdns

import (
	"context"
	"net/http"

	"github.com/DNSControl/dnscontrol/v4/models"
	"github.com/mittwald/go-powerdns/apis/zones"
	"github.com/mittwald/go-powerdns/pdnshttp"
)

// GetNameservers returns the nameservers for a domain.
func (dsp *powerdnsProvider) GetNameservers(string) ([]*models.Nameserver, error) {
	var r []string
	for _, j := range dsp.nameservers {
		r = append(r, j.Name)
	}
	return models.ToNameservers(r)
}

// GetZoneRecords gets the records of a zone and returns them in RecordConfig format.
func (dsp *powerdnsProvider) GetZoneRecords(dc *models.DomainConfig) (models.Records, error) {
	domain := dc.Name

	curRecords := models.Records{}
	domainVariant := dsp.zoneName(domain, dc.Tag)
	zone, err := dsp.client.Zones().GetZone(context.Background(), dsp.ServerName, domainVariant)
	if err != nil {
		if _, ok := err.(pdnshttp.ErrNotFound); ok {
			// Zone is not found, but everything else is okay so return no records
			return curRecords, nil
		}
		return nil, err
	}

	// loop over grouped records by type, called RRSet
	for _, rrset := range zone.ResourceRecordSets {
		// loop over single records of this group and create records
		for _, pdnsRecord := range rrset.Records {
			r, err := toRecordConfig(domain, pdnsRecord, rrset.TTL, rrset.Name, rrset.Type)
			if err != nil {
				return nil, err
			}
			curRecords = append(curRecords, r)
		}
	}

	return curRecords, nil
}

// GetZoneRecordsCorrections returns a list of corrections that will turn existing records into dc.Records.
func (dsp *powerdnsProvider) GetZoneRecordsCorrections(dc *models.DomainConfig, existing models.Records) ([]*models.Correction, int, error) {
	corrections, actualChangeCount, err := dsp.getDiff2DomainCorrections(dc, existing)
	if err != nil {
		return nil, 0, err
	}

	// DNSSec corrections
	dnssecCorrections, err := dsp.getDNSSECCorrections(dc)
	if err != nil {
		return nil, 0, err
	}
	actualChangeCount += len(dnssecCorrections)

	return append(corrections, dnssecCorrections...), actualChangeCount, nil
}

// EnsureZoneExists creates a zone if it does not exist.
func (dsp *powerdnsProvider) EnsureZoneExists(dc *models.DomainConfig) error {
	domain := dc.Name
	domainVariant := dsp.zoneName(domain, dc.Tag)
	if _, err := dsp.client.Zones().GetZone(context.Background(), dsp.ServerName, domainVariant); err != nil {
		if e, ok := err.(pdnshttp.ErrUnexpectedStatus); ok {
			if e.StatusCode != http.StatusNotFound {
				return err
			}
		}
	} else { // zone seems to exist
		return nil
	}

	_, err := dsp.client.Zones().CreateZone(context.Background(), dsp.ServerName, zones.Zone{
		Name:        domainVariant,
		Type:        zones.ZoneTypeZone,
		DNSSec:      dsp.DNSSecOnCreate,
		Nameservers: dsp.DefaultNS,
		Kind:        dsp.ZoneKind,
		SOAEditAPI:  dsp.SOAEditAPI,
	})
	return err
}
