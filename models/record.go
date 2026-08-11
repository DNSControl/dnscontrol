package models

import (
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"github.com/DNSControl/dnscontrol/v4/pkg/txtutil"
	"github.com/jinzhu/copier"
	dnsv1 "github.com/miekg/dns"
	dnsutilv1 "github.com/miekg/dns/dnsutil"
	"github.com/qdm12/reprint"
)

// RecordConfig stores a DNS record whether it was created from data downloaded from
// a provider's API ("actual") or from user input in dndsconfig.js ("desired").
type RecordConfig struct {
	// Type is the DNS record type (rtype), all caps, "A", "MX", etc.
	Type string `json:"type"`

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
	Comparable string `json:"comparable,omitempty"` // Cache of ToComparableNoTTL()

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

// MarshalJSON marshals RecordConfig.
func (rc *RecordConfig) MarshalJSON() ([]byte, error) {
	recj := &struct {
		RecordConfig
		Target string `json:"target"`
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
		Target string `json:"target"`

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
	if rc.IsModernType() {
		return rc.Comparable
	}

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
		if s := addr.AsSlice(); len(s) == 4 {
			rr.(*dnsv1.A).A = s
		}
	case dnsv1.TypeAAAA:
		addr := rc.GetTargetIP()
		if s := addr.AsSlice(); len(s) == 16 {
			rr.(*dnsv1.AAAA).AAAA = s
		}
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
		panic("DS should have been handled as modern type")
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

// GetSVCBValue returns the SVCB Key/Values as a list of Key/Values.
func (rc *RecordConfig) GetSVCBValue() []dnsv1.SVCBKeyValue {
	if !strings.Contains(rc.SvcParams, "IGNORE+DNSCONTROL") {
		rc.SvcParams = strings.ReplaceAll(rc.SvcParams, "ech=IGNORE", "ech=IGNORE+DNSCONTROL+++")
	}

	record, err := dnsv1.NewRR(fmt.Sprintf("%s %s %d %s %s", rc.NameFQDN, rc.Type, rc.SvcPriority, rc.target, rc.SvcParams))
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
