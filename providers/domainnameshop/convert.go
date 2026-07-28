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

	dnsR := &domainNameShopRecord{
		ID:       0,
		Host:     rc.GetLabel(),
		TTL:      uint16(fixTTL(rc.TTL)),
		Type:     rc.Type,
		DomainID: domainID,
	}

	switch rc.Type {
	case "CAA":
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
	case "MX":
		f := rc.AsMX()
		dnsR.Priority = strconv.Itoa(int(f.Preference))
		dnsR.Data = f.Mx
	case "SRV":
		f := rc.AsSRV()
		dnsR.Priority = strconv.Itoa(int(f.Priority))
		dnsR.Weight = strconv.Itoa(int(f.Weight))
		dnsR.Port = strconv.Itoa(int(f.Port))
		dnsR.ActualWeight = f.Weight
		dnsR.ActualPort = f.Port
		dnsR.Data = f.Target
	case "TXT":
		dnsR.Data = rc.GetTargetTXTJoined()
	default:
		dnsR.Data = rc.GetRDATA().String()
	}

	return dnsR, nil
}
