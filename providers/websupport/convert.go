package websupport

import (
	"fmt"
	"strings"

	dnsv2 "codeberg.org/miekg/dns"
	"github.com/DNSControl/dnscontrol/v5/models"
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

	switch rc.TypeNum {
	case dnsv2.TypeMX:
		f := rc.AsMX()
		r.Priority = intPtr(f.Preference)
		r.Content = trimDot(f.Mx)
	case dnsv2.TypeSRV:
		f := rc.AsSRV()
		r.Priority = intPtr(f.Priority)
		r.Weight = intPtr(f.Weight)
		r.Port = intPtr(f.Port)
		r.Content = trimDot(f.Target)
	case dnsv2.TypeTXT:
		r.Content = rc.GetTargetTXTJoined()
	case dnsv2.TypeCNAME:
		f := rc.AsCNAME()
		r.Content = trimDot(f.Target)
	default:
		r.Content = rc.GetRDATA().String()
		// TODO(tlim): This was the original:
		//r.Content = rc.Get TargetField()
	}

	return r, nil
}

// toRecordConfig converts a WebSupport native record into a dnscontrol RecordConfig.
func toRecordConfig(dc *models.DomainConfig, n nativeRecord) (*models.RecordConfig, error) {
	content := n.Content
	if fqdnTypes[n.Type] {
		content = ensureDot(content)
	}

	label := dc.LabelFromFQDNNoDot(n.Name)
	ttl := n.TTL
	var rc *models.RecordConfig
	var err error
	switch n.Type {
	case "MX":
		rc, err = dc.NewRecordConfig(label, ttl, n.Type, derefInt(n.Priority), content)
	case "SRV":
		rc, err = dc.NewRecordConfig(label, ttl, n.Type, derefInt(n.Priority), derefInt(n.Weight), derefInt(n.Port), content)
	case "TXT":
		rc, err = dc.NewRecordConfig(label, ttl, n.Type, n.Content)
	default:
		rc, err = dc.NewRecordConfig(label, ttl, n.Type, content)
	}
	if err != nil {
		return nil, fmt.Errorf("WEBSUPPORT: %s record %q: %w", n.Type, n.Name, err)
	}
	rc.Original = n
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
