package cloudpress

import (
	"errors"
	"fmt"

	"github.com/DNSControl/dnscontrol/v4/models"
	"github.com/DNSControl/dnscontrol/v4/pkg/diff2"
	"github.com/DNSControl/dnscontrol/v4/pkg/printer"
)

// GetZoneRecords downloads the records of a zone and returns them as RecordConfigs.
func (c *cloudpressProvider) GetZoneRecords(dc *models.DomainConfig) (models.Records, error) {
	z, err := c.findZoneByDomain(dc.Name)
	if err != nil {
		return nil, err
	}

	// Fetch the full zone, which includes the nested records array.
	full, err := c.getZone(z.ID)
	if err != nil {
		return nil, err
	}

	recs := make(models.Records, 0, len(full.Records)+len(full.Nameservers))
	for i := range full.Records {
		nativeRec := &full.Records[i]

		// Unsupported record types are ignored with a warning and left untouched.
		if !isSupported(nativeRec.RecordType) {
			printer.Warnf("CLOUDPRESS: ignoring unsupported record type %s\n", recordTypeToString(nativeRec.RecordType))
			continue
		}

		// Apex NS records are managed by CloudPress and surfaced via GetNameservers,
		// so we don't manage them as ordinary records to avoid clobbering them.
		if nativeRec.RecordType == recordTypeNS && (nativeRec.Name == "" || nativeRec.Name == "@") {
			continue
		}

		rc, err := toRecordConfig(z.Name, nativeRec)
		if err != nil {
			return nil, err
		}
		recs = append(recs, rc)
	}

	// CloudPress does not return the apex NS records via the records API; they
	// live in the separate "nameservers" field. DNSControl adds the provider's
	// nameservers to the desired config, so we add matching implicit NS records
	// here (with TTL 0) to avoid a spurious diff. They carry an empty Original,
	// so any attempt to modify or delete them is rejected in the corrections.
	for _, ns := range full.Nameservers {
		rc := &models.RecordConfig{Type: "NS", TTL: 0, Original: &record{}}
		rc.SetLabel("@", z.Name)
		if err := rc.SetTarget(ns + "."); err != nil {
			return nil, err
		}
		recs = append(recs, rc)
	}

	return recs, nil
}

// GetZoneRecordsCorrections returns the corrections needed to make the zone match dc.Records.
func (c *cloudpressProvider) GetZoneRecordsCorrections(dc *models.DomainConfig, existing models.Records) ([]*models.Correction, int, error) {
	z, err := c.findZoneByDomain(dc.Name)
	if err != nil {
		return nil, 0, err
	}

	// The implicit apex NS records returned by GetZoneRecords have TTL 0 (no TTL
	// is configurable for them), so force the desired apex NS records to TTL 0
	// as well to avoid a spurious diff.
	for _, rc := range dc.Records {
		if rc.Name == "@" && rc.Type == "NS" {
			rc.TTL = 0
		}
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
			corrections = append(corrections, c.mkCreateCorrection(z.ID, inst.New[0], inst.Msgs[0]))
		case diff2.CHANGE:
			corrections = append(corrections, c.mkChangeCorrection(z.ID, inst.Old[0], inst.New[0], inst.Msgs[0]))
		case diff2.DELETE:
			corrections = append(corrections, c.mkDeleteCorrection(z.ID, inst.Old[0], inst.Msgs[0]))
		default:
			panic(fmt.Sprintf("unhandled inst.Type %s", inst.Type))
		}
	}

	dnssecCorrections := c.getDNSSECCorrections(dc, z)
	corrections = append(corrections, dnssecCorrections...)

	return corrections, actualChangeCount, nil
}

func (c *cloudpressProvider) mkCreateCorrection(zoneID string, newRec *models.RecordConfig, msg string) *models.Correction {
	return &models.Correction{
		Msg: msg,
		F: func() error {
			return c.createRecord(zoneID, fromRecordConfig(newRec))
		},
	}
}

func (c *cloudpressProvider) mkChangeCorrection(zoneID string, oldRec, newRec *models.RecordConfig, msg string) *models.Correction {
	return &models.Correction{
		Msg: msg,
		F: func() error {
			existingID := oldRec.Original.(*record).ID
			if existingID == "" {
				return errors.New("CLOUDPRESS: cannot change record without an ID")
			}
			return c.modifyRecord(zoneID, existingID, fromRecordConfig(newRec))
		},
	}
}

func (c *cloudpressProvider) mkDeleteCorrection(zoneID string, oldRec *models.RecordConfig, msg string) *models.Correction {
	return &models.Correction{
		Msg: msg,
		F: func() error {
			existingID := oldRec.Original.(*record).ID
			if existingID == "" {
				return errors.New("CLOUDPRESS: cannot delete record without an ID")
			}
			return c.deleteRecord(zoneID, existingID)
		},
	}
}
