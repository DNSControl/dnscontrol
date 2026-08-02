package netcup

import (
	"encoding/json"
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

func fromRecordConfig(rc *models.RecordConfig) *record {

	ncRec := &record{
		Hostname: rc.GetLabel(),
		Type:     rc.Type,
		Delete:   false,
		State:    "",
	}

	switch ncRec.Type {
	case "CAA":
		f := rc.AsCAA()
		ncRec.Destination = strconv.Itoa(int(f.Flag)) + " " + f.Tag + " \"" + f.Value + "\""
		// TODO(tlim): Try this instead:
		//ncRec.Destination = f.String()
	case "CNAME":
		f := rc.AsCNAME()
		ncRec.Destination = strings.TrimSuffix(f.Target, ".")
	case "MX":
		f := rc.AsMX()
		ncRec.Priority = strconv.Itoa(int(f.Preference))
		ncRec.Destination = strings.TrimSuffix(f.Mx, ".")
	case "NS":
		return nil // API ignores NS records
	case "SRV":
		f := rc.AsSRV()
		ncRec.Destination = strconv.Itoa(int(f.Priority)) + " " + strconv.Itoa(int(f.Weight)) + " " + strconv.Itoa(int(f.Port)) + " " + f.Target
		// TODO(tlim): Try this instead:
		//ncRec.Destination = f.String()
	case "SSHFP":
		f := rc.AsSSHFP()
		ncRec.Destination = f.String()
	case "TLSA":
		f := rc.AsTLSA()
		ncRec.Destination = strconv.Itoa(int(f.Usage)) + " " + strconv.Itoa(int(f.Selector)) + " " + strconv.Itoa(int(f.MatchingType)) + " " + f.Certificate
		// TODO(tlim): Try this instead:
		//ncRec.Destination = f.String()
	case "TXT":
		ncRec.Destination = rc.GetTargetTXTJoined()
	default:
		ncRec.Destination = rc.GetRDATA().String()
	}

	return ncRec
}
