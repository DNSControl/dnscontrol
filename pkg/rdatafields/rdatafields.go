// Package rdatafields classifies the fields of an RDATA struct so that code
// can normalize them without a hand-maintained switch statement per rtype.
//
// The classification is derived from the struct tags that
// codeberg.org/miekg/dns/rdata already puts on its fields, plus a small
// exclusion list (tagged fields that are domain names but are NOT hostnames)
// and a small inclusion list (fields that are hostnames but carry no usable
// tag, mostly in pkg/privatetypes/rdata).
//
// build/normalizergen uses Fields to generate
// models.normalizeRDATA without runtime reflection.
package rdatafields

import (
	"reflect"
	"strings"
	"sync"
)

// Kind says how a field must be normalized before it is compared.
type Kind int

const (
	// KindOther means "leave it alone".
	KindOther Kind = iota

	// KindTargetHost is a domain name that names a host: CNAME.Target,
	// MX.Mx, SRV.Target, ALIAS.Target, and so on. It is case-insensitive
	// and fully qualified, so we store it downcased and ending in ".".
	KindTargetHost

	// KindMailbox is an email address encoded as a domain name
	// (SOA.Mbox, RP.Mbox). It is case-insensitive on the wire, but the
	// local part is conventionally preserved, and it does not name a
	// host. We do not rewrite these.
	KindMailbox

	// KindHexLower is a hex-encoded blob (DS.Digest, SSHFP.FingerPrint,
	// TLSA.Certificate). Case-insensitive; we store it downcased.
	KindHexLower

	// KindOpaqueName is a domain name that is neither a host nor a
	// mailbox: a DNSSEC internal (RRSIG.SignerName, NSEC.NextDomain) or
	// an identifier that merely borrows the wire format
	// (TSIG.Algorithm). These must be preserved byte for byte.
	KindOpaqueName
)

func (k Kind) String() string {
	switch k {
	case KindTargetHost:
		return "TargetHost"
	case KindMailbox:
		return "Mailbox"
	case KindHexLower:
		return "HexLower"
	case KindOpaqueName:
		return "OpaqueName"
	}
	return "Other"
}

// Field is one classified field of an rdata struct.
type Field struct {
	Name  string // Go field name, e.g. "Target"
	Index int    // index into the struct, for reflect.Value.Field
	Kind  Kind
	Slice bool // true if the field is a []string, e.g. HIP.RendezvousServers
}

// Fields returns the classified fields of rd's struct type. rd may be a
// struct or a pointer to one. Results are cached per type.
func Fields(rd any) []Field {
	// What is this?
	t := reflect.TypeOf(rd)
	for t != nil && t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	if t == nil || t.Kind() != reflect.Struct {
		return nil
	}

	// Do we have a cached answer?
	if v, ok := cache.Load(t); ok {
		return v.([]Field)
	}

	// Generate the answer.
	var out []Field
	for i := range t.NumField() {
		f := t.Field(i)
		k := Classify(t.Name(), f)
		if k == KindOther {
			continue
		}
		out = append(out, Field{
			Name:  f.Name,
			Index: i,
			Kind:  k,
			Slice: f.Type.Kind() == reflect.Slice,
		})
	}

	// Cache the answer.
	cache.Store(t, out)
	return out
}

// tagKind maps a `dns:"..."` tag value to its default classification.
//
// "cname", "name" and "mname" are all domain names on the wire; miekg uses
// the three spellings to control compression and escaping, not meaning, so
// the split below is ours:
//
//	cname, name -> a hostname, unless overridden by notHost
//	mname       -> a mailbox (SOA.Mbox, RP.Mbox, and the obsolete MG/MR/MINFO)
var tagKind = map[string]Kind{
	"cname": KindTargetHost,
	"name":  KindTargetHost,
	"mname": KindMailbox,
	"hex":   KindHexLower,
}

