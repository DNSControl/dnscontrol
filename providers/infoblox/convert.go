package infoblox

import (
	"encoding/json"
	"fmt"
	"strings"

	dnsv2 "codeberg.org/miekg/dns"
	"github.com/DNSControl/dnscontrol/v5/models"
)

// supportedTypes lists the Infoblox record types we fetch and manage.
// NS is excluded because Infoblox requires an 'addresses' field (nameserver IPs)
// when creating NS delegation records, which dnscontrol does not provide.
var supportedTypes = []string{"a", "aaaa", "cname", "mx", "txt", "srv", "ptr", "caa"}

// infobloxRecord is the common set of fields returned by all Infoblox record types.
// Type-specific fields are extracted in the per-type converters.
type infobloxRecord struct {
	Ref    string `json:"_ref"`
	Name   string `json:"name"`
	TTL    uint32 `json:"ttl"`
	UseTTL bool   `json:"use_ttl"`
	View   string `json:"view"`
}

// Type-specific Infoblox record structs.

type ibRecordA struct {
	infobloxRecord
	IPv4Addr string `json:"ipv4addr"`
}

type ibRecordAAAA struct {
	infobloxRecord
	IPv6Addr string `json:"ipv6addr"`
}

type ibRecordCNAME struct {
	infobloxRecord
	Canonical string `json:"canonical"`
}

type ibRecordMX struct {
	infobloxRecord
	MailExchanger string `json:"mail_exchanger"`
	Preference    uint16 `json:"preference"`
}

type ibRecordTXT struct {
	infobloxRecord
	Text string `json:"text"`
}

type ibRecordSRV struct {
	infobloxRecord
	Target   string `json:"target"`
	Priority uint16 `json:"priority"`
	Weight   uint16 `json:"weight"`
	Port     uint16 `json:"port"`
}

type ibRecordNS struct {
	Ref        string `json:"_ref"`
	Name       string `json:"name"`
	View       string `json:"view"`
	Nameserver string `json:"nameserver"`
}

type ibRecordPTR struct {
	infobloxRecord
	PtrDName string `json:"ptrdname"`
}

type ibRecordCAA struct {
	infobloxRecord
	CaFlag  uint8  `json:"ca_flag"`
	CaTag   string `json:"ca_tag"`
	CaValue string `json:"ca_value"`
}

// toRecordConfig converts a raw JSON Infoblox record of the given type to a DNSControl RecordConfig.
func toRecordConfig(recType string, raw json.RawMessage, dc *models.DomainConfig, defaultTTL uint32) (*models.RecordConfig, error) {
	switch recType {
	case "a":
		return convertA(raw, dc, defaultTTL)
	case "aaaa":
		return convertAAAA(raw, dc, defaultTTL)
	case "cname":
		return convertCNAME(raw, dc, defaultTTL)
	case "mx":
		return convertMX(raw, dc, defaultTTL)
	case "txt":
		return convertTXT(raw, dc, defaultTTL)
	case "srv":
		return convertSRV(raw, dc, defaultTTL)
	case "ns":
		return convertNS(raw, dc, defaultTTL)
	case "ptr":
		return convertPTR(raw, dc, defaultTTL)
	case "caa":
		return convertCAA(raw, dc, defaultTTL)
	default:
		return nil, fmt.Errorf("unsupported Infoblox record type: %s", recType)
	}
}

func effectiveTTL(useTTL bool, ttl, defaultTTL uint32) uint32 {
	if useTTL {
		return ttl
	}
	return defaultTTL
}

// ensureTrailingDot adds a trailing dot if not already present.
// Infoblox sometimes returns FQDNs without trailing dots for target fields.
func ensureTrailingDot(s string) string {
	if s == "" || s[len(s)-1] == '.' {
		return s
	}
	return s + "."
}

