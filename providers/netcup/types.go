package netcup

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	dnsv2 "codeberg.org/miekg/dns"
	"github.com/DNSControl/dnscontrol/v5/models"
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

func toRecordConfig(dc *models.DomainConfig, r *record) (*models.RecordConfig, error) {
	label := dc.LabelFromShort(r.Hostname)
	var rc *models.RecordConfig
	var err error
	switch rtype := r.Type; rtype { // #rtype_variations
	case "TXT":
		rc, err = dc.NewRecordConfig(label, 0, dnsv2.TypeTXT, r.Destination)
	case "NS", "ALIAS", "CNAME", "MX":
		if r.Type == "MX" {
			rc, err = dc.NewRecordConfig(label, 0, dnsv2.TypeMX, r.Priority, addTailingDot(r.Destination))
		} else {
			rc, err = dc.NewRecordConfig(label, 0, r.Type, addTailingDot(r.Destination))
		}
	// case "SRV":
	// 	parts := strings.Split(r.Destination, " ")
	// 	rc, err = dc.NewRecordConfig(label, 0, dnsv2.TypeSRV, parts[0], parts[1], parts[2], parts[3])
	// case "CAA":
	// 	parts := strings.Split(r.Destination, " ")
	// 	rc, err = dc.NewRecordConfig(label, 0, dnsv2.TypeCAA, parts[0], parts[1], strings.Trim(parts[2], "\""))
	// case "TLSA":
	// 	parts := strings.Split(r.Destination, " ")
	// 	rc, err = dc.NewRecordConfig(label, 0, dnsv2.TypeTLSA, parts[0], parts[1], parts[2], parts[3])
	default:
		rc, err = dc.NewRecordConfigParse(label, 0, r.Type, r.Destination)
	}
	if err != nil {
		return nil, err
	}
	rc.Original = r
	return rc, nil
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
