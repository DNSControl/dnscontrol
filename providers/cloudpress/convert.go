package cloudpress

import (
	"fmt"
	"slices"
	"strings"

	dnsv2 "codeberg.org/miekg/dns"
	dnsrdatav2 "codeberg.org/miekg/dns/rdata"
	"github.com/DNSControl/dnscontrol/v5/models"
)

// recordType is the integer record-type discriminator used by the CloudPress
// API. The numbering mirrors the Bunny DNS scheme (CloudPress is derived from
// the same platform; both use record_type == 7 for pull zones).
type recordType int

const (
	recordTypeA        recordType = 0
	recordTypeAAAA     recordType = 1
	recordTypeCNAME    recordType = 2
	recordTypeTXT      recordType = 3
	recordTypeMX       recordType = 4
	recordTypeRedirect recordType = 5
	recordTypeFlatten  recordType = 6
	recordTypePullZone recordType = 7
	recordTypeSRV      recordType = 8
	recordTypeCAA      recordType = 9
	recordTypePTR      recordType = 10
	recordTypeScript   recordType = 11
	recordTypeNS       recordType = 12
)

// fqdnTypes are record types whose Value holds a hostname that CloudPress
// stores fully-qualified and without a trailing dot.
var fqdnTypes = []recordType{recordTypeCNAME, recordTypeMX, recordTypeNS, recordTypePTR, recordTypeSRV}

// recordTypeFromString maps a DNSControl rtype to its CloudPress integer.
// It panics on unsupported types: those are filtered out before conversion via
// the capabilities list, so reaching this default indicates a programming error.
func recordTypeFromString(t string) recordType {
	switch t {
	case "A":
		return recordTypeA
	case "AAAA":
		return recordTypeAAAA
	case "CNAME":
		return recordTypeCNAME
	case "TXT":
		return recordTypeTXT
	case "MX":
		return recordTypeMX
	case "SRV":
		return recordTypeSRV
	case "CAA":
		return recordTypeCAA
	case "PTR":
		return recordTypePTR
	case "NS":
		return recordTypeNS
	default:
		panic(fmt.Errorf("CLOUDPRESS: rtype %v unimplemented", t))
	}
}

// recordTypeToString maps a CloudPress integer to a DNSControl rtype. Known but
// unsupported types return a name so they can be reported and skipped; a truly
// unknown integer panics so a silently-changed API surfaces during testing.
func recordTypeToString(t recordType) string {
	switch t {
	case recordTypeA:
		return "A"
	case recordTypeAAAA:
		return "AAAA"
	case recordTypeCNAME:
		return "CNAME"
	case recordTypeTXT:
		return "TXT"
	case recordTypeMX:
		return "MX"
	case recordTypeSRV:
		return "SRV"
	case recordTypeCAA:
		return "CAA"
	case recordTypePTR:
		return "PTR"
	case recordTypeNS:
		return "NS"
	case recordTypeRedirect:
		return "REDIRECT"
	case recordTypeFlatten:
		return "FLATTEN"
	case recordTypePullZone:
		return "PULLZONE"
	case recordTypeScript:
		return "SCRIPT"
	default:
		panic(fmt.Errorf("CLOUDPRESS: native rtype %v unimplemented", t))
	}
}

// isSupported reports whether a native record type can be represented as a
// standard DNSControl record. Unsupported types are left untouched in the zone.
func isSupported(t recordType) bool {
	switch t {
	case recordTypeA, recordTypeAAAA, recordTypeCNAME, recordTypeTXT,
		recordTypeMX, recordTypeSRV, recordTypeCAA, recordTypePTR, recordTypeNS:
		return true
	default:
		return false
	}
}

func deref16(p *uint16) uint16 {
	if p == nil {
		return 0
	}
	return *p
}

func deref8(p *uint8) uint8 {
	if p == nil {
		return 0
	}
	return *p
}

// fromRecordConfig converts a DNSControl record into the CloudPress wire format.
// CloudPress expects the fully-qualified name and strips the zone suffix itself
// (so the apex is sent as the bare zone name).
func fromRecordConfig(rc *models.RecordConfig) *record {
	r := record{
		RecordType: recordTypeFromString(rc.Type),
		Name:       rc.GetLabelFQDN(),
		TTL:        rc.TTL,
	}

	switch r.RecordType {
	case recordTypeSRV:
		rd := rc.GetRDATA().(dnsrdatav2.SRV)
		r.Priority = &rd.Priority
		r.Weight = &rd.Weight
		r.Port = &rd.Port
		r.Value = rd.Target
	case recordTypeMX:
		rd := rc.GetRDATA().(dnsrdatav2.MX)
		r.Priority = &rd.Preference
		r.Value = rd.Mx
	case recordTypeCAA:
		rd := rc.GetRDATA().(dnsrdatav2.CAA)
		r.Flags = &rd.Flag
		r.Tag = rd.Tag
		r.Value = rd.Value
	case recordTypeTXT:
		r.Value = rc.GetTargetTXTJoined()
	default:
		r.Value = rc.GetRDATA().String()
	}

	// CloudPress stores hostnames without a trailing dot, so strip it. The
	// exception is a bare "." which is a null target (NullMX, null SRV target):
	// CloudPress accepts and preserves it, while an empty value is rejected.
	if slices.Contains(fqdnTypes, r.RecordType) && r.Value != "." {
		r.Value = strings.TrimSuffix(r.Value, ".")
	}

	return &r
}

// toRecordConfig converts a CloudPress record into a DNSControl record.
func toRecordConfig(dc *models.DomainConfig, r *record) (*models.RecordConfig, error) {
	rtype := recordTypeToString(r.RecordType)

	// CloudPress returns the short label for sub-records ("www") and the bare
	// zone name for the apex ("example.com"). Normalize both to a DNSControl
	// label ("@" for the apex).
	label := r.Name
	switch {
	case label == dc.Name:
		label = "@"
	case strings.HasSuffix(label, "."+dc.Name):
		label = strings.TrimSuffix(label, "."+dc.Name)
	}
	label = dc.LabelFromShort(label)

	// CloudPress returns hostnames without a trailing dot. Add the dot back so
	// the value is an absolute target DNSControl can parse.
	value := r.Value
	if slices.Contains(fqdnTypes, r.RecordType) && !strings.HasSuffix(value, ".") {
		value += "."
	}

	var rc *models.RecordConfig
	var err error
	switch rtype {
	case "CAA":
		rc, err = dc.NewRecordConfig(label, r.TTL, dnsv2.TypeCAA, deref8(r.Flags), r.Tag, value)
	case "MX":
		rc, err = dc.NewRecordConfig(label, r.TTL, dnsv2.TypeMX, deref16(r.Priority), value)
	case "SRV":
		rc, err = dc.NewRecordConfig(label, r.TTL, dnsv2.TypeSRV, deref16(r.Priority), deref16(r.Weight), deref16(r.Port), value)
	case "TXT":
		rc, err = dc.NewRecordConfig(label, r.TTL, dnsv2.TypeTXT, value)
	default:
		rc, err = dc.NewRecordConfigParse(label, r.TTL, rtype, value)
	}
	if err != nil {
		return nil, err
	}

	rc.Original = r
	return rc, nil
}
