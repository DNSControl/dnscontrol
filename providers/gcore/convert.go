package gcore

// Convert the provider's native record description to models.RecordConfig.

import (
	"errors"
	"fmt"

	"github.com/DNSControl/dnscontrol/v4/models"
	"github.com/DNSControl/dnscontrol/v4/pkg/printer"
	dnssdk "github.com/G-Core/gcore-dns-sdk-go"
)

// nativeToRecord takes a DNS record from G-Core and returns a native RecordConfig struct.
func nativeToRecords(n gcoreRRSetExtended, zoneName string) ([]*models.RecordConfig, error) {
	var rcs []*models.RecordConfig
	recName := n.Name
	recType := n.Type

	// Split G-Core's RRset into individual records
	for _, value := range n.Records {
		metadata, err := nativeMetadataToRecords(&n, &value)
		if err != nil {
			return nil, fmt.Errorf("unparsable record received from G-Core: %w", err)
		}
		rc := &models.RecordConfig{
			TTL:      uint32(n.TTL),
			Original: n,
			Metadata: metadata,
		}
		rc.SetLabelFromFQDN(recName, zoneName)
		switch recType {
		case "CAA": // G-Core API don't need quotes around CAA with whitespace
			if len(value.Content) != 3 {
				return nil, errors.New("incorrect number of fields in G-Core's CAA record")
			}

			parts := make([]string, len(value.Content))
			for i := range value.Content {
				parts[i] = fmt.Sprint(value.Content[i])
			}

			flag, tag, target := parts[0], parts[1], parts[2]
			if err := rc.SetTargetCAAStrings(flag, tag, target); err != nil {
				return nil, fmt.Errorf("unparsable record received from G-Core: %w", err)
			}

		case "TXT": // Avoid double quoting for TXT records
			if err := rc.SetTargetTXTs(convertSdkAnySliceToTxtSlice(value.Content)); err != nil {
				return nil, fmt.Errorf("unparsable record received from G-Core: %w", err)
			}

		case "SCVB": // GCore mistypes "SVCB" as "SCVB"
			if err := rc.PopulateFromString("SVCB", value.ContentToString(), zoneName); err != nil {
				return nil, fmt.Errorf("unparsable record received from G-Core: %w", err)
			}

		default: //  "A", "AAAA", "CAA", "NS", "CNAME", "MX", "PTR", "SRV"
			if err := rc.PopulateFromString(recType, value.ContentToString(), zoneName); err != nil {
				return nil, fmt.Errorf("unparsable record received from G-Core: %w", err)
			}
		}
		rcs = append(rcs, rc)
	}

	return rcs, nil
}

func recordsToNative(rcs []*models.RecordConfig, expectedKey models.RecordKey) (*dnssdk.RRSet, error) {
	// Merge DNSControl records into G-Core RRsets

	var result *dnssdk.RRSet
	var resultRRSetFilters []dnssdk.RecordFilter = nil
	var resultRRSetMeta map[string]any = nil
	var resultRRSetMetaSourceRecord *models.RecordConfig = nil

	for _, r := range rcs {
		key := r.Key()

		if key != expectedKey {
			continue
		}

		rrsetFilters, rrsetMeta, recordMeta, err := recordsMetadataToNative(r.Metadata)
		if err != nil {
			return nil, err
		}

		if resultRRSetMeta == nil {
			resultRRSetFilters = rrsetFilters
			resultRRSetMeta = rrsetMeta
			resultRRSetMetaSourceRecord = r
		} else {
			isRRSetFilterEqual, err := isListStructEqual(resultRRSetFilters, rrsetFilters)
			if err != nil {
				return nil, err
			}
			if !isRRSetFilterEqual {
				return nil, fmt.Errorf("filter is not consistent between %s and %s in RRSet %s", resultRRSetMetaSourceRecord, r, expectedKey)
			}

			isRRSetMetaEqual, err := isStructEqual(resultRRSetMeta, rrsetMeta)
			if err != nil {
				return nil, err
			}
			if !isRRSetMetaEqual {
				return nil, fmt.Errorf("metadata is not consistent between %s and %s in RRSet %s", resultRRSetMetaSourceRecord, r, expectedKey)
			}
		}

		var rr dnssdk.ResourceRecord
		switch key.Type {
		case "CAA": // G-Core API don't need quotes around CAA with whitespace
			rr = dnssdk.ResourceRecord{
				Content: []any{
					int64(r.CaaFlag),
					r.CaaTag,
					r.GetTargetField(),
				},
				Meta:    recordMeta,
				Enabled: true,
			}
		case "TXT": // Avoid double quoting for TXT records
			rr = dnssdk.ResourceRecord{
				Content: convertTxtSliceToSdkAnySlice(r.GetTargetTXTJoined()),
				Meta:    recordMeta,
				Enabled: true,
			}
		case "SVCB":
			// GCore mistypes "SVCB" as "SCVB"
			rr = dnssdk.ResourceRecord{
				Content: dnssdk.ContentFromValue("SCVB", r.GetTargetCombined()),
				Meta:    recordMeta,
				Enabled: true,
			}
		default:
			rr = dnssdk.ResourceRecord{
				Content: dnssdk.ContentFromValue(key.Type, r.GetTargetCombined()),
				Meta:    recordMeta,
				Enabled: true,
			}
		}

		if result == nil {
			result = &dnssdk.RRSet{
				TTL:     int(r.TTL),
				Records: []dnssdk.ResourceRecord{rr},
			}
		} else {
			result.Records = append(result.Records, rr)

			if int(r.TTL) != result.TTL {
				printer.Warnf("All TTLs for a rrset (%v) must be the same. Using smaller of %v and %v.\n", key, r.TTL, result.TTL)
				if int(r.TTL) < result.TTL {
					result.TTL = int(r.TTL)
				}
			}
		}
	}

	if result != nil {
		result.Filters = resultRRSetFilters
		result.Meta = resultRRSetMeta
	}

	return result, nil
}

func convertTxtSliceToSdkAnySlice(record string) []any {
	result := []any{record}
	return result
}

func convertSdkAnySliceToTxtSlice(records []any) []string {
	result := []string{}
	for _, record := range records {
		result = append(result, record.(string))
	}
	return result
}
