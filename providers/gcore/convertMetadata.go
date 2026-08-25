package gcore

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	dnssdk "github.com/G-Core/gcore-dns-sdk-go"
)

type gcoreFailoverMetadata struct {
	Protocol       *string `json:"protocol,omitempty"`
	Port           *int    `json:"port,omitempty"`
	Frequency      *int    `json:"frequency,omitempty"`
	Timeout        *int    `json:"timeout,omitempty"`
	Method         *string `json:"method,omitempty"`
	Command        *string `json:"command,omitempty"`
	URL            *string `json:"url,omitempty"`
	TLS            *bool   `json:"tls,omitempty"`
	RegExp         *string `json:"regexp,omitempty"`
	HTTPStatusCode *int    `json:"http_status_code,omitempty"`
	Host           *string `json:"host,omitempty"`
}

type gcoreMetadata struct {
	// These fields only apply to record metadata
	ASN        []int       `json:"asn,omitempty"`
	Continents []string    `json:"continents,omitempty"`
	Countries  []string    `json:"countries,omitempty"`
	LatLong    *[2]float64 `json:"latlong,omitempty"`
	Fallback   *bool       `json:"fallback,omitempty"`
	Backup     *bool       `json:"backup,omitempty"`
	Notes      *string     `json:"notes,omitempty"`
	Weight     *float64    `json:"weight,omitempty"`
	IP         []string    `json:"ip,omitempty"`
	// Failover only applies to RRSet metadata
	Failover *gcoreFailoverMetadata `json:"failover,omitempty"`
}

func nativeMetadataToRecords(rrset *gcoreRRSetExtended, rec *dnssdk.ResourceRecord) (map[string]string, error) {
	result := map[string]string{}

	if len(rrset.Filters) > 0 {
		result[metaFilters] = serializeRecordFilter(rrset.Filters)
	}

	// RRSet only supports failover field
	if rrset.Meta != nil && rrset.Meta.Failover != nil {
		failover := *rrset.Meta.Failover
		if failover.Protocol != nil {
			result[metaFailoverProtocol] = *failover.Protocol
		}
		if failover.Port != nil {
			result[metaFailoverPort] = strconv.Itoa(*failover.Port)
		}
		if failover.Frequency != nil {
			result[metaFailoverFrequency] = strconv.Itoa(*failover.Frequency)
		}
		if failover.Timeout != nil {
			result[metaFailoverTimeout] = strconv.Itoa(*failover.Timeout)
		}
		if failover.Method != nil {
			result[metaFailoverMethod] = *failover.Method
		}
		if failover.Command != nil {
			result[metaFailoverCommand] = *failover.Command
		}
		if failover.URL != nil {
			result[metaFailoverURL] = *failover.URL
		}
		if failover.TLS != nil {
			result[metaFailoverTLS] = strconv.FormatBool(*failover.TLS)
		}
		if failover.RegExp != nil {
			result[metaFailoverRegexp] = *failover.RegExp
		}
		if failover.HTTPStatusCode != nil {
			result[metaFailoverHTTPStatusCode] = strconv.Itoa(*failover.HTTPStatusCode)
		}
		if failover.Host != nil {
			result[metaFailoverHost] = *failover.Host
		}
	}

	// ResourceRecord supports all fields except failover
	if rec.Meta != nil {
		// Convert GCore SDK's type to our metadata type
		metaJSON, err := json.Marshal(rec.Meta)
		if err != nil {
			return nil, err
		}

		meta := gcoreMetadata{}
		err = json.Unmarshal(metaJSON, &meta)
		if err != nil {
			return nil, err
		}

		if meta.ASN != nil {
			// This is probably a ton of memory copies, but I cannot think of a better solution for now
			asnArray := []string{}
			for _, asn := range meta.ASN {
				asnArray = append(asnArray, strconv.Itoa(asn))
			}
			result[metaASN] = strings.Join(asnArray, ",")
		}
		if meta.Continents != nil {
			result[metaContinents] = strings.Join(meta.Continents, ",")
		}
		if meta.Countries != nil {
			result[metaCountries] = strings.Join(meta.Countries, ",")
		}
		if meta.LatLong != nil {
			result[metaLatitude] = strconv.FormatFloat(meta.LatLong[0], 'f', -1, 64)
			result[metaLongitude] = strconv.FormatFloat(meta.LatLong[1], 'f', -1, 64)
		}
		if meta.Fallback != nil {
			result[metaFallback] = strconv.FormatBool(*meta.Fallback)
		}
		if meta.Backup != nil {
			result[metaBackup] = strconv.FormatBool(*meta.Backup)
		}
		if meta.Notes != nil {
			result[metaNotes] = *meta.Notes
		}
		if meta.Weight != nil {
			result[metaWeight] = strconv.FormatFloat(*meta.Weight, 'f', -1, 64)
		}
		if meta.IP != nil {
			result[metaIP] = strings.Join(meta.IP, ",")
		}
	}

	return result, nil
}

