package mittwald

import (
	"fmt"
	"strings"

	dnsv2 "codeberg.org/miekg/dns"
	"github.com/DNSControl/dnscontrol/v5/models"
	"github.com/mittwald/api-client-go/mittwaldv2/generated/clients/domainclientv2"
	mwdns "github.com/mittwald/api-client-go/mittwaldv2/generated/schemas/dnsv2"
)

// autoTTL is what mStudio's nameservers serve for a record set whose TTL is "auto".
const autoTTL = 60

// Record set names of the API, one per slot of a zone.
const (
	setA     = domainclientv2.UpdateRecordSetRequestPathRecordSetA
	setCAA   = domainclientv2.UpdateRecordSetRequestPathRecordSetCaa
	setCNAME = domainclientv2.UpdateRecordSetRequestPathRecordSetCname
	setMX    = domainclientv2.UpdateRecordSetRequestPathRecordSetMx
	setSRV   = domainclientv2.UpdateRecordSetRequestPathRecordSetSrv
	setTXT   = domainclientv2.UpdateRecordSetRequestPathRecordSetTxt
)

// slotOf returns the record set that holds records of a type. A and AAAA
// share one set.
func slotOf(typeNum uint16) (domainclientv2.UpdateRecordSetRequestPathRecordSet, error) {
	switch typeNum {
	case dnsv2.TypeA, dnsv2.TypeAAAA:
		return setA, nil
	case dnsv2.TypeCAA:
		return setCAA, nil
	case dnsv2.TypeCNAME:
		return setCNAME, nil
	case dnsv2.TypeMX:
		return setMX, nil
	case dnsv2.TypeSRV:
		return setSRV, nil
	case dnsv2.TypeTXT:
		return setTXT, nil
	}
	return "", fmt.Errorf("record type %s is not supported", dnsv2.TypeToString[typeNum])
}

func ttlOf(s mwdns.RecordSettings) uint32 {
	if s.Ttl != nil && s.Ttl.AlternativeTtlSeconds != nil {
		return uint32(s.Ttl.AlternativeTtlSeconds.Seconds)
	}
	return autoTTL
}

func settingsFor(ttl uint32) mwdns.RecordSettings {
	return mwdns.RecordSettings{Ttl: &mwdns.RecordSettingsTtl{
		AlternativeTtlSeconds: &mwdns.TtlSeconds{Seconds: int64(ttl)},
	}}
}

// fqdn returns a target the way the API reads it: without the trailing dot.
func fqdn(target string) string {
	return strings.TrimSuffix(target, ".")
}

/*
The generated client decodes a oneOf into every alternative that accepts the
JSON, and an unset record set ({}) is accepted by several of them: it comes
back as RecordUnset and, for example, as a CNAME with an empty target. A set
is therefore only read when it is not unset.
*/

// hasManagedSet reports whether mStudio manages a record set of the zone
// (addresses of an ingress, mail exchangers of mittwald's mail service).
// Those sets are not returned as records and are never touched.
func hasManagedSet(z mwdns.Zone) bool {
	a, mx := z.RecordSet.CombinedARecords, z.RecordSet.Mx
	return (a.AlternativeRecordUnset == nil && a.AlternativeCombinedAManaged != nil) ||
		(mx.AlternativeRecordUnset == nil && mx.AlternativeRecordMXManaged != nil && mx.AlternativeRecordMXManaged.Managed)
}

// toRC converts the custom record sets of a zone into records.
func toRC(dc *models.DomainConfig, z mwdns.Zone) (models.Records, error) {
	label := dc.LabelFromFQDNNoDot(z.Domain)
	var recs models.Records
	add := func(ttl uint32, typeNum uint16, args ...any) error {
		rc, err := dc.NewRecordConfig(label, ttl, typeNum, args...)
		if err != nil {
			return fmt.Errorf("zone %s: %w", z.Domain, err)
		}
		recs = append(recs, rc)
		return nil
	}

	rs := z.RecordSet
	if a := rs.CombinedARecords.AlternativeCombinedACustom; a != nil && rs.CombinedARecords.AlternativeRecordUnset == nil {
		ttl := ttlOf(a.Settings)
		for _, ip := range a.A {
			if err := add(ttl, dnsv2.TypeA, string(ip)); err != nil {
				return nil, err
			}
		}
		for _, ip := range a.Aaaa {
			if err := add(ttl, dnsv2.TypeAAAA, string(ip)); err != nil {
				return nil, err
			}
		}
	}
	if c := rs.Cname.AlternativeRecordCNAMEComponent; c != nil && rs.Cname.AlternativeRecordUnset == nil {
		if err := add(ttlOf(c.Settings), dnsv2.TypeCNAME, c.Fqdn+"."); err != nil {
			return nil, err
		}
	}
	if m := rs.Mx.AlternativeRecordMXCustom; m != nil && rs.Mx.AlternativeRecordUnset == nil {
		for _, r := range m.Records {
			if err := add(ttlOf(m.Settings), dnsv2.TypeMX, uint16(r.Priority), r.Fqdn+"."); err != nil {
				return nil, err
			}
		}
	}
	if t := rs.Txt.AlternativeRecordTXTComponent; t != nil && rs.Txt.AlternativeRecordUnset == nil {
		for _, e := range t.Entries {
			if err := add(ttlOf(t.Settings), dnsv2.TypeTXT, e); err != nil {
				return nil, err
			}
		}
	}
	if s := rs.Srv.AlternativeRecordSRVComponent; s != nil && rs.Srv.AlternativeRecordUnset == nil {
		for _, r := range s.Records {
			var priority, weight uint16
			if r.Priority != nil {
				priority = uint16(*r.Priority)
			}
			if r.Weight != nil {
				weight = uint16(*r.Weight)
			}
			if err := add(ttlOf(s.Settings), dnsv2.TypeSRV, priority, weight, uint16(r.Port), r.Fqdn+"."); err != nil {
				return nil, err
			}
		}
	}
	if c := rs.Caa.AlternativeRecordCAAComponent; c != nil && rs.Caa.AlternativeRecordUnset == nil {
		for _, r := range c.Records {
			if err := add(ttlOf(c.Settings), dnsv2.TypeCAA, uint8(r.Flags), string(r.Tag), r.Value); err != nil {
				return nil, err
			}
		}
	}
	return recs, nil
}

