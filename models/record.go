package models

import (
	"encoding/json"
	"fmt"
	"log"
	"strings"

	dnsv2 "codeberg.org/miekg/dns"
	dnsutilv2 "codeberg.org/miekg/dns/dnsutil"
	dnsrdatav2 "codeberg.org/miekg/dns/rdata"
	"github.com/DNSControl/dnscontrol/v4/pkg/mustbe"
	"github.com/DNSControl/dnscontrol/v4/pkg/privatetypes"
	privatetypesrdata "github.com/DNSControl/dnscontrol/v4/pkg/privatetypes/rdata"
	"github.com/DNSControl/dnscontrol/v4/pkg/txtutil"
	"github.com/jinzhu/copier"
	dnsv1 "github.com/miekg/dns"
	dnsutilv1 "github.com/miekg/dns/dnsutil"
	"github.com/qdm12/reprint"
	"golang.org/x/net/idna"
)

// RecordConfig stores a DNS record whether it was created from data downloaded from
// a provider's API ("actual") or from user input in dndsconfig.js ("desired").
type RecordConfig struct {

	// Type is the DNS record type (rtype), all caps, "A", "MX", etc.
	Type string `json:"type"`

	// TypeNum is the assigned number of the record's type. 1 for A, 5 for CNAME, etc. See dnsv2.TypeToString and dnsv2.StringToType.
	// NB(tlim): Not currently used. Placeholder for future feature.
	TypeNum uint16 `json:"typenum,omitempty"`

	// RDATA is (the fields of the record).
	// NB(tlim): Not currently used. Placeholder for future feature.
	RDATA dnsv2.RDATA `json:"rdata,omitempty"`

	// ComparableV3 is an opaque string that can be used to compare two
	// RecordConfigs for equality. Typically this is the Zonefile line
	// minus the label and TTL.
	// The V3 distingues itself from .Comparable, which it will eventually replace.
	// NB(tlim): Not currently used. Placeholder for future feature.
	ComparableV3 string `json:"comparablev3,omitempty"`

	// TTL is the DNS record's TTL in seconds. 0 means provider default.
	TTL uint32 `json:"ttl,omitempty"`

	// Name is the shortname i.e. the FQDN without the parent domains's suffix.
	// It should never be "".  Record at the apex (naked domain) are represented by "@".
	NameRaw     string `json:"name_raw,omitempty"`     // .Name as the user entered it in dnsconfig.js
	Name        string `json:"name"`                   // The short name, PunyCode. See above.
	NameUnicode string `json:"name_unicode,omitempty"` // .Name as Unicode (downcased, then convertedot Unicode).

	// This is the FQDN version of .Name. It should never have a trailing ".".
	NameFQDNRaw     string `json:"-"` // .NameFQDN as the user entered it in dnsconfig.js (downcased).
	NameFQDN        string `json:"-"` // Must end with ".$origin".
	NameFQDNUnicode string `json:"-"` // .NameFQDN as Unicode (downcased, then convertedot Unicode).

	// F is the binary representation of the record's data usually a dns.XYZ struct.
	// Always stored in Punycode, not Unicode. Downcased where applicable.
	F any `json:"fields,omitempty"`
	//FieldsAsRaw     []string // Fields as received from the dnsconfig.js file, converted to strings.
	//FieldsAsUnicode []string // fields with IDN fields converted to Unicode for display purposes.

	// Comparable is an opaque string that can be used to compare two
	// RecordConfigs for equality. Typically this is the Zonefile line minus the
	// label and TTL.
	//xComparable string `json:"comparable,omitempty"` // Cache of ToComparableNoTTL()

	// ZonefilePartial is the partial zonefile line for this record, excluding
	// the label and TTL.  If this is not an official RR type, we invent the format.
	ZonefilePartial string `json:"zonfefilepartial,omitempty"`

	//// Fields only relevant when RecordConfig was created from data in dnsconfig.js:

	// Metadata (desired) added to the record via dnsconfig.js. For example: A("foo", "1.2.3.4", {metakey: "value"})
	Metadata map[string]string `json:"meta,omitempty"`

	// FilePos (desired) is "filename:line:char" of the record in dnsconfig.js (desired).
	FilePos string `json:"filepos"`

	// Subdomain (if non-empty) contains the subdomain path for this record.
	// When .Name* fields are updated to include the subdomain, this field is
	// cleared.
	SubDomain string `json:"subdomain,omitempty"`

	//// Fields only relevant when RecordConfig was created from data downloaded from a provider:

	// Original is a pointer to the provider-specific record object. When
	// getting the records via the API, we store the original object here.
	// Later if we need to pull out an ID or other provider-specific field, we
	// can.  Typically deleting or updating a record requires knowing its ID.
	Original any `json:"-"`

	//// Legacy fields we hope to remove someday

	// If you add a field to this struct, also add it to the list in the UnmarshalJSON function.
	target             string            // If a name, must end with "."
	MxPreference       uint16            `json:"mxpreference,omitempty"`
	SrvPriority        uint16            `json:"srvpriority,omitempty"`
	SrvWeight          uint16            `json:"srvweight,omitempty"`
	SrvPort            uint16            `json:"srvport,omitempty"`
	CaaTag             string            `json:"caatag,omitempty"`
	CaaFlag            uint8             `json:"caaflag,omitempty"`
	DsKeyTag           uint16            `json:"dskeytag,omitempty"`
	DsAlgorithm        uint8             `json:"dsalgorithm,omitempty"`
	DsDigestType       uint8             `json:"dsdigesttype,omitempty"`
	DsDigest           string            `json:"dsdigest,omitempty"`
	DnskeyFlags        uint16            `json:"dnskeyflags,omitempty"`
	DnskeyProtocol     uint8             `json:"dnskeyprotocol,omitempty"`
	DnskeyAlgorithm    uint8             `json:"dnskeyalgorithm,omitempty"`
	DnskeyPublicKey    string            `json:"dnskeypublickey,omitempty"`
	LocVersion         uint8             `json:"locversion,omitempty"`
	LocSize            uint8             `json:"locsize,omitempty"`
	LocHorizPre        uint8             `json:"lochorizpre,omitempty"`
	LocVertPre         uint8             `json:"locvertpre,omitempty"`
	LocLatitude        uint32            `json:"loclatitude,omitempty"`
	LocLongitude       uint32            `json:"loclongitude,omitempty"`
	LocAltitude        uint32            `json:"localtitude,omitempty"`
	LuaRType           string            `json:"luartype,omitempty"`
	NaptrOrder         uint16            `json:"naptrorder,omitempty"`
	NaptrPreference    uint16            `json:"naptrpreference,omitempty"`
	NaptrFlags         string            `json:"naptrflags,omitempty"`
	NaptrService       string            `json:"naptrservice,omitempty"`
	NaptrRegexp        string            `json:"naptrregexp,omitempty"`
	SmimeaUsage        uint8             `json:"smimeausage,omitempty"`
	SmimeaSelector     uint8             `json:"smimeaselector,omitempty"`
	SmimeaMatchingType uint8             `json:"smimeamatchingtype,omitempty"`
	SshfpAlgorithm     uint8             `json:"sshfpalgorithm,omitempty"`
	SshfpFingerprint   uint8             `json:"sshfpfingerprint,omitempty"`
	SoaMbox            string            `json:"soambox,omitempty"`
	SoaSerial          uint32            `json:"soaserial,omitempty"`
	SoaRefresh         uint32            `json:"soarefresh,omitempty"`
	SoaRetry           uint32            `json:"soaretry,omitempty"`
	SoaExpire          uint32            `json:"soaexpire,omitempty"`
	SoaMinttl          uint32            `json:"soaminttl,omitempty"`
	SvcPriority        uint16            `json:"svcpriority,omitempty"`
	SvcParams          string            `json:"svcparams,omitempty"`
	TlsaUsage          uint8             `json:"tlsausage,omitempty"`
	TlsaSelector       uint8             `json:"tlsaselector,omitempty"`
	TlsaMatchingType   uint8             `json:"tlsamatchingtype,omitempty"`
	R53Alias           map[string]string `json:"r53_alias,omitempty"`
	AzureAlias         map[string]string `json:"azure_alias,omitempty"`
	AnswerType         string            `json:"answer_type,omitempty"`
	UnknownTypeName    string            `json:"unknown_type_name,omitempty"`
}

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

	//rd, err := privatetypes.TypeToMakeRDATA[typeNum](dc.Name, args...)
	f, ok := privatetypes.TypeToMakeRDATA[typeNum]
	if !ok {
		return nil, fmt.Errorf("NewRecordConfig: failed TypeToMakeRDATA[%d] == nil\n", typeNum)
	}
	rd, err := f(dc.Name, nil, args...)
	if err != nil {
		log.Fatalf("NewRecordConfig: Failed to create RDATA for type %d: %+v", typeNum, err)
	}

	return newRecordConfigHelper(dc.Name, name, ttl, typeNum, rd, nil)
}

