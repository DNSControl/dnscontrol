package cloudpress

import (
	"fmt"
	"slices"
	"strings"

	"github.com/DNSControl/dnscontrol/v4/models"
	dnsutilv1 "github.com/miekg/dns/dnsutil"
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

func u16(v uint16) *uint16 { return &v }
func u8(v uint8) *uint8     { return &v }

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
		Value:      rc.GetTargetField(),
		TTL:        rc.TTL,
	}

	switch r.RecordType {
	case recordTypeSRV:
		r.Priority = u16(rc.SrvPriority)
		r.Weight = u16(rc.SrvWeight)
		r.Port = u16(rc.SrvPort)
	case recordTypeMX:
		r.Priority = u16(rc.MxPreference)
	case recordTypeCAA:
		r.Flags = u8(rc.CaaFlag)
		r.Tag = rc.CaaTag
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
func toRecordConfig(domain string, r *record) (*models.RecordConfig, error) {
	rc := models.RecordConfig{
		Type:     recordTypeToString(r.RecordType),
		TTL:      r.TTL,
		Original: r,
	}

	// CloudPress returns the short label for sub-records ("www") and the bare
	// zone name for the apex ("example.com"). Normalize both to a DNSControl
	// label ("@" for the apex).
	label := r.Name
	switch {
	case label == domain:
		label = "@"
	case strings.HasSuffix(label, "."+domain):
		label = strings.TrimSuffix(label, "."+domain)
	}
	rc.SetLabel(label, domain)

	// CloudPress returns hostnames without a trailing dot. Add the dot back so
	// the value is an absolute target DNSControl can parse.
	value := r.Value
	if slices.Contains(fqdnTypes, r.RecordType) && !strings.HasSuffix(value, ".") {
		value = dnsutilv1.AddOrigin(value+".", domain)
	}

	var err error
	switch rc.Type {
	case "CAA":
		err = rc.SetTargetCAA(deref8(r.Flags), r.Tag, value)
	case "MX":
		err = rc.SetTargetMX(deref16(r.Priority), value)
	case "SRV":
		err = rc.SetTargetSRV(deref16(r.Priority), deref16(r.Weight), deref16(r.Port), value)
	default:
		err = rc.PopulateFromStringFunc(rc.Type, value, domain, nil)
	}
	if err != nil {
		return nil, err
	}

	return &rc, nil
}