func convertA(raw json.RawMessage, dc *models.DomainConfig, defaultTTL uint32) (*models.RecordConfig, error) {
	var r ibRecordA
	if err := json.Unmarshal(raw, &r); err != nil {
		return nil, fmt.Errorf("failed to parse A record: %w", err)
	}

	rc, err := dc.NewRecordConfig(dc.LabelFromFQDNNoDot(r.Name), effectiveTTL(r.UseTTL, r.TTL, defaultTTL),
		dnsv2.TypeA, r.IPv4Addr)
	if err != nil {
		return nil, err
	}
	rc.Original = r.Ref

	return rc, nil
}

func convertAAAA(raw json.RawMessage, dc *models.DomainConfig, defaultTTL uint32) (*models.RecordConfig, error) {
	var r ibRecordAAAA
	if err := json.Unmarshal(raw, &r); err != nil {
		return nil, fmt.Errorf("failed to parse AAAA record: %w", err)
	}

	rc, err := dc.NewRecordConfig(dc.LabelFromFQDNNoDot(r.Name), effectiveTTL(r.UseTTL, r.TTL, defaultTTL),
		dnsv2.TypeAAAA, r.IPv6Addr)
	if err != nil {
		return nil, err
	}
	rc.Original = r.Ref

	return rc, nil
}

func convertCNAME(raw json.RawMessage, dc *models.DomainConfig, defaultTTL uint32) (*models.RecordConfig, error) {
	var r ibRecordCNAME
	if err := json.Unmarshal(raw, &r); err != nil {
		return nil, fmt.Errorf("failed to parse CNAME record: %w", err)
	}

	rc, err := dc.NewRecordConfig(dc.LabelFromFQDNNoDot(r.Name), effectiveTTL(r.UseTTL, r.TTL, defaultTTL),
		dnsv2.TypeCNAME, r.Canonical+".")
	if err != nil {
		return nil, err
	}
	rc.Original = r.Ref

	return rc, nil
}

func convertMX(raw json.RawMessage, dc *models.DomainConfig, defaultTTL uint32) (*models.RecordConfig, error) {
	var r ibRecordMX
	if err := json.Unmarshal(raw, &r); err != nil {
		return nil, fmt.Errorf("failed to parse MX record: %w", err)
	}

	rc, err := dc.NewRecordConfig(dc.LabelFromFQDNNoDot(r.Name), effectiveTTL(r.UseTTL, r.TTL, defaultTTL),
		dnsv2.TypeMX, r.Preference, ensureTrailingDot(r.MailExchanger))
	if err != nil {
		return nil, err
	}
	rc.Original = r.Ref

	return rc, nil
}

func convertTXT(raw json.RawMessage, dc *models.DomainConfig, defaultTTL uint32) (*models.RecordConfig, error) {
	var r ibRecordTXT
	if err := json.Unmarshal(raw, &r); err != nil {
		return nil, fmt.Errorf("failed to parse TXT record: %w", err)
	}

	rc, err := dc.NewRecordConfig(dc.LabelFromFQDNNoDot(r.Name), effectiveTTL(r.UseTTL, r.TTL, defaultTTL),
		dnsv2.TypeTXT, r.Text)
	if err != nil {
		return nil, err
	}
	rc.Original = r.Ref

	return rc, nil
}

func convertSRV(raw json.RawMessage, dc *models.DomainConfig, defaultTTL uint32) (*models.RecordConfig, error) {
	var r ibRecordSRV
	if err := json.Unmarshal(raw, &r); err != nil {
		return nil, fmt.Errorf("failed to parse SRV record: %w", err)
	}

	rc, err := dc.NewRecordConfig(dc.LabelFromFQDNNoDot(r.Name), effectiveTTL(r.UseTTL, r.TTL, defaultTTL),
		dnsv2.TypeSRV, r.Priority, r.Weight, r.Port, ensureTrailingDot(r.Target))
	if err != nil {
		return nil, err
	}
	rc.Original = r.Ref

	return rc, nil
}