func recordsMetadataToNative(meta map[string]string) ([]dnssdk.RecordFilter, map[string]any, map[string]any, error) {
	var rrsetFilters []dnssdk.RecordFilter = nil
	rrsetMeta := gcoreMetadata{}
	recordMeta := gcoreMetadata{}

	for k, v := range meta {
		// Copy string to avoid it later changed by other logic
		vCopy := strings.Clone(v)
		switch k {
		case metaFilters:
			var err error
			rrsetFilters, err = parseRecordFilter(v)
			if err != nil {
				return nil, nil, nil, err
			}
		case metaFailoverProtocol:
			if rrsetMeta.Failover == nil {
				rrsetMeta.Failover = &gcoreFailoverMetadata{}
			}
			rrsetMeta.Failover.Protocol = &vCopy
		case metaFailoverPort:
			if rrsetMeta.Failover == nil {
				rrsetMeta.Failover = &gcoreFailoverMetadata{}
			}
			value, err := strconv.Atoi(v)
			if err != nil {
				return nil, nil, nil, err
			}
			rrsetMeta.Failover.Port = &value
		case metaFailoverFrequency:
			if rrsetMeta.Failover == nil {
				rrsetMeta.Failover = &gcoreFailoverMetadata{}
			}
			value, err := strconv.Atoi(v)
			if err != nil {
				return nil, nil, nil, err
			}
			rrsetMeta.Failover.Frequency = &value
		case metaFailoverTimeout:
			if rrsetMeta.Failover == nil {
				rrsetMeta.Failover = &gcoreFailoverMetadata{}
			}
			value, err := strconv.Atoi(v)
			if err != nil {
				return nil, nil, nil, err
			}
			rrsetMeta.Failover.Timeout = &value
		case metaFailoverMethod:
			if rrsetMeta.Failover == nil {
				rrsetMeta.Failover = &gcoreFailoverMetadata{}
			}
			rrsetMeta.Failover.Method = &vCopy
		case metaFailoverCommand:
			if rrsetMeta.Failover == nil {
				rrsetMeta.Failover = &gcoreFailoverMetadata{}
			}
			rrsetMeta.Failover.Command = &vCopy
		case metaFailoverURL:
			if rrsetMeta.Failover == nil {
				rrsetMeta.Failover = &gcoreFailoverMetadata{}
			}
			rrsetMeta.Failover.URL = &vCopy
		case metaFailoverTLS:
			if rrsetMeta.Failover == nil {
				rrsetMeta.Failover = &gcoreFailoverMetadata{}
			}
			value, err := strconv.ParseBool(v)
			if err != nil {
				return nil, nil, nil, err
			}
			rrsetMeta.Failover.TLS = &value
		case metaFailoverRegexp:
			if rrsetMeta.Failover == nil {
				rrsetMeta.Failover = &gcoreFailoverMetadata{}
			}
			rrsetMeta.Failover.RegExp = &vCopy
		case metaFailoverHTTPStatusCode:
			if rrsetMeta.Failover == nil {
				rrsetMeta.Failover = &gcoreFailoverMetadata{}
			}
			value, err := strconv.Atoi(v)
			if err != nil {
				return nil, nil, nil, err
			}
			rrsetMeta.Failover.HTTPStatusCode = &value
		case metaFailoverHost:
			if rrsetMeta.Failover == nil {
				rrsetMeta.Failover = &gcoreFailoverMetadata{}
			}
			rrsetMeta.Failover.Host = &vCopy
		case metaASN:
			// This is probably a ton of memory copies, but I cannot think of a better solution for now
			asnArray := []int{}
			for asn := range strings.SplitSeq(v, ",") {
				if len(asn) > 0 {
					value, err := strconv.Atoi(asn)
					if err != nil {
						return nil, nil, nil, err
					}
					asnArray = append(asnArray, value)
				}
			}
			recordMeta.ASN = asnArray
		case metaContinents:
			continents := strings.Split(v, ",")
			recordMeta.Continents = continents
		case metaCountries:
			countries := strings.Split(v, ",")
			recordMeta.Countries = countries
		case metaLatitude:
			value, err := strconv.ParseFloat(v, 64)
			if err != nil {
				return nil, nil, nil, err
			}
			if recordMeta.LatLong == nil {
				recordMeta.LatLong = &[2]float64{0, 0}
			}
			recordMeta.LatLong[0] = value
		case metaLongitude:
			value, err := strconv.ParseFloat(v, 64)
			if err != nil {
				return nil, nil, nil, err
			}
			if recordMeta.LatLong == nil {
				recordMeta.LatLong = &[2]float64{0, 0}
			}
			recordMeta.LatLong[1] = value
		case metaFallback:
			value, err := strconv.ParseBool(v)
			if err != nil {
				return nil, nil, nil, err
			}
			recordMeta.Fallback = &value
		case metaBackup:
			value, err := strconv.ParseBool(v)
			if err != nil {
				return nil, nil, nil, err
			}
			recordMeta.Backup = &value
		case metaNotes:
			recordMeta.Notes = &vCopy
		case metaWeight:
			value, err := strconv.ParseFloat(v, 64)
			if err != nil {
				return nil, nil, nil, err
			}
			recordMeta.Weight = &value
		case metaIP:
			ips := strings.Split(v, ",")
			recordMeta.IP = ips
		}
	}

	// Convert metadata to the map[string]any type SDK wants
	rrsetMetaSDK, err := convertToDnssdkMeta(rrsetMeta)
	if err != nil {
		return nil, nil, nil, err
	}
	recordMetaSDK, err := convertToDnssdkMeta(recordMeta)
	if err != nil {
		return nil, nil, nil, err
	}

	return rrsetFilters, rrsetMetaSDK, recordMetaSDK, nil
}

