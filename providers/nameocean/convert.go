package nameocean

import (
	"fmt"
	"strconv"
	"strings"

	dnsv2 "codeberg.org/miekg/dns"
	"github.com/DNSControl/dnscontrol/v5/models"
)

// nativeRecord is one record of a NameOcean zone document.
type nativeRecord struct {
	Name       string `json:"name"`
	Type       string `json:"type"`
	Value      string `json:"value"`
	TTL        string `json:"ttl,omitempty"`
	Priority   *int   `json:"priority,omitempty"`
	Weight     *int   `json:"weight,omitempty"`
	Port       *int   `json:"port,omitempty"`
	Tag        string `json:"tag,omitempty"`
	Flags      *int   `json:"flags,omitempty"`
	KeyTag     *int   `json:"key_tag,omitempty"`
	Algorithm  *int   `json:"algorithm,omitempty"`
	DigestType *int   `json:"digest_type,omitempty"`
	Params     string `json:"params,omitempty"`
}

// managedTypes are the record types this provider converts. Everything else
// in a zone document (REDIRECT, PTR, anything added later) is passed through
// to the apply untouched.
var managedTypes = map[string]bool{
	"A":     true,
	"AAAA":  true,
	"ALIAS": true,
	"CAA":   true,
	"CNAME": true,
	"DS":    true,
	"HTTPS": true,
	"MX":    true,
	"NS":    true,
	"SRV":   true,
	"SVCB":  true,
	"TXT":   true,
}

// defaultTTL is what NameOcean uses when a record has no ttl.
const defaultTTL = 3600

// ttlNames maps the TTLs NameOcean accepts to their spelling in a zone document.
var ttlNames = map[uint32]string{
	30:    "30", // TXT only
	300:   "5m",
	900:   "15m",
	1800:  "30m",
	3600:  "1h",
	86400: "1d",
}

// allowedTTLs lists the TTLs NameOcean accepts, shortest first. TXT records may also use 30.
var allowedTTLs = []uint32{300, 900, 1800, 3600, 86400}

// fixTTL snaps ttl to the nearest TTL NameOcean accepts for rtype. A ttl
// exactly between two allowed values goes to the shorter one.
func fixTTL(rtype string, ttl uint32) uint32 {
	candidates := allowedTTLs
	if rtype == "TXT" {
		candidates = append([]uint32{30}, allowedTTLs...)
	}
	best := candidates[0]
	for _, c := range candidates[1:] {
		if absDiff(ttl, c) < absDiff(ttl, best) {
			best = c
		}
	}
	return best
}

func absDiff(a, b uint32) uint32 {
	if a > b {
		return a - b
	}
	return b - a
}

// parseTTL turns a zone document ttl ("5m", "1h", "1d", "30", or none) into seconds.
func parseTTL(s string) (uint32, error) {
	if s == "" {
		return defaultTTL, nil
	}
	for secs, name := range ttlNames {
		if s == name {
			return secs, nil
		}
	}
	if n, err := strconv.ParseUint(s, 10, 32); err == nil {
		return uint32(n), nil
	}
	return 0, fmt.Errorf("unknown ttl %q", s)
}

// formatTTL turns seconds into a zone document ttl. The TTL must already be
// one NameOcean accepts (see fixTTL).
func formatTTL(ttl uint32) (string, error) {
	if name, ok := ttlNames[ttl]; ok {
		return name, nil
	}
	return "", fmt.Errorf("ttl %d is not one NameOcean accepts", ttl)
}

// hostFromNative turns a NameOcean target ("mail.example.com", or "@" for the
// zone apex) into a FQDN with a trailing dot.
func hostFromNative(dc *models.DomainConfig, value string) string {
	value = strings.ToLower(strings.TrimSuffix(strings.TrimSpace(value), "."))
	if value == "" || value == "@" {
		return dc.Name + "."
	}
	return value + "."
}

// hostToNative turns a FQDN into the NameOcean form: no trailing dot, "@" for the zone apex.
func hostToNative(dc *models.DomainConfig, fqdn string) string {
	host := strings.ToLower(strings.TrimSuffix(fqdn, "."))
	if host == dc.Name {
		return "@"
	}
	return host
}

// intField reads an optional integer field and checks its range.
func intField(v *int, field string, def, maxValue int) (int, error) {
	if v == nil {
		if def < 0 {
			return 0, fmt.Errorf("missing %s", field)
		}
		return def, nil
	}
	if *v < 0 || *v > maxValue {
		return 0, fmt.Errorf("%s %d out of range 0-%d", field, *v, maxValue)
	}
	return *v, nil
}

//go:fix inline

// toRC converts a NameOcean record to a RecordConfig. It must not be called
// for types outside managedTypes.
func toRC(dc *models.DomainConfig, r nativeRecord) (*models.RecordConfig, error) {
	rc, err := toRCHelper(dc, r)
	if err != nil {
		return nil, fmt.Errorf("nameocean: %s record %q: %w", r.Type, r.Name, err)
	}
	return rc, nil
}

