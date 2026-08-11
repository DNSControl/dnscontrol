package openwrt

import (
	"fmt"
	"net/netip"
	"strconv"

	"github.com/DNSControl/dnscontrol/v4/models"
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

func toRc(domain string, r nativeRecord) (*models.RecordConfig, error) {
	rc := &models.RecordConfig{
		TTL:      300,
		Original: r,
	}

	recDomain, _ := r.getDomain()
	rc.SetLabelFromFQDN(recDomain, domain)

	switch r.Type {
	case "domain":
		addr, err := netip.ParseAddr(r.IP)
		if err != nil {
			return nil, err
		}

		rc.SetTargetIP(addr)
		switch {
		case addr.Is4():
			rc.Type = "A"
		case addr.Is6():
			rc.Type = "AAAA"
		}

	case "cname":
		rc.Type = "CNAME"
		rc.SetTarget(r.Target)

	case "mxhost":
		rc.Type = "MX"
		pref, err := strconv.ParseUint(r.Pref, 10, 16)
		if err != nil {
			return nil, err
		}
		rc.SetTargetMX(uint16(pref), r.Relay)

	case "srvhost":
		rc.Type = "SRV"
		priority, err := strconv.ParseUint(r.Priority, 10, 16)
		if err != nil {
			return nil, err
		}
		weight, err := strconv.ParseUint(r.Weight, 10, 16)
		if err != nil {
			return nil, err
		}
		port, err := strconv.ParseUint(r.Port, 10, 16)
		if err != nil {
			return nil, err
		}
		rc.SetTargetSRV(uint16(priority), uint16(weight), uint16(port), r.Target)

	default:
		return nil, fmt.Errorf("unhandled record type: %s", r.Type)
	}

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
