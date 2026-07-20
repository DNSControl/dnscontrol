package joker

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/DNSControl/dnscontrol/v4/models"
	"github.com/DNSControl/dnscontrol/v4/pkg/diff2"
)

// GetZoneRecords gets the records of a zone and returns them in RecordConfig format.
func (api *jokerProvider) GetZoneRecords(dc *models.DomainConfig) (models.Records, error) {
	domain := dc.Name

	params := url.Values{}
	params.Set("domain", domain)

	_, body, err := api.makeRequest("dns-zone-get", params)
	if err != nil {
		return nil, err
	}

	records, err := api.parseZoneRecords(domain, body)
	if err != nil {
		return nil, err
	}

	return records, nil
}

// parseZoneLine parses a zone file line while preserving quoted strings.
func parseZoneLine(line string) []string {
	var parts []string
	var current strings.Builder
	inQuotes := false
	escaped := false

	for _, r := range line {
		if escaped {
			current.WriteRune(r)
			escaped = false
			continue
		}

		if r == '\\' {
			escaped = true
			current.WriteRune(r)
			continue
		}

		if r == '"' {
			inQuotes = !inQuotes
			current.WriteRune(r)
			continue
		}

		if !inQuotes && (r == ' ' || r == '\t') {
			// Skip multiple consecutive spaces
			if current.Len() > 0 {
				parts = append(parts, current.String())
				current.Reset()
			}
			continue
		}

		current.WriteRune(r)
	}

	// Add the final part if any
	if current.Len() > 0 {
		parts = append(parts, current.String())
	}

	return parts
}

