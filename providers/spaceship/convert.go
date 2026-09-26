package spaceship

import (
	"fmt"
	"strconv"
	"strings"

	dnsv2 "codeberg.org/miekg/dns"
	"github.com/DNSControl/dnscontrol/v5/models"
	"github.com/DNSControl/dnscontrol/v5/pkg/nrc"
	"github.com/DNSControl/dnscontrol/v5/pkg/privatetypes"
	"github.com/namecheap/go-spaceship-sdk/client"
)

func toRC(dc *models.DomainConfig, rec client.DNSRecord) (*models.RecordConfig, error) {
	ttl := clampTTL(uint32(rec.TTL))
	label := dc.LabelFromShort(rec.Name)

	var rc *models.RecordConfig
	var err error
	switch rtype := strings.ToUpper(rec.Type); rtype {
	case "A":
		rc, err = dc.NewRecordConfig(label, ttl, dnsv2.TypeA, rec.Address)
	case "AAAA":
		rc, err = dc.NewRecordConfig(label, ttl, dnsv2.TypeAAAA, rec.Address)
	case "CNAME":
		rc, err = dc.NewRecordConfig(label, ttl, dnsv2.TypeCNAME, rec.CName,
			nrc.Flags{TargetIsFqdnNoDot: true})
	case "ALIAS":
		rc, err = dc.NewRecordConfig(label, ttl, privatetypes.TypeALIAS, ensureDot(rec.AliasName))
	case "MX":
		rc, err = dc.NewRecordConfig(label, ttl, dnsv2.TypeMX, derefInt(rec.Preference), ensureDot(rec.Exchange))
	case "TXT":
		rc, err = dc.NewRecordConfig(label, ttl, dnsv2.TypeTXT, rec.Value)
	case "NS":
		rc, err = dc.NewRecordConfig(label, ttl, dnsv2.TypeNS, ensureDot(rec.Nameserver))
	case "PTR":
		rc, err = dc.NewRecordConfig(label, ttl, dnsv2.TypePTR, ensureDot(rec.Pointer))
	case "CAA":
		rc, err = dc.NewRecordConfig(label, ttl, dnsv2.TypeCAA, derefInt(rec.Flag), rec.Tag, rec.Value)
	case "SRV":
		label = joinPrefixedLabel([]string{rec.Service, rec.Protocol}, rec.Name)
		port, portErr := srvPort(rec.Port)
		if portErr != nil {
			return nil, fmt.Errorf("spaceship.toRC: %w", portErr)
		}
		rc, err = dc.NewRecordConfig(label, ttl, dnsv2.TypeSRV, derefInt(rec.Priority), derefInt(rec.Weight), port, ensureDot(rec.Target))
	case "HTTPS", "SVCB":
		label = joinPrefixedLabel([]string{portString(rec.Port), rec.Scheme}, rec.Name)
		rc, err = dc.NewRecordConfig(dc.LabelFromShort(label), ttl, rtype, derefInt(rec.SvcPriority), httpsTarget(rec.TargetName), rec.SvcParams)
	case "TLSA":
		label = joinPrefixedLabel([]string{portString(rec.Port), rec.Protocol}, rec.Name)
		rc, err = dc.NewRecordConfig(dc.LabelFromShort(label), ttl, dnsv2.TypeTLSA, derefInt(rec.Usage), derefInt(rec.Selector), derefInt(rec.Matching), rec.AssociationData)
	default:
		return nil, fmt.Errorf("spaceship.toRC: unsupported record type %q", rec.Type)
	}
	if err != nil {
		return nil, err
	}
	rc.Original = rec
	return rc, nil
}

