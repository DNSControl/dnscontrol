package websupport

import (
	"fmt"
	"strings"

	"github.com/DNSControl/dnscontrol/v4/models"
)

// fqdnTypes are record types whose `content` holds a hostname that dnscontrol
// represents as a fully-qualified, dot-terminated target. WebSupport stores
// these without the trailing dot, so it is stripped on write and restored on
// read.
var fqdnTypes = map[string]bool{
	"CNAME": true,
	"MX":    true,
	"SRV":   true,
}

func intPtr(i uint16) *int {
	v := int(i)
	return &v
}

func derefInt(p *int) uint16 {
	if p == nil {
		return 0
	}
	return uint16(*p)
}

// toNative converts a dnscontrol RecordConfig into the WebSupport API shape.
func toNative(rc *models.RecordConfig) (nativeRecord, error) {
	// The WebSupport API is asymmetric about record names: GET returns the
	// fully-qualified name, but POST/PUT expect the relative label (the API
	// appends the zone itself). dnscontrol's GetLabel() returns "@" for the
	// apex, which the API accepts.
	r := nativeRecord{
		Type: rc.Type,
		Name: rc.GetLabel(),
		TTL:  rc.TTL,
	}

	switch rc.Type {
	case "MX":
		r.Content = trimDot(rc.GetTargetField())
		r.Priority = intPtr(rc.MxPreference)
	case "SRV":
		r.Content = trimDot(rc.GetTargetField())
		r.Priority = intPtr(rc.SrvPriority)
		r.Weight = intPtr(rc.SrvWeight)
		r.Port = intPtr(rc.SrvPort)
	case "TXT":
		r.Content = rc.GetTargetTXTJoined()
	case "CNAME":
		r.Content = trimDot(rc.GetTargetField())
	default:
		r.Content = rc.GetTargetField()
	}

	return r, nil
}

// toRecordConfig converts a WebSupport native record into a dnscontrol RecordConfig.
func toRecordConfig(domain string, n nativeRecord) (*models.RecordConfig, error) {
	rc := &models.RecordConfig{
		Type:     n.Type,
		TTL:      n.TTL,
		Original: n,
	}
	rc.SetLabelFromFQDN(n.Name, domain)

	content := n.Content
	if fqdnTypes[n.Type] {
		content = ensureDot(content)
	}

	var err error
	switch n.Type {
	case "MX":
		err = rc.SetTargetMX(derefInt(n.Priority), content)
	case "SRV":
		err = rc.SetTargetSRV(derefInt(n.Priority), derefInt(n.Weight), derefInt(n.Port), content)
	case "TXT":
		err = rc.SetTargetTXT(n.Content)
	default:
		err = rc.SetTarget(content)
	}
	if err != nil {
		return nil, fmt.Errorf("WEBSUPPORT: %s record %q: %w", n.Type, n.Name, err)
	}
	return rc, nil
}

func trimDot(s string) string {
	return strings.TrimSuffix(s, ".")
}

func ensureDot(s string) string {
	if s == "" || strings.HasSuffix(s, ".") {
		return s
	}
	return s + "."
}