func serializeRecordFilter(f []dnssdk.RecordFilter) string {
	result := []string{}

	for _, v := range f {
		filterString := v.Type

		if v.Strict {
			filterString += ",true"
		} else {
			filterString += ",false"
		}

		if v.Limit != 0 {
			filterString += "," + strconv.Itoa(int(v.Limit))
		}

		result = append(result, filterString)
	}

	return strings.Join(result, ";")
}

func parseRecordFilter(filterString string) ([]dnssdk.RecordFilter, error) {
	result := []dnssdk.RecordFilter{}

	for s := range strings.SplitSeq(filterString, ";") {
		var err error

		fields := strings.Split(s, ",")
		if len(fields) < 2 || len(fields) > 3 {
			return nil, fmt.Errorf("filter %s has invalid format, correct format is \"type,strict[,limit]\"", s)
		}

		record := dnssdk.RecordFilter{
			Type: fields[0]}

		record.Strict, err = strconv.ParseBool(fields[1])
		if err != nil {
			return nil, err
		}

		if len(fields) == 3 {
			limit, err := strconv.Atoi(fields[2])
			if err != nil {
				return nil, err
			}
			record.Limit = uint(limit)
		}

		result = append(result, record)
	}

	return result, nil
}

func isStructEqual[T any](a T, b T) (bool, error) {
	aJSON, err := json.Marshal(a)
	if err != nil {
		return false, err
	}
	bJSON, err := json.Marshal(b)
	if err != nil {
		return false, err
	}
	return bytes.Equal(aJSON, bJSON), nil
}

func isListStructEqual[T any](a []T, b []T) (bool, error) {
	if len(a) != len(b) {
		return true, nil
	}

	for i := range len(a) {
		result, err := isStructEqual(a[i], b[i])
		if err != nil {
			return false, err
		}

		if !result {
			return false, nil
		}
	}

	return true, nil
}

func convertToDnssdkMeta(v any) (map[string]any, error) {
	marshaled, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	result := map[string]any{}
	err = json.Unmarshal(marshaled, &result)
	if err != nil {
		return nil, err
	}
	return result, nil
}
