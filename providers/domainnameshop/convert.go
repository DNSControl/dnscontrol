package domainnameshop

import (
	"strconv"

	"github.com/DNSControl/dnscontrol/v4/models"
	dnsutilv1 "github.com/miekg/dns/dnsutil"
)

func toRecordConfig(domain string, currentRecord *domainNameShopRecord) (*models.RecordConfig, error) {
	name := dnsutilv1.AddOrigin(currentRecord.Host, domain)

	target := currentRecord.Data

	t := &models.RecordConfig{
		Type:         currentRecord.Type,
		TTL:          fixTTL(uint32(currentRecord.TTL)),
		MxPreference: uint16(currentRecord.ActualPriority),
		SrvPriority:  uint16(currentRecord.ActualPriority),
		SrvWeight:    uint16(currentRecord.ActualWeight),
		SrvPort:      uint16(currentRecord.ActualPort),
		Original:     currentRecord,
		CaaTag:       currentRecord.CAATag,
		CaaFlag:      uint8(currentRecord.CAAFlag),
	}

	if err := t.SetTarget(target); err != nil {
		return nil, err
	}
	t.SetLabelFromFQDN(name, domain)

	switch rtype := currentRecord.Type; rtype {
	case "TXT":
		if err := t.SetTargetTXT(target); err != nil {
			return nil, err
		}
	case "CAA":
		switch currentRecord.CAATag {
		case "0":
			t.CaaTag = "issue"
		case "1":
			t.CaaTag = "issuewild"
		default:
			t.CaaTag = "iodef"
		}
	default:
		// nothing additional required
	}
	return t, nil
}

func (api *domainNameShopProvider) fromRecordConfig(domainName string, rc *models.RecordConfig) (*domainNameShopRecord, error) {
	domainID, err := api.getDomainID(domainName)
	if err != nil {
		return nil, err
	}

	data := ""
	if rc.Type == "TXT" {
		data = rc.GetTargetTXTJoined()
	} else {
		data = rc.GetTargetField()
	}

	dnsR := &domainNameShopRecord{
		ID:            0,
		Host:          rc.GetLabel(),
		TTL:           uint16(fixTTL(rc.TTL)),
		Type:          rc.Type,
		Data:          data,
		Weight:        strconv.Itoa(int(rc.SrvWeight)),
		Port:          strconv.Itoa(int(rc.SrvPort)),
		ActualWeight:  rc.SrvWeight,
		ActualPort:    rc.SrvPort,
		CAAFlag:       uint64(int(rc.CaaFlag)),
		ActualCAAFlag: strconv.Itoa(int(rc.CaaFlag)),
		DomainID:      domainID,
	}

	switch rc.Type {
	case "CAA":
		// Actual CAA FLAG
		switch rc.CaaTag {
		case "issue":
			dnsR.CAATag = "0"
		case "issuewild":
			dnsR.CAATag = "1"
		case "iodef":
			dnsR.CAATag = "2"
		}
	case "MX":
		dnsR.Priority = strconv.Itoa(int(rc.MxPreference))
	case "SRV":
		dnsR.Priority = strconv.Itoa(int(rc.SrvPriority))
	default:
		// pass through
	}

	return dnsR, nil
}
