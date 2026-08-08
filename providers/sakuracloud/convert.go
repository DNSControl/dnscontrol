package sakuracloud

import (
	"strings"

	dnsv2 "codeberg.org/miekg/dns"
	"github.com/DNSControl/dnscontrol/v5/models"
)

const defaultTTL = uint32(3600)

func toRc(dc *models.DomainConfig, r domainRecord) (*models.RecordConfig, error) {
	ttl := r.TTL
	if ttl == 0 {
		ttl = defaultTTL
	}
	label := dc.LabelFromShort(r.Name)

	var rc *models.RecordConfig
	var err error
	switch r.Type {
	case "TXT":
		// TXT records are stored verbatim; no quoting/escaping to parse.
		rc, err = dc.NewRecordConfig(label, ttl, r.Type, r.RData)
	default:
		rc, err = dc.NewRecordConfigParse(label, ttl, r.Type, r.RData)
	}
	if err != nil {
		return nil, err
	}
	rc.Original = r
	return rc, nil
}

func toNative(rc *models.RecordConfig) domainRecord {
	contentsOrig := rc.GetRDATA().String()
	contents := strings.ReplaceAll(contentsOrig, `"`, ``)
	rr := domainRecord{
		Name:  rc.GetLabel(),
		Type:  rc.Type,
		RData: contents,
	}
	if rc.TTL != defaultTTL {
		rr.TTL = rc.TTL
	}

	switch rc.TypeNum {
	case dnsv2.TypeTXT:
		rr.RData = rc.GetTargetTXTJoined()
	case dnsv2.TypeCAA:
		// SakuraCloud requires the CAA value to remain quoted, e.g.
		// `0 issue "letsencrypt.org"`. The generic quote-stripping above
		// produces `0 issue letsencrypt.org`, which the API rejects as
		// malformed.
		rr.RData = contentsOrig
	}
	return rr
}
