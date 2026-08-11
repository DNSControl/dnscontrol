package huaweicloud

import (
	"fmt"
	"net/url"
	"slices"

	"github.com/DNSControl/dnscontrol/v4/models"
	"github.com/DNSControl/dnscontrol/v4/pkg/diff2"
	"github.com/huaweicloud/huaweicloud-sdk-go-v3/services/dns/v2/model"
)

// GetZoneRecords gets the records of a zone and returns them in RecordConfig format.
func (c *huaweicloudProvider) GetZoneRecords(dc *models.DomainConfig) (models.Records, error) {
	domain := dc.Name

	if err := c.getZones(); err != nil {
		return nil, err
	}
	zoneID, ok := c.zoneIDByDomain[domain]
	if !ok {
		return nil, fmt.Errorf("zone %s not found", domain)
	}
	records, err := c.fetchZoneRecordsFromRemote(zoneID)
	if err != nil {
		return nil, err
	}

	// Convert rrsets to DNSControl's RecordConfig
	existingRecords := []*models.RecordConfig{}
	for _, rec := range *records {
		if *rec.Type == "SOA" {
			continue
		}
		nativeRecords, err := nativeToRecords(&rec, domain)
		if err != nil {
			return nil, err
		}
		existingRecords = append(existingRecords, nativeRecords...)
	}

	return existingRecords, nil
}

// GenerateDomainCorrections takes the desired and existing records
// and produces a Correction list.  The correction list is simply
// a list of functions to call to actually make the desired
// correction, and a message to output to the user when the change is
// made.
func (c *huaweicloudProvider) GetZoneRecordsCorrections(dc *models.DomainConfig, existing models.Records) ([]*models.Correction, int, error) {
	if err := c.getZones(); err != nil {
		return nil, 0, err
	}
	zoneID, ok := c.zoneIDByDomain[dc.Name]
	if !ok {
		return nil, 0, fmt.Errorf("zone %s not found", dc.Name)
	}

	addDefaultMeta(dc.Records)

	// Make delete happen earlier than creates & updates.
	var corrections []*models.Correction
	var deletions []*models.Correction
	var reports []*models.Correction

	changes, actualChangeCount, err := diff2.ByRecordSet(existing, dc, genComparable)
	if err != nil {
		return nil, 0, err
	}

	for _, change := range changes {
		switch change.Type {
		case diff2.REPORT:
			reports = append(reports, &models.Correction{Msg: change.MsgsJoined})
		case diff2.CREATE, diff2.CHANGE:
			newRecordsColl := collectRecordsByLineAndWeightAndKey(change.New)
			oldRecordsColl := collectRecordsByLineAndWeightAndKey(change.Old)
			corrections = append(corrections, &models.Correction{
				Msg: change.MsgsJoined,
				F: func() error {
					// delete old records if not exist in new records
					for key, oldRecords := range oldRecordsColl {
						if _, ok := newRecordsColl[key]; !ok {
							rrsetIDOld := getRRSetIDFromRecords(oldRecords)
							err := c.deleteRRSets(zoneID, rrsetIDOld)
							if err != nil {
								return err
							}
						}
					}
					// modify or create new records
					for key, newRecords := range newRecordsColl {
						records, err := recordsToNative(newRecords, change.Key)
						if err != nil {
							return err
						}
						oldRecords := oldRecordsColl[key]
						rrsetIDOld := getRRSetIDFromRecords(oldRecords)

						if len(rrsetIDOld) == 1 {
							// update existing rrset
							err = c.updateRRSet(zoneID, rrsetIDOld[0], records)
							if err != nil {
								return err
							}
						} else {
							// create new rrset or combine multiple rrsets into one
							err := c.deleteRRSets(zoneID, rrsetIDOld)
							if err != nil {
								return err
							}
							err = c.createRRSet(zoneID, records)
							if err != nil {
								return err
							}
						}
					}
					return nil
				},
			})
		case diff2.DELETE:
			rrsetsID := getRRSetIDFromRecords(change.Old)
			deletions = append(deletions, &models.Correction{
				Msg: change.MsgsJoined,
				F: func() error {
					return c.deleteRRSets(zoneID, rrsetsID)
				},
			})
		default:
			panic(fmt.Sprintf("unhandled change.Type %s", change.Type))
		}
	}

	dnssecCorrections, dnssecChangeCount, err := c.getDNSSECCorrections(dc)
	if err != nil {
		return nil, 0, err
	}

	result := append(reports, deletions...)
	result = append(result, corrections...)
	result = append(result, dnssecCorrections...)
	return result, actualChangeCount + dnssecChangeCount, nil
}

