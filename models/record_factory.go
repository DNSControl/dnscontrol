package models

import (
	"fmt"
	"log"
	"strings"

	dnsv2 "codeberg.org/miekg/dns"
	dnsutilv2 "codeberg.org/miekg/dns/dnsutil"
	"github.com/DNSControl/dnscontrol/v4/pkg/mustbe"
	"github.com/DNSControl/dnscontrol/v4/pkg/privatetypes"
	"golang.org/x/net/idna"
)

// NewRecordConfig constructs a models.NewRecord().
//
// It may seem odd that this is a method of DomainConfig but it makes sense if
// you consider that a RecordConfig lives in the context of its DomainConfig.
// For example, the need to shorten a FQDN requires knowing the domain's name,
// which is stored in a DomainConfig. If you need to create a RecordConfig
// outside of a DomainConfig, consider using models.MakeTestRC() or
// models.MakeTestRCParse() (both in record_helpers_test.go).
func (dc *DomainConfig) NewRecordConfig(name string, ttl uint32, typeAny any, args ...any) (*RecordConfig, error) {
	mustbe.ValidArgs(args)
	typeNum, err := anyToTypeNum(typeAny)
	if err != nil {
		return nil, err
	}

	f, ok := privatetypes.TypeToMakeRDATA[typeNum]
	if !ok {
		fmt.Printf("NewRecordConfig: failed TypeToMakeRDATA[%d] == nil", typeNum)
		return nil, fmt.Errorf("NewRecordConfig: failed TypeToMakeRDATA[%d] == nil", typeNum)
	}
	rd, err := f(dc.Name, nil, args...)
	if err != nil {
		log.Printf("NewRecordConfig: Failed to create RDATA for type %d: %+v\n", typeNum, err)
		log.Fatalf("NewRecordConfig: Failed to create RDATA for type %d: %+v", typeNum, err)
	}
	//fmt.Printf("DEBUG rd=%T\n", rd)

	return newRecordConfigHelper(dc.Name, name, ttl, typeNum, rd, nil)
}

// NewRecordConfigParse is like NewRecordConfig but the fields of the record
// come from parsing data which is assumed to be in RFC1038 Zonefile format.
func (dc *DomainConfig) NewRecordConfigParse(name string, ttl uint32, typeAny any, data string) (*RecordConfig, error) {
	typeNum, err := anyToTypeNum(typeAny)
	if err != nil {
		return nil, err
	}
	rd, err := MyNewData(typeNum, data, dc.Name)
	if err != nil {
		return nil, err
	}
	return newRecordConfigHelper(dc.Name, name, ttl, typeNum, rd, nil)
}

// NewRecordConfigFromDnsconfigjs is only for use by dnsrr.go.
//
// subdomain is the D_EXTEND() subdomain this record was declared under ("" if
// none). Relative targets (CNAME/MX/NS/SRV/ALIAS) are canonicalized relative to
// subdomain.zone, while the label is already relative to the zone, so the label
// machinery still uses dc.Name.
func (dc *DomainConfig) NewRecordConfigFromDnsconfigjs(name string, ttl uint32, typeNum uint16, args []any, metadata map[string]string, subdomain string) (*RecordConfig, error) {

	targetOrigin := dc.Name
	if subdomain != "" {
		targetOrigin = subdomain + "." + dc.Name
	}
	rd, err := privatetypes.TypeToMakeRDATA[typeNum](targetOrigin, metadata, args...)
	if err != nil {
		fmt.Printf("NewRecordConfigFromDnsconfigjs: Failed to create RDATA for type %s: %v\n", dnsutilv2.TypeToString(typeNum), err)
		log.Fatalf("NewRecordConfigFromDnsconfigjs: Failed to create RDATA for type %s: %v", dnsutilv2.TypeToString(typeNum), err)
	}
	return newRecordConfigHelper(dc.Name, name, ttl, typeNum, rd, metadata)
}