func toRCHelper(dc *models.DomainConfig, r nativeRecord) (*models.RecordConfig, error) {
	ttl, err := parseTTL(r.TTL)
	if err != nil {
		return nil, err
	}
	label := "@"
	if r.Name != "" && r.Name != "@" {
		label = dc.LabelFromShort(r.Name)
	}

	var rc *models.RecordConfig
	switch rtype := strings.ToUpper(r.Type); rtype { // #rtype_variations
	case "A", "AAAA":
		rc, err = dc.NewRecordConfig(label, ttl, rtype, strings.TrimSpace(r.Value))
	case "TXT":
		rc, err = dc.NewRecordConfig(label, ttl, dnsv2.TypeTXT, r.Value)
	case "CNAME", "ALIAS", "NS":
		rc, err = dc.NewRecordConfig(label, ttl, rtype, hostFromNative(dc, r.Value))
	case "MX":
		var prio int
		if prio, err = intField(r.Priority, "priority", 10, 65535); err != nil {
			return nil, err
		}
		rc, err = dc.NewRecordConfig(label, ttl, dnsv2.TypeMX, uint16(prio), hostFromNative(dc, r.Value))
	case "SRV":
		var prio, weight, port int
		if prio, err = intField(r.Priority, "priority", -1, 65535); err != nil {
			return nil, err
		}
		if weight, err = intField(r.Weight, "weight", -1, 65535); err != nil {
			return nil, err
		}
		if port, err = intField(r.Port, "port", -1, 65535); err != nil {
			return nil, err
		}
		rc, err = dc.NewRecordConfig(label, ttl, dnsv2.TypeSRV,
			uint16(prio), uint16(weight), uint16(port), hostFromNative(dc, r.Value))
	case "CAA":
		var flags int
		if flags, err = intField(r.Flags, "flags", 0, 255); err != nil {
			return nil, err
		}
		rc, err = dc.NewRecordConfig(label, ttl, dnsv2.TypeCAA, uint8(flags), strings.ToLower(r.Tag), r.Value)
	case "DS":
		var keyTag, alg, digestType int
		if keyTag, err = intField(r.KeyTag, "key_tag", -1, 65535); err != nil {
			return nil, err
		}
		if alg, err = intField(r.Algorithm, "algorithm", -1, 255); err != nil {
			return nil, err
		}
		if digestType, err = intField(r.DigestType, "digest_type", -1, 255); err != nil {
			return nil, err
		}
		rc, err = dc.NewRecordConfig(label, ttl, dnsv2.TypeDS,
			uint16(keyTag), uint8(alg), uint8(digestType), r.Value)
	case "HTTPS", "SVCB":
		var prio int
		if prio, err = intField(r.Priority, "priority", 1, 65535); err != nil {
			return nil, err
		}
		target := strings.TrimSpace(r.Value)
		if target != "." {
			target = hostFromNative(dc, target)
		}
		rc, err = dc.NewRecordConfigParse(label, ttl, rtype,
			strings.TrimSpace(fmt.Sprintf("%d %s %s", prio, target, r.Params)))
	default:
		return nil, fmt.Errorf("unsupported record type %q", r.Type)
	}
	if err != nil {
		return nil, err
	}
	return rc, nil
}

// toNative converts a RecordConfig to a NameOcean record. The TTL must
// already have been snapped with fixTTL.
func toNative(dc *models.DomainConfig, rc *models.RecordConfig) (nativeRecord, error) {
	ttl, err := formatTTL(rc.TTL)
	if err != nil {
		return nativeRecord{}, fmt.Errorf("nameocean: %s record %q: %w", rc.Type, rc.GetLabel(), err)
	}
	r := nativeRecord{
		Name: rc.GetLabel(),
		Type: rc.Type,
		TTL:  ttl,
	}

	switch rc.Type { // #rtype_variations
	case "A", "AAAA":
		r.Value = rc.GetRDATA().String()
	case "TXT":
		r.Value = rc.GetTargetTXTJoined()
	case "CNAME":
		r.Value = hostToNative(dc, rc.AsCNAME().Target)
	case "ALIAS":
		r.Value = hostToNative(dc, rc.AsALIAS().Target)
	case "NS":
		r.Value = hostToNative(dc, rc.AsNS().Ns)
	case "MX":
		f := rc.AsMX()
		r.Value = hostToNative(dc, f.Mx)
		r.Priority = new(int(f.Preference))
	case "SRV":
		f := rc.AsSRV()
		r.Value = hostToNative(dc, f.Target)
		r.Priority = new(int(f.Priority))
		r.Weight = new(int(f.Weight))
		r.Port = new(int(f.Port))
	case "CAA":
		f := rc.AsCAA()
		r.Value = f.Value
		r.Tag = f.Tag
		r.Flags = new(int(f.Flag))
	case "DS":
		f := rc.AsDS()
		r.Value = strings.ToLower(f.Digest)
		r.KeyTag = new(int(f.KeyTag))
		r.Algorithm = new(int(f.Algorithm))
		r.DigestType = new(int(f.DigestType))
	case "HTTPS", "SVCB":
		f := rc.AsSVCB()
		if f.Target == "." {
			r.Value = "."
		} else {
			r.Value = hostToNative(dc, f.Target)
		}
		r.Priority = new(int(f.Priority))
		r.Params = models.Svcbv2ValueToString(f.Value)
	default:
		return nativeRecord{}, fmt.Errorf("nameocean: record type %q is not supported", rc.Type)
	}
	return r, nil
}