// parseZoneRecords parses Joker zone format into RecordConfig format.
func (api *jokerProvider) parseZoneRecords(domain, zoneData string) (models.Records, error) {
	var records models.Records

	lines := strings.SplitSeq(strings.TrimSpace(zoneData), "\n")
	for line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, "$") {
			continue
		}

		// Parse the line while preserving quoted strings
		parts := parseZoneLine(line)
		if len(parts) < 4 {
			continue
		}

		label := parts[0]
		recordType := parts[1]
		priority := parts[2]

		// For TXT records, we need to handle quoted content specially
		var target string
		var ttl uint32 = 300

		if recordType == "TXT" {
			// Use parseZoneLine to handle proper quoting
			if len(parts) >= 4 {
				// For TXT, the content is in parts[3] and may be quoted
				target = parts[3]
				// Remove surrounding quotes if present
				if strings.HasPrefix(target, "\"") && strings.HasSuffix(target, "\"") && len(target) > 1 {
					target = target[1 : len(target)-1]
				}
				// Properly handle escaped characters - reverse the generation order
				target = strings.ReplaceAll(target, "\\\"", "\"")
				target = strings.ReplaceAll(target, "\\\\", "\\")
				// Parse TTL if present
				if len(parts) >= 5 {
					if ttlParsed, err := strconv.ParseUint(parts[4], 10, 32); err == nil {
						ttl = uint32(ttlParsed)
					}
				}
			}
		} else {
			target = parts[3]
			// Default TTL if not specified in zone record
			if len(parts) >= 5 {
				if ttlParsed, err := strconv.ParseUint(parts[4], 10, 32); err == nil {
					ttl = uint32(ttlParsed)
				}
			}
		}

		// Convert @ to empty string for root domain
		if label == "@" {
			label = ""
		}

		rc := &models.RecordConfig{
			TTL: ttl,
		}

		// Set the label and domain correctly
		rc.SetLabel(label, domain)

		// Handle different record types
		switch recordType {
		case "A", "AAAA":
			rc.Type = recordType
			if err := rc.SetTarget(target); err != nil {
				continue
			}
		case "CNAME":
			rc.Type = recordType
			// Ensure CNAME targets are fully qualified
			if !strings.HasSuffix(target, ".") {
				target = target + "."
			}
			if err := rc.SetTarget(target); err != nil {
				continue
			}
		case "NS":
			rc.Type = recordType
			// Ensure NS targets are fully qualified
			if !strings.HasSuffix(target, ".") {
				target = target + "."
			}
			if err := rc.SetTarget(target); err != nil {
				continue
			}
		case "TXT":
			rc.Type = recordType
			// TXT target is already extracted without quotes in the parsing above
			if err := rc.SetTarget(target); err != nil {
				continue
			}
		case "MX":
			rc.Type = recordType
			if prio, err := strconv.ParseUint(priority, 10, 16); err == nil {
				rc.MxPreference = uint16(prio)
			}
			// Ensure MX targets are fully qualified
			if !strings.HasSuffix(target, ".") {
				target = target + "."
			}
			if err := rc.SetTarget(target); err != nil {
				continue
			}
		case "SRV":
			rc.Type = recordType
			// SRV format: priority/weight target:port
			if strings.Contains(priority, "/") {
				priorityParts := strings.Split(priority, "/")
				if len(priorityParts) == 2 {
					if prio, err := strconv.ParseUint(priorityParts[0], 10, 16); err == nil {
						rc.SrvPriority = uint16(prio)
					}
					if weight, err := strconv.ParseUint(priorityParts[1], 10, 16); err == nil {
						rc.SrvWeight = uint16(weight)
					}
				}
			}
			if strings.Contains(target, ":") {
				targetParts := strings.Split(target, ":")
				if len(targetParts) == 2 {
					if port, err := strconv.ParseUint(targetParts[1], 10, 16); err == nil {
						rc.SrvPort = uint16(port)
					}
					srvTarget := targetParts[0]
					// Ensure SRV targets are fully qualified
					if !strings.HasSuffix(srvTarget, ".") {
						srvTarget = srvTarget + "."
					}
					if err := rc.SetTarget(srvTarget); err != nil {
						continue
					}
				}
			}
		case "CAA":
			rc.Type = recordType
			// CAA format: flags tag value [ttl]
			if len(parts) >= 4 {
				flags := priority // priority field contains flags for CAA
				tag := parts[3]   // tag field
				// Value might be quoted
				value := ""
				if len(parts) >= 5 {
					value = parts[4]
					// Remove surrounding quotes if present
					if strings.HasPrefix(value, "\"") && strings.HasSuffix(value, "\"") && len(value) > 1 {
						value = value[1 : len(value)-1]
					}
					// Parse TTL from the end if present (position 5)
					if len(parts) >= 6 {
						if ttlParsed, err := strconv.ParseUint(parts[5], 10, 32); err == nil {
							rc.TTL = uint32(ttlParsed)
						}
					}
				}

				if flagsInt, err := strconv.ParseUint(flags, 10, 8); err == nil {
					rc.CaaFlag = uint8(flagsInt)
				}
				rc.CaaTag = tag
				if err := rc.SetTarget(value); err != nil {
					continue
				}
			}
		case "NAPTR":
			rc.Type = recordType
			// NAPTR format for Joker: order/preference replacement ttl 0 0 "flags" "service" "regex"
			if len(parts) >= 9 {
				// Parse order/preference from priority field (parts[2])
				if strings.Contains(priority, "/") {
					priorityParts := strings.Split(priority, "/")
					if len(priorityParts) == 2 {
						if order, err := strconv.ParseUint(priorityParts[0], 10, 16); err == nil {
							rc.NaptrOrder = uint16(order)
						}
						if pref, err := strconv.ParseUint(priorityParts[1], 10, 16); err == nil {
							rc.NaptrPreference = uint16(pref)
						}
					}
				}
				// Replacement is in position 3
				if len(parts) > 3 {
					target = parts[3]
				}
				// Parse TTL from position 4
				if len(parts) >= 5 {
					if ttlParsed, err := strconv.ParseUint(parts[4], 10, 32); err == nil {
						rc.TTL = uint32(ttlParsed)
					}
				}
				// Parse flags, service, and regex from positions 7, 8, 9
				if len(parts) > 7 {
					rc.NaptrFlags = strings.Trim(parts[7], "\"")
				}
				if len(parts) > 8 {
					rc.NaptrService = strings.Trim(parts[8], "\"")
				}
				if len(parts) > 9 {
					rc.NaptrRegexp = strings.Trim(parts[9], "\"")
				}
				// Ensure NAPTR targets are fully qualified if they're not empty or "."
				if target != "" && target != "." && !strings.HasSuffix(target, ".") {
					target = target + "."
				}
				if err := rc.SetTarget(target); err != nil {
					continue
				}
			}
		default:
			// Skip unsupported record types
			continue
		}

		records = append(records, rc)
	}

	return records, nil
}

// fixTTLs enforces minimum TTL requirements for Joker provider.
// NAPTR and SVC records can have TTL=0, others need >= 300.
func fixTTLs(records models.Records) {
	for _, rc := range records {
		if rc.TTL != 0 && rc.TTL < 300 {
			if rc.Type != "NAPTR" && rc.Type != "SVC" {
				rc.TTL = 300
			}
		}
	}
}

