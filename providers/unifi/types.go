package unifi

import (
	"fmt"
	"strings"

	"github.com/DNSControl/dnscontrol/v5/models"
)

// legacyDNSRecord represents a DNS record in the OLD UniFi API format (v2/api/site/{site}/static-dns).
// This API is available in UniFi Network 8.2+.
type legacyDNSRecord struct {
	ID         string `json:"_id,omitempty"`
	Enabled    bool   `json:"enabled"`
	Key        string `json:"key"`         // FQDN (e.g., "test.example.com")
	RecordType string `json:"record_type"` // A, AAAA, CNAME, MX, TXT, SRV, NS
	Value      string `json:"value"`       // Record value
	TTL        int    `json:"ttl"`         // 0 = default
	Port       int    `json:"port"`        // SRV port
	Priority   int    `json:"priority"`    // MX/SRV priority
	Weight     int    `json:"weight"`      // SRV weight
}

// New API record types (Network 10.1+).
const (
	NewAPITypeA     = "A_RECORD"
	NewAPITypeAAAA  = "AAAA_RECORD"
	NewAPITypeCNAME = "CNAME_RECORD"
	NewAPITypeMX    = "MX_RECORD"
	NewAPITypeTXT   = "TXT_RECORD"
	NewAPITypeSRV   = "SRV_RECORD"
)

// dnsPolicyMetadata represents metadata for a DNS policy record.
type dnsPolicyMetadata struct {
	Origin string `json:"origin,omitempty"` // "USER_DEFINED"
}

// dnsPolicyRecord represents a DNS record in the NEW UniFi API format (integration/v1/sites/{siteId}/dns/policies).
// This API is available in UniFi Network 10.1+.
// The record is polymorphic - different fields are used depending on the type.
type dnsPolicyRecord struct {
	Type     string             `json:"type"`               // A_RECORD, AAAA_RECORD, CNAME_RECORD, MX_RECORD, TXT_RECORD, SRV_RECORD
	ID       string             `json:"id,omitempty"`       // UUID
	Enabled  bool               `json:"enabled"`            // Whether the record is enabled
	Metadata *dnsPolicyMetadata `json:"metadata,omitempty"` // Metadata (origin, read-only in API responses)
	Domain   string             `json:"domain"`             // FQDN (e.g., "test.example.com")

	// TTL in seconds. Only A/AAAA/CNAME accept it in the new API; for
	// MX/TXT/SRV the property is rejected, so it is omitted when zero.
	TTLSeconds int `json:"ttlSeconds,omitempty"`

	// Type-specific fields
	IPv4Address      string `json:"ipv4Address,omitempty"`      // A record
	IPv6Address      string `json:"ipv6Address,omitempty"`      // AAAA record
	TargetDomain     string `json:"targetDomain,omitempty"`     // CNAME record
	MailServerDomain string `json:"mailServerDomain,omitempty"` // MX record
	Text             string `json:"text,omitempty"`             // TXT record
	ServerDomain     string `json:"serverDomain,omitempty"`     // SRV record

	// MX/SRV specific
	Priority int `json:"priority,omitempty"` // MX/SRV priority

	// SRV specific. The new API splits the "_service._proto.name" label into
	// separate fields; service/protocol keep their leading underscore.
	Service  string `json:"service,omitempty"`  // SRV service, e.g. "_sip"
	Protocol string `json:"protocol,omitempty"` // SRV protocol, e.g. "_tcp"
	Weight   int    `json:"weight,omitempty"`   // SRV weight
	Port     int    `json:"port,omitempty"`     // SRV port
}

// dnsPolicyResponse wraps the response from the new API list endpoint.
type dnsPolicyResponse struct {
	Data []dnsPolicyRecord `json:"data"`
}

// siteInfo represents a site from the new API.
type siteInfo struct {
	ID                string `json:"id"`
	InternalReference string `json:"internalReference"` // This is "default", "site2", etc.
	Name              string `json:"name"`              // This is "Default", "Site 2", etc.
}

