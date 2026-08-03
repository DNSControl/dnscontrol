package dnsrr

import (
	"fmt"
	"strings"

	dnsv2 "codeberg.org/miekg/dns"
	dnsrdatav2 "codeberg.org/miekg/dns/rdata"
	"github.com/DNSControl/dnscontrol/v5/models"
	"github.com/DNSControl/dnscontrol/v5/pkg/nrc"
)

// RRv2toRC converts dnsv2.RR to models.RecordConfig. It assumes:
// - The header.Name is a fully qualified domain name (FQDN) with a trailing dot.
// - The fields in the RDATA are already in the correct format (e.g., IDNA for domain names), FQDN+"." for targets.
func RRv2toRC(dc *models.DomainConfig, rr dnsv2.RR) (*models.RecordConfig, error) {
	// Convert's dnsv2.RR into DNSControl's models.RecordConfig struct.

	header := rr.Header()
	ttl := header.TTL
	typeNum := dnsv2.RRToType(rr)
	rd := rr.Data()

	var err error
	switch v := rd.(type) {
	case dnsrdatav2.TXT:
		// DNSControl stores a TXT value as a single string, so join the
		// parser's 255-octet chunks. The patched dns library (codeberg.org/miekg/dns)
		// unescapes backslashes correctly, so no further fix-up is needed
		// (previously we compensated for github.com/miekg/dns/issues/1384 here).
		rd = dnsrdatav2.TXT{Txt: []string{strings.Join(v.Txt, "")}}
	case dnsrdatav2.TLSA:
		// TLSA is a special case because we need to normalize the certificate data to uppercase.
		rd, err = models.MakeTLSA(dc.Name, nil, nrc.Flags{}, v.Usage, v.Selector, v.MatchingType, v.Certificate)
		if err != nil {
			return nil, fmt.Errorf("error creating TLSA record: %w", err)
		}
	}

	rec, err := dc.NewRecordConfigForRRv2toRC(dc.LabelFromFQDNWithDot(header.Name), ttl, typeNum, rd)
	if err != nil {
		return nil, fmt.Errorf("error creating record config: %w", err)
	}
	return rec, nil
}
