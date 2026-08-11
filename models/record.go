package models

import (
	"encoding/json"
	"fmt"
	"log"
	"strings"

	dnsv2 "codeberg.org/miekg/dns"
	"github.com/DNSControl/dnscontrol/v5/pkg/nameutil"
	"github.com/qdm12/reprint"
)

// RecordConfig stores a DNS record whether it was created from data downloaded from
// a provider's API ("actual") or from user input in dndsconfig.js ("desired").
type RecordConfig struct {

	// TypeNum is the assigned number of the record's type. 1 for A, 5 for CNAME, etc. See dnsv2.TypeToString and dnsv2.StringToType.
	TypeNum uint16 `json:"typenum,omitempty"`

	// Type is the DNS record type (rtype), all caps, "A", "MX", etc. (Deprecated. Use .TypeNum)
	Type string `json:"type"`

	// rdata is the fields of the record.
	rdata dnsv2.RDATA

	// TTL is the DNS record's TTL in seconds. 0 means provider default.
	TTL uint32 `json:"ttl,omitempty"`

	// Name is the shortname i.e. the FQDN without the parent domains's suffix.
	// It should never be "".  Record at the apex (naked domain) are represented by "@".
	Name        string `json:"name"`                   // The short name, PunyCode. See above.
	NameUnicode string `json:"name_unicode,omitempty"` // .Name as Unicode (downcased, then convertedot Unicode).

	// This is the FQDN version of .Name. It should never have a trailing ".".
	NameFQDNRaw     string `json:"-"` // .NameFQDN as the user entered it in dnsconfig.js (downcased).
	NameFQDN        string `json:"-"` // Must end with ".$origin".
	NameFQDNUnicode string `json:"-"` // .NameFQDN as Unicode (downcased, then convertedot Unicode).

	// ComparableV3 is an opaque string that can be used to compare two
	// RecordConfigs for equality. Typically this is the Zonefile line
	// minus the label and TTL.
	// The V3 distingues itself from .Comparable, which it will eventually replace.
	// NB(tlim): Not currently used. Placeholder for future feature.
	ComparableV3 string `json:"comparablev3,omitempty"`

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

	UnknownTypeName string `json:"unknown_type_name,omitempty"`

	// FYI: If you add a field to this struct, also add it to the list in the UnmarshalJSON function.
}

// MarshalJSON marshals RecordConfig.
func (rc *RecordConfig) MarshalJSON() ([]byte, error) {
	recj := &struct {
		RecordConfig
		RDATA dnsv2.RDATA `json:"rdata,omitempty"`
	}{
		RecordConfig: *rc,
	}
	j, err := json.Marshal(*recj)
	if err != nil {
		return nil, err
	}
	return j, nil
}

// FixPosition takes the string representation of a position in a file that
// comes from dnsconfig.js's initial execution, and reduces it down to just the
// line/position we display to the user. The input is not well-defined, thus if
// we find something we don't expect, we just return the original input.
// TODO: Move this closer to where it is used.
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
	newR.rdata = rc.rdata
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
	// TODO(tlim): This also needs to make .NameUnicode / .NameFQDNUnicode
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

// GroupedByFQDN returns a map of keys to records, grouped by FQDN.
func (recs Records) GroupedByFQDN() map[string]Records {
	groups := map[string]Records{}
	for _, rec := range recs {
		namefqdn := rec.GetLabelFQDN()
		groups[namefqdn] = append(groups[namefqdn], rec)
	}
	return groups
}

// GetAllDependencies concatinates all dependencies of all records.
func (recs Records) GetAllDependencies() []string {
	var dependencies []string
	for _, rec := range recs {
		dependencies = append(dependencies, rec.GetDependencies()...)
	}

	return dependencies
}