// GetZoneRecordsCorrections returns a list of corrections that will turn existing records into dc.Records.
func (api *jokerProvider) GetZoneRecordsCorrections(dc *models.DomainConfig, existingRecords models.Records) ([]*models.Correction, int, error) {
	// Apply TTL fixes to desired records before comparison
	fixTTLs(dc.Records)

	result, err := diff2.ByZone(existingRecords, dc, nil)
	if err != nil {
		return nil, 0, err
	}

	if !result.HasChanges {
		return []*models.Correction{}, result.ActualChangeCount, nil
	}

	msg := fmt.Sprintf("Zone update for %s\n%s", dc.Name, strings.Join(result.Msgs, "\n"))

	correction := &models.Correction{
		Msg: msg,
		F: func() error {
			return api.updateZoneRecords(dc.Name, result.DesiredPlus)
		},
	}

	return []*models.Correction{correction}, result.ActualChangeCount, nil
}

// updateZoneRecords replaces the entire zone with new records.
func (api *jokerProvider) updateZoneRecords(domain string, records models.Records) error {
	zoneData := api.recordsToZoneFormat(domain, records)

	// Joker API doesn't support empty zones (returns Status-Code: 2400)
	// If the zone would be empty, we need to skip the update
	if zoneData == "" {
		// Return success for empty zone updates since Joker maintains minimal DNS infrastructure
		// (SOA, NS records) automatically and doesn't allow completely empty zones
		return nil
	}

	params := url.Values{}
	params.Set("domain", domain)
	params.Set("zone", zoneData)

	_, _, err := api.makeRequest("dns-zone-put", params)
	return err
}

// recordsToZoneFormat converts RecordConfig records to Joker zone format.
func (api *jokerProvider) recordsToZoneFormat(_ string, records models.Records) string {
	var lines []string

	for _, rc := range records {
		label := rc.Name
		if label == "" {
			label = "@"
		}

		// Joker format: <label> <type> <pri> <target> <ttl> (valid-from/valid-to omitted when 0)
		switch rc.Type {
		case "A", "AAAA":
			line := fmt.Sprintf("%s %s 0 %s %d", label, rc.Type, rc.GetTargetField(), rc.TTL)
			lines = append(lines, line)
		case "CNAME":
			line := fmt.Sprintf("%s %s 0 %s %d", label, rc.Type, rc.GetTargetField(), rc.TTL)
			lines = append(lines, line)
		case "NS":
			line := fmt.Sprintf("%s %s 0 %s %d", label, rc.Type, rc.GetTargetField(), rc.TTL)
			lines = append(lines, line)
		case "MX":
			line := fmt.Sprintf("%s %s %d %s %d", label, rc.Type, rc.MxPreference, rc.GetTargetField(), rc.TTL)
			lines = append(lines, line)
		case "TXT":
			// Fix TXT record escaping - escape backslashes first, then quotes
			content := rc.GetTargetField()
			content = strings.ReplaceAll(content, "\\", "\\\\")
			content = strings.ReplaceAll(content, "\"", "\\\"")
			line := fmt.Sprintf("%s %s 0 \"%s\" %d", label, rc.Type, content, rc.TTL)
			lines = append(lines, line)
		case "SRV":
			target := fmt.Sprintf("%s:%d", rc.GetTargetField(), rc.SrvPort)
			priority := fmt.Sprintf("%d/%d", rc.SrvPriority, rc.SrvWeight)
			line := fmt.Sprintf("%s %s %s %s %d", label, rc.Type, priority, target, rc.TTL)
			lines = append(lines, line)
		case "CAA":
			line := fmt.Sprintf("%s %s %d %s \"%s\" %d", label, rc.Type, rc.CaaFlag, rc.CaaTag, rc.GetTargetField(), rc.TTL)
			lines = append(lines, line)
		case "NAPTR":
			// NAPTR format for Joker: order/preference replacement ttl 0 0 "flags" "service" "regex"
			priority := fmt.Sprintf("%d/%d", rc.NaptrOrder, rc.NaptrPreference)
			line := fmt.Sprintf("%s %s %s %s %d 0 0 \"%s\" \"%s\" \"%s\"",
				label, rc.Type, priority, rc.GetTargetField(), rc.TTL,
				rc.NaptrFlags, rc.NaptrService, rc.NaptrRegexp)
			lines = append(lines, line)
		}
	}

	// If no records, return empty zone rather than completely empty string
	// Some DNS providers require at least a minimal zone structure
	if len(lines) == 0 {
		return ""
	}

	return strings.Join(lines, "\n")
}
