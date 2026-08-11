package hostingde

import (
	"encoding/json"
	"fmt"
	"log"
	"net" // Needed for communicating with provider API.
	"strings"

	"github.com/DNSControl/dnscontrol/v4/models"
	"github.com/pkg/errors"
)

var errZoneNotFound = errors.Errorf("zone not found")

type request struct {
	AuthToken      string  `json:"authToken"`
	OwnerAccountID string  `json:"ownerAccountId,omitempty"`
	Filter         *filter `json:"filter,omitempty"`
	Limit          uint    `json:"limit,omitempty"`
	Page           uint    `json:"page,omitempty"`

	// Update Zone
	ZoneConfig      *zoneConfig `json:"zoneConfig,omitempty"`
	RecordsToAdd    []*record   `json:"recordsToAdd,omitempty"`
	RecordsToModify []*record   `json:"recordsToModify,omitempty"`
	RecordsToDelete []*record   `json:"recordsToDelete,omitempty"`

	// Create Zone
	Records []*record `json:"records,omitempty"`

	DomainName string        `json:"domainName,omitempty"`
	Add        []dnsSecEntry `json:"add,omitempty"`
	Remove     []dnsSecEntry `json:"remove,omitempty"`

	// Domain
	Domain *domainConfig `json:"domain"`

	DNSSECOptions *dnsSecOptions `json:"dnsSecOptions,omitempty"`
}

type filter struct {
	Field    string `json:"field"`
	Value    string `json:"value"`
	Relation string `json:"relation,omitempty"`
}

type nameserver struct {
	Name string   `json:"name"`
	IPs  []net.IP `json:"ips"`
}

type domainConfig struct {
	Name                string          `json:"name"`
	Contacts            json.RawMessage `json:"contacts"`
	Nameservers         []nameserver    `json:"nameservers"`
	DNSSecEntries       []dnsSecEntry   `json:"dnsSecEntries"`
	TransferLockEnabled bool            `json:"transferLockEnabled"`
}

type dnsSecEntry struct {
	KeyData dnsSecKey `json:"keyData"`
	Comment string    `json:"comment"`
	KeyTag  uint32    `json:"keyTag"`
}

type zoneConfig struct {
	ID                    string          `json:"id"`
	DNSSECMode            string          `json:"dnsSecMode"`
	EmailAddress          string          `json:"emailAddress,omitempty"`
	MasterIP              string          `json:"masterIp"`
	Name                  string          `json:"name"` // Not required per docs, but required IRL
	NameUnicode           string          `json:"nameUnicode"`
	SOAValues             soaValues       `json:"soaValues"`
	Type                  string          `json:"type"`
	TemplateValues        json.RawMessage `json:"templateValues,omitempty"`
	ZoneTransferWhitelist []string        `json:"zoneTransferWhitelist"`
}

type soaValues struct {
	Refresh     uint32 `json:"refresh"`
	Retry       uint32 `json:"retry"`
	Expire      uint32 `json:"expire"`
	NegativeTTL uint32 `json:"negativeTtl"`
	TTL         uint32 `json:"ttl"`
}

type zone struct {
	ZoneConfig zoneConfig `json:"zoneConfig"`
	Records    []record   `json:"records"`
}

type dnsSecOptions struct {
	Keys       []dnsSecEntry `json:"keys,omitempty"`
	Algorithms []string      `json:"algorithms,omitempty"`
	NSECMode   string        `json:"nsecMode"`
	PublishKSK bool          `json:"publishKsk"`
}

type dnsSecKey struct {
	Flags     uint32 `json:"flags"`
	Protocol  uint32 `json:"protocol"`
	Algorithm uint32 `json:"algorithm"`
	PublicKey string `json:"publicKey"`
}

type record struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Type     string `json:"type"`
	Content  string `json:"content"`
	TTL      uint32 `json:"ttl"`
	Priority uint16 `json:"priority"`
}

type response struct {
	Errors   []apiError    `json:"errors"`
	Response *responseData `json:"response"`
	Status   string        `json:"status"`
}

type apiError struct {
	Code          int    `json:"code"`
	ContextObject string `json:"contextObject"`
	ContextPath   string `json:"contextPath"`
	Text          string `json:"text"`
	Value         string `json:"value"`
}

type responseData struct {
	Data json.RawMessage `json:"data"`
	Type string          `json:"type"`

	Limit      uint `json:"limit"`
	Page       uint `json:"page"`
	TotalPages uint `json:"totalPages"`
}

func (r record) nativeToRecord(domain string) (*models.RecordConfig, error) {
	// normalize cname,mx,ns records with dots to be consistent with our config format.
	if r.Type == "ALIAS" || r.Type == "CNAME" || r.Type == "MX" || r.Type == "NS" || r.Type == "SRV" {
		if r.Content != "." {
			r.Content = r.Content + "."
		}
	}

	rc := &models.RecordConfig{
		Type:         "",
		TTL:          r.TTL,
		MxPreference: r.Priority,
		SrvPriority:  r.Priority,
		Original:     r,
	}
	rc.SetLabelFromFQDN(r.Name, domain)

	var err error
	switch r.Type {
	case "ALIAS":
		rc.Type = r.Type
		err = rc.SetTarget(r.Content)
	case "NULLMX":
		err = rc.PopulateFromString("MX", "0 .", domain)
	case "MX":
		err = rc.SetTargetMX(uint16(r.Priority), r.Content)
	case "PTR":
		rc.Type = r.Type
		err = rc.SetTarget(r.Content + ".")
	case "SRV":
		err = rc.SetTargetSRVPriorityString(uint16(r.Priority), r.Content)
	default:
		err = rc.PopulateFromString(r.Type, r.Content, domain)
	}
	return rc, err
}

func recordToNative(rc *models.RecordConfig) *record {
	record := &record{
		Name:    rc.NameFQDN,
		Type:    rc.Type,
		Content: strings.TrimSuffix(rc.GetTargetCombined(), "."),
		TTL:     rc.TTL,
	}

	switch rc.Type { // #rtype_variations
	case "A", "AAAA", "ALIAS", "CAA", "CNAME", "DNSKEY", "DS", "NS", "NSEC", "NSEC3", "NSEC3PARAM", "PTR", "RRSIG", "SSHFP", "TSLA":
		// Nothing special.
	case "TXT":
		// TODO(tlim): Move this to a function with unit tests.
		txtStrings := make([]string, rc.GetTargetTXTSegmentCount())
		copy(txtStrings, rc.GetTargetTXTSegmented())

		// Escape quotes
		for i := range txtStrings {
			txtStrings[i] = fmt.Sprintf(`"%s"`, strings.ReplaceAll(txtStrings[i], `"`, `\"`))
		}

		record.Content = strings.Join(txtStrings, " ")
	case "MX":
		record.Priority = rc.MxPreference
		record.Content = strings.TrimSuffix(rc.GetTargetField(), ".")
		if record.Content == "" {
			record.Type = "NULLMX"
			record.Priority = 10
		}
	case "SRV":
		record.Priority = rc.SrvPriority
		record.Content = fmt.Sprintf("%d %d %s", rc.SrvWeight, rc.SrvPort, strings.TrimSuffix(rc.GetTargetField(), "."))
	default:
		log.Printf("hosting.de rtype %v unimplemented", rc.Type)
	}

	return record
}
