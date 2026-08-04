package domainnameshop

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/DNSControl/dnscontrol/v5/models"
	"github.com/DNSControl/dnscontrol/v5/pkg/diff2"
)

func (api *domainNameShopProvider) GetZoneRecords(dc *models.DomainConfig) (models.Records, error) {
	domain := dc.Name

	records, err := api.getDNS(domain)
	if err != nil {
		return nil, err
	}

	var existingRecords []*models.RecordConfig
	for i := range records {
		rC, err := toRecordConfig(dc, &records[i])
		if err != nil {
			return nil, err
		}
		existingRecords = append(existingRecords, rC)
	}

	return existingRecords, nil
}

// GetZoneRecordsCorrections returns a list of corrections that will turn existing records into dc.Records.
func (api *domainNameShopProvider) GetZoneRecordsCorrections(dc *models.DomainConfig, existingRecords models.Records) ([]*models.Correction, int, error) {
	// NB(tlim): Commenting this out. GetTargetTXTJoined() should be sufficient.
	// Merge TXT strings to one string
	// for _, rc := range dc.Records {
	// 	if rc.HasFormatIdenticalToTXT() {
	// 		if err := rc.Set|TargetTXT(rc.GetTargetTXTJoined()); err != nil {
	// 			return nil, 0, err
	// 		}
	// 	}
	// }

	// Domainnameshop doesn't allow arbitrary TTLs they must be a multiple of 60.
	for _, record := range dc.Records {
		record.TTL = fixTTL(record.TTL)
	}

	changes, actualChangeCount, err := diff2.ByRecord(existingRecords, dc, nil)
	if err != nil {
		return nil, 0, err
	}

	var corrections []*models.Correction
	for _, change := range changes {
		switch change.Type {
		case diff2.REPORT:
			corrections = append(corrections, &models.Correction{Msg: change.MsgsJoined})

		case diff2.CREATE:
			// Retrieve the domain name that is targeted. I.e. example.com instead of sub.example.com
			des := change.New[0]
			domainName := strings.ReplaceAll(des.GetLabelFQDN(), des.GetLabel()+".", "")

			dnsR, err := api.fromRecordConfig(domainName, des)
			if err != nil {
				return nil, 0, err
			}
			corrections = append(corrections, &models.Correction{
				Msg: change.Msgs[0],
				F:   func() error { return api.CreateRecord(domainName, dnsR) },
			})

		case diff2.CHANGE:
			des := change.New[0]
			domainName := strings.ReplaceAll(des.GetLabelFQDN(), des.GetLabel()+".", "")

			dnsR, err := api.fromRecordConfig(domainName, des)
			if err != nil {
				return nil, 0, err
			}
			dnsR.ID = change.Old[0].Original.(*domainNameShopRecord).ID
			corrections = append(corrections, &models.Correction{
				Msg: change.Msgs[0],
				F:   func() error { return api.UpdateRecord(dnsR) },
			})

		case diff2.DELETE:
			original := change.Old[0].Original.(*domainNameShopRecord)
			domainID := original.DomainID
			recordID := strconv.Itoa(original.ID)
			corrections = append(corrections, &models.Correction{
				Msg: fmt.Sprintf("%s, record id: %s", change.Msgs[0], recordID),
				F:   func() error { return api.deleteRecord(domainID, recordID) },
			})

		default:
			panic(fmt.Sprintf("unhandled change.Type %s", change.Type))
		}
	}

	return corrections, actualChangeCount, nil
}

func (api *domainNameShopProvider) GetNameservers(domain string) ([]*models.Nameserver, error) {
	ns, err := api.getNS(domain)
	if err != nil {
		return nil, err
	}
	return models.ToNameservers(ns)
}

const (
	minAllowedTTL = 60
	maxAllowedTTL = 604800
	multiplierTTL = 60
)

func fixTTL(ttl uint32) uint32 {
	// if the TTL is larger than the largest allowed value, return the largest allowed value
	if ttl > maxAllowedTTL {
		return maxAllowedTTL
	} else if ttl < 60 {
		return minAllowedTTL
	}

	// Return closest rounded down possible

	return (ttl / multiplierTTL) * multiplierTTL
}