// legacyToRecord converts a UniFi legacy API record to a dnscontrol RecordConfig.
func legacyToRecord(dc *models.DomainConfig, r *legacyDNSRecord) (*models.RecordConfig, error) {
	// Set TTL (UniFi uses 0 for default, we map to 300)
	ttl := uint32(300)
	if r.TTL > 0 {
		ttl = uint32(r.TTL)
	}
	label := dc.LabelFromFQDNNoDot(r.Key)

	var rc *models.RecordConfig
	var err error
	switch r.RecordType {
	case "A", "AAAA":
		rc, err = dc.NewRecordConfig(label, ttl, r.RecordType, r.Value)

	case "CNAME", "NS":
		target := r.Value
		if !strings.HasSuffix(target, ".") {
			target = target + "."
		}
		rc, err = dc.NewRecordConfig(label, ttl, r.RecordType, target)

	case "MX":
		target := r.Value
		if !strings.HasSuffix(target, ".") {
			target = target + "."
		}
		rc, err = dc.NewRecordConfig(label, ttl, r.RecordType, r.Priority, target)

	case "TXT":
		rc, err = dc.NewRecordConfig(label, ttl, r.RecordType, r.Value)

	case "SRV":
		target := r.Value
		if !strings.HasSuffix(target, ".") {
			target = target + "."
		}
		rc, err = dc.NewRecordConfig(label, ttl, r.RecordType, r.Priority, r.Weight, r.Port, target)

	default:
		err = fmt.Errorf("unsupported record type: %s", r.RecordType)
	}
	if err == nil {
		rc.Original = r
	}
	return rc, err
}

// recordToLegacyMap converts a dnscontrol RecordConfig to a map for API requests.
// UniFi is strict about which fields can be set for each record type.
func recordToLegacyMap(rc *models.RecordConfig) (map[string]any, error) {
	m := map[string]any{
		"enabled":     true,
		"key":         rc.NameFQDN,
		"record_type": rc.Type,
		"value":       "",
	}

	switch rc.Type {
	case "A":
		m["value"] = rc.GetTargetField()
		// A records can have TTL
		if rc.TTL > 0 {
			m["ttl"] = int(rc.TTL)
		}

	case "AAAA":
		m["value"] = rc.GetTargetField()
		// AAAA records can have TTL
		if rc.TTL > 0 {
			m["ttl"] = int(rc.TTL)
		}

	case "CNAME":
		m["value"] = strings.TrimSuffix(rc.GetTargetField(), ".")
		// CNAME records can have TTL
		if rc.TTL > 0 {
			m["ttl"] = int(rc.TTL)
		}

	case "NS":
		m["value"] = strings.TrimSuffix(rc.GetTargetField(), ".")
		// NS records can have TTL
		if rc.TTL > 0 {
			m["ttl"] = int(rc.TTL)
		}

	case "MX":
		// MX records: only enabled, key, record_type, value, priority allowed
		m["value"] = strings.TrimSuffix(rc.GetTargetField(), ".")
		m["priority"] = int(rc.MxPreference)

	case "TXT":
		// TXT records: only enabled, key, record_type, value allowed
		m["value"] = rc.GetTargetTXTJoined()

	case "SRV":
		// SRV records: enabled, key, record_type, value, priority, weight, port allowed
		m["value"] = strings.TrimSuffix(rc.GetTargetField(), ".")
		m["priority"] = int(rc.SrvPriority)
		m["weight"] = int(rc.SrvWeight)
		m["port"] = int(rc.SrvPort)

	default:
		return nil, fmt.Errorf("unsupported record type: %s", rc.Type)
	}

	return m, nil
}

// getRecordID extracts the UniFi record ID from the Original field.
func getRecordID(rc *models.RecordConfig) string {
	if rc.Original == nil {
		return ""
	}
	if r, ok := rc.Original.(*legacyDNSRecord); ok {
		return r.ID
	}
	if r, ok := rc.Original.(*dnsPolicyRecord); ok {
		return r.ID
	}
	return ""
}