func collectRecordsByLineAndWeightAndKey(records models.Records) map[string]models.Records {
	recordsByLineAndWeight := make(map[string]models.Records)
	for _, rec := range records {
		line := rec.Metadata[metaLine]
		weight := rec.Metadata[metaWeight]
		rrsetKey := rec.Metadata[metaKey]
		key := weight + "," + line + "," + rrsetKey
		recordsByLineAndWeight[key] = append(recordsByLineAndWeight[key], rec)
	}
	return recordsByLineAndWeight
}

func addDefaultMeta(recs models.Records) {
	for _, r := range recs {
		if r.Metadata == nil {
			r.Metadata = make(map[string]string)
		}
		if r.Metadata[metaLine] == "" {
			r.Metadata[metaLine] = defaultLine
		}
		// apex ns should not have weight
		isApexNS := r.Type == "NS" && r.Name == "@"
		if !isApexNS && r.Metadata[metaWeight] == "" {
			r.Metadata[metaWeight] = defaultWeight
		}
	}
}

func genComparable(rec *models.RecordConfig) string {
	// apex ns
	if rec.Type == "NS" && rec.Name == "@" {
		return ""
	}
	weight := rec.Metadata[metaWeight]
	line := rec.Metadata[metaLine]
	key := rec.Metadata[metaKey]
	if weight == "" {
		weight = defaultWeight
	}
	if line == "" {
		line = defaultLine
	}
	return "weight=" + weight + " line=" + line + " key=" + key
}

func (c *huaweicloudProvider) deleteRRSets(zoneID string, rrsets []string) error {
	for _, rrset := range rrsets {
		deletePayload := &model.DeleteRecordSetsRequest{
			ZoneId:      zoneID,
			RecordsetId: rrset,
		}
		var err error
		withRetry(func() error {
			_, err = c.client.DeleteRecordSets(deletePayload)
			return err
		})
		if err != nil {
			return err
		}
	}
	return nil
}

func (c *huaweicloudProvider) createRRSet(zoneID string, rc *model.ShowRecordSetByZoneResp) error {
	createPayload := &model.CreateRecordSetWithLineRequest{
		ZoneId: zoneID,
		Body: &model.CreateRecordSetWithLineRequestBody{
			Name:        *rc.Name,
			Type:        *rc.Type,
			Ttl:         rc.Ttl,
			Records:     rc.Records,
			Weight:      rc.Weight,
			Line:        rc.Line,
			Description: rc.Description,
		},
	}
	var err error
	withRetry(func() error {
		_, err = c.client.CreateRecordSetWithLine(createPayload)
		return err
	})
	return err
}

func (c *huaweicloudProvider) updateRRSet(zoneID, rrsetID string, rc *model.ShowRecordSetByZoneResp) error {
	updatePayload := &model.UpdateRecordSetsRequest{
		ZoneId:      zoneID,
		RecordsetId: rrsetID,
		Body: &model.UpdateRecordSetsReq{
			Name:        *rc.Name,
			Type:        *rc.Type,
			Ttl:         rc.Ttl,
			Records:     rc.Records,
			Weight:      rc.Weight,
			Description: rc.Description,
		},
	}
	var err error
	withRetry(func() error {
		_, err = c.client.UpdateRecordSets(updatePayload)
		return err
	})
	return err
}

func parseMarkerFromURL(link string) (string, error) {
	// Parse the marker params from the URL
	// Example: https://dns.myhuaweicloud.com/v2/zones?marker=abcdefg
	url, err := url.Parse(link)
	if err != nil {
		return "", err
	}
	marker := url.Query().Get("marker")
	if marker == "" {
		return "", fmt.Errorf("marker not found in URL %s", link)
	}
	return marker, nil
}

func (c *huaweicloudProvider) fetchZoneRecordsFromRemote(zoneID string) (*[]model.ShowRecordSetByZoneResp, error) {
	var nextMarker *string
	existingRecords := []model.ShowRecordSetByZoneResp{}
	availableStatus := []string{"ACTIVE", "PENDING_CREATE", "PENDING_UPDATE"}

	for {
		payload := model.ShowRecordSetByZoneRequest{
			ZoneId: zoneID,
			Marker: nextMarker,
		}
		var res *model.ShowRecordSetByZoneResponse
		var err error
		withRetry(func() error {
			res, err = c.client.ShowRecordSetByZone(&payload)
			return err
		})
		if err != nil {
			return nil, err
		}
		if res.Recordsets == nil {
			return &existingRecords, nil
		}
		for _, record := range *res.Recordsets {
			if record.Records == nil {
				continue
			}
			if !slices.Contains(availableStatus, *record.Status) {
				continue
			}
			existingRecords = append(existingRecords, record)
		}

		// if has next page, continue to get next page
		if res.Links.Next != nil {
			marker, err := parseMarkerFromURL(*res.Links.Next)
			if err != nil {
				return nil, err
			}
			nextMarker = &marker
		} else {
			return &existingRecords, nil
		}
	}
}
