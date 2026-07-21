package sakuracloud

import (
	"strings"

	"github.com/DNSControl/dnscontrol/v5/models"
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
	contents := rc.String()
	contents = strings.ReplaceAll(contents, `"`, ``)
	rr := domainRecord{
		Name:  rc.GetLabel(),
		Type:  rc.Type,
		RData: contents,
	}
	if rc.TTL != defaultTTL {
		rr.TTL = rc.TTL
	}

	switch rc.Type {
	case "TXT":
		rr.RData = rc.GetTargetTXTJoined()
	case "CAA":
		// SakuraCloud requires the CAA value to remain quoted, e.g.
		// `0 issue "letsencrypt.org"`. The generic quote-stripping above
		// produces `0 issue letsencrypt.org`, which the API rejects as
		// malformed.
		rr.RData = rc.GetTargetCombined()
	}
	return rr
}