// NewRecordConfigForRRtoRC is only for use by dnsrr.go.
func NewRecordConfigForRRtoRC(origin, name string, ttl uint32, typeNum uint16, args ...any) (*RecordConfig, error) {
	mustbe.ValidArgs(args)

	rd, err := privatetypes.TypeToMakeRDATA[typeNum](origin, nil, args...)
	if err != nil {
		log.Fatalf("NewRecordConfigForRRtoRC: Failed to create RDATA for type %s: %v", dnsutilv2.TypeToString(typeNum), err)
	}
	return newRecordConfigHelper(origin, name, ttl, typeNum, rd, nil)
}

// // newRecordConfigHelper creates a RecordConfig using a dnsv2.RDATA.
// // This is risky because it assumes the caller has done a lot of the prep work
// // that is automatic with NewRecordConfig and NewRecordConfigParse.  In
// // partiular, any hostnames must be converted to ASCII (IDN PunyCode) and must
// // be FQDNs (usually with a "." at the end, but not for all record types!) and
// // not shortnames.
//
// // We're commenting this out until someone actually needs this functionality, most likely AXFRDDNS.
// // (Note to self: Maybe it should take an dnsv2.RR so that it can validate the label, ttl, etc?)
//
//	func (dc *DomainConfig) NewRecordConfigRDATA(name string, ttl uint32, typeNum uint16, rd dnsv2.RDATA) (*RecordConfig, error) {
//		return newRecordConfigHelper(origin, ttl, typeNum, rd)
//	}

// newRecordConfigHelper is a helper.  if rd != nil, args is ignored.
// All valid RecordConfig structs come through this function. Everything else is questionable.
func newRecordConfigHelper(origin, name string, ttl uint32, typeNum uint16, rd dnsv2.RDATA, metadata map[string]string) (*RecordConfig, error) {
	rc := &RecordConfig{
		TypeNum:  typeNum,
		Type:     dnsutilv2.TypeToString(typeNum),
		TTL:      ttl,
		Metadata: metadata,
	}
	rc.SetRDATA(rd)

	rc.Name = name
	rc.NameUnicode = makeLabelNameUnicode(name)
	rc.NameFQDN = makeLabelNameFQDN(origin, name)
	rc.NameFQDNUnicode = makeNameFQDNUnicode(rc.NameFQDN)

	rc.FixUp(origin)    // Add .ComparableV3
	err := backfill(rc) // Fill in the legacy rc.${TYPE}{Field} fields.
	if err != nil {
		return nil, err
	}

	return rc, nil
}

func newRecordConfigHelperRC(rc *RecordConfig, typeName string, contents string, origin string) error {
	typeNum, err := dnsutilv2.StringToType(typeName)
	if err != nil {
		return err
	}
	rc.TypeNum = typeNum
	rc.Type = typeName

	rd, err := MyNewData(typeNum, contents, origin)
	if err != nil {
		return err
	}
	rc.SetRDATA(rd)

	rc.FixUp(origin) // Add .RDATA and .ComparableV3
	err = backfill(rc)
	if err != nil {
		return err
	}
	return nil
}

func anyToTypeNum(a any) (uint16, error) {
	switch v := a.(type) {
	case uint16:
		return v, nil
	case int:
		return uint16(v), nil
	case string:
		typeNum, err := dnsutilv2.StringToType(v)
		if err == nil {
			return typeNum, nil
		} else {
			return 0, fmt.Errorf("anyToTypeNum(%q) failed: %w", v, err)
		}
	}
	return 0, fmt.Errorf("anyToTypeNum called with unknown type: %T", a)
}

func makeLabelNameFQDN(origin, name string) string {
	if name == "@" {
		return origin
	}
	if strings.HasSuffix(name, ".") { // only needed by TestWriteZoneFileEach() and may be removed when that's gone.
		return name[:len(name)-1]
	}
	return name + "." + origin
}

func makeLabelNameUnicode(name string) string {
	nameUnicode, err := idna.ToUnicode(name)
	if err != nil {
		panic(err) // should not happen
	}
	return nameUnicode
}

func makeNameFQDNUnicode(nameFQDN string) string {
	// TODO(tlim): If this is too slow, we could join name + originFQDN
	nameUnicode, err := idna.ToUnicode(nameFQDN)
	if err != nil {
		panic(err) // should not happen
	}
	return nameUnicode
}
