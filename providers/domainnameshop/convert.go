package domainnameshop

import (
	"strconv"

	"github.com/DNSControl/dnscontrol/v5/models"
)

func toRecordConfig(dc *models.DomainConfig, currentRecord *domainNameShopRecord) (*models.RecordConfig, error) {
	target := currentRecord.Data
	var rc *models.RecordConfig
	var err error
	switch currentRecord.Type {
	case "TXT":
		rc, err = dc.NewRecordConfig(currentRecord.Host, fixTTL(uint32(currentRecord.TTL)), currentRecord.Type, target)
	case "MX":
		rc, err = dc.NewRecordConfig(currentRecord.Host, fixTTL(uint32(currentRecord.TTL)), currentRecord.Type, currentRecord.ActualPriority, target)
	case "SRV":
		rc, err = dc.NewRecordConfig(currentRecord.Host, fixTTL(uint32(currentRecord.TTL)), currentRecord.Type, currentRecord.ActualPriority, currentRecord.ActualWeight, currentRecord.ActualPort, target)
	case "CAA":
		tag := "iodef"
		switch currentRecord.CAATag {
		case "0":
			tag = "issue"
		case "1":
			tag = "issuewild"
		}
		rc, err = dc.NewRecordConfig(currentRecord.Host, fixTTL(uint32(currentRecord.TTL)), currentRecord.Type, uint8(currentRecord.CAAFlag), tag, target)
	default:
		rc, err = dc.NewRecordConfig(currentRecord.Host, fixTTL(uint32(currentRecord.TTL)), currentRecord.Type, target)
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
