package scaleway

import (
	"fmt"

	"github.com/DNSControl/dnscontrol/v4/models"
	"github.com/DNSControl/dnscontrol/v4/pkg/diff2"
	domain "github.com/scaleway/scaleway-sdk-go/api/domain/v2beta1"
)

const pageSize = uint32(1000)

// GetZoneRecords gets the records of a zone and returns them in RecordConfig format.
func (s *scalewayProvider) GetZoneRecords(dc *models.DomainConfig) (models.Records, error) {
	zone := dc.Name

	page := int32(1)
	ps := pageSize
	var records models.Records
	for {
		p := page
		resp, err := s.client.ListDNSZoneRecords(&domain.ListDNSZoneRecordsRequest{
			DNSZone:  zone,
			Page:     &p,
			PageSize: &ps,
		})
		if err != nil {
			return nil, fmt.Errorf("SCALEWAY: could not list records for %q: %w", zone, err)
		}
		for _, r := range resp.Records {
			// Skip SOA — Scaleway manages it.
			if string(r.Type) == "SOA" {
				continue
			}
			rc, err := toRecordConfig(zone, r)
			if err != nil {
				return nil, err
			}
			records = append(records, rc)
		}
		if uint32(len(records)) >= resp.TotalCount {
			break
		}
		page++
	}
	return records, nil
}

// GetZoneRecordsCorrections returns a list of corrections that will turn existing records into dc.Records.
func (s *scalewayProvider) GetZoneRecordsCorrections(dc *models.DomainConfig, existing models.Records) ([]*models.Correction, int, error) {
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
			rec := fromRecordConfig(inst.New[0])
			msg := inst.Msgs[0]
			corrections = append(corrections, &models.Correction{
				Msg: msg,
				F: func() error {
					return s.applyChanges(dc.Name, []*domain.RecordChange{{
						Add: &domain.RecordChangeAdd{Records: []*domain.Record{&rec}},
					}})
				},
			})

		case diff2.CHANGE:
			oldRec, ok := inst.Old[0].Original.(*domain.Record)
			if !ok {
				return nil, 0, fmt.Errorf("SCALEWAY: missing original record for change")
			}
			rec := fromRecordConfig(inst.New[0])
			id := oldRec.ID
			msg := inst.Msgs[0]
			corrections = append(corrections, &models.Correction{
				Msg: msg,
				F: func() error {
					return s.applyChanges(dc.Name, []*domain.RecordChange{
						{Delete: &domain.RecordChangeDelete{ID: &id}},
						{Add: &domain.RecordChangeAdd{Records: []*domain.Record{&rec}}},
					})
				},
			})

		case diff2.DELETE:
			oldRec, ok := inst.Old[0].Original.(*domain.Record)
			if !ok {
				return nil, 0, fmt.Errorf("SCALEWAY: missing original record for delete")
			}
			id := oldRec.ID
			msg := inst.Msgs[0]
			corrections = append(corrections, &models.Correction{
				Msg: msg,
				F: func() error {
					return s.applyChanges(dc.Name, []*domain.RecordChange{{
						Delete: &domain.RecordChangeDelete{ID: &id},
					}})
				},
			})

		default:
			panic(fmt.Sprintf("unhandled inst.Type %s", inst.Type))
		}
	}

	return corrections, actualChangeCount, nil
}

func (s *scalewayProvider) applyChanges(zone string, changes []*domain.RecordChange) error {
	_, err := s.client.UpdateDNSZoneRecords(&domain.UpdateDNSZoneRecordsRequest{
		DNSZone: zone,
		Changes: changes,
	})
	if err != nil {
		return fmt.Errorf("SCALEWAY: update for %q failed: %w", zone, err)
	}
	return nil
}