// NewRecordConfigParse is like NewRecordConfig but the fields of the record come from parsing a string (data).
func (dc *DomainConfig) NewRecordConfigParse(name string, ttl uint32, typeAny any, data string) (*RecordConfig, error) {
	typeNum, err := anyToTypeNum(typeAny)
	if err != nil {
		return nil, err
	}
	rd, err := dnsv2.NewData(typeNum, data, dc.Name)
	if err != nil {
		return nil, err
	}
	return newRecordConfigHelper(dc.Name, name, ttl, typeNum, rd, nil)
}

// NewRecordConfigForRRtoRC is only for use by dnsrr.go.
func (dc *DomainConfig) NewRecordConfigFromDnsconfigjs(name string, ttl uint32, typeNum uint16, args []any, metadata map[string]string) (*RecordConfig, error) {

	rd, err := privatetypes.TypeToMakeRDATA[typeNum](dc.Name, metadata, args...)
	if err != nil {
		log.Fatalf("NewRecordConfigForRRtoRC: Failed to create RDATA for type %s: %v", dnsutilv2.TypeToString(typeNum), err)
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
		RDATA:    rd,
		Metadata: metadata,
	}
	rc.Name = name
	rc.NameUnicode = makeLabelNameUnicode(name)
	rc.NameFQDN = makeLabelNameFQDN(origin, name)
	rc.NameFQDNUnicode = makeNameFQDNUnicode(rc.NameFQDN)

	rc.FixUp(origin) // Add .ComparableV3

	// Hack to back-fill legacy fields. This will go away eventually.
	switch rd := rc.RDATA.(type) {
	case dnsrdatav2.A:
		rc.SetTargetIP(rd.Addr)
	case dnsrdatav2.CAA:
		rc.SetTargetCAA(rd.Flag, rd.Tag, rd.Value)
	case dnsrdatav2.CNAME:
		rc.SetTarget(rd.Target)
	case dnsrdatav2.DS:
		rc.SetTargetDS(rd.KeyTag, rd.Algorithm, rd.DigestType, rd.Digest)
	case dnsrdatav2.DNSKEY:
		rc.SetTargetDNSKEY(rd.Flags, rd.Protocol, rd.Algorithm, rd.PublicKey)
	case dnsrdatav2.LOC:
		rc.SetTargetLOC(rd.Version, rd.Latitude, rd.Longitude, rd.Altitude, rd.Size, rd.HorizPre, rd.VertPre)
	case dnsrdatav2.MX:
		rc.SetTargetMX(rd.Preference, rd.Mx)
	case dnsrdatav2.NAPTR:
		rc.SetTargetNAPTR(rd.Order, rd.Preference, rd.Flags, rd.Service, rd.Regexp, rd.Service)
	case dnsrdatav2.RP:
		// noop -- no legacy fields
	case dnsrdatav2.SMIMEA:
		rc.SetTargetSMIMEA(rd.Usage, rd.Selector, rd.MatchingType, rd.Certificate)
	case dnsrdatav2.SOA:
		rc.SetTargetSOA(rd.Ns, rd.Mbox, rd.Serial, rd.Refresh, rd.Retry, rd.Expire, rd.Minttl)
	case dnsrdatav2.SRV:
		rc.SetTargetSRV(rd.Priority, rd.Weight, rd.Port, rd.Target)
	case dnsrdatav2.SVCB: // There is no dnsrdatav2.HTTPS
		rc.SvcPriority = rd.Priority
		rc.SetTarget(rd.Target)
		rc.SvcParams = svcbv2ValueToString(rd.Value)
	case dnsrdatav2.SSHFP:
		rc.SetTargetSSHFP(rd.Algorithm, rd.Type, rd.FingerPrint)
	case dnsrdatav2.TLSA:
		rc.SetTargetTLSA(rd.Usage, rd.Selector, rd.MatchingType, rd.Certificate)
	default:
		switch rc.Type {
		case "CLOUDFLAREAPI_SINGLE_REDIRECT":
			// no-op
		case "PORKBUN_URLFWD":
			p := rd.(privatetypesrdata.PORKBUNURLFWD)
			if rc.Metadata == nil {
				rc.Metadata = map[string]string{}
			}
			rc.Metadata["type"] = p.TypeName
			rc.Metadata["includePath"] = p.IncludePath
			rc.Metadata["wildcard"] = p.Wildcard
		case "URL":
			u := rd.(privatetypesrdata.URL)
			rc.SetTarget(u.Location)
			if rc.Metadata == nil {
				rc.Metadata = map[string]string{}
			}
			rc.Metadata["includePath"] = fmt.Sprintf("%t", u.PorkbunIncludePath)
			rc.Metadata["wildcard"] = fmt.Sprintf("%t", u.PorkbunWildCard)
		case "URL301":
			u := rd.(privatetypesrdata.URL301)
			rc.SetTarget(u.Location)
		default:
			return nil, fmt.Errorf("assertion failed: NewRecordConfig back-fill has not implemented type %T", rd)
			// TODO:
			//case privatetypes..AzureAlias:
			//case privatetypes..LUA:
			//case privatetypes..R53Alias:
			//case privatetypes..AKAMAITLC:
		}
	}

	return rc, nil
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

// func anyToType(a any) (uint16, string, error) {
// 	switch v := a.(type) {
// 	case uint16:
// 		return v, dnsutilv2.TypeToString(v), nil
// 	case string:
// 		typeNum, err := dnsutilv2.StringToType(v)
// 		if err == nil {
// 			return typeNum, v, nil
// 		} else {
// 			return 0, "", fmt.Errorf("anyToTypeNum(%q) failed: %w", v, err)
// 		}
// 	}
// 	return 0, "", fmt.Errorf("anyToTypeNum called with unknown type: %T", a)
// }

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

// MarshalJSON marshals RecordConfig.
func (rc *RecordConfig) MarshalJSON() ([]byte, error) {
	recj := &struct {
		RecordConfig
		Target string `json:"target,omitempty"`
	}{
		RecordConfig: *rc,
		Target:       rc.GetTargetField(),
	}
	j, err := json.Marshal(*recj)
	if err != nil {
		return nil, err
	}
	return j, nil
}

// UnmarshalJSON unmarshals RecordConfig.
func (rc *RecordConfig) UnmarshalJSON(b []byte) error {
	recj := &struct {
		Target string `json:"target,omitempty"`

		Type      string            `json:"type"` // All caps rtype name.
		Name      string            `json:"name"` // The short name. See above.
		SubDomain string            `json:"subdomain,omitempty"`
		NameFQDN  string            `json:"-"` // Must end with ".$origin". See above.
		target    string            // If a name, must end with "."
		TTL       uint32            `json:"ttl,omitempty"`
		Metadata  map[string]string `json:"meta,omitempty"`
		FilePos   string            `json:"filepos"` // Where in the file this record was defined.
		Original  any               `json:"-"`       // Store pointer to provider-specific record object. Used in diffing.
		Args      []any             `json:"args,omitempty"`

		MxPreference       uint16            `json:"mxpreference,omitempty"`
		SrvPriority        uint16            `json:"srvpriority,omitempty"`
		SrvWeight          uint16            `json:"srvweight,omitempty"`
		SrvPort            uint16            `json:"srvport,omitempty"`
		CaaTag             string            `json:"caatag,omitempty"`
		CaaFlag            uint8             `json:"caaflag,omitempty"`
		DsKeyTag           uint16            `json:"dskeytag,omitempty"`
		DsAlgorithm        uint8             `json:"dsalgorithm,omitempty"`
		DsDigestType       uint8             `json:"dsdigesttype,omitempty"`
		DsDigest           string            `json:"dsdigest,omitempty"`
		DnskeyFlags        uint16            `json:"dnskeyflags,omitempty"`
		DnskeyProtocol     uint8             `json:"dnskeyprotocol,omitempty"`
		DnskeyAlgorithm    uint8             `json:"dnskeyalgorithm,omitempty"`
		DnskeyPublicKey    string            `json:"dnskeypublickey,omitempty"`
		LocVersion         uint8             `json:"locversion,omitempty"`
		LocSize            uint8             `json:"locsize,omitempty"`
		LocHorizPre        uint8             `json:"lochorizpre,omitempty"`
		LocVertPre         uint8             `json:"locvertpre,omitempty"`
		LocLatitude        uint32            `json:"loclatitude,omitempty"`
		LocLongitude       uint32            `json:"loclongitude,omitempty"`
		LocAltitude        uint32            `json:"localtitude,omitempty"`
		LuaRType           string            `json:"luartype,omitempty"`
		NaptrOrder         uint16            `json:"naptrorder,omitempty"`
		NaptrPreference    uint16            `json:"naptrpreference,omitempty"`
		NaptrFlags         string            `json:"naptrflags,omitempty"`
		NaptrService       string            `json:"naptrservice,omitempty"`
		NaptrRegexp        string            `json:"naptrregexp,omitempty"`
		SmimeaUsage        uint8             `json:"smimeausage,omitempty"`
		SmimeaSelector     uint8             `json:"smimeaselector,omitempty"`
		SmimeaMatchingType uint8             `json:"smimeamatchingtype,omitempty"`
		SshfpAlgorithm     uint8             `json:"sshfpalgorithm,omitempty"`
		SshfpFingerprint   uint8             `json:"sshfpfingerprint,omitempty"`
		SoaMbox            string            `json:"soambox,omitempty"`
		SoaSerial          uint32            `json:"soaserial,omitempty"`
		SoaRefresh         uint32            `json:"soarefresh,omitempty"`
		SoaRetry           uint32            `json:"soaretry,omitempty"`
		SoaExpire          uint32            `json:"soaexpire,omitempty"`
		SoaMinttl          uint32            `json:"soaminttl,omitempty"`
		SvcPriority        uint16            `json:"svcpriority,omitempty"`
		SvcParams          string            `json:"svcparams,omitempty"`
		TlsaUsage          uint8             `json:"tlsausage,omitempty"`
		TlsaSelector       uint8             `json:"tlsaselector,omitempty"`
		TlsaMatchingType   uint8             `json:"tlsamatchingtype,omitempty"`
		R53Alias           map[string]string `json:"r53_alias,omitempty"`
		AzureAlias         map[string]string `json:"azure_alias,omitempty"`
		AnswerType         string            `json:"answer_type,omitempty"`
		UnknownTypeName    string            `json:"unknown_type_name,omitempty"`

		EnsureAbsent bool `json:"ensure_absent,omitempty"` // Override NO_PURGE and delete this record

		// NB(tlim): If anyone can figure out how to do this without listing all
		// the fields, please let us know!
	}{}
	if err := json.Unmarshal(b, &recj); err != nil {
		return err
	}

	recj.FilePos = FixPosition(recj.FilePos)

	// Copy the exported fields.
	if err := copier.CopyWithOption(&rc, &recj, copier.Option{IgnoreEmpty: true, DeepCopy: true}); err != nil {
		return err
	}
	// Set each unexported field.
	if err := rc.SetTarget(recj.Target); err != nil {
		return err
	}

	// Some sanity checks:
	if recj.Type != rc.Type {
		panic("DEBUG: TYPE NOT COPIED\n")
	}
	if recj.Type == "" {
		panic("DEBUG: TYPE BLANK\n")
	}
	if recj.Name != rc.Name {
		panic("DEBUG: NAME NOT COPIED\n")
	}

	return nil
}

// FixPosition takes the string representation of a position in a file that
// comes from dnsconfig.js's initial execution, and reduces it down to just the
// line/position we display to the user. The input is not well-defined, thus if
// we find something we don't expect, we just return the original input.
// TODO: Move this to rtypecontrol or a similar package.
func FixPosition(str string) string {
	if str == "" {
		return ""
	}
	str = strings.TrimSpace(str)
	str = strings.ReplaceAll(str, "\n", " ")
	str = strings.ReplaceAll(str, "<anonymous>", "line")
	str = strings.TrimPrefix(str, "at ")
	return fmt.Sprintf("[%s]", str)
}

// Copy returns a deep copy of a RecordConfig.
func (rc *RecordConfig) Copy() (*RecordConfig, error) {
	newR := &RecordConfig{}
	// Copy the exported fields.
	err := reprint.FromTo(rc, newR) // Deep copy
	// Set each unexported field.
	newR.target = rc.target
	return newR, err
}

// SetLabel sets the .Name/.NameFQDN fields given a short name and origin.
// origin must not have a trailing dot: The entire code base maintains dc.Name
// without the trailig dot. Finding a dot here means something is very wrong.
//
// short must not have a training dot: That would mean you have a FQDN, and
// shouldn't be using SetLabel().  Maybe SetLabelFromFQDN()?
func (rc *RecordConfig) SetLabel(short, origin string) {
	// Assertions that make sure the function is being used correctly:
	if strings.HasSuffix(origin, ".") {
		panic(fmt.Errorf("origin (%s) is not supposed to end with a dot", origin))
	}
	if strings.HasSuffix(short, ".") {
		if strings.HasSuffix(short, origin+".") {
			fmt.Printf("DEBUG: ******** SetLabel on FQDNdot: %q origin=%q\n", short, origin)

		}
		if short != "**current-domain**" {
			panic(fmt.Errorf("short (%s) is not supposed to end with a dot", short))
		}
	}

	// TODO(tlim): We should add more validation here or in a separate validation
	// module.  We might want to check things like (\w+\.)+

	short = strings.ToLower(short)
	origin = strings.ToLower(origin)
	if short == "" || short == "@" {
		rc.Name = "@"
		rc.NameFQDN = origin
	} else {
		rc.Name = short
		rc.NameFQDN = dnsutilv1.AddOrigin(short, origin)
	}
}

// SetLabelFromFQDN sets the .Name/.NameFQDN fields given a FQDN and origin.
// fqdn may have a trailing "." but it is not required.
// origin may not have a trailing dot.
func (rc *RecordConfig) SetLabelFromFQDN(fqdn, origin string) {
	// Assertions that make sure the function is being used correctly:
	if strings.HasSuffix(origin, ".") {
		panic(fmt.Errorf("origin (%s) is not supposed to end with a dot", origin))
	}
	if strings.HasSuffix(fqdn, "..") {
		panic(fmt.Errorf("fqdn (%s) is not supposed to end with double dots", origin))
	}

	// Trim off a trailing dot.
	fqdn = strings.TrimSuffix(fqdn, ".")

	fqdn = strings.ToLower(fqdn)
	origin = strings.ToLower(origin)
	rc.Name = dnsutilv1.TrimDomainName(fqdn, origin)
	rc.NameFQDN = fqdn
}

// GetLabel returns the shortname of the label associated with this RecordConfig.
// It will never end with ".". It does not need further shortening (i.e. if it
// returns "foo.com" and the domain is "foo.com" then the FQDN is actually
// "foo.com.foo.com"). It will never be "" (the apex is returned as "@").
func (rc *RecordConfig) GetLabel() string {
	return rc.Name
}

// GetLabelFQDN returns the FQDN of the label associated with this RecordConfig.
// It will not end with ".".
func (rc *RecordConfig) GetLabelFQDN() string {
	return rc.NameFQDN
}

// ToComparableNoTTL returns a comparison string. If you need to compare two
// RecordConfigs, you can simply compare the string returned by this function.
// The comparison includes all fields except TTL and any provider-specific
// metafields.  Provider-specific metafields like CF_PROXY are not the same as
// pseudo-records like ANAME or R53_ALIAS.
func (rc *RecordConfig) ToComparableNoTTL() string {
	if rc.ComparableV3 != "" {
		return rc.ComparableV3
	}
	// if rc.IsModernType() {
	// 	return rc.Comparable
	// }

	switch rc.Type {
	case "SOA":
		return fmt.Sprintf("%s %v %d %d %d %d", rc.target, rc.SoaMbox, rc.SoaRefresh, rc.SoaRetry, rc.SoaExpire, rc.SoaMinttl)
		// SoaSerial is not included because it isn't used in comparisons.
	case "TXT":
		// fmt.Fprintf(os.Stdout, "DEBUG: ToComNoTTL raw txts=%s q=%q\n", rc.target, rc.target)
		r := txtutil.EncodeSingle(rc.target)
		// fmt.Fprintf(os.Stdout, "DEBUG: ToComNoTTL cmp txts=%s q=%q\n", r, r)
		return r
	case "LUA":
		return rc.luaCombined()
	case "UNKNOWN":
		return fmt.Sprintf("rtype=%s rdata=%s", rc.UnknownTypeName, rc.target)
	case "HTTPS", "SVCB":
		//panic("unused ToComparableNoTTL for SVCB/HTTPS. Should be using .ComparableV3 instead")
		return rc.targetCombinedSVCBRaw()

	}
	return rc.GetTargetCombined()
}

// ToRR converts a RecordConfig to a dns.RR.
func (rc *RecordConfig) ToRR() dnsv1.RR {
	// Function is not valid on pseudo-types.
	rdtype, ok := dnsv1.StringToType[rc.Type]
	if !ok {
		log.Fatalf("No such DNS type as (%#v)\n", rc.Type)
	}

	// If this IsModernType, the dns.RR is already in rc.F.
	if rr, ok := rc.F.(dnsv1.RR); ok {
		rr.Header().Name = rc.NameFQDN + "."
		rr.Header().Rrtype = rdtype
		rr.Header().Class = dnsv1.ClassINET
		rr.Header().Ttl = rc.TTL
		if rc.TTL == 0 {
			rr.Header().Ttl = DefaultTTL
		}
		return rr
	}

	// Magically create an RR of the correct type.
	rr := dnsv1.TypeToRR[rdtype]()

	// Fill in the header.
	rr.Header().Name = rc.NameFQDN + "."
	rr.Header().Rrtype = rdtype
	rr.Header().Class = dnsv1.ClassINET
	rr.Header().Ttl = rc.TTL
	if rc.TTL == 0 {
		rr.Header().Ttl = DefaultTTL
	}

	// Fill in the data.
	switch rdtype { // #rtype_variations
	case dnsv1.TypeA:
		addr := rc.GetTargetIP()
		s := addr.AsSlice()
		rr.(*dnsv1.A).A = s[0:4]
	case dnsv1.TypeAAAA:
		addr := rc.GetTargetIP()
		s := addr.AsSlice()
		rr.(*dnsv1.AAAA).AAAA = s[0:16]
	case dnsv1.TypeCAA:
		rr.(*dnsv1.CAA).Flag = rc.CaaFlag
		rr.(*dnsv1.CAA).Tag = rc.CaaTag
		rr.(*dnsv1.CAA).Value = rc.GetTargetField()
	case dnsv1.TypeCNAME:
		rr.(*dnsv1.CNAME).Target = rc.GetTargetField()
	case dnsv1.TypeDHCID:
		rr.(*dnsv1.DHCID).Digest = rc.GetTargetField()
	case dnsv1.TypeDNAME:
		rr.(*dnsv1.DNAME).Target = rc.GetTargetField()
	case dnsv1.TypeDS:
		rr.(*dnsv1.DS).KeyTag = rc.DsKeyTag
		rr.(*dnsv1.DS).Algorithm = rc.DsAlgorithm
		rr.(*dnsv1.DS).DigestType = rc.DsDigestType
		rr.(*dnsv1.DS).Digest = rc.DsDigest
	case dnsv1.TypeDNSKEY:
		rr.(*dnsv1.DNSKEY).Flags = rc.DnskeyFlags
		rr.(*dnsv1.DNSKEY).Protocol = rc.DnskeyProtocol
		rr.(*dnsv1.DNSKEY).Algorithm = rc.DnskeyAlgorithm
		rr.(*dnsv1.DNSKEY).PublicKey = rc.DnskeyPublicKey
	case dnsv1.TypeHTTPS:
		rr.(*dnsv1.HTTPS).Priority = rc.SvcPriority
		rr.(*dnsv1.HTTPS).Target = rc.GetTargetField()
		rr.(*dnsv1.HTTPS).Value = rc.GetSVCBValue()
	case dnsv1.TypeLOC:
		// fmt.Printf("ToRR long: %d, lat:%d, sz: %d, hz:%d, vt:%d\n", rc.LocLongitude, rc.LocLatitude, rc.LocSize, rc.LocHorizPre, rc.LocVertPre)
		// fmt.Printf("ToRR rc: %+v\n", *rc)
		rr.(*dnsv1.LOC).Version = rc.LocVersion
		rr.(*dnsv1.LOC).Longitude = rc.LocLongitude
		rr.(*dnsv1.LOC).Latitude = rc.LocLatitude
		rr.(*dnsv1.LOC).Altitude = rc.LocAltitude
		rr.(*dnsv1.LOC).Size = rc.LocSize
		rr.(*dnsv1.LOC).HorizPre = rc.LocHorizPre
		rr.(*dnsv1.LOC).VertPre = rc.LocVertPre
	case dnsv1.TypeMX:
		rr.(*dnsv1.MX).Preference = rc.MxPreference
		rr.(*dnsv1.MX).Mx = rc.GetTargetField()
	case dnsv1.TypeNAPTR:
		rr.(*dnsv1.NAPTR).Order = rc.NaptrOrder
		rr.(*dnsv1.NAPTR).Preference = rc.NaptrPreference
		rr.(*dnsv1.NAPTR).Flags = rc.NaptrFlags
		rr.(*dnsv1.NAPTR).Service = rc.NaptrService
		rr.(*dnsv1.NAPTR).Regexp = rc.NaptrRegexp
		rr.(*dnsv1.NAPTR).Replacement = rc.GetTargetField()
	case dnsv1.TypeNS:
		rr.(*dnsv1.NS).Ns = rc.GetTargetField()
	case dnsv1.TypeOPENPGPKEY:
		rr.(*dnsv1.OPENPGPKEY).PublicKey = rc.GetTargetField()
	case dnsv1.TypePTR:
		rr.(*dnsv1.PTR).Ptr = rc.GetTargetField()
	case dnsv1.TypeSMIMEA:
		rr.(*dnsv1.SMIMEA).Usage = rc.SmimeaUsage
		rr.(*dnsv1.SMIMEA).MatchingType = rc.SmimeaMatchingType
		rr.(*dnsv1.SMIMEA).Selector = rc.SmimeaSelector
		rr.(*dnsv1.SMIMEA).Certificate = rc.GetTargetField()
	case dnsv1.TypeSOA:
		rr.(*dnsv1.SOA).Ns = rc.GetTargetField()
		rr.(*dnsv1.SOA).Mbox = rc.SoaMbox
		rr.(*dnsv1.SOA).Serial = rc.SoaSerial
		rr.(*dnsv1.SOA).Refresh = rc.SoaRefresh
		rr.(*dnsv1.SOA).Retry = rc.SoaRetry
		rr.(*dnsv1.SOA).Expire = rc.SoaExpire
		rr.(*dnsv1.SOA).Minttl = rc.SoaMinttl
	case dnsv1.TypeSPF:
		rr.(*dnsv1.SPF).Txt = rc.GetTargetTXTSegmented()
	case dnsv1.TypeSRV:
		rr.(*dnsv1.SRV).Priority = rc.SrvPriority
		rr.(*dnsv1.SRV).Weight = rc.SrvWeight
		rr.(*dnsv1.SRV).Port = rc.SrvPort
		rr.(*dnsv1.SRV).Target = rc.GetTargetField()
	case dnsv1.TypeSSHFP:
		rr.(*dnsv1.SSHFP).Algorithm = rc.SshfpAlgorithm
		rr.(*dnsv1.SSHFP).Type = rc.SshfpFingerprint
		rr.(*dnsv1.SSHFP).FingerPrint = rc.GetTargetField()
	case dnsv1.TypeSVCB:
		rr.(*dnsv1.SVCB).Priority = rc.SvcPriority
		rr.(*dnsv1.SVCB).Target = rc.GetTargetField()
		rr.(*dnsv1.SVCB).Value = rc.GetSVCBValue()
	case dnsv1.TypeTLSA:
		rr.(*dnsv1.TLSA).Usage = rc.TlsaUsage
		rr.(*dnsv1.TLSA).MatchingType = rc.TlsaMatchingType
		rr.(*dnsv1.TLSA).Selector = rc.TlsaSelector
		rr.(*dnsv1.TLSA).Certificate = rc.GetTargetField()
	case dnsv1.TypeTXT:
		rr.(*dnsv1.TXT).Txt = rc.GetTargetTXTSegmented()
	default:
		panic(fmt.Sprintf("ToRR: Unimplemented rtype %v", rc.Type))
		// We panic so that we quickly find any switch statements
		// that have not been updated for a new RR type.
	}

	return rr
}

// GetDependencies returns the FQDNs on which this record dependents.
func (rc *RecordConfig) GetDependencies() []string {
	switch rc.Type {
	// #rtype_variations
	case "NS", "SRV", "CNAME", "DNAME", "MX", "ALIAS", "AZURE_ALIAS", "R53_ALIAS":
		return []string{
			rc.target,
		}
	}

	return []string{}
}

// RecordKey represents a resource record in a format used by some systems.
type RecordKey struct {
	NameFQDN string
	Type     string
}

func (rk *RecordKey) String() string {
	return rk.NameFQDN + ":" + rk.Type
}

// Key converts a RecordConfig into a RecordKey.
func (rc *RecordConfig) Key() RecordKey {
	t := rc.Type
	if rc.R53Alias != nil {
		if v, ok := rc.R53Alias["type"]; ok {
			// Route53 aliases append their alias type, so that records for the same
			// label with different alias types are considered separate.
			t = fmt.Sprintf("%s_%s", t, v)
		}
	} else if rc.AzureAlias != nil {
		if v, ok := rc.AzureAlias["type"]; ok {
			// Azure aliases append their alias type, so that records for the same
			// label with different alias types are considered separate.
			t = fmt.Sprintf("%s_%s", t, v)
		}
	}
	// Route 53 weighted/failover routing: records with different
	// SetIdentifiers are separate ResourceRecordSets in the R53 API,
	// so they must have distinct keys for the diff engine.
	if sid, ok := rc.Metadata["r53_set_identifier"]; ok && sid != "" {
		t = fmt.Sprintf("%s!%s", t, sid)
	}
	return RecordKey{rc.NameFQDN, t}
}

// func (rc *RecordConfig) GetSVCBValueV2() []svcbv2.Pair {
// 	switch v := rc.RDATA.(type) {
// 	case *dnsrdatav2.SVCB:
// 		return v.Value
// 	default:
// 		panic(fmt.Sprintf("GetSVCBValueV2 failed. Unknown rdata type: %T", rc.RDATA))
// 	}
// 	//return rc.RDATA.(*dnsrdatav2.SVCB).Value

// 	return nil
// }

// GetSVCBValue returns the SVCB Key/Values as a list of Key/Values.
// Used to construct dnsv.RR of type SVCB or HTTPS. (This is legacy code that should go away eventualy).
func (rc *RecordConfig) GetSVCBValue() []dnsv1.SVCBKeyValue {
	// if !strings.Contains(rc.SvcParams, "IGNORE+DNSCONTROL") {
	// 	rc.SvcParams = strings.ReplaceAll(rc.SvcParams, "ech=IGNORE", "ech=IGNORE+DNSCONTROL+++")
	// }
	// if strings.Contains(rc.SvcParams, "IGNORE") {
	// 	p := rc.SvcParams
	// 	p = strings.ReplaceAll(" "+p+" ", " ech=IGNORE ", " ech=1000 ")
	// 	p = strings.ReplaceAll(" "+p+" ", ` ech="IGNORE" `, " ech=1000 ")
	// 	rc.SvcParams = strings.TrimSpace(p)
	// }

	var s string
	if rc.RDATA != nil {
		s = fmt.Sprintf("%s %s %s", rc.NameFQDN, rc.Type, rc.RDATA.String())
	} else {
		s = fmt.Sprintf("%s %s %d %s %s", rc.NameFQDN, rc.Type, rc.SvcPriority, rc.target, rc.SvcParams)
	}
	fmt.Printf("DEBUG: GetSVCBValue: s=%q\n", s)
	record, err := dnsv1.NewRR(s)
	if err != nil {
		log.Fatalf("could not parse SVCB record: %s", err)
	}
	switch r := record.(type) {
	case *dnsv1.HTTPS:
		return r.Value
	case *dnsv1.SVCB:
		return r.Value
	}

	return nil
}

// IsModernType returns true if this RecordConfig is a record type implemented
// in the new ("Modern") style (i.e. uses the RecordConfig .F field to store
// the rdata of the record).
//
// Since this relies on .F, it must be used only after the RecordConfig
// has been populated. Otherwise, use rtypecontrol.IsModernType(recordTypeName),
// which takes the type name as input.
//
// NOTE: Do not confuse this with rtypeinfo.IsModernType() which provides
// similar functionality.  This function is used to have a RecordConfig reveal
// if it uses a modern type.  rtypeinfo.IsModernType() takes the rtype name as
// a string argument.
//
// FUTURE(tlim): Once all record types have been migrated to use ".F", this function can be removed.
func (rc *RecordConfig) IsModernType() bool {
	return rc.F != nil
}

// Records is a list of *RecordConfig.
type Records []*RecordConfig

// HasRecordTypeName returns True if there is a record with this rtype and name.
func (recs Records) HasRecordTypeName(rtype, name string) bool {
	for _, r := range recs {
		if r.Type == rtype && r.Name == name {
			return true
		}
	}
	return false
}

// GetByType returns the records that match rtype typeName.
func (recs Records) GetByType(typeName string) Records {
	results := Records{}
	for _, rec := range recs {
		if rec.Type == typeName {
			results = append(results, rec)
		}
	}
	return results
}

// GroupedByKey returns a map of keys to records.
func (recs Records) GroupedByKey() map[RecordKey]Records {
	groups := map[RecordKey]Records{}
	for _, rec := range recs {
		groups[rec.Key()] = append(groups[rec.Key()], rec)
	}
	return groups
}

// GroupedByFQDN returns a map of keys to records, grouped by FQDN.
func (recs Records) GroupedByFQDN() ([]string, map[string]Records) {
	order := []string{}
	groups := map[string]Records{}
	for _, rec := range recs {
		namefqdn := rec.GetLabelFQDN()
		if _, found := groups[namefqdn]; !found {
			order = append(order, namefqdn)
		}
		groups[namefqdn] = append(groups[namefqdn], rec)
	}
	return order, groups
}

// GetAllDependencies concatinates all dependencies of all records.
func (recs Records) GetAllDependencies() []string {
	var dependencies []string
	for _, rec := range recs {
		dependencies = append(dependencies, rec.GetDependencies()...)
	}

	return dependencies
}

// PostProcessRecords does any post-processing of the downloaded DNS records.
// Deprecated. zonerecords.CorrectZoneRecords() calls Downcase directly.
func PostProcessRecords(recs []*RecordConfig) {
	Downcase(recs)
}

// Downcase converts all labels and targets to lowercase in a list of RecordConfig.
func Downcase(recs []*RecordConfig) {
	for _, r := range recs {
		if r.IsModernType() {
			continue
		}

		r.Name = strings.ToLower(r.Name)
		r.NameFQDN = strings.ToLower(r.NameFQDN)
		switch r.Type { // #rtype_variations
		case "AKAMAICDN", "AKAMAITLC", "ALIAS", "AAAA", "ANAME", "CNAME", "DNAME", "DS", "DNSKEY", "MX", "NS", "NAPTR", "SMIMEA", "PTR", "SRV", "TLSA", "AZURE_ALIAS":
			// Target is case insensitive. Downcase it.
			r.target = strings.ToLower(r.target)
			// BUGFIX(tlim): isn't ALIAS in the wrong case statement?
		case "A", "CAA", "CLOUDFLAREAPI_SINGLE_REDIRECT", "CF_REDIRECT", "CF_TEMP_REDIRECT", "CF_WORKER_ROUTE", "DHCID", "IMPORT_TRANSFORM", "LOC", "OPENPGPKEY", "SSHFP", "TXT", "ADGUARDHOME_A_PASSTHROUGH", "ADGUARDHOME_AAAA_PASSTHROUGH":
			// Do nothing. (IP address or case sensitive target)
		case "SOA":
			if r.target != "DEFAULT_NOT_SET." {
				r.target = strings.ToLower(r.target) // .target stores the Ns
			}
			if r.SoaMbox != "DEFAULT_NOT_SET." {
				r.SoaMbox = strings.ToLower(r.SoaMbox)
			}
		default:
			// TODO: we'd like to panic here, but custom record types complicate things.
		}
	}
}

// CanonicalizeTargets turns Targets into FQDNs.
func CanonicalizeTargets(recs []*RecordConfig, origin string) {
	originFQDN := origin + "."

	for _, r := range recs {

		if r.IsModernType() {
			continue
		}

		switch r.Type { // #rtype_variations
		case "ALIAS", "ANAME", "CNAME", "DNAME", "DS", "DNSKEY", "MX", "NS", "NAPTR", "PTR", "SRV":
			// Target is a hostname that might be a shortname. Turn it into a FQDN.
			r.target = dnsutilv1.AddOrigin(r.target, originFQDN)
		case "A", "AKAMAICDN", "AKAMAITLC", "CAA", "DHCID", "CLOUDFLAREAPI_SINGLE_REDIRECT", "CF_REDIRECT", "CF_TEMP_REDIRECT", "CF_WORKER_ROUTE", "HTTPS", "IMPORT_TRANSFORM", "LOC", "OPENPGPKEY", "SMIMEA", "SSHFP", "SVCB", "TLSA", "TXT", "ADGUARDHOME_A_PASSTHROUGH", "ADGUARDHOME_AAAA_PASSTHROUGH":
			// Do nothing.
		case "SOA":
			if r.target != "DEFAULT_NOT_SET." {
				r.target = dnsutilv1.AddOrigin(r.target, originFQDN) // .target stores the Ns
			}
			if r.SoaMbox != "DEFAULT_NOT_SET." {
				r.SoaMbox = dnsutilv1.AddOrigin(r.SoaMbox, originFQDN)
			}
		default:
			// TODO: we'd like to panic here, but custom record types complicate things.
		}
	}
}