// toNative groups the records of one name into the record sets the API
// expects. All records of a set must have the same TTL.
func toNative(recs models.Records) (map[domainclientv2.UpdateRecordSetRequestPathRecordSet]domainclientv2.UpdateRecordSetRequestBody, error) {
	ttls := map[domainclientv2.UpdateRecordSetRequestPathRecordSet]uint32{}
	var (
		a     *mwdns.CombinedACustom
		cname *mwdns.RecordCNAMEComponent
		mx    *mwdns.RecordMXCustom
		txt   *mwdns.RecordTXTComponent
		srv   *mwdns.RecordSRVComponent
		caa   *mwdns.RecordCAAComponent
	)
	for _, rc := range recs {
		set, err := slotOf(rc.TypeNum)
		if err != nil {
			return nil, err
		}
		if ttl, ok := ttls[set]; ok && ttl != rc.TTL {
			return nil, fmt.Errorf("%s: mStudio keeps one TTL per record set (%s); found %d and %d", rc.GetLabelFQDN(), set, ttl, rc.TTL)
		}
		ttls[set] = rc.TTL
		settings := settingsFor(rc.TTL)

		switch rc.TypeNum {
		case dnsv2.TypeA, dnsv2.TypeAAAA:
			if a == nil {
				a = &mwdns.CombinedACustom{A: []mwdns.CombinedAManagedARecord{}, Aaaa: []mwdns.CombinedAManagedAAAARecord{}, Settings: settings}
			}
			if rc.TypeNum == dnsv2.TypeA {
				a.A = append(a.A, mwdns.CombinedAManagedARecord(rc.AsA().Addr.String()))
			} else {
				a.Aaaa = append(a.Aaaa, mwdns.CombinedAManagedAAAARecord(rc.AsAAAA().Addr.String()))
			}
		case dnsv2.TypeCNAME:
			cname = &mwdns.RecordCNAMEComponent{Fqdn: fqdn(rc.AsCNAME().Target), Settings: settings}
		case dnsv2.TypeMX:
			if mx == nil {
				mx = &mwdns.RecordMXCustom{Settings: settings}
			}
			f := rc.AsMX()
			mx.Records = append(mx.Records, mwdns.RecordMXRecord{Priority: int64(f.Preference), Fqdn: fqdn(f.Mx)})
		case dnsv2.TypeTXT:
			if txt == nil {
				txt = &mwdns.RecordTXTComponent{Settings: settings}
			}
			txt.Entries = append(txt.Entries, rc.GetTargetTXTJoined())
		case dnsv2.TypeSRV:
			if srv == nil {
				srv = &mwdns.RecordSRVComponent{Settings: settings}
			}
			f := rc.AsSRV()
			priority, weight := int64(f.Priority), int64(f.Weight)
			srv.Records = append(srv.Records, mwdns.RecordSRVRecord{
				Priority: &priority, Weight: &weight, Port: int64(f.Port), Fqdn: fqdn(f.Target),
			})
		case dnsv2.TypeCAA:
			if caa == nil {
				caa = &mwdns.RecordCAAComponent{Settings: settings}
			}
			f := rc.AsCAA()
			caa.Records = append(caa.Records, mwdns.RecordCAARecord{
				Flags: int64(f.Flag), Tag: mwdns.RecordCAARecordTag(f.Tag), Value: f.Value,
			})
		}
	}

	bodies := map[domainclientv2.UpdateRecordSetRequestPathRecordSet]domainclientv2.UpdateRecordSetRequestBody{}
	if a != nil {
		bodies[setA] = domainclientv2.UpdateRecordSetRequestBody{AlternativeCombinedACustom: a}
	}
	if cname != nil {
		bodies[setCNAME] = domainclientv2.UpdateRecordSetRequestBody{AlternativeRecordCNAMEComponent: cname}
	}
	if mx != nil {
		bodies[setMX] = domainclientv2.UpdateRecordSetRequestBody{AlternativeRecordMXCustom: mx}
	}
	if txt != nil {
		bodies[setTXT] = domainclientv2.UpdateRecordSetRequestBody{AlternativeRecordTXTComponent: txt}
	}
	if srv != nil {
		bodies[setSRV] = domainclientv2.UpdateRecordSetRequestBody{AlternativeRecordSRVComponent: srv}
	}
	if caa != nil {
		bodies[setCAA] = domainclientv2.UpdateRecordSetRequestBody{AlternativeRecordCAAComponent: caa}
	}
	return bodies, nil
}
