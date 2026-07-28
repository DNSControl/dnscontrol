package fortigate

import (
	"fmt"
	"strings"

	"github.com/DNSControl/dnscontrol/v5/models"
)

// nativeToRecord – convert an fgDNSRecord coming from FortiGate into a *models.RecordConfig that dnscontrol understands.
func nativeToRecord(dc *models.DomainConfig, n fgDNSRecord) (*models.RecordConfig, error) {

	rtype := strings.ToUpper(n.Type)

	var label string
	hostname := n.Hostname
	if hostname == "@" || hostname == "" {
		label = "@"
	} else if hostname[len(hostname)-1] == '.' {
		label = dc.LabelFromFQDNWithDot(n.Hostname)
	} else {
		label = dc.LabelFromShort(n.Hostname)
	}
	ttl := n.TTL

	// Type-specific fields
	var rc *models.RecordConfig
	var err error
	switch rtype {
	case "A":
		rc, err = dc.NewRecordConfig(label, ttl, rtype, n.IP)
		if err != nil {
			return nil, fmt.Errorf("[FORTIGATE] Invalid IPv4 address %q in %+v", n.IP, n)
		}

	case "AAAA":
		rc, err = dc.NewRecordConfig(label, ttl, rtype, n.IPv6)
		if err != nil {
			return nil, fmt.Errorf("[FORTIGATE] Invalid IPv6 address %q in %+v", n.IPv6, n)
		}

	case "CNAME":
		if n.CanonicalName == "" {
			return nil, fmt.Errorf("[FORTIGATE] CNAME record without canonical-name (id=%d)", n.ID)
		}
		rc, err = dc.NewRecordConfig(label, ttl, rtype, n.CanonicalName)
		if err != nil {
			return nil, err
		}

	case "NS":
		if n.Hostname == "" {
			return nil, fmt.Errorf("[FORTIGATE] NS record missing hostname (id=%d)", n.ID)
		}

		rc, err = dc.NewRecordConfig("@", ttl, rtype, n.Hostname)
		if err != nil {
			return nil, err
		}

	case "MX":
		if n.Hostname == "" {
			return nil, fmt.Errorf("[FORTIGATE] MX record missing hostname (id=%d)", n.ID)
		}

		rc, err = dc.NewRecordConfig("@", ttl, rtype, n.Preference, n.Hostname)
		if err != nil {
			return nil, err
		}

	default:
		// Not supported due to FortiGate limitations
		return nil, fmt.Errorf("[FORTIGATE] Record type %q is not supported by fortigate provider", rtype)
	}
	rc.Original = n

	// Status → Metadata
	if strings.ToLower(n.Status) != "enable" {
		rc.Metadata["fortigate_status"] = "disable"
	}

	return rc, nil
}

func recordsToNative(recs models.Records) ([]*fgDNSRecord, []error) {
	var resourceRecords []*fgDNSRecord
	var errors []error

	id := 1

	for _, record := range recs {

		n := &fgDNSRecord{
			Status: "enable",
			Type:   strings.ToUpper(record.Type),
		}

		// TTL
		if ttl := record.TTL; ttl > 0 {
			n.TTL = ttl
		}

		// Wildcard support
		if strings.Contains(record.GetLabelFQDN(), "*") {
			errors = append(errors, fmt.Errorf("[FORTIGATE] Wildcard records are not supported: %s", record.GetLabelFQDN()))
			continue
		}

		// Status from Metadata
		if v, ok := record.Metadata["fortigate_status"]; ok && strings.ToLower(v) == "disable" {
			n.Status = "disable"
		}

		// Hostname (Label)
		label := record.GetLabel()
		if label == "" {
			label = "@"
		}
		n.Hostname = label

		// Type-specific fields
		switch n.Type {
		case "A":
			ip := record.GetTargetIP()
			if !ip.Is4() {
				errors = append(errors, fmt.Errorf("[FORTIGATE] A record is missing a valid IPv4 address: %s", record.GetLabelFQDN()))
				continue
			}
			n.IP = ip.String()

		case "AAAA":
			ip := record.GetTargetIP()
			if !ip.Is6() {
				errors = append(errors, fmt.Errorf("[FORTIGATE] AAAA record is missing a valid IPv6 address: %s", record.GetLabelFQDN()))
				continue
			}
			n.IPv6 = ip.String()

		case "CNAME":
			target := record.AsCNAME().String()
			//if ascii, err := idna.ToASCII(target); err == nil {
			//	target = ascii
			//}
			n.CanonicalName = target

		case "NS":
			target := record.AsNS().String()
			//if ascii, err := idna.ToASCII(target); err == nil {
			//	target = ascii
			//}
			n.Hostname = target
			n.CanonicalName = ""

		case "MX":
			mx := record.AsMX()
			//if ascii, err := idna.ToASCII(target); err == nil {
			//	target = ascii
			//}
			n.Hostname = mx.Mx
			n.Preference = mx.Preference
			n.CanonicalName = ""

		default:
			errors = append(errors, fmt.Errorf("[FORTIGATE] Record type %q is not supported: %s", n.Type, record.GetLabelFQDN()))
			continue
		}

		n.ID = id
		id++

		resourceRecords = append(resourceRecords, n)
	}

	return resourceRecords, errors
}
