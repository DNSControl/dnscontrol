package hostingde

import (
	"encoding/json"
	"fmt"
	"net" // Needed for communicating with provider API.
	"strings"

	dnsv2 "codeberg.org/miekg/dns"
	"github.com/DNSControl/dnscontrol/v5/models"
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

func (r record) nativeToRecord(dc *models.DomainConfig) (*models.RecordConfig, error) {
	// normalize cname,mx,ns records with dots to be consistent with our config format.
	if r.Type == "ALIAS" || r.Type == "CNAME" || r.Type == "MX" || r.Type == "NS" || r.Type == "SRV" {
		if r.Content != "." {
			r.Content = r.Content + "."
		}
	}

	label := dc.ToShort(r.Name)
	var rc *models.RecordConfig
	var err error
	ttl := r.TTL
	switch r.Type {
	case "ALIAS":
		rc, err = dc.NewRecordConfig(label, ttl, r.Type, r.Content)
	case "NULLMX":
		rc, err = dc.NewRecordConfig(label, ttl, "MX", 0, ".")
	case "MX":
		rc, err = dc.NewRecordConfig(label, ttl, r.Type, r.Priority, r.Content)
	case "PTR":
		rc, err = dc.NewRecordConfig(label, ttl, r.Type, r.Content+".")
	case "SRV":
		rc, err = dc.NewRecordConfigParse(label, ttl, r.Type, fmt.Sprintf("%d %s", r.Priority, r.Content))
	case "TXT":
		rc, err = dc.NewRecordConfig(label, ttl, r.Type, r.Content)
	default:
		rc, err = dc.NewRecordConfigParse(label, ttl, r.Type, r.Content)
	}
	if err == nil {
		rc.Original = r
	}
	return rc, err
}

func recordToNative(rc *models.RecordConfig) *record {
	record := &record{
		Name:    rc.NameFQDN,
		Type:    rc.Type,
		Content: strings.TrimSuffix(rc.GetRDATA().String(), "."),
		TTL:     rc.TTL,
	}

	switch rc.TypeNum {
	case dnsv2.TypeTXT:
		// TODO(tlim): I think all of this can be replaced by:
		// record.Content = rc.AsTXT().String()

		// TODO(tlim): Move this to a function with unit tests.
		txtStrings := make([]string, rc.GetTargetTXTSegmentCount())
		copy(txtStrings, rc.GetTargetTXTSegmented())

		// Escape quotes
		for i := range txtStrings {
			txtStrings[i] = fmt.Sprintf(`"%s"`, strings.ReplaceAll(txtStrings[i], `"`, `\"`))
		}

		record.Content = strings.Join(txtStrings, " ")
	case dnsv2.TypeMX:
		mx := rc.AsMX()
		record.Priority = mx.Preference
		record.Content = strings.TrimSuffix(mx.Mx, ".")
		if record.Content == "" {
			record.Type = "NULLMX"
			record.Priority = 10
		}
	case dnsv2.TypeSRV:
		srv := rc.AsSRV()
		record.Priority = srv.Priority
		record.Content = fmt.Sprintf("%d %d %s", srv.Weight, srv.Port, strings.TrimSuffix(srv.Target, "."))
	default:
		// no-op
	}

	return record
}
