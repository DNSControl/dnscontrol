package loopia

// Convert the provider's native record description to models.RecordConfig.

import (
	"fmt"

	"github.com/DNSControl/dnscontrol/v4/models"
	dnsutilv1 "github.com/miekg/dns/dnsutil"
)

// nativeToRecord takes a DNS record from Loopia and returns a native RecordConfig struct.
func nativeToRecord(zr zoneRecord, origin string, subdomain string) (rc *models.RecordConfig, err error) {
	record := zr.GetZR()

	rc = &models.RecordConfig{
		TTL:      record.TTL,
		Original: record,
		Type:     record.Type,
	}
	rc.SetLabel(subdomain, origin)
	if err := rc.SetTarget(record.Rdata); err != nil {
		return nil, err
	}

	switch rtype := record.Type; rtype {
	case "CAA":
		err = rc.SetTargetCAAString(record.Rdata)
	case "MX":
		// See dnscontrol issue #2218
		err = rc.SetTargetMX(record.Priority, dnsutilv1.AddOrigin(record.Rdata, origin)+".")
	case "NAPTR":
		err = rc.SetTargetNAPTRString(record.Rdata)
	case "TXT":
		err = rc.SetTargetTXT(record.Rdata)
	default:
		err = rc.PopulateFromString(rtype, record.Rdata, origin)
	}
	if err != nil {
		return nil, fmt.Errorf("unparsable record received from loopia: %w", err)
	}

	return rc, nil
}

func recordToNative(rc *models.RecordConfig, id ...uint32) paramStruct {
	// rc is the record from dnscontrol to loopia
	zrec := zRec{}
	zrec.Type = rc.Type
	zrec.TTL = rc.TTL
	zrec.Rdata = rc.GetTargetCombined()

	if rc.Original != nil {
		zrec.RecordID = rc.Original.(*zRec).RecordID
	} else if len(id) > 0 {
		zrec.RecordID = id[0]
	}
	switch zrec.Type {
	case "TXT":
		zrec.Rdata = rc.GetTargetTXTJoined()
	case "MX":
		zrec.Priority = rc.MxPreference
		zrec.Rdata = rc.GetTargetField()
	case "SRV":
		zrec.Priority = rc.SrvPriority
	}
	// fmt.Printf("r2n:zr %+v\n", zrec)

	return zrec.SetPS()
}
