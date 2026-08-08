package websupport

import (
	"fmt"

	"github.com/DNSControl/dnscontrol/v5/models"
	"github.com/DNSControl/dnscontrol/v5/pkg/diff2"
	"github.com/DNSControl/dnscontrol/v5/pkg/providers"
)

// GetZoneRecords returns the current records for a zone.
func (c *websupportProvider) GetZoneRecords(dc *models.DomainConfig) (models.Records, error) {
	svcID, err := c.serviceID(dc.Name)
	if err != nil {
		return nil, err
	}

	nativeRecs, err := c.getAllRecords(svcID)
	if err != nil {
		return nil, err
	}

	recs := make(models.Records, 0, len(nativeRecs))
	for _, n := range nativeRecs {
		before := providers.BeginToRC(c.observer, "toRecordConfig", n)
		rc, err := toRecordConfig(dc, n)
		providers.EndToRC(c.observer, "toRecordConfig", before, n, models.Records{rc}, err)
		if err != nil {
			return nil, err
		}
		recs = append(recs, rc)
	}
	return recs, nil
}

// GetZoneRecordsCorrections computes the changes needed to bring the zone in
// sync with the desired configuration.
func (c *websupportProvider) GetZoneRecordsCorrections(dc *models.DomainConfig, existing models.Records) ([]*models.Correction, int, error) {
	svcID, err := c.serviceID(dc.Name)
	if err != nil {
		return nil, 0, err
	}

	instructions, actualChangeCount, err := diff2.ByRecord(existing, dc, nil)
	if err != nil {
		return nil, 0, err
	}

	var corrections []*models.Correction
	for _, inst := range instructions {
		switch inst.Type {
		case diff2.REPORT:
			corrections = append(corrections, &models.Correction{Msg: inst.MsgsJoined})
		case diff2.CREATE:
			corrections = append(corrections, c.mkCreateCorrection(svcID, inst.New[0], inst.Msgs[0]))
		case diff2.CHANGE:
			corrections = append(corrections, c.mkChangeCorrection(svcID, inst.Old[0], inst.New[0], inst.Msgs[0]))
		case diff2.DELETE:
			corrections = append(corrections, c.mkDeleteCorrection(svcID, inst.Old[0], inst.Msgs[0]))
		default:
			panic(fmt.Sprintf("unhandled inst.Type %s", inst.Type))
		}
	}

	return corrections, actualChangeCount, nil
}

func (c *websupportProvider) mkCreateCorrection(svcID int64, newRec *models.RecordConfig, msg string) *models.Correction {
	return &models.Correction{
		Msg: msg,
		F: func() error {
			input := models.Records{newRec}
			before := providers.BeginToNative(c.observer, "toNative", input)
			n, err := toNative(newRec)
			providers.EndToNative(c.observer, "toNative", before, input, n, err)
			if err != nil {
				return err
			}
			return c.createRecord(svcID, n)
		},
	}
}

func (c *websupportProvider) mkChangeCorrection(svcID int64, oldRec, newRec *models.RecordConfig, msg string) *models.Correction {
	return &models.Correction{
		Msg: msg,
		F: func() error {
			id := oldRec.Original.(nativeRecord).ID
			if id == 0 {
				return fmt.Errorf("WEBSUPPORT: cannot update record without an id")
			}
			input := models.Records{newRec}
			before := providers.BeginToNative(c.observer, "toNative", input)
			n, err := toNative(newRec)
			providers.EndToNative(c.observer, "toNative", before, input, n, err)
			if err != nil {
				return err
			}
			return c.updateRecord(svcID, id, n)
		},
	}
}

func (c *websupportProvider) mkDeleteCorrection(svcID int64, oldRec *models.RecordConfig, msg string) *models.Correction {
	return &models.Correction{
		Msg: msg,
		F: func() error {
			id := oldRec.Original.(nativeRecord).ID
			if id == 0 {
				return fmt.Errorf("WEBSUPPORT: cannot delete record without an id")
			}
			return c.deleteRecord(svcID, id)
		},
	}
}
