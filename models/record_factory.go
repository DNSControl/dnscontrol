package models

import (
	"fmt"
	"log"
	"slices"
	"strings"

	dnsv2 "codeberg.org/miekg/dns"
	dnsutilv2 "codeberg.org/miekg/dns/dnsutil"
	"github.com/DNSControl/dnscontrol/v5/pkg/mustbe"
	nrc "github.com/DNSControl/dnscontrol/v5/pkg/nrc"
	"github.com/DNSControl/dnscontrol/v5/pkg/privatetypes"
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
// Behavior can be modified by sending an optional nrc.Flag struct as the last arg.
func (dc *DomainConfig) NewRecordConfig(name string, ttl uint32, typeAny any, args ...any) (*RecordConfig, error) {
	mustbe.ValidArgs(args)
	typeNum, err := anyToTypeNum(typeAny)
	if err != nil {
		return nil, err
	}

	// if last arg is type nrc.Flags, assign it to isEnabled then remove it from the args array.
	var isEnabled nrc.Flags
	if len(args) > 0 {
		if f, ok := args[len(args)-1].(nrc.Flags); ok {
			isEnabled = f
			args = slices.Delete(args, len(args)-1, len(args))
		}
	}

	// SrvWeirdSplit
	if isEnabled.SrvWeirdSplit && len(args) == 2 {
		isEnabled.SrvWeirdSplit = false
		return dc.NewRecordConfigParse(name, ttl, typeNum, fmt.Sprintf("%d %s", args[0], args[1].(string)), isEnabled)
	}
	// TargetIsFqdnNoDot
	// Passed to downstream functions.

	// TxtDontParse
	if isEnabled.TxtDontParse {
		panic("NewRecordConfig() incompatible with TxtDontParse")
	}

	f, ok := privatetypes.TypeToMakeRDATA[typeNum]
	if !ok {
		return nil, fmt.Errorf("NewRecordConfig: failed TypeToMakeRDATA[%d] == nil", typeNum)
	}
	rd, err := f(dc.Name, nil, isEnabled, args...)
	if err != nil {
		return nil, fmt.Errorf("NewRecordConfig: Failed to create RDATA for type %d: %w", typeNum, err)
	}

	return newRecordConfigHelper(dc.Name, name, ttl, typeNum, rd, nil)
}

// NewRecordConfigParse is like NewRecordConfig but the fields of the record
// come from parsing data which is assumed to be in RFC1038 Zonefile format.
// Behavior can be modified by sending an optional rfc.Flag struct.
func (dc *DomainConfig) NewRecordConfigParse(name string, ttl uint32, typeAny any, data string, rcflag ...nrc.Flags) (*RecordConfig, error) {
	typeNum, err := anyToTypeNum(typeAny)
	if err != nil {
		return nil, err
	}

	var isEnabled nrc.Flags
	switch len(rcflag) {
	case 0:
	case 1:
		isEnabled = rcflag[0]
	default:
		panic(fmt.Sprintf("NewRecordConfigParse() called with multiple flags: %v", rcflag))
	}

	// SrvWeirdSplit
	if isEnabled.SrvWeirdSplit {
		panic("NewRecordConfigParse() incompatible with SrvWeirdSplit")
	}

	// TargetIsFqdnNoDot
	origin := dc.Name
	if isEnabled.TargetIsFqdnNoDot {
		origin = ""
	}

	// TxtDontParse
	if isEnabled.TxtDontParse && typeNum == dnsv2.TypeTXT {
		isEnabled.TxtDontParse = false
		return dc.NewRecordConfig(name, ttl, typeNum, data, isEnabled)
	}

	rd, err := MyNewData(typeNum, data, origin)
	if err != nil {
		return nil, err
	}
	return newRecordConfigHelper(dc.Name, name, ttl, typeNum, rd, nil)
}

// NewRecordConfigForRRv2toRC is like NewRecordConfig but takes an RDATA. It
// should only be used by RRv2toRC. It is not intended for general use.
func (dc *DomainConfig) NewRecordConfigForRRv2toRC(name string, ttl uint32, typeNum uint16, rd dnsv2.RDATA) (*RecordConfig, error) {
	return newRecordConfigHelper(dc.Name, name, ttl, typeNum, rd, nil)
}

// NewRecordConfigForRRtoRC is only for use by dnsrr.go. Do not use this. The signature may change at any time.
func NewRecordConfigForRRtoRC(origin, name string, ttl uint32, typeNum uint16, args ...any) (*RecordConfig, error) {
	mustbe.ValidArgs(args)

	// Make sure label is a shortname.
	name = strings.ToLower(name)
	if before, found := strings.CutSuffix(name, "."+origin+"."); found {
		name = before
	}
	if name == origin+"." {
		name = "@"
	}

	isEnabled := nrc.Flags{}

	rd, err := privatetypes.TypeToMakeRDATA[typeNum](origin, nil, isEnabled, args...)
	if err != nil {
		log.Fatalf("NewRecordConfigForRRtoRC: Failed to create RDATA for type %s: %v", dnsutilv2.TypeToString(typeNum), err)
	}
	return newRecordConfigHelper(origin, name, ttl, typeNum, rd, nil)
}

// newRecordConfigFromDnsconfigjs is only for use by models.ImportRawRecords().
//
// This is similar to NewRecordConfig plus:
// * Processes name with the "dot rules" of dnsconfig.js.
// * Processes metadata.
// * Handles the D_EXTEND() "subdomain" concept on both the name and any targets.
//
// subdomain is the D_EXTEND() subdomain this record was declared under ("" if
// none). Relative targets (CNAME/MX/NS/SRV/ALIAS) are canonicalized relative to
// subdomain.zone, while the label is already relative to the zone, so the label
// machinery still uses dc.Name.
func (dc *DomainConfig) newRecordConfigFromDnsconfigjs(name string, ttl uint32, typeNum uint16, args []any, metadata map[string]string, subdomain string) (*RecordConfig, error) {

	targetOrigin := dc.Name
	if subdomain != "" {
		targetOrigin = subdomain + "." + dc.Name
	}
	rd, err := privatetypes.TypeToMakeRDATA[typeNum](targetOrigin, metadata, nrc.Flags{}, args...)
	if err != nil {
		fmt.Printf("NewRecordConfigFromDnsconfigjs: Failed to create RDATA for type %s: %v\n", dnsutilv2.TypeToString(typeNum), err)
		log.Fatalf("NewRecordConfigFromDnsconfigjs: Failed to create RDATA for type %s: %v", dnsutilv2.TypeToString(typeNum), err)
	}
	return newRecordConfigHelper(dc.Name, name, ttl, typeNum, rd, metadata)
}

// newRecordConfigHelper is a helper.  if rd != nil, args is ignored.
// All valid RecordConfig structs come through this function. Everything else is questionable.
func newRecordConfigHelper(origin, name string, ttl uint32, typeNum uint16, rd dnsv2.RDATA, metadata map[string]string) (*RecordConfig, error) {
	rc := &RecordConfig{
		Type:        dnsutilv2.TypeToString(typeNum),
		TypeNum:     typeNum,
		TTL:         ttl,
		Name:        name,
		NameUnicode: makeLabelNameUnicode(name),
		NameFQDN:    makeLabelNameFQDN(origin, name),
		Metadata:    metadata,
	}
	if rc.Metadata == nil {
		rc.Metadata = map[string]string{}
	}
	rc.NameFQDNUnicode = makeNameFQDNUnicode(rc.NameFQDN)
	rc.SetRDATA(rd)
	return rc, nil
}

func legacySetTargetArgsTXT(rc *RecordConfig, args ...any) error {

	rc.TypeNum = dnsv2.TypeTXT
	rc.Type = "TXT"

	if rc.Metadata == nil {
		rc.Metadata = map[string]string{}
	}

	rd, err := MakeTXT("", nil, nrc.Flags{}, args...)
	if err != nil {
		log.Fatalf("legacySetTargetArgs: Failed to create RDATA for type %s: %+v", rc.Type, err)
	}
	rc.SetRDATA(rd)

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