// notHost lists "Type.Field" pairs whose dns tag says "name"/"cname" but
// which do not name a host. Without this list the tag rule would rewrite
// DNSSEC material and protocol identifiers.
var notHost = map[string]Kind{
	// DNSSEC internals. The signature is computed over these bytes.
	// (SIG reuses rdata.RRSIG and NXT reuses rdata.NSEC, so they need no
	// entries of their own.)
	"RRSIG.SignerName": KindOpaqueName,
	"NSEC.NextDomain":  KindOpaqueName,

	// Algorithm identifiers that merely use the domain-name wire format,
	// e.g. "hmac-sha256.". Also meta-RRs that never live in a zone.
	"TSIG.Algorithm": KindOpaqueName,
	"TKEY.Algorithm": KindOpaqueName,

	// RP.Txt is the owner name of a TXT record, not a host.
	"RP.Txt": KindOpaqueName,

	// TALINK is obsolete and its names are zone-ordering internals.
	"TALINK.NextName":     KindOpaqueName,
	"TALINK.PreviousName": KindOpaqueName,
}

// isHost lists "Type.Field" pairs that ARE hostnames but carry no dns tag.
// This is the escape hatch for hand-written rdata structs.
//
// Generated pkg/privatetypes/rdata structs do not need an entry here: the
// generator emits `dnscontrol:"targethost"` on every field declared
// `type: TargetHost` in types_generate.yaml.
//
// Do not add a field here just because it is named "Target". In
// pkg/privatetypes most Target fields are deliberately NOT hostnames:
// AZURE_ALIAS.Target is an Azure resource id, BUNNY_DNS_RDR.Target and
// FRAME.Target are URLs, MIKROTIK_FORWARDER.Target is an IP address.
var isHost = map[string]Kind{}

// obsolete lists rdata struct names that dnscontrol does not normalize:
// record types that are formally obsolete, were deprecated before they were
// deployed, or were never standardized. Classify reports KindOther for every
// field of these, so generated code never mentions them.
//
// They are skipped rather than classified because each one would otherwise
// need its hostname/mailbox decision reviewed and tested for no practical
// benefit. Note this is keyed by STRUCT name: SIG shares rdata.RRSIG and NXT
// shares rdata.NSEC, so neither can be skipped without also skipping its
// modern counterpart.
//
// Deliberately NOT here: AFSDB, RT, KX, NAPTR, LP, HIP, DSYNC and RP. Those
// are rare but live, standards-track, and users do write them.
var obsolete = map[string]bool{
	"MD": true, // Obsoleted by RFC 973.
	"MF": true, // Obsoleted by RFC 973.

	// The RFC 1035 mailbox experiment. Never deployed; MX won.
	"MB":    true,
	"MG":    true,
	"MR":    true,
	"MINFO": true,

	"NXT":     true, // Obsoleted by RFC 3755 in favor of NSEC.
	"NSAPPTR": true, // NSAP-PTR; NSAP support was deprecated by RFC 1637.
	"PX":      true, // RFC 2163 X.400 mapping. Dead with X.400.
	"TALINK":  true, // Expired draft, never standardized.
	"EID":     true, // Nimrod. Never standardized.
	"NIMLOC":  true, // Nimrod. Never standardized.
	"TA":      true, // DNSSEC trust anchor experiment. Never standardized.
}

// IsObsolete reports whether structName is on the skip list above. Exported
// so a generator can log what it left out.
func IsObsolete(structName string) bool { return obsolete[structName] }

var cache sync.Map // reflect.Type -> []Field

// Classify returns the Kind of one field of the rdata struct named
// structName. structName is the Go struct name ("CNAME"), not the rtype
// name, so that types which share a struct (HTTPS and SVCB, CDS and DS) are
// classified once.
func Classify(structName string, f reflect.StructField) Kind {
	if obsolete[structName] {
		return KindOther
	}

	key := structName + "." + f.Name

	// 1. Explicit overrides win over any tag.
	if k, ok := notHost[key]; ok {
		return k
	}
	if k, ok := isHost[key]; ok {
		return k
	}

	// 2. Our own tag, used by generated privatetypes structs.
	if v := f.Tag.Get("dnscontrol"); v != "" {
		if v == "targethost" {
			return KindTargetHost
		}
	}

	// 3. The tag miekg already maintains.
	tag := f.Tag.Get("dns")
	if i := strings.IndexByte(tag, ','); i >= 0 {
		tag = tag[:i]
	}
	if k, ok := tagKind[tag]; ok {
		return k
	}
	return KindOther
}