func toNative(rc *models.RecordConfig) (client.DNSRecord, error) {
	rec := client.DNSRecord{
		Type: rc.Type,
		Name: nativeName(rc.GetLabel()),
		TTL:  int(clampTTL(rc.TTL)),
	}

	switch rc.TypeNum {
	case dnsv2.TypeA:
		rec.Address = rc.AsA().Addr.String()
	case dnsv2.TypeAAAA:
		rec.Address = rc.AsAAAA().Addr.String()
	case dnsv2.TypeCNAME:
		rec.CName = trimDot(rc.AsCNAME().Target)
	case privatetypes.TypeALIAS:
		rec.AliasName = trimDot(rc.AsALIAS().Target)
	case dnsv2.TypeMX:
		f := rc.AsMX()
		rec.Preference = new(int(f.Preference))
		rec.Exchange = hostTargetOut(f.Mx)
	case dnsv2.TypeTXT:
		rec.Value = rc.GetTargetTXTJoined()
	case dnsv2.TypeNS:
		rec.Nameserver = trimDot(rc.AsNS().Ns)
	case dnsv2.TypePTR:
		rec.Pointer = trimDot(rc.AsPTR().Ptr)
	case dnsv2.TypeCAA:
		f := rc.AsCAA()
		rec.Flag = new(int(f.Flag))
		rec.Tag = f.Tag
		rec.Value = f.Value
	case dnsv2.TypeSRV:
		f := rc.AsSRV()
		prefix, name, err := splitPrefixedLabel(rc.GetLabel(), 2)
		if err != nil {
			return rec, fmt.Errorf("spaceship.toNative SRV %q: %w", rc.GetLabel(), err)
		}
		rec.Service = prefix[0]
		rec.Protocol = prefix[1]
		rec.Name = nativeName(name)
		rec.Priority = new(int(f.Priority))
		rec.Weight = new(int(f.Weight))
		rec.Port = client.NewIntPortValue(int(f.Port))
		rec.Target = hostTargetOut(f.Target)
	case dnsv2.TypeHTTPS, dnsv2.TypeSVCB:
		f := rc.AsSVCB()
		prefix, name, ok := optionalPrefixedLabel(rc.GetLabel(), 2)
		if ok {
			rec.Port = client.NewStringPortValue(prefix[0])
			rec.Scheme = prefix[1]
			rec.Name = nativeName(name)
		}
		rec.SvcPriority = new(int(f.Priority))
		rec.TargetName = httpsTargetOut(f.Target)
		rec.SvcParams = models.Svcbv2ValueToString(f.Value)
	case dnsv2.TypeTLSA:
		f := rc.AsTLSA()
		prefix, name, err := splitPrefixedLabel(rc.GetLabel(), 2)
		if err != nil {
			return rec, fmt.Errorf("spaceship.toNative TLSA %q: %w", rc.GetLabel(), err)
		}
		rec.Port = client.NewStringPortValue(prefix[0])
		rec.Protocol = prefix[1]
		rec.Name = nativeName(name)
		rec.Usage = new(int(f.Usage))
		rec.Selector = new(int(f.Selector))
		rec.Matching = new(int(f.MatchingType))
		rec.AssociationData = f.Certificate
	default:
		return rec, fmt.Errorf("spaceship.toNative: unsupported record type %q", rc.Type)
	}

	return rec, nil
}

func nativeName(label string) string {
	if label == "" {
		return "@"
	}
	return label
}

func joinPrefixedLabel(parts []string, name string) string {
	var labelParts []string
	for _, part := range parts {
		if part != "" {
			labelParts = append(labelParts, part)
		}
	}
	if name != "" && name != "@" {
		labelParts = append(labelParts, name)
	}
	if len(labelParts) == 0 {
		return "@"
	}
	return strings.Join(labelParts, ".")
}

func splitPrefixedLabel(label string, n int) (prefix []string, rest string, err error) {
	if label == "" || label == "@" {
		return nil, "@", fmt.Errorf("expected %d underscore-prefixed labels", n)
	}
	parts := strings.Split(label, ".")
	if len(parts) < n {
		return nil, "", fmt.Errorf("expected %d underscore-prefixed labels, got %q", n, label)
	}
	prefix = make([]string, n)
	for i := range n {
		if !strings.HasPrefix(parts[i], "_") {
			return nil, "", fmt.Errorf("expected %d underscore-prefixed labels, got %q", n, label)
		}
		prefix[i] = parts[i]
	}
	if len(parts) == n {
		return prefix, "@", nil
	}
	return prefix, strings.Join(parts[n:], "."), nil
}

func optionalPrefixedLabel(label string, n int) (prefix []string, rest string, ok bool) {
	prefix, rest, err := splitPrefixedLabel(label, n)
	if err != nil {
		return nil, label, false
	}
	return prefix, rest, true
}

func ensureDot(s string) string {
	if s == "" || s == "." || strings.HasSuffix(s, ".") {
		return s
	}
	return s + "."
}

func trimDot(s string) string {
	return strings.TrimSuffix(s, ".")
}

// hostTargetOut keeps a bare "." so null MX (RFC 7505) and null SRV are not
// serialized as an empty field. strings.TrimSuffix(".", ".") is "" and the
// SDK omits empty exchange/target, which the API then 422s as required.
func hostTargetOut(s string) string {
	if s == "." {
		return "."
	}
	return trimDot(s)
}

func httpsTarget(s string) string {
	if s == "" {
		return "."
	}
	if s == "." {
		return s
	}
	return ensureDot(s)
}

func httpsTargetOut(s string) string {
	if s == "" || s == "." {
		return "."
	}
	return trimDot(s)
}

func derefInt(p *int) int {
	if p == nil {
		return 0
	}
	return *p
}

func portString(p *client.PortValue) string {
	if p == nil {
		return ""
	}
	if p.String != nil {
		return *p.String
	}
	if p.Int != nil {
		return "_" + strconv.Itoa(*p.Int)
	}
	return ""
}

func srvPort(p *client.PortValue) (uint16, error) {
	if p == nil {
		return 0, nil
	}
	if p.Int != nil {
		if *p.Int < 0 || *p.Int > 65535 {
			return 0, fmt.Errorf("SRV port %d is outside the valid range", *p.Int)
		}
		return uint16(*p.Int), nil
	}
	if p.String != nil {
		raw := strings.TrimPrefix(*p.String, "_")
		n, err := strconv.ParseUint(raw, 10, 16)
		if err != nil {
			return 0, fmt.Errorf("invalid SRV port %q: %w", *p.String, err)
		}
		return uint16(n), nil
	}
	return 0, nil
}
