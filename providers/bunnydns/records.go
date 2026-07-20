package bunnydns

import (
	"errors"
	"fmt"
	"slices"

	"github.com/DNSControl/dnscontrol/v4/models"
	"github.com/DNSControl/dnscontrol/v4/pkg/diff2"
	"github.com/DNSControl/dnscontrol/v4/pkg/printer"
)

func (b *bunnydnsProvider) GetZoneRecords(dc *models.DomainConfig) (models.Records, error) {
	domain := dc.Name

	zone, err := b.findZoneByDomain(domain)
	if err != nil {
		return nil, err
	}

	nativeRecs, err := b.getAllRecords(zone.ID)
	if err != nil {
		return nil, err
	}

	implicitRecs, err := b.getImplicitRecordConfigs(zone)
	if err != nil {
		return nil, err
	}

	recs := make(models.Records, 0, len(nativeRecs)+len(implicitRecs))
	recs = append(recs, implicitRecs...)

	// Define a list of record types that are currently not supported by this provider.
	unsupportedTypes := []recordType{
		recordTypeFlatten,
		recordTypeScript,
	}

	// Loop through all native records and convert them to standardized RecordConfigs
	// Unsupported record types are ignored with a warning and will remain untouched in the zone.
	for _, nativeRec := range nativeRecs {
		if slices.Contains(unsupportedTypes, nativeRec.Type) {
			printer.Warnf("BUNNY_DNS: ignoring unsupported record type %s\n", recordTypeToString(nativeRec.Type))
			continue
		}

		rc, err := toRecordConfig(zone.Domain, nativeRec)
		if err != nil {
			return nil, err
		}
		recs = append(recs, rc)
	}

	return recs, nil
}

func (b *bunnydnsProvider) GetZoneRecordsCorrections(dc *models.DomainConfig, existing models.Records) ([]*models.Correction, int, error) {
	// Bunny DNS never returns NS records for the apex domain, so these are artificially added when retrieving records.
	// As no TTL can be configured or retrieved for these NS records, we set it to 0 to avoid unnecessary updates.
	for _, rc := range dc.Records {
		if rc.Name == "@" && rc.Type == "NS" {
			rc.TTL = 0
		}

		if rc.Type == "BUNNY_DNS_RDR" {
			rc.TTL = 0
		}

		if rc.Type == "ALIAS" {
			rc.ChangeType("CNAME", dc.Name)
		}
	}

	zone, err := b.findZoneByDomain(dc.Name)
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
			corrections = append(corrections, &models.Correction{
				Msg: inst.MsgsJoined,
			})
		case diff2.CREATE:
			corrections = append(corrections, b.mkCreateCorrection(
				zone.ID, inst.New[0], inst.Msgs[0],
			))
		case diff2.CHANGE:
			corrections = append(corrections, b.mkChangeCorrection(
				zone.ID, inst.Old[0], inst.New[0], inst.Msgs[0],
			))
		case diff2.DELETE:
			corrections = append(corrections, b.mkDeleteCorrection(
				zone.ID, inst.Old[0], inst.Msgs[0],
			))
		default:
			panic(fmt.Sprintf("unhandled inst.Type %s", inst.Type))
		}
	}

	dnssecCorrections, err := b.getDNSSECCorrections(dc, zone)
	if err != nil {
		return nil, 0, err
	}
	corrections = append(corrections, dnssecCorrections...)

	return corrections, actualChangeCount, nil
}

func (b *bunnydnsProvider) mkCreateCorrection(zoneID int64, newRec *models.RecordConfig, msg string) *models.Correction {
	return &models.Correction{
		Msg: msg,
		F: func() error {
			desired, err := fromRecordConfig(newRec)
			if err != nil {
				return err
			}

			return b.createRecord(zoneID, desired)
		},
	}
}

func (b *bunnydnsProvider) mkChangeCorrection(zoneID int64, oldRec, newRec *models.RecordConfig, msg string) *models.Correction {
	return &models.Correction{
		Msg: msg,
		F: func() error {
			existingID := oldRec.Original.(*record).ID
			if existingID == 0 {
				return errors.New("BUNNY_DNS: cannot change implicit records")
			}

			desired, err := fromRecordConfig(newRec)
			if err != nil {
				return err
			}

			return b.modifyRecord(zoneID, existingID, desired)
		},
	}
}

func (b *bunnydnsProvider) mkDeleteCorrection(zoneID int64, oldRec *models.RecordConfig, msg string) *models.Correction {
	return &models.Correction{
		Msg: msg,
		F: func() error {
			existingID := oldRec.Original.(*record).ID
			if existingID == 0 {
				return errors.New("BUNNY_DNS: cannot delete implicit records")
			}

			return b.deleteRecord(zoneID, existingID)
		},
	}
}
