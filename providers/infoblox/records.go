package infoblox

import (
	"fmt"

	"github.com/DNSControl/dnscontrol/v4/models"
	"github.com/DNSControl/dnscontrol/v4/pkg/diff2"
)

// GetZoneRecords gets the records of a zone and returns them in RecordConfig format.
func (p *infobloxProvider) GetZoneRecords(dc *models.DomainConfig) (models.Records, error) {
	domain := dc.Name

	// Verify the zone exists in the configured view.
	_, err := p.api.getZoneAuth(domain)
	if err != nil {
		return nil, fmt.Errorf("failed to find zone %q: %w", domain, err)
	}

	// Fetch the zone default TTL for records that inherit TTL.
	defaultTTL, err := p.api.getZoneDefaultTTL(domain)
	if err != nil {
		return nil, fmt.Errorf("failed to get zone default TTL for %q: %w", domain, err)
	}
	if defaultTTL == 0 {
		defaultTTL = 300 // Fallback default
	}

	var records models.Records

	for _, recType := range supportedTypes {
		raws, err := p.api.getRecords(recType, domain)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch %s records for %q: %w", recType, domain, err)
		}

		for _, raw := range raws {
			rc, err := toRecordConfig(recType, raw, domain, defaultTTL)
			if err != nil {
				return nil, err
			}
			// toRecordConfig returns nil for records we want to skip (e.g. apex NS).
			if rc != nil {
				records = append(records, rc)
			}
		}
	}

	return records, nil
}

// GetZoneRecordsCorrections returns a list of corrections to turn existing records into dc.Records.
func (p *infobloxProvider) GetZoneRecordsCorrections(dc *models.DomainConfig, existing models.Records) ([]*models.Correction, int, error) {
	var corrections []*models.Correction

	instructions, actualChangeCount, err := diff2.ByRecord(existing, dc, nil)
	if err != nil {
		return nil, 0, err
	}

	for _, inst := range instructions {
		switch inst.Type {
		case diff2.REPORT:
			corrections = append(corrections, inst.CreateMessage())

		case diff2.CREATE:
			rc := inst.New[0]
			msg := inst.MsgsJoined
			corrections = append(corrections, &models.Correction{
				Msg: msg,
				F: func() error {
					recType, body, err := buildRecordBody(rc, dc.Name, p.api.view, true)
					if err != nil {
						return err
					}
					_, err = p.api.createRecord(recType, body)
					return err
				},
			})

		case diff2.CHANGE:
			old := inst.Old[0]
			rc := inst.New[0]
			msg := inst.MsgsJoined
			ref, ok := old.Original.(string)
			if !ok {
				return nil, 0, fmt.Errorf("original record missing _ref for CHANGE operation")
			}
			corrections = append(corrections, &models.Correction{
				Msg: msg,
				F: func() error {
					_, body, err := buildRecordBody(rc, dc.Name, p.api.view, false)
					if err != nil {
						return err
					}
					return p.api.updateRecord(ref, body)
				},
			})

		case diff2.DELETE:
			old := inst.Old[0]
			msg := inst.MsgsJoined
			ref, ok := old.Original.(string)
			if !ok {
				return nil, 0, fmt.Errorf("original record missing _ref for DELETE operation")
			}
			corrections = append(corrections, &models.Correction{
				Msg: msg,
				F: func() error {
					return p.api.deleteRecord(ref)
				},
			})

		default:
			panic(fmt.Sprintf("unhandled diff2 verb: %d", inst.Type))
		}
	}

	return corrections, actualChangeCount, nil
}
