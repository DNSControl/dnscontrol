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
	//NameFQDNRaw     string `json:"-"` // .NameFQDN as the user entered it in dnsconfig.js (downcased).
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
	//fmt.Printf("DEBUG: MARSHALING %v\n", rc.Name)
	recj := &struct {
		RecordConfig
		RDATA dnsv2.RDATA `json:"rdata,omitempty"`
	}{
		RecordConfig: *rc,
	}
	recj.RDATA = rc.GetRDATA()
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
		rc.NameFQDN = nameutil.ToFqdnNoDot(short, origin)
	}
	// TODO(tlim): This also needs to make .NameUnicode / .NameFQDNUnicode
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

// ToRRv2 converts a RecordConfig to a dnsv2.RR.
func (rc *RecordConfig) ToRRv2() dnsv2.RR {
	// Function is only valid on defined types.
	rdtype, ok := dnsv2.StringToType[rc.Type]
	if !ok {
		log.Fatalf("No such DNS type as (%#v)\n", rc.Type)
	}
	if rdtype != rc.TypeNum {
		panic("should not happen: ToRRv2")
	}

	ttl := rc.TTL
	if ttl == 0 {
		ttl = DefaultTTL
	}

	// Make the header
	hdr := dnsv2.Header{
		Name:  rc.NameFQDN + ".",
		TTL:   ttl,
		Class: dnsv2.ClassINET,
	}

	rd := rc.GetRDATA()

	rr := dnsv2.TypeToRR[rdtype]()    // Magically create an RR of the correct type.
	*rr.Header() = hdr                // Point the header at the header we created.
	dnsv2.TypeToRDATA[rdtype](rr, rd) // Copy rd into the fields.

	return rr
}

// GetDependencies returns the FQDNs on which this record dependents.
// For example, some providers won't create a CNAME until the target already exists.
// DNSControl will assure that the target exists before the CNAME is created if
// this function returns the target name when called on a CNAME record.
// The reverse is true for deletions. DNSControl will delete the records for
// rc.GetDependencies() before deleting the rc.
func (rc *RecordConfig) GetDependencies() []string {
	switch rc.Type {
	case "NS":
		return []string{rc.AsNS().Ns}
	case "SRV":
		return []string{rc.AsSRV().Target}
	case "CNAME":
		return []string{rc.AsCNAME().Target}
	case "DNAME":
		return []string{rc.AsDNAME().Target}
	case "MX":
		return []string{rc.AsMX().Mx}
	case "ALIAS":
		return []string{rc.AsALIAS().Target}
	case "AZURE_ALIAS":
		return []string{rc.AsAZUREALIAS().Target}
	case "R53_ALIAS":
		return []string{rc.AsR53ALIAS().Target}
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
	if rc.GetRDATA() != nil {
		switch rc.Type {
		case "R53_ALIAS":
			// Route53 aliases append their alias type, so that records for the same
			// label with different alias types are considered separate.
			t = fmt.Sprintf("%s_%s", t, rc.AsR53ALIAS().AliasType)
		case "AZURE_ALIAS":
			// Azure aliases append their alias type, so that records for the same
			// label with different alias types are considered separate.
			t = fmt.Sprintf("%s_%s", t, rc.AsAZUREALIAS().AliasType)
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

func (rc *RecordConfig) IsTTLSignificant() bool {
	// "private types" don't really have a useful TTL.
	// There may be better ways to determine this.  Right now
	// this only affects checkRecordSetHasMultipleTTLs().
	return rc.TypeNum < 65280
}

// Records is a list of *RecordConfig.
type Records []*RecordConfig

// StringEach returns a list of strings, one for each RecordConfig in recs.
func (recs Records) StringEach() []string {
	r := make([]string, 0, len(recs))
	for _, rc := range recs {
		r = append(r, rc.GetRDATA().String())
	}
	return r
}

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
