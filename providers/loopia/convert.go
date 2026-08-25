package loopia

// Convert the provider's native record description to models.RecordConfig.

import (
	"fmt"

	"github.com/DNSControl/dnscontrol/v5/models"
)

// nativeToRecord takes a DNS record from Loopia and returns a native RecordConfig struct.
func nativeToRecord(zr zoneRecord, dc *models.DomainConfig, subdomain string) (rc *models.RecordConfig, err error) {
	record := zr.GetZR()
	label := subdomain
	ttl := record.TTL

	switch rtype := record.Type; rtype {
	// case "CAA":
	// 	rc, err = dc.NewRecordConfigParse(label, ttl, rtype, record.Rdata)
	case "MX":
		// See dnscontrol issue #2218
		rc, err = dc.NewRecordConfig(label, ttl, rtype, record.Priority, dc.ToFqdnWithDot(record.Rdata))
	// case "NAPTR":
	// 	rc, err = dc.NewRecordConfigParse(label, ttl, rtype, record.Rdata)
	case "TXT":
		rc, err = dc.NewRecordConfig(label, ttl, rtype, record.Rdata)
	default:
		rc, err = dc.NewRecordConfigParse(label, ttl, rtype, record.Rdata)
	}
	if err != nil {
		return nil, fmt.Errorf("unparsable record received from loopia: %w", err)
	}
	rc.Original = record

	return rc, nil
}

func recordToNative(rc *models.RecordConfig, id ...uint32) paramStruct {
	// rc is the record from dnscontrol to loopia
	zrec := zRec{
		Type:  rc.Type,
		TTL:   rc.TTL,
		Rdata: rc.GetRDATA().String()}

	if rc.Original != nil {
		zrec.RecordID = rc.Original.(*zRec).RecordID
	} else if len(id) > 0 {
		zrec.RecordID = id[0]
	}
	switch zrec.Type {
	case "TXT":
		zrec.Rdata = rc.GetTargetTXTJoined()
	case "MX":
		f := rc.AsMX()
		zrec.Priority = f.Preference
		zrec.Rdata = f.Mx
	case "SRV":
		f := rc.AsSRV()
		zrec.Priority = f.Priority
		zrec.Rdata = f.Target
		// if that doesn't work, try:
		//zrec.Rdata = fmt.Sprintf("%d %d %s", f.Weight, f.Port, f.Target)
	}
	// fmt.Printf("r2n:zr %+v\n", zrec)

	return zrec.SetPS()
}
