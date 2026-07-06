package dnsrr

import (
	"fmt"
	"strings"

	dnsv2 "codeberg.org/miekg/dns"
	dnsrdatav2 "codeberg.org/miekg/dns/rdata"
	"github.com/DNSControl/dnscontrol/v4/models"
	dnsv1 "github.com/miekg/dns"
)

// RRtoRC converts dns.RR to models.RecordConfig.
func RRtoRC(rr dnsv1.RR, origin string) (models.RecordConfig, error) {
	return helperRRtoRC(rr, origin, false)
}

// RRtoRCTxtBug converts dns.RR to models.RecordConfig. Compensates for the backslash bug in github.com/miekg/dns/issues/1384.
func RRtoRCTxtBug(rr dnsv1.RR, origin string) (models.RecordConfig, error) {
	return helperRRtoRC(rr, origin, true)
}

// helperRRtoRC converts dns.RR to models.RecordConfig. If fixBug is true, replaces `\\` to `\` in TXT records to compensate for github.com/miekg/dns/issues/1384.
func helperRRtoRC(rr dnsv1.RR, origin string, fixBug bool) (models.RecordConfig, error) {
	// Convert's dns.RR into DNSControl's models.RecordConfig struct.

	header := rr.Header()
	typeNum := header.Rrtype

	rc := new(models.RecordConfig)
	rc.Type = dnsv2.TypeToString[header.Rrtype]
	rc.TTL = header.Ttl
	rc.Original = rr
	rc.SetLabelFromFQDN(header.Name, origin)
	var err error
	switch v := rr.(type) { // #rtype_variations

	case *dnsv1.A:
		if rec, err := models.NewRecordConfigForRRtoRC(origin, header.Name, header.Ttl, typeNum,
			v.A); err == nil {
			return *rec, nil
		}
	case *dnsv1.AAAA:
		if rec, err := models.NewRecordConfigForRRtoRC(origin, header.Name, header.Ttl, typeNum,
			v.AAAA); err == nil {
			return *rec, nil
		}

	case *dnsv1.CAA:
		if rec, err := models.NewRecordConfigForRRtoRC(origin, header.Name, header.Ttl, typeNum,
			v.Flag, v.Tag, v.Value); err == nil {
			return *rec, nil
		}
	case *dnsv1.CNAME:
		if rec, err := models.NewRecordConfigForRRtoRC(origin, header.Name, header.Ttl, typeNum,
			v.Target); err == nil {
			return *rec, nil
		}

	case *dnsv1.DHCID:
		if rec, err := models.NewRecordConfigForRRtoRC(origin, header.Name, header.Ttl, typeNum,
			v.Digest); err == nil {
			return *rec, nil
		}
	case *dnsv1.DNAME:
		if rec, err := models.NewRecordConfigForRRtoRC(origin, header.Name, header.Ttl, typeNum,
			v.Target); err == nil {
			return *rec, nil
		}
	case *dnsv1.DS:
		if rec, err := models.NewRecordConfigForRRtoRC(origin, header.Name, header.Ttl, typeNum,
			v.KeyTag, v.Algorithm, v.DigestType, v.Digest); err == nil {
			return *rec, nil
		}
	case *dnsv1.DNSKEY:
		if rec, err := models.NewRecordConfigForRRtoRC(origin, header.Name, header.Ttl, typeNum,
			v.Flags, v.Protocol, v.Algorithm, v.PublicKey); err == nil {
			return *rec, nil
		}

	case *dnsv1.HTTPS:
		if rec, err := models.NewRecordConfigForRRtoRC(origin, header.Name, header.Ttl, typeNum,
			v.Priority, v.Target, v.Value); err == nil {
			return *rec, nil
		}

	case *dnsv1.LOC:
		if rec, err := models.NewRecordConfigForRRtoRC(origin, header.Name, header.Ttl, typeNum,
			v.Version, v.Size, v.HorizPre, v.VertPre, v.Latitude, v.Longitude, v.Altitude); err == nil {
			return *rec, nil
		}

	case *dnsv1.MX:
		if rec, err := models.NewRecordConfigForRRtoRC(origin, header.Name, header.Ttl, typeNum,
			v.Preference, v.Mx); err == nil {
			return *rec, nil
		}

	case *dnsv1.NAPTR:
		if rec, err := models.NewRecordConfigForRRtoRC(origin, header.Name, header.Ttl, typeNum,
			v.Order, v.Preference, v.Flags, v.Service, v.Regexp, v.Replacement); err == nil {
			return *rec, nil
		}

	case *dnsv1.OPENPGPKEY:
		if rec, err := models.NewRecordConfigForRRtoRC(origin, header.Name, header.Ttl, typeNum,
			v.PublicKey); err == nil {
			return *rec, nil
		}

	case *dnsv1.NS:
		if rec, err := models.NewRecordConfigForRRtoRC(origin, header.Name, header.Ttl, typeNum,
			v.Ns); err == nil {
			return *rec, nil
		}

	case *dnsv1.PTR:
		if rec, err := models.NewRecordConfigForRRtoRC(origin, header.Name, header.Ttl, typeNum,
			v.Ptr); err == nil {
			return *rec, nil
		}
	case *dnsv1.RP:
		if rec, err := models.NewRecordConfigForRRtoRC(origin, header.Name, header.Ttl, typeNum, v.Mbox, v.Txt); err == nil {
			return *rec, nil
		}
	case *dnsv1.SMIMEA:
		if rec, err := models.NewRecordConfigForRRtoRC(origin, header.Name, header.Ttl, typeNum,
			v.Usage, v.Selector, v.MatchingType, v.Certificate); err == nil {
			return *rec, nil
		}
	case *dnsv1.SOA:
		if rec, err := models.NewRecordConfigForRRtoRC(origin, header.Name, header.Ttl, typeNum,
			v.Ns, v.Mbox, v.Serial, v.Refresh, v.Retry, v.Expire, v.Minttl); err == nil {
			return *rec, nil
		}
	case *dnsv1.SRV:
		if rec, err := models.NewRecordConfigForRRtoRC(origin, header.Name, header.Ttl, typeNum,
			v.Priority, v.Weight, v.Port, v.Target); err == nil {
			return *rec, nil
		}
	case *dnsv1.SSHFP:
		if rec, err := models.NewRecordConfigForRRtoRC(origin, header.Name, header.Ttl, typeNum,
			v.Algorithm, v.Type, v.FingerPrint); err == nil {
			return *rec, nil
		}
	case *dnsv1.SVCB:
		if rec, err := models.NewRecordConfigForRRtoRC(origin, header.Name, header.Ttl, typeNum,
			v.Priority, v.Target, v.Value); err == nil {
			return *rec, nil
		}

	case *dnsv1.TLSA:
		if rec, err := models.NewRecordConfigForRRtoRC(origin, header.Name, header.Ttl, typeNum,
			v.Usage, v.Selector, v.MatchingType, v.Certificate); err == nil {
			return *rec, nil
		}
	case *dnsv1.TXT:
		if fixBug {
			t := strings.Join(v.Txt, "")
			te := t
			te = strings.ReplaceAll(te, `\\`, `\`)
			te = strings.ReplaceAll(te, `\"`, `"`)
			err = rc.SetTargetTXT(te)
		} else {
			err = rc.SetTargetTXTs(v.Txt)
		}
	default:
		return *rc, fmt.Errorf("rrToRecord: Unimplemented zone record type=%s (%v)", rc.Type, rr)
	}
	if err != nil {
		return *rc, fmt.Errorf("unparsable record received: %w", err)
	}
	return *rc, nil
}

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
		// TXT is special case because we need to work around the backslack.
		rd = dnsrdatav2.TXT{Txt: []string{fixRRv2TXT(v)}}
	case dnsrdatav2.TLSA:
		// TLSA is a special case because we need to normalize the certificate data to uppercase.
		rd, err = models.MakeTLSA(dc.Name, nil, v.Usage, v.Selector, v.MatchingType, v.Certificate)
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

func fixRRv2TXT(rd dnsrdatav2.TXT) string {
	j := strings.Join(rd.Txt, "")
	j = strings.ReplaceAll(j, `\\`, `\`)
	return j
}