func convertNS(raw json.RawMessage, dc *models.DomainConfig, defaultTTL uint32) (*models.RecordConfig, error) {
	var r ibRecordNS
	if err := json.Unmarshal(raw, &r); err != nil {
		return nil, fmt.Errorf("failed to parse NS record: %w", err)
	}

	// Skip NS records at the zone apex — Infoblox manages these internally.
	if r.Name == dc.Name || r.Name == dc.Name+"." {
		return nil, nil
	}

	rc, err := dc.NewRecordConfig(dc.LabelFromFQDNNoDot(r.Name), defaultTTL,
		dnsv2.TypeNS, ensureTrailingDot(r.Nameserver))
	if err != nil {
		return nil, err
	}
	rc.Original = r.Ref

	return rc, nil
}

func convertPTR(raw json.RawMessage, dc *models.DomainConfig, defaultTTL uint32) (*models.RecordConfig, error) {
	var r ibRecordPTR
	if err := json.Unmarshal(raw, &r); err != nil {
		return nil, fmt.Errorf("failed to parse PTR record: %w", err)
	}

	rc, err := dc.NewRecordConfig(dc.LabelFromFQDNNoDot(r.Name), effectiveTTL(r.UseTTL, r.TTL, defaultTTL),
		dnsv2.TypePTR, ensureTrailingDot(r.PtrDName))
	if err != nil {
		return nil, err
	}
	rc.Original = r.Ref

	return rc, nil
}

func convertCAA(raw json.RawMessage, dc *models.DomainConfig, defaultTTL uint32) (*models.RecordConfig, error) {
	var r ibRecordCAA
	if err := json.Unmarshal(raw, &r); err != nil {
		return nil, fmt.Errorf("failed to parse CAA record: %w", err)
	}

	rc, err := dc.NewRecordConfig(dc.LabelFromFQDNNoDot(r.Name), effectiveTTL(r.UseTTL, r.TTL, defaultTTL),
		dnsv2.TypeCAA, r.CaFlag, r.CaTag, r.CaValue)
	if err != nil {
		return nil, err
	}
	rc.Original = r.Ref

	return rc, nil
}

// buildRecordBody constructs the JSON body for a create or update API call
// from a DNSControl RecordConfig.
func buildRecordBody(rc *models.RecordConfig, view string, includeView bool) (string, map[string]any, error) {
	fqdn := rc.GetLabelFQDN()
	ttl := rc.TTL

	body := map[string]any{
		"name": fqdn,
	}
	if includeView {
		body["view"] = view
	}
	if ttl > 0 {
		body["use_ttl"] = true
		body["ttl"] = ttl
	}

	var recType string

	switch rc.Type {
	case "A":
		recType = "a"
		body["ipv4addr"] = rc.AsA().Addr.String()
	case "AAAA":
		recType = "aaaa"
		body["ipv6addr"] = rc.AsAAAA().Addr.String()
	case "CNAME":
		recType = "cname"
		body["canonical"] = strings.TrimSuffix(rc.AsCNAME().Target, ".")
	case "MX":
		recType = "mx"
		rd := rc.AsMX()
		body["mail_exchanger"] = strings.TrimSuffix(rd.Mx, ".")
		body["preference"] = rd.Preference
	case "TXT":
		recType = "txt"
		body["text"] = rc.GetTargetTXTJoined()
	case "SRV":
		recType = "srv"
		rd := rc.AsSRV()
		body["target"] = strings.TrimSuffix(rd.Target, ".")
		body["priority"] = rd.Priority
		body["weight"] = rd.Weight
		body["port"] = rd.Port
	case "NS":
		recType = "ns"
		body["nameserver"] = strings.TrimSuffix(rc.AsNS().Ns, ".")
		// NS records in Infoblox don't support ttl/use_ttl fields.
		delete(body, "use_ttl")
		delete(body, "ttl")
	case "PTR":
		recType = "ptr"
		body["ptrdname"] = strings.TrimSuffix(rc.AsPTR().Ptr, ".")
	case "CAA":
		recType = "caa"
		rd := rc.AsCAA()
		body["ca_flag"] = rd.Flag
		body["ca_tag"] = rd.Tag
		body["ca_value"] = rd.Value
	default:
		return "", nil, fmt.Errorf("unsupported record type for Infoblox: %s", rc.Type)
	}

	return recType, body, nil
}
