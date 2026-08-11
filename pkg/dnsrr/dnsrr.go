package dnsrr

import (
	"fmt"
	"strings"

	"github.com/DNSControl/dnscontrol/v4/models"
	"github.com/DNSControl/dnscontrol/v4/pkg/domaintags"
	"github.com/DNSControl/dnscontrol/v4/pkg/rtypecontrol"
	"github.com/DNSControl/dnscontrol/v4/pkg/rtypeinfo"
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
	ty := dnsv1.TypeToString[header.Rrtype]

	if rtypeinfo.IsModernType(ty) {
		switch v := rr.(type) {
		default:
			rec, err := rtypecontrol.NewRecordConfigFromStruct(strings.TrimSuffix(header.Name, origin), header.Ttl, dnsv1.TypeToString[header.Rrtype], v, domaintags.MakeDomainNameVarieties(origin))
			return *rec, err
		}
	}

	rc := new(models.RecordConfig)
	rc.Type = dnsv1.TypeToString[header.Rrtype]
	rc.TTL = header.Ttl
	rc.Original = rr
	rc.SetLabelFromFQDN(strings.TrimSuffix(header.Name, "."), origin)
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
		panic("DS should be handled as modern type")
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
		panic("RP should be handled as modern type")
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
