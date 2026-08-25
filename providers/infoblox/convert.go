package infoblox

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/DNSControl/dnscontrol/v4/models"
)

// supportedTypes lists the Infoblox record types we fetch and manage.
var supportedTypes = []string{"a", "aaaa", "cname", "mx", "txt", "srv", "ns", "ptr", "caa"}

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
	TTL        uint32 `json:"ttl"`
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
func toRecordConfig(recType string, raw json.RawMessage, domain string, defaultTTL uint32) (*models.RecordConfig, error) {
	switch recType {
	case "a":
		return convertA(raw, domain, defaultTTL)
	case "aaaa":
		return convertAAAA(raw, domain, defaultTTL)
	case "cname":
		return convertCNAME(raw, domain, defaultTTL)
	case "mx":
		return convertMX(raw, domain, defaultTTL)
	case "txt":
		return convertTXT(raw, domain, defaultTTL)
	case "srv":
		return convertSRV(raw, domain, defaultTTL)
	case "ns":
		return convertNS(raw, domain, defaultTTL)
	case "ptr":
		return convertPTR(raw, domain, defaultTTL)
	case "caa":
		return convertCAA(raw, domain, defaultTTL)
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

func convertA(raw json.RawMessage, domain string, defaultTTL uint32) (*models.RecordConfig, error) {
	var r ibRecordA
	if err := json.Unmarshal(raw, &r); err != nil {
		return nil, fmt.Errorf("failed to parse A record: %w", err)
	}

	rc := &models.RecordConfig{
		Type:     "A",
		TTL:      effectiveTTL(r.UseTTL, r.TTL, defaultTTL),
		Original: r.Ref,
	}
	rc.SetLabelFromFQDN(r.Name, domain)
	if err := rc.PopulateFromString("A", r.IPv4Addr, domain); err != nil {
		return nil, fmt.Errorf("failed to populate A record: %w", err)
	}
	return rc, nil
}

func convertAAAA(raw json.RawMessage, domain string, defaultTTL uint32) (*models.RecordConfig, error) {
	var r ibRecordAAAA
	if err := json.Unmarshal(raw, &r); err != nil {
		return nil, fmt.Errorf("failed to parse AAAA record: %w", err)
	}

	rc := &models.RecordConfig{
		Type:     "AAAA",
		TTL:      effectiveTTL(r.UseTTL, r.TTL, defaultTTL),
		Original: r.Ref,
	}
	rc.SetLabelFromFQDN(r.Name, domain)
	if err := rc.PopulateFromString("AAAA", r.IPv6Addr, domain); err != nil {
		return nil, fmt.Errorf("failed to populate AAAA record: %w", err)
	}
	return rc, nil
}

func convertCNAME(raw json.RawMessage, domain string, defaultTTL uint32) (*models.RecordConfig, error) {
	var r ibRecordCNAME
	if err := json.Unmarshal(raw, &r); err != nil {
		return nil, fmt.Errorf("failed to parse CNAME record: %w", err)
	}

	rc := &models.RecordConfig{
		Type:     "CNAME",
		TTL:      effectiveTTL(r.UseTTL, r.TTL, defaultTTL),
		Original: r.Ref,
	}
	rc.SetLabelFromFQDN(r.Name, domain)
	if err := rc.PopulateFromString("CNAME", ensureTrailingDot(r.Canonical), domain); err != nil {
		return nil, fmt.Errorf("failed to populate CNAME record: %w", err)
	}
	return rc, nil
}

func convertMX(raw json.RawMessage, domain string, defaultTTL uint32) (*models.RecordConfig, error) {
	var r ibRecordMX
	if err := json.Unmarshal(raw, &r); err != nil {
		return nil, fmt.Errorf("failed to parse MX record: %w", err)
	}

	rc := &models.RecordConfig{
		Type:     "MX",
		TTL:      effectiveTTL(r.UseTTL, r.TTL, defaultTTL),
		Original: r.Ref,
	}
	rc.SetLabelFromFQDN(r.Name, domain)
	if err := rc.SetTargetMX(r.Preference, ensureTrailingDot(r.MailExchanger)); err != nil {
		return nil, fmt.Errorf("failed to populate MX record: %w", err)
	}
	return rc, nil
}

func convertTXT(raw json.RawMessage, domain string, defaultTTL uint32) (*models.RecordConfig, error) {
	var r ibRecordTXT
	if err := json.Unmarshal(raw, &r); err != nil {
		return nil, fmt.Errorf("failed to parse TXT record: %w", err)
	}

	rc := &models.RecordConfig{
		Type:     "TXT",
		TTL:      effectiveTTL(r.UseTTL, r.TTL, defaultTTL),
		Original: r.Ref,
	}
	rc.SetLabelFromFQDN(r.Name, domain)
	if err := rc.SetTargetTXT(r.Text); err != nil {
		return nil, fmt.Errorf("failed to populate TXT record: %w", err)
	}
	return rc, nil
}

func convertSRV(raw json.RawMessage, domain string, defaultTTL uint32) (*models.RecordConfig, error) {
	var r ibRecordSRV
	if err := json.Unmarshal(raw, &r); err != nil {
		return nil, fmt.Errorf("failed to parse SRV record: %w", err)
	}

	rc := &models.RecordConfig{
		Type:     "SRV",
		TTL:      effectiveTTL(r.UseTTL, r.TTL, defaultTTL),
		Original: r.Ref,
	}
	rc.SetLabelFromFQDN(r.Name, domain)
	if err := rc.SetTargetSRV(r.Priority, r.Weight, r.Port, ensureTrailingDot(r.Target)); err != nil {
		return nil, fmt.Errorf("failed to populate SRV record: %w", err)
	}
	return rc, nil
}

func convertNS(raw json.RawMessage, domain string, defaultTTL uint32) (*models.RecordConfig, error) {
	var r ibRecordNS
	if err := json.Unmarshal(raw, &r); err != nil {
		return nil, fmt.Errorf("failed to parse NS record: %w", err)
	}

	// Skip NS records at the zone apex — Infoblox manages these internally.
	if r.Name == domain || r.Name == domain+"." {
		return nil, nil
	}

	rc := &models.RecordConfig{
		Type:     "NS",
		TTL:      r.TTL,
		Original: r.Ref,
	}
	if rc.TTL == 0 {
		rc.TTL = defaultTTL
	}
	rc.SetLabelFromFQDN(r.Name, domain)
	if err := rc.PopulateFromString("NS", ensureTrailingDot(r.Nameserver), domain); err != nil {
		return nil, fmt.Errorf("failed to populate NS record: %w", err)
	}
	return rc, nil
}

func convertPTR(raw json.RawMessage, domain string, defaultTTL uint32) (*models.RecordConfig, error) {
	var r ibRecordPTR
	if err := json.Unmarshal(raw, &r); err != nil {
		return nil, fmt.Errorf("failed to parse PTR record: %w", err)
	}

	rc := &models.RecordConfig{
		Type:     "PTR",
		TTL:      effectiveTTL(r.UseTTL, r.TTL, defaultTTL),
		Original: r.Ref,
	}
	rc.SetLabelFromFQDN(r.Name, domain)
	if err := rc.PopulateFromString("PTR", ensureTrailingDot(r.PtrDName), domain); err != nil {
		return nil, fmt.Errorf("failed to populate PTR record: %w", err)
	}
	return rc, nil
}

func convertCAA(raw json.RawMessage, domain string, defaultTTL uint32) (*models.RecordConfig, error) {
	var r ibRecordCAA
	if err := json.Unmarshal(raw, &r); err != nil {
		return nil, fmt.Errorf("failed to parse CAA record: %w", err)
	}

	rc := &models.RecordConfig{
		Type:     "CAA",
		TTL:      effectiveTTL(r.UseTTL, r.TTL, defaultTTL),
		Original: r.Ref,
	}
	rc.SetLabelFromFQDN(r.Name, domain)
	if err := rc.SetTargetCAA(r.CaFlag, r.CaTag, r.CaValue); err != nil {
		return nil, fmt.Errorf("failed to populate CAA record: %w", err)
	}
	return rc, nil
}

// buildRecordBody constructs the JSON body for a create or update API call
// from a DNSControl RecordConfig.
func buildRecordBody(rc *models.RecordConfig, domain, view string, includeView bool) (string, map[string]any, error) {
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
		body["ipv4addr"] = rc.GetTargetField()
	case "AAAA":
		recType = "aaaa"
		body["ipv6addr"] = rc.GetTargetField()
	case "CNAME":
		recType = "cname"
		body["canonical"] = strings.TrimSuffix(rc.GetTargetField(), ".")
	case "MX":
		recType = "mx"
		body["mail_exchanger"] = strings.TrimSuffix(rc.GetTargetField(), ".")
		body["preference"] = rc.MxPreference
	case "TXT":
		recType = "txt"
		body["text"] = rc.GetTargetTXTJoined()
	case "SRV":
		recType = "srv"
		body["target"] = strings.TrimSuffix(rc.GetTargetField(), ".")
		body["priority"] = rc.SrvPriority
		body["weight"] = rc.SrvWeight
		body["port"] = rc.SrvPort
	case "NS":
		recType = "ns"
		body["nameserver"] = strings.TrimSuffix(rc.GetTargetField(), ".")
	case "PTR":
		recType = "ptr"
		body["ptrdname"] = strings.TrimSuffix(rc.GetTargetField(), ".")
	case "CAA":
		recType = "caa"
		body["ca_flag"] = rc.CaaFlag
		body["ca_tag"] = rc.CaaTag
		body["ca_value"] = rc.GetTargetField()
	default:
		return "", nil, fmt.Errorf("unsupported record type for Infoblox: %s", rc.Type)
	}

	return recType, body, nil
}
