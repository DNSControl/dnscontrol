package dnsrr

import (
	"fmt"
	"strings"

	dnsv2 "codeberg.org/miekg/dns"
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
	//typeName := dnsv2.TypeToString[header.Rrtype]

	// if rtypeinfo.IsModernType(ty) {
	// 	switch v := rr.(type) {
	// 	default:
	// 		rec, err := rtypecontrol.NewRecordConfigFromStruct(strings.TrimSuffix(header.Name, origin), header.Ttl, dnsv2.TypeToString[header.Rrtype], v, domaintags.MakeDomainNameVarieties(origin))
	// 		return *rec, err
	// 	}
	// }

	rc := new(models.RecordConfig)
	rc.Type = dnsv2.TypeToString[header.Rrtype]
	rc.TTL = header.Ttl
	rc.Original = rr
	//rc.SetLabelFromFQDN(strings.TrimSuffix(header.Name, "."), origin)
	rc.SetLabelFromFQDN(header.Name, origin)
	var err error
	switch v := rr.(type) { // #rtype_variations
	case *dnsv1.A:
		err = rc.SetTarget(v.A.String())
	case *dnsv1.AAAA:
		err = rc.SetTarget(v.AAAA.String())
	case *dnsv1.CAA:
		err = rc.SetTargetCAA(v.Flag, v.Tag, v.Value)
	case *dnsv1.CNAME:
		err = rc.SetTarget(v.Target)
	case *dnsv1.DHCID:
		err = rc.SetTarget(v.Digest)
	case *dnsv1.DNAME:
		err = rc.SetTarget(v.Target)
	case *dnsv1.DS:
		if rec, err := models.NewRecordConfigForRRtoRC(origin, header.Name, header.Ttl, typeNum, v.KeyTag, v.Algorithm, v.DigestType, v.Digest); err == nil {
			return *rec, nil
		}
	case *dnsv1.DNSKEY:
		err = rc.SetTargetDNSKEY(v.Flags, v.Protocol, v.Algorithm, v.PublicKey)
	case *dnsv1.HTTPS:
		err = rc.SetTargetSVCB(v.Priority, v.Target, v.Value)
	case *dnsv1.LOC:
		err = rc.SetTargetLOC(v.Version, v.Latitude, v.Longitude, v.Altitude, v.Size, v.HorizPre, v.VertPre)
	case *dnsv1.MX:
		err = rc.SetTargetMX(v.Preference, v.Mx)
	case *dnsv1.NAPTR:
		err = rc.SetTargetNAPTR(v.Order, v.Preference, v.Flags, v.Service, v.Regexp, v.Replacement)
	case *dnsv1.OPENPGPKEY:
		err = rc.SetTarget(v.PublicKey)
	case *dnsv1.NS:
		err = rc.SetTarget(v.Ns)
	case *dnsv1.PTR:
		err = rc.SetTarget(v.Ptr)
	case *dnsv1.RP:
		if rec, err := models.NewRecordConfigForRRtoRC(origin, header.Name, header.Ttl, typeNum, v.Mbox, v.Txt); err == nil {
			return *rec, nil
		}
	case *dnsv1.SMIMEA:
		err = rc.SetTargetSMIMEA(v.Usage, v.Selector, v.MatchingType, v.Certificate)
	case *dnsv1.SOA:
		err = rc.SetTargetSOA(v.Ns, v.Mbox, v.Serial, v.Refresh, v.Retry, v.Expire, v.Minttl)
	case *dnsv1.SRV:
		err = rc.SetTargetSRV(v.Priority, v.Weight, v.Port, v.Target)
	case *dnsv1.SSHFP:
		err = rc.SetTargetSSHFP(v.Algorithm, v.Type, v.FingerPrint)
	case *dnsv1.SVCB:
		err = rc.SetTargetSVCB(v.Priority, v.Target, v.Value)
	case *dnsv1.TLSA:
		err = rc.SetTargetTLSA(v.Usage, v.Selector, v.MatchingType, v.Certificate)
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

func RRtoRCV2(rr dnsv2.RR, origin string) (models.RecordConfig, error) {
	// Convert's dns.RR into DNSControl's models.RecordConfig struct.

	header := rr.Header()
	ttl := header.TTL
	typeName := dnsv2.TypeToString[dnsv2.RRToType(rr)]
	typeNum := dnsv2.RRToType(rr)

	// if rtypeinfo.IsModernType(ty) {
	// 	switch v := rr.(type) {
	// 	default:
	// 		rec, err := rtypecontrol.NewRecordConfigFromStruct(strings.TrimSuffix(header.Name, origin), header.TTL, ty, v, domaintags.MakeDomainNameVarieties(origin))
	// 		return *rec, err
	// 	}
	// }

	rc := new(models.RecordConfig)
	rc.Type = typeName
	rc.TypeNum = typeNum
	rc.TTL = ttl
	rc.Original = rr
	rc.SetLabelFromFQDN(strings.TrimSuffix(header.Name, "."), origin)
	var err error
	switch v := rr.(type) { // #rtype_variations
	case *dnsv2.A:
		err = rc.SetTarget(v.A.String())
	case *dnsv2.AAAA:
		err = rc.SetTarget(v.AAAA.String())
	case *dnsv2.CAA:
		err = rc.SetTargetCAA(v.Flag, v.Tag, v.Value)
	case *dnsv2.CNAME:
		err = rc.SetTarget(v.Target)
	case *dnsv2.DHCID:
		err = rc.SetTarget(v.Digest)
	case *dnsv2.DNAME:
		err = rc.SetTarget(v.Target)
	case *dnsv2.DS:
		if rec, err := models.NewRecordConfigForRRtoRC(origin, rc.Name, ttl, typeNum, v.KeyTag, v.Algorithm, v.DigestType, v.Digest); err == nil {
			return *rec, nil
		}
	case *dnsv2.DNSKEY:
		err = rc.SetTargetDNSKEY(v.Flags, v.Protocol, v.Algorithm, v.PublicKey)
	case *dnsv2.HTTPS:
		//err = rc.SetTargetSVCB(v.Priority, v.Target, v.Value)
		//err = fmt.Errorf("RR type %s should be handled as modern type", ty)
		rc.RDATA = rr.Data()
	case *dnsv2.LOC:
		err = rc.SetTargetLOC(v.Version, v.Latitude, v.Longitude, v.Altitude, v.Size, v.HorizPre, v.VertPre)
	case *dnsv2.MX:
		err = rc.SetTargetMX(v.Preference, v.Mx)
	case *dnsv2.NAPTR:
		err = rc.SetTargetNAPTR(v.Order, v.Preference, v.Flags, v.Service, v.Regexp, v.Replacement)
	case *dnsv2.OPENPGPKEY:
		err = rc.SetTarget(v.PublicKey)
	case *dnsv2.NS:
		err = rc.SetTarget(v.Ns)
	case *dnsv2.PTR:
		err = rc.SetTarget(v.Ptr)
	case *dnsv2.RP:
		if rec, err := models.NewRecordConfigForRRtoRC(origin, header.Name, ttl, typeNum, v.Mbox, v.Txt); err == nil {
			return *rec, nil
		}
	case *dnsv2.SMIMEA:
		err = rc.SetTargetSMIMEA(v.Usage, v.Selector, v.MatchingType, v.Certificate)
	case *dnsv2.SOA:
		err = rc.SetTargetSOA(v.Ns, v.Mbox, v.Serial, v.Refresh, v.Retry, v.Expire, v.Minttl)
	case *dnsv2.SRV:
		err = rc.SetTargetSRV(v.Priority, v.Weight, v.Port, v.Target)
	case *dnsv2.SSHFP:
		err = rc.SetTargetSSHFP(v.Algorithm, v.Type, v.FingerPrint)
	case *dnsv2.SVCB:
		//err = rc.SetTargetSVCB(v.Priority, v.Target, v.Value)
		//err = fmt.Errorf("RR type %s should be handled as modern type", ty)
		rc.RDATA = rr.Data()
	case *dnsv2.TLSA:
		err = rc.SetTargetTLSA(v.Usage, v.Selector, v.MatchingType, v.Certificate)
	case *dnsv2.TXT:
		if true {
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
