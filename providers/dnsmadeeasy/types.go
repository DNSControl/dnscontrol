package dnsmadeeasy

import (
	"strconv"

	"github.com/DNSControl/dnscontrol/v5/models"
	"github.com/DNSControl/dnscontrol/v5/pkg/txtutil"
)

// DNS Made Easy does not allow the system name servers to be edited, and said records appear to always have a fixed TTL of 86400.
const fixedNameServerRecordTTL = 86400

type singleDomainResponse struct {
	ID                  int                              `json:"id"`
	Name                string                           `json:"name"`
	DelegateNameServers []string                         `json:"delegateNameServers"`
	NameServers         []singleDomainResponseNameServer `json:"nameServers"`
	ProcessMulti        bool                             `json:"processMulti"`
	ActiveThirdParties  []any                            `json:"activeThirdParties"`
	PendingActionID     int                              `json:"pendingActionId"`
	GtdEnabled          bool                             `json:"gtdEnabled"`
	Created             int64                            `json:"created"`
	Updated             int64                            `json:"updated"`
}

type singleDomainResponseNameServer struct {
	Fqdn string `json:"fqdn"`
	Ipv4 string `json:"ipv4"`
	Ipv6 string `json:"ipv6"`
}

type singleDomainRequestData struct {
	Name string `json:"name"`
}

type multiDomainResponse struct {
	TotalRecords int                            `json:"totalRecords"`
	TotalPages   int                            `json:"totalPages"`
	Data         []multiDomainResponseDataEntry `json:"data"`
	Page         int                            `json:"page"`
}

type multiDomainResponseDataEntry struct {
	ID                 int    `json:"id"`
	Name               string `json:"name"`
	FolderID           int    `json:"folderId"`
	GtdEnabled         bool   `json:"gtdEnabled"`
	ProcessMulti       bool   `json:"processMulti"`
	ActiveThirdParties []any  `json:"activeThirdParties"`
	PendingActionID    int    `json:"pendingActionId"`
	VanityID           int    `json:"vanityId,omitempty"`
	Created            int64  `json:"created"`
	Updated            int64  `json:"updated"`
}

type recordResponse struct {
	TotalRecords int                       `json:"totalRecords"`
	TotalPages   int                       `json:"totalPages"`
	Data         []recordResponseDataEntry `json:"data"`
	Page         int                       `json:"page"`
}

type recordResponseDataEntry struct {
	ID    int    `json:"id"`
	Name  string `json:"name"`
	Type  string `json:"type"`
	Value string `json:"value"`
	TTL   int    `json:"ttl"`

	Source   int `json:"source"`
	SourceID int `json:"sourceId"`

	DynamicDNS bool   `json:"dynamicDns"`
	Password   string `json:"password"`

	// A records
	Monitor  bool `json:"monitor"`
	Failover bool `json:"failover"`
	Failed   bool `json:"failed"`

	// Global Traffic Director
	GtdLocation string `json:"gtdLocation"`

	// HTTPRED records
	Description  string `json:"description"`
	Keywords     string `json:"keywords"`
	Title        string `json:"title"`
	RedirectType string `json:"redirectType"`
	HardLink     bool   `json:"hardLink"`

	// MX records
	MxLevel int `json:"mxLevel"`

	// SRV records
	Weight   int `json:"weight"`
	Priority int `json:"Priority"`
	Port     int `json:"port"`

	// CAA records
	CaaType        string `json:"caaType"`
	IssuerCritical int    `json:"issuerCritical"`
}

type recordRequestData struct {
	ID    int    `json:"id"`
	Name  string `json:"name"`
	Type  string `json:"type"`
	Value string `json:"value"`
	TTL   int    `json:"ttl"`

	// Global Traffic Director
	GtdLocation string `json:"gtdLocation"`

	// MX records
	MxLevel int `json:"mxLevel"`

	// SRV records
	Weight   int `json:"weight,omitempty"`
	Priority int `json:"priority,omitempty"`
	Port     int `json:"port,omitempty"`

	// CAA records
	CaaType        string `json:"caaType"`
	IssuerCritical int    `json:"issuerCritical"`
}

func toRecordConfig(domain string, record *recordResponseDataEntry) *models.RecordConfig {
	recordType := record.Type
	if recordType == "ANAME" {
		// ANAME is DNS Made Easy's name for ALIAS (inverse of the ALIAS->ANAME
		// conversion in GetZoneRecordsCorrections).
		recordType = "ALIAS"
	}

	rc := &models.RecordConfig{
		Type:     recordType,
		TTL:      uint32(record.TTL),
		Original: record,
	}

	rc.SetLabel(record.Name, domain)

	var err error
	switch recordType {
	case "MX":
		err = rc.SetTargetMX(uint16(record.MxLevel), record.Value)
	case "SRV":
		err = rc.SetTargetSRV(uint16(record.Priority), uint16(record.Weight), uint16(record.Port), record.Value)
	case "CAA":
		value, unquoteErr := strconv.Unquote(record.Value)
		if unquoteErr != nil {
			panic(unquoteErr)
		}
		err = rc.SetTargetCAA(uint8(record.IssuerCritical), record.CaaType, value)
	default:
		err = rc.PopulateFromStringFunc(recordType, record.Value, domain, txtutil.ParseQuoted)
	}

	if err != nil {
		panic(err)
	}

	return rc
}

func fromRecordConfig(rc *models.RecordConfig) *recordRequestData {
	label := rc.GetLabel()
	if label == "@" {
		label = ""
	}

	recordType := rc.Type
	if recordType == "ALIAS" {
		// ALIAS is called ANAME on DNS Made Easy. Converting on write only
		// keeps the diff comparing ALIAS against ALIAS.
		recordType = "ANAME"
	}

	record := &recordRequestData{
		Type:        recordType,
		TTL:         int(rc.TTL),
		GtdLocation: "DEFAULT",
		Name:        label,
		// DNS Made Easy stores TXT as opaque text and mangles multi-chunk
		// values, so send the whole value as one quoted string.
		Value: rc.GetTargetCombinedFunc(txtutil.EncodeSingle),
	}

	switch record.Type {
	case "MX":
		record.MxLevel = int(rc.MxPreference)
		record.Value = rc.GetTargetField()
	case "SRV":
		target := rc.GetTargetField()
		if target == "." {
			target += "."
		}

		record.Priority = int(rc.SrvPriority)
		record.Weight = int(rc.SrvWeight)
		record.Port = int(rc.SrvPort)
		record.Value = target
	case "CAA":
		record.IssuerCritical = int(rc.CaaFlag)
		record.CaaType = rc.CaaTag
		record.Value = rc.GetTargetField()
	}

	return record
}

func systemNameServerToRecordConfig(domain string, nameServer string) *models.RecordConfig {
	target := nameServer + "."
	return toRecordConfig(domain, &recordResponseDataEntry{Type: "NS", Value: target, TTL: fixedNameServerRecordTTL})
}
