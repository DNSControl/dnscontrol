package sakuracloud

import (
	"github.com/DNSControl/dnscontrol/v4/models"
)

const defaultTTL = uint32(3600)

func toRc(domain string, r domainRecord) (*models.RecordConfig, error) {
	rc := &models.RecordConfig{
		Type:     r.Type,
		TTL:      r.TTL,
		Original: r,
	}
	if r.TTL == 0 {
		rc.TTL = defaultTTL
	}

	rc.SetLabel(r.Name, domain)

	var err error
	switch r.Type {
	case "TXT":
		// TXT records are stored verbatim; no quoting/escaping to parse.
		err = rc.SetTargetTXT(r.RData)
	default:
		err = rc.PopulateFromString(r.Type, r.RData, domain)
	}
	return rc, err
}

func toNative(rc *models.RecordConfig) domainRecord {
	rr := domainRecord{
		Name:  rc.GetLabel(),
		Type:  rc.Type,
		RData: rc.String(),
	}
	if rc.TTL != defaultTTL {
		rr.TTL = rc.TTL
	}

	switch rc.Type {
	case "TXT":
		rr.RData = rc.GetTargetTXTJoined()
	}
	return rr
}
