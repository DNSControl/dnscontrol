package openwrt

import (
	"fmt"
	"net/netip"
	"strconv"

	dnsv2 "codeberg.org/miekg/dns"
	"github.com/DNSControl/dnscontrol/v5/models"
)

type nativeRecord struct {
	Section string `json:".name,omitempty"`
	Type    string `json:".type,omitempty"`

	// A
	Name string `json:"name,omitempty"`
	IP   string `json:"ip,omitempty"`

	// CNAME
	Cname  string `json:"cname,omitempty"`
	Target string `json:"target,omitempty"`

	// MX
	Domain string `json:"domain,omitempty"`
	Relay  string `json:"relay,omitempty"`
	Pref   string `json:"pref,omitempty"`

	// SRV
	Srv      string `json:"srv,omitempty"`
	Priority string `json:"class,omitempty"`
	Weight   string `json:"weight,omitempty"`
	Port     string `json:"port,omitempty"`
	// Shares Target attribute with CNAME.
}

func (r *nativeRecord) isRecord() bool {
	return r.Type == "domain" || r.Type == "cname" || r.Type == "mxhost" || r.Type == "srvhost"
}

// The domain is a different attribute based on the record type.
func (r *nativeRecord) getDomain() (string, error) {
	var recDomain string

	switch r.Type {
	case "domain":
		recDomain = r.Name
	case "cname":
		recDomain = r.Cname
	case "mxhost":
		recDomain = r.Domain
	case "srvhost":
		recDomain = r.Srv
	default:
		return "", fmt.Errorf("no valid domain could be forund %s", r.Type)
	}

	return recDomain, nil
}

func toRc(dc *models.DomainConfig, r nativeRecord) (*models.RecordConfig, error) {
	recDomain, _ := r.getDomain()
	label := dc.LabelFromFQDNNoDot(recDomain)
	var rc *models.RecordConfig
	var err error

	switch r.Type {
	case "domain":
		addr, parseErr := netip.ParseAddr(r.IP)
		if parseErr != nil {
			return nil, parseErr
		}
		switch {
		case addr.Is4():
			rc, err = dc.NewRecordConfig(label, 300, dnsv2.TypeA, addr)
		case addr.Is6():
			rc, err = dc.NewRecordConfig(label, 300, dnsv2.TypeAAAA, addr)
		}

	case "cname":
		rc, err = dc.NewRecordConfig(label, 300, dnsv2.TypeCNAME, r.Target)

	case "mxhost":
		if _, parseErr := strconv.ParseUint(r.Pref, 10, 16); parseErr != nil {
			return nil, parseErr
		}
		rc, err = dc.NewRecordConfig(label, 300, dnsv2.TypeMX, r.Pref, r.Relay)

	case "srvhost":
		if _, parseErr := strconv.ParseUint(r.Priority, 10, 16); parseErr != nil {
			return nil, parseErr
		}
		if _, parseErr := strconv.ParseUint(r.Weight, 10, 16); parseErr != nil {
			return nil, parseErr
		}
		if _, parseErr := strconv.ParseUint(r.Port, 10, 16); parseErr != nil {
			return nil, parseErr
		}
		rc, err = dc.NewRecordConfig(label, 300, dnsv2.TypeSRV, r.Priority, r.Weight, r.Port, r.Target)

	default:
		return nil, fmt.Errorf("unhandled record type: %s", r.Type)
	}
	if err != nil {
		return nil, err
	}
	rc.Original = r
	return rc, nil
}

func toNative(rc *models.RecordConfig) (nativeRecord, string, error) {
	var r nativeRecord
	var recType string
	var err error

	// omits .type and .name
	switch rc.Type {
	case "A", "AAAA":
		recType = "domain"
		r.Name = rc.NameFQDN
		r.IP = rc.GetTargetIP().String()

	case "CNAME":
		recType = "cname"
		r.Cname = rc.NameFQDN
		r.Target = rc.GetTargetField()

	case "SRV":
		recType = "srvhost"
		r.Srv = rc.NameFQDN
		r.Priority = string(strconv.Itoa(int(rc.SrvPriority)))
		r.Weight = strconv.Itoa(int(rc.SrvWeight))
		r.Port = strconv.Itoa(int(rc.SrvPort))
		r.Target = rc.GetTargetField()

	case "MX":
		recType = "mxhost"
		r.Domain = rc.NameFQDN
		r.Pref = strconv.Itoa(int(rc.MxPreference))
		r.Relay = rc.GetTargetField()

	default:
		err = fmt.Errorf("unhandled record type: %s", rc.Type)
	}

	return r, recType, err
}
