package netcup

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/DNSControl/dnscontrol/v4/models"
	dnsutilv1 "github.com/miekg/dns/dnsutil"
)

type request struct {
	Action string `json:"action"`
	Param  any    `json:"param"`
}

type paramLogin struct {
	Key            string `json:"apikey"`
	Password       string `json:"apipassword"`
	CustomerNumber string `json:"customernumber"`
}

type paramGetRecords struct {
	Key            string `json:"apikey"`
	SessionID      string `json:"apisessionid"`
	CustomerNumber string `json:"customernumber"`
	DomainName     string `json:"domainname"`
}

type paramUpdateRecords struct {
	Key            string  `json:"apikey"`
	SessionID      string  `json:"apisessionid"`
	CustomerNumber string  `json:"customernumber"`
	DomainName     string  `json:"domainname"`
	RecordSet      records `json:"dnsrecordset"`
}

type records struct {
	Records []record `json:"dnsrecords"`
}

type record struct {
	ID          string `json:"id"`
	Hostname    string `json:"hostname"`
	Type        string `json:"type"`
	Priority    string `json:"priority"`
	Destination string `json:"destination"`
	Delete      bool   `json:"deleterecord"`
	State       string `json:"state"`
}

type response struct {
	ServerRequestID string          `json:"serverrequestid"`
	ClientRequestID string          `json:"clientrequestid"`
	Action          string          `json:"action"`
	Status          string          `json:"status"`
	StatusCode      int             `json:"statuscode"`
	ShortMessage    string          `json:"shortmessage"`
	LongMessage     string          `json:"longmessage"`
	Data            json.RawMessage `json:"responsedata"`
}

type responseLogin struct {
	SessionID string `json:"apisessionid"`
}

// addTailingDot adds a dot if it's missing from what the netcup api has returned to us.
func addTailingDot(destination string) string {
	if destination == "@" || len(destination) == 0 {
		return destination
	}
	if destination[len(destination)-1:] != "." {
		return destination + "."
	}
	return destination
}

func toRecordConfig(domain string, r *record) *models.RecordConfig {
	priority, _ := strconv.ParseUint(r.Priority, 10, 16)

	rc := &models.RecordConfig{
		Type:         r.Type,
		TTL:          uint32(0),
		MxPreference: uint16(priority),
		SrvPriority:  uint16(priority),
		SrvWeight:    uint16(0),
		SrvPort:      uint16(0),
		Original:     r,
	}
	rc.SetLabel(r.Hostname, domain)

	switch rtype := r.Type; rtype { // #rtype_variations
	case "TXT":
		_ = rc.SetTargetTXT(r.Destination)
	case "NS", "ALIAS", "CNAME", "MX":
		_ = rc.SetTarget(dnsutilv1.AddOrigin(addTailingDot(r.Destination), domain))
	case "SRV":
		parts := strings.Split(r.Destination, " ")
		priority, _ := strconv.ParseUint(parts[0], 10, 16)
		weight, _ := strconv.ParseUint(parts[1], 10, 16)
		port, _ := strconv.ParseUint(parts[2], 10, 16)
		rc.SrvPriority = uint16(priority)
		rc.SrvWeight = uint16(weight)
		rc.SrvPort = uint16(port)
		_ = rc.SetTarget(parts[3])
	case "CAA":
		parts := strings.Split(r.Destination, " ")
		caaFlag, _ := strconv.ParseUint(parts[0], 10, 8)
		rc.CaaFlag = uint8(caaFlag)
		rc.CaaTag = parts[1]
		_ = rc.SetTarget(strings.Trim(parts[2], "\""))
	case "TLSA":
		parts := strings.Split(r.Destination, " ")
		tlsaUsage, _ := strconv.ParseUint(parts[0], 10, 8)
		tlsaSelector, _ := strconv.ParseUint(parts[1], 10, 8)
		tlsaMatchingType, _ := strconv.ParseUint(parts[2], 10, 8)
		rc.TlsaUsage = uint8(tlsaUsage)
		rc.TlsaSelector = uint8(tlsaSelector)
		rc.TlsaMatchingType = uint8(tlsaMatchingType)
		_ = rc.SetTarget(parts[3])
	default:
		_ = rc.SetTarget(r.Destination)
	}
	return rc
}

func fromRecordConfig(in *models.RecordConfig) *record {
	rc := &record{
		Hostname:    in.GetLabel(),
		Type:        in.Type,
		Destination: in.GetTargetField(),
		Delete:      false,
		State:       "",
	}

	switch rc.Type { // #rtype_variations
	case "A", "AAAA", "PTR", "TXT", "SOA", "ALIAS":
		// Nothing special.
	case "CAA":
		rc.Destination = strconv.Itoa(int(in.CaaFlag)) + " " + in.CaaTag + " \"" + in.GetTargetField() + "\""
	case "CNAME":
		rc.Destination = strings.TrimSuffix(in.GetTargetField(), ".")
	case "MX":
		rc.Destination = strings.TrimSuffix(in.GetTargetField(), ".")
		rc.Priority = strconv.Itoa(int(in.MxPreference))
	case "NS":
		return nil // API ignores NS records
	case "SRV":
		rc.Destination = strconv.Itoa(int(in.SrvPriority)) + " " + strconv.Itoa(int(in.SrvWeight)) + " " + strconv.Itoa(int(in.SrvPort)) + " " + in.GetTargetField()
	case "SSHFP":
		rc.Destination = strconv.Itoa(int(in.SshfpAlgorithm)) + " " + strconv.Itoa(int(in.SshfpFingerprint))
	case "TLSA":
		rc.Destination = strconv.Itoa(int(in.TlsaUsage)) + " " + strconv.Itoa(int(in.TlsaSelector)) + " " + strconv.Itoa(int(in.TlsaMatchingType)) + " " + in.GetTargetField()
	default:
		msg := fmt.Sprintf("ClouDNS.toReq rtype %v unimplemented", rc.Type)
		panic(msg)
		// We panic so that we quickly find any switch statements
	}
	return rc
}
