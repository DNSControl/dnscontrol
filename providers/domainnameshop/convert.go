package domainnameshop

import (
	"strconv"

	dnsv2 "codeberg.org/miekg/dns"
	"github.com/DNSControl/dnscontrol/v5/models"
)

func toRecordConfig(dc *models.DomainConfig, currentRecord *domainNameShopRecord) (*models.RecordConfig, error) {
	label := dc.LabelFromShort(currentRecord.Host)
	ttl := fixTTL(uint32(currentRecord.TTL))
	target := currentRecord.Data

	var rc *models.RecordConfig
	var err error
	switch currentRecord.Type {
	// case "TXT":
	// 	rc, err = dc.NewRecordConfig(label, ttl, currentRecord.Type, target)
	case "MX":
		rc, err = dc.NewRecordConfig(label, ttl, currentRecord.Type, currentRecord.ActualPriority, target)
	case "SRV":
		rc, err = dc.NewRecordConfig(label, ttl, currentRecord.Type, currentRecord.ActualPriority, currentRecord.ActualWeight, currentRecord.ActualPort, target)
	case "CAA":
		tag := "iodef"
		switch currentRecord.CAATag {
		case "0":
			tag = "issue"
		case "1":
			tag = "issuewild"
		}
		rc, err = dc.NewRecordConfig(label, ttl, currentRecord.Type, uint8(currentRecord.CAAFlag), tag, target)
	default:
		rc, err = dc.NewRecordConfig(label, ttl, currentRecord.Type, target)
	}
	if err != nil {
		return nil, err
	}

	rc.Original = currentRecord

	return rc, nil
}

func (api *domainNameShopProvider) fromRecordConfig(domainName string, rc *models.RecordConfig) (*domainNameShopRecord, error) {
	domainID, err := api.getDomainID(domainName)
	if err != nil {
		return nil, err
	}

	dnsR := &domainNameShopRecord{
		ID:       0,
		Host:     rc.GetLabel(),
		TTL:      uint16(fixTTL(rc.TTL)),
		Type:     rc.Type,
		DomainID: domainID,
	}

	switch rc.TypeNum {
	case dnsv2.TypeCAA:
		f := rc.AsCAA()
		// Actual CAA FLAG
		switch f.Tag {
		case "issue":
			dnsR.CAATag = "0"
		case "issuewild":
			dnsR.CAATag = "1"
		case "iodef":
			dnsR.CAATag = "2"
		}
		dnsR.CAAFlag = uint64(int(f.Flag))
		dnsR.ActualCAAFlag = strconv.Itoa(int(f.Flag))
		dnsR.Data = f.Value
	case dnsv2.TypeMX:
		f := rc.AsMX()
		dnsR.Priority = strconv.Itoa(int(f.Preference))
		dnsR.Data = f.Mx
	case dnsv2.TypeSRV:
		f := rc.AsSRV()
		dnsR.Priority = strconv.Itoa(int(f.Priority))
		dnsR.Weight = strconv.Itoa(int(f.Weight))
		dnsR.Port = strconv.Itoa(int(f.Port))
		dnsR.ActualWeight = f.Weight
		dnsR.ActualPort = f.Port
		dnsR.Data = f.Target
	case dnsv2.TypeTXT:
		dnsR.Data = rc.GetTargetTXTJoined()
	default:
		dnsR.Data = rc.GetRDATA().String()
	}

	return dnsR, nil
}