// newToRecord converts a UniFi new API record to a dnscontrol RecordConfig.
func newToRecord(dc *models.DomainConfig, r *dnsPolicyRecord) (*models.RecordConfig, error) {
	// Map new API type to standard type
	var rtype string
	switch r.Type {
	case NewAPITypeA:
		rtype = "A"
	case NewAPITypeAAAA:
		rtype = "AAAA"
	case NewAPITypeCNAME:
		rtype = "CNAME"
	case NewAPITypeMX:
		rtype = "MX"
	case NewAPITypeTXT:
		rtype = "TXT"
	case NewAPITypeSRV:
		rtype = "SRV"
	default:
		return nil, fmt.Errorf("unsupported new API record type: %s", r.Type)
	}

	// Set TTL (UniFi uses 0 for default, we map to 300)
	ttl := uint32(300)
	if r.TTLSeconds > 0 {
		ttl = uint32(r.TTLSeconds)
	}

	// Set label from FQDN. For SRV the new API splits the label, so rebuild
	// "_service._proto.name" from the separate fields.
	fqdn := r.Domain
	if r.Type == NewAPITypeSRV && r.Service != "" && r.Protocol != "" {
		fqdn = r.Service + "." + r.Protocol + "." + r.Domain
	}
	label := dc.LabelFromFQDNNoDot(fqdn)

	var rc *models.RecordConfig
	var err error
	switch r.Type {
	case NewAPITypeA:
		rc, err = dc.NewRecordConfig(label, ttl, rtype, r.IPv4Address)

	case NewAPITypeAAAA:
		rc, err = dc.NewRecordConfig(label, ttl, rtype, r.IPv6Address)

	case NewAPITypeCNAME:
		target := r.TargetDomain
		if !strings.HasSuffix(target, ".") {
			target = target + "."
		}
		rc, err = dc.NewRecordConfig(label, ttl, rtype, target)

	case NewAPITypeMX:
		target := r.MailServerDomain
		if !strings.HasSuffix(target, ".") {
			target = target + "."
		}
		rc, err = dc.NewRecordConfig(label, ttl, rtype, r.Priority, target)

	case NewAPITypeTXT:
		rc, err = dc.NewRecordConfig(label, ttl, rtype, r.Text)

	case NewAPITypeSRV:
		target := r.ServerDomain
		if !strings.HasSuffix(target, ".") {
			target = target + "."
		}
		rc, err = dc.NewRecordConfig(label, ttl, rtype, r.Priority, r.Weight, r.Port, target)
	}
	if err == nil {
		rc.Original = r
	}
	return rc, err
}

// recordToNew converts a dnscontrol RecordConfig to a UniFi new API record.
func recordToNew(rc *models.RecordConfig) (*dnsPolicyRecord, error) {
	r := &dnsPolicyRecord{
		Enabled: true,
		Domain:  rc.NameFQDN,
	}

	// The new API only accepts ttlSeconds for A/AAAA/CNAME; sending it for
	// MX/TXT/SRV is rejected with "Unknown request body property '$.ttlSeconds'".
	switch rc.Type {
	case "A", "AAAA", "CNAME":
		if rc.TTL > 0 {
			r.TTLSeconds = int(rc.TTL)
		} else {
			r.TTLSeconds = 300
		}
	}

	switch rc.Type {
	case "A":
		r.Type = NewAPITypeA
		r.IPv4Address = rc.GetTargetField()

	case "AAAA":
		r.Type = NewAPITypeAAAA
		r.IPv6Address = rc.GetTargetField()

	case "CNAME":
		r.Type = NewAPITypeCNAME
		r.TargetDomain = strings.TrimSuffix(rc.GetTargetField(), ".")

	case "MX":
		r.Type = NewAPITypeMX
		r.Priority = int(rc.MxPreference)
		r.MailServerDomain = strings.TrimSuffix(rc.GetTargetField(), ".")

	case "TXT":
		r.Type = NewAPITypeTXT
		r.Text = rc.GetTargetTXTJoined()

	case "SRV":
		r.Type = NewAPITypeSRV
		// The new API wants the "_service._proto.name" label split apart, e.g.
		// "_sip._tcp.example.com" => service="_sip", protocol="_tcp",
		// domain="example.com".
		labels := strings.SplitN(rc.NameFQDN, ".", 3)
		if len(labels) < 3 {
			return nil, fmt.Errorf("SRV record %q is not in _service._proto.name form", rc.NameFQDN)
		}
		r.Service = labels[0]
		r.Protocol = labels[1]
		r.Domain = labels[2]
		r.Priority = int(rc.SrvPriority)
		r.Weight = int(rc.SrvWeight)
		r.Port = int(rc.SrvPort)
		r.ServerDomain = strings.TrimSuffix(rc.GetTargetField(), ".")

	default:
		return nil, fmt.Errorf("unsupported record type for new API: %s", rc.Type)
	}

	return r, nil
}
