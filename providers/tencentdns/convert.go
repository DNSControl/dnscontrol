package tencentdns

import (
	"fmt"

	"github.com/DNSControl/dnscontrol/v4/models"
	"github.com/DNSControl/dnscontrol/v4/pkg/txtutil"
	dnspod "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/dnspod/v20210323"
)

func nativeToRecord(r *dnspod.RecordListItem, domainName string) (*models.RecordConfig, error) {
	rc := &models.RecordConfig{
		TTL:      uint32(*r.TTL),
		Original: r,
	}
	rc.SetLabel(*r.Name, domainName)

	val := *r.Value
	switch *r.Type {
	case "A", "AAAA", "CNAME", "NS", "PTR", "TXT", "CAA", "SRV":
	case "MX":
		if r.MX != nil {
			val = fmt.Sprintf("%d %s", *r.MX, *r.Value)
		}
	default:
		return nil, fmt.Errorf("unsupported record type: %s", *r.Type)
	}

	// DNSPod does not have a native ALIAS record type. DNSControl uses
	// ALIAS("@") to model apex CNAME flattening, which DNSPod represents
	// as a CNAME record at "@".
	// See https://docs.dnspod.com/dns/faq-dns-resolution/?lang=en.
	rtype := *r.Type
	if rtype == "CNAME" && *r.Name == "@" {
		rtype = "ALIAS"
	}

	if err := rc.PopulateFromStringFunc(rtype, val, domainName, txtutil.ParseQuoted); err != nil {
		return nil, err
	}

	return rc, nil
}

func recordToCreateRequest(rc *models.RecordConfig) *dnspod.CreateRecordRequest {
	req := dnspod.NewCreateRecordRequest()
	req.SubDomain = new(rc.GetLabel())
	req.RecordType = new(rc.Type)
	if rc.Type == "ALIAS" {
		req.RecordType = new("CNAME")
	}
	req.RecordLine = new("默认")

	val := rc.GetTargetCombinedFunc(txtutil.EncodeQuoted)
	if rc.Type == "MX" {
		val = rc.GetTargetField()
		req.MX = new(uint64(rc.MxPreference))
	}
	req.Value = new(val)
	req.TTL = new(uint64(rc.TTL))

	return req
}

func recordToModifyRequest(rc *models.RecordConfig, recordID uint64) *dnspod.ModifyRecordRequest {
	req := dnspod.NewModifyRecordRequest()
	req.RecordId = new(recordID)
	req.SubDomain = new(rc.GetLabel())
	req.RecordType = new(rc.Type)
	if rc.Type == "ALIAS" {
		req.RecordType = new("CNAME")
	}
	req.RecordLine = new("默认")

	val := rc.GetTargetCombinedFunc(txtutil.EncodeQuoted)
	if rc.Type == "MX" {
		val = rc.GetTargetField()
		req.MX = new(uint64(rc.MxPreference))
	}
	req.Value = new(val)
	req.TTL = new(uint64(rc.TTL))

	return req
}
