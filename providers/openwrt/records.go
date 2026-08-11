package openwrt

import (
	"fmt"
	"strings"

	"github.com/DNSControl/dnscontrol/v4/models"
	"github.com/DNSControl/dnscontrol/v4/pkg/diff2"
	"github.com/DNSControl/dnscontrol/v4/pkg/printer"
)

// GetZoneRecordsCorrections returns a list of corrections that will turn existing records into dc.Records.
func (c *openwrtProvider) GetZoneRecordsCorrections(dc *models.DomainConfig, existingRecords models.Records) ([]*models.Correction, int, error) {
	// TTLs don't matter in OPENWRT and
	// we use the default value of 300
	for _, record := range dc.Records {
		record.TTL = 300
	}

	var corrections []*models.Correction

	changes, actualChangeCount, err := diff2.ByRecord(existingRecords, dc,
		func(rec *models.RecordConfig) string { return "" },
	)
	if err != nil {
		return nil, 0, err
	}
	for _, change := range changes {
		var corr *models.Correction
		switch change.Type {
		case diff2.REPORT:
			printer.Warnf("diff2 report message\n")
			corr = &models.Correction{Msg: change.MsgsJoined}

		case diff2.CREATE:
			r, recType, err := toNative(change.New[0])
			if err != nil {
				return nil, 0, err
			}

			corr = &models.Correction{
				Msg: change.Msgs[0],
				F: func() error {
					_, err := c.uciSection(recType, r)
					return err
				},
			}

		case diff2.DELETE:
			section := change.Old[0].Original.(nativeRecord).Section
			corr = &models.Correction{
				Msg: change.Msgs[0],
				F: func() error {
					fmt.Println(section)
					_, err := c.uciDelete(section)
					return err
				},
			}

		case diff2.CHANGE:
			section := change.Old[0].Original.(nativeRecord).Section
			r, _, err := toNative(change.New[0])
			if err != nil {
				return nil, 0, err
			}
			corr = &models.Correction{
				Msg: change.Msgs[0],
				F: func() error {
					_, err := c.uciTset(section, r)
					return err
				},
			}

		default:
			panic(fmt.Sprintf("unhandled change.Type %s", change.Type))
		}

		corrections = append(corrections, corr)
	}

	// Apply changes last, changes cannot be applied incrementally
	// because doing so shifts the section names, making deleting
	// records unreliable
	if actualChangeCount > 0 {
		corrections = append(corrections, &models.Correction{
			Msg: "Applying changes",
			F: func() error {
				_, err := c.uciApply()
				return err
			},
		})
	}

	return corrections, actualChangeCount, nil
}

// GetZoneRecords gets the records of a zone and returns them in RecordConfig format.
func (c *openwrtProvider) GetZoneRecords(dc *models.DomainConfig) (models.Records, error) {
	domain := dc.Name

	resp, err := c.uciGetAll()
	if err != nil {
		return nil, fmt.Errorf("failed to fetch records from openwrt: %w", err)
	}

	nativeRecords := make([]nativeRecord, 0)
	for _, section := range resp {
		if !section.isRecord() {
			continue
		}

		recDomain, err := section.getDomain()
		if err != nil {
			return nil, fmt.Errorf("openwrt returned an invalid record: %w", err)
		}

		if !strings.HasSuffix(recDomain, "."+domain) && recDomain != domain {
			continue
		}

		nativeRecords = append(nativeRecords, section)
	}

	records := make([]*models.RecordConfig, 0)
	for _, r := range nativeRecords {
		rc, err := toRc(domain, r)
		if err != nil {
			return nil, err
		}
		records = append(records, rc)
	}

	return records, nil
}
