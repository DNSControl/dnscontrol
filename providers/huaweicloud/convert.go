package huaweicloud

import (
	"errors"
	"fmt"
	"slices"
	"strconv"

	"github.com/DNSControl/dnscontrol/v4/models"
	"github.com/DNSControl/dnscontrol/v4/pkg/printer"
	"github.com/huaweicloud/huaweicloud-sdk-go-v3/services/dns/v2/model"
)

func getRRSetIDFromRecords(rcs models.Records) []string {
	ids := []string{}
	for _, r := range rcs {
		if r.Original == nil {
			continue
		}
		rrset := r.Original.(*model.ShowRecordSetByZoneResp)
		if rrset.Id == nil {
			printer.Warnf("RecordSet ID is nil for record %+v\n", r)
			continue
		}
		ids = append(ids, *rrset.Id)
	}
	slices.Sort(ids)
	return slices.Compact(ids)
}

func nativeToRecords(n *model.ShowRecordSetByZoneResp, zoneName string) (models.Records, error) {
	if n.Name == nil || n.Type == nil || n.Records == nil || n.Ttl == nil {
		return nil, fmt.Errorf("missing required fields in Huaweicloud's RRset: %+v", n)
	}
	var rcs models.Records
	recName := *n.Name
	recType := *n.Type

	// Split into records
	for _, value := range *n.Records {
		rc := &models.RecordConfig{
			TTL:      uint32(*n.Ttl),
			Original: n,
			Metadata: map[string]string{},
		}
		rc.SetLabelFromFQDN(recName, zoneName)
		if err := rc.PopulateFromString(recType, value, zoneName); err != nil {
			return nil, fmt.Errorf("unparsable record received from Huaweicloud: %w", err)
		}
		if n.Line != nil {
			rc.Metadata[metaLine] = *n.Line
		}
		if n.Weight != nil {
			rc.Metadata[metaWeight] = strconv.Itoa(int(*n.Weight))
		}
		if n.Description != nil {
			rc.Metadata[metaKey] = *n.Description
		}
		rcs = append(rcs, rc)
	}

	return rcs, nil
}

func recordsToNative(rcs models.Records, expectedKey models.RecordKey) (*model.ShowRecordSetByZoneResp, error) {
	// rcs length is guaranteed to be > 0
	if len(rcs) == 0 {
		return nil, errors.New("empty record set")
	}
	// line and weight should be the same for all records in the rrset
	line := rcs[0].Metadata[metaLine]
	weightStr := rcs[0].Metadata[metaWeight]
	for _, r := range rcs {
		if r.Metadata[metaLine] != line {
			return nil, fmt.Errorf("all records in the rrset must have the same line %s", line)
		}
		if r.Metadata[metaWeight] != weightStr {
			return nil, fmt.Errorf("all records in the rrset must have the same weight %s", weightStr)
		}
	}

	// parse weight to int32
	var weight *int32
	if weightStr != "" {
		weightInt, err := strconv.ParseInt(weightStr, 10, 32)
		if err != nil {
			return nil, fmt.Errorf("failed to parse weight %s to int32", weightStr)
		}
		weightInt32 := int32(weightInt)
		// weight should be 0-1000
		if weightInt32 < 0 || weightInt32 > 1000 {
			return nil, errors.New("weight must be between 0 and 1000")
		}
		weight = &weightInt32
	}

	resultTTL := int32(0)
	resultVal := []string{}
	name := expectedKey.NameFQDN + "."
	key := rcs[0].Metadata[metaKey]
	result := &model.ShowRecordSetByZoneResp{
		Name:        &name,
		Type:        &expectedKey.Type,
		Ttl:         &resultTTL,
		Records:     &resultVal,
		Line:        &line,
		Weight:      weight,
		Description: &key,
	}

	for _, r := range rcs {
		key := r.Key()
		if key != expectedKey {
			continue
		}
		val := r.GetTargetCombined()
		// special case for empty TXT records
		if key.Type == "TXT" && len(val) == 0 {
			val = "\"\""
		}

		resultVal = append(resultVal, val)
		if resultTTL == 0 {
			resultTTL = int32(r.TTL)
		}

		// Check if all TTLs are the same
		if int32(r.TTL) != resultTTL {
			printer.Warnf("All TTLs for a rrset (%v) must be the same. Using smaller of %v and %v.\n", key, r.TTL, resultTTL)
			if int32(r.TTL) < resultTTL {
				resultTTL = int32(r.TTL)
			}
		}
	}

	return result, nil
}
