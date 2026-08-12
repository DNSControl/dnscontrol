package nexdns

import (
	"fmt"
	"strings"

	"github.com/DNSControl/dnscontrol/v5/models"
	"github.com/DNSControl/dnscontrol/v5/pkg/diff2"
	"github.com/DNSControl/dnscontrol/v5/pkg/printer"
)

// GetZoneRecords gets the records of a zone and returns them in RecordConfig format.
//
// The SOA record and the NS records at the zone apex are left out: both are
// maintained by the platform and rejected by the API, so reporting them would
// make every run plan a change it cannot apply.
func (n *nexdnsProvider) GetZoneRecords(dc *models.DomainConfig) (models.Records, error) {
	zone, err := n.getZone(dc.Name)
	if err != nil {
		return nil, err
	}

	nativeRecs, err := n.client.listRecords(zone.ID)
	if err != nil {
		return nil, err
	}

	recs := make(models.Records, 0, len(nativeRecs))
	for _, nativeRec := range nativeRecs {
		if nativeRec.Type == "SOA" {
			continue
		}
		if nativeRec.Type == "NS" && nativeRec.Name == apexLabel {
			continue
		}

		rc, err := toRecordConfig(dc, nativeRec)
		if err != nil {
			return nil, fmt.Errorf("NEXDNS: %s %s: %w", nativeRec.Type, nativeRec.Name, err)
		}
		recs = append(recs, rc)
	}

	return recs, nil
}

// GetZoneRecordsCorrections returns a list of corrections that will turn existing records into dc.Records.
func (n *nexdnsProvider) GetZoneRecordsCorrections(dc *models.DomainConfig, existing models.Records) ([]*models.Correction, int, error) {
	zone, err := n.getZone(dc.Name)
	if err != nil {
		return nil, 0, err
	}

	filterApexNS(dc)

	instructions, actualChangeCount, err := diff2.ByRecord(existing, dc, nil)
	if err != nil {
		return nil, 0, err
	}

	var corrections []*models.Correction
	for _, inst := range instructions {
		switch inst.Type {
		case diff2.REPORT:
			corrections = append(corrections, &models.Correction{
				Msg: inst.MsgsJoined,
			})

		case diff2.CREATE:
			newRec := inst.New[0]
			corrections = append(corrections, &models.Correction{
				Msg: inst.Msgs[0],
				F: func() error {
					return n.client.createRecord(zone.ID, fromRecordConfig(newRec))
				},
			})

		case diff2.CHANGE:
			oldID := inst.Old[0].Original.(apiRecord).ID
			newRec := inst.New[0]
			corrections = append(corrections, &models.Correction{
				Msg: inst.Msgs[0],
				F: func() error {
					return n.client.updateRecord(zone.ID, oldID, fromRecordConfig(newRec))
				},
			})

		case diff2.DELETE:
			oldID := inst.Old[0].Original.(apiRecord).ID
			corrections = append(corrections, &models.Correction{
				Msg: inst.Msgs[0],
				F: func() error {
					return n.client.deleteRecord(zone.ID, oldID)
				},
			})

		default:
			panic(fmt.Sprintf("unhandled inst.Type %s", inst.Type))
		}
	}

	return corrections, actualChangeCount, nil
}

// filterApexNS removes the NS records at the zone apex from dc.Records.
// The apex NS set is the delegation the platform serves and the API rejects a
// write to it, so DNSControl cannot manage it. A record that repeats one of the
// zone's declared nameservers is dropped silently; any other apex NS record is
// dropped with a warning, because the user asked for something that will not
// happen.
func filterApexNS(dc *models.DomainConfig) {
	declared := make(map[string]bool, len(dc.Nameservers))
	for _, ns := range dc.Nameservers {
		declared[strings.TrimSuffix(ns.Name, ".")] = true
	}

	kept := make([]*models.RecordConfig, 0, len(dc.Records))
	for _, rec := range dc.Records {
		if rec.Type == "NS" && rec.GetLabel() == apexLabel {
			target := strings.TrimSuffix(rec.AsNS().Ns, ".")
			if !declared[target] {
				printer.Warnf("NEXDNS does not support changing the NS records at the zone apex. %s will not be added.\n", rec.AsNS().Ns)
			}
			continue
		}
		kept = append(kept, rec)
	}
	dc.Records = kept
}
