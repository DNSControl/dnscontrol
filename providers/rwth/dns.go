package rwth

import (
	"github.com/DNSControl/dnscontrol/v4/models"
	"github.com/DNSControl/dnscontrol/v4/pkg/diff2"
)

// RWTHDefaultNs is the default DNS NS for this provider.
var RWTHDefaultNs = []string{"dns-1.dfn.de", "dns-2.dfn.de", "zs1.rz.rwth-aachen.de", "zs2.rz.rwth-aachen.de"}

// GetZoneRecords gets the records of a zone and returns them in RecordConfig format.
func (api *rwthProvider) GetZoneRecords(dc *models.DomainConfig) (models.Records, error) {
	domain := dc.Name

	records, err := api.getAllRecords(domain)
	if err != nil {
		return nil, err
	}
	foundRecords := models.Records{}
	for i := range records {
		foundRecords = append(foundRecords, &records[i])
	}
	return foundRecords, nil
}

// GetNameservers returns the default nameservers for RWTH.
func (api *rwthProvider) GetNameservers(domain string) ([]*models.Nameserver, error) {
	return models.ToNameservers(RWTHDefaultNs)
}

// GetZoneRecordsCorrections returns a list of corrections that will turn existing records into dc.Records.
func (api *rwthProvider) GetZoneRecordsCorrections(dc *models.DomainConfig, existingRecords models.Records) ([]*models.Correction, int, error) {
	domain := dc.Name

	instructions, actualChangeCount, err := diff2.ByRecord(existingRecords, dc, nil)
	if err != nil {
		return nil, 0, err
	}
	var corrections []*models.Correction

	for _, inst := range instructions {
		switch inst.Type {
		case diff2.REPORT:
			corrections = append(corrections, &models.Correction{Msg: inst.MsgsJoined})
		case diff2.CREATE:
			rec := inst.New[0]
			corrections = append(corrections, &models.Correction{
				Msg: inst.MsgsJoined,
				F:   func() error { return api.createRecord(rec) },
			})
		case diff2.DELETE:
			existingRecord := inst.Old[0].Original.(RecordReply)
			corrections = append(corrections, &models.Correction{
				Msg: inst.MsgsJoined,
				F:   func() error { return api.destroyRecord(existingRecord) },
			})
		case diff2.CHANGE:
			rec := inst.New[0]
			existingID := inst.Old[0].Original.(RecordReply).ID
			corrections = append(corrections, &models.Correction{
				Msg: inst.MsgsJoined,
				F:   func() error { return api.updateRecord(existingID, *rec) },
			})
		default:
			panic("unhandled instruction type")
		}
	}

	// And deploy if any corrections were applied
	if actualChangeCount > 0 {
		corrections = append(corrections, &models.Correction{
			Msg: "Deploy zone " + domain,
			F:   func() error { return api.deployZone(domain) },
		})
	}

	return corrections, actualChangeCount, nil
}
