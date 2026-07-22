package tencentdns

import (
	"fmt"
	"strconv"

	"github.com/DNSControl/dnscontrol/v5/models"
	"github.com/DNSControl/dnscontrol/v5/pkg/txtutil"
	dnspod "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/dnspod/v20210323"
)

func nativeToRecord(r *dnspod.RecordListItem, dc *models.DomainConfig) (*models.RecordConfig, error) {
	label := dc.LabelFromShort(*r.Name)
	ttl := uint32(*r.TTL)
	metadata := map[string]string{}
	if r.Line != nil && *r.Line != "" {
		metadata[metaRecordLine] = *r.Line
	}
	if r.LineId != nil && *r.LineId != "" {
		metadata[metaRecordLineID] = *r.LineId
	}
	if r.Weight != nil {
		metadata[metaRecordWeight] = strconv.FormatUint(*r.Weight, 10)
	}

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

	var rc *models.RecordConfig
	var err error
	switch rtype {
	case "TXT":
		var txt string
		txt, err = txtutil.ParseQuoted(val)
		if err == nil {
			rc, err = dc.NewRecordConfig(label, ttl, rtype, txt)
		}
	case "ALIAS":
		rc, err = dc.NewRecordConfig(label, ttl, rtype, val)
	default:
		rc, err = dc.NewRecordConfigParse(label, ttl, rtype, val)
	}
	if err != nil {
		return nil, err
	}
	rc.Original = r
	rc.Metadata = metadata

	return rc, nil
}

func recordLineMetadata(rc *models.RecordConfig) (line, lineID string) {
	line = defaultRecordLine
	if rc.Metadata == nil {
		return line, ""
	}
	if configuredLine := rc.Metadata[metaRecordLine]; configuredLine != "" {
		line = configuredLine
	}
	return line, rc.Metadata[metaRecordLineID]
}

func recordWeightMetadata(rc *models.RecordConfig) (uint64, bool) {
	if rc == nil || rc.Metadata == nil || rc.Metadata[metaRecordWeight] == "" {
		return 0, false
	}
	weight, err := strconv.ParseUint(rc.Metadata[metaRecordWeight], 10, 64)
	if err != nil || weight > 100 {
		return 0, false
	}
	return weight, true
}

// comparableRecordWeight treats an omitted weight and weight 0 as equivalent,
// because DNSPod defines 0 as disabling weighted routing.
func comparableRecordWeight(rc *models.RecordConfig) string {
	weight, ok := recordWeightMetadata(rc)
	if !ok || weight == 0 {
		return ""
	}
	return strconv.FormatUint(weight, 10)
}

func recordToCreateRequest(rc *models.RecordConfig) *dnspod.CreateRecordRequest {
	req := dnspod.NewCreateRecordRequest()
	req.SubDomain = new(rc.GetLabel())
	req.RecordType = new(rc.Type)
	if rc.Type == "ALIAS" {
		req.RecordType = new("CNAME")
	}
	line, lineID := recordLineMetadata(rc)
	req.RecordLine = new(line)
	if lineID != "" {
		req.RecordLineId = new(lineID)
	}
	if weight, ok := recordWeightMetadata(rc); ok {
		req.Weight = new(weight)
	}

	val := rc.GetRDATA().String()
	if rc.Type == "TXT" {
		val = txtutil.EncodeQuoted(rc.GetTargetTXTJoined())
	}
	if rc.Type == "MX" {
		val = rc.GetTargetField()
		req.MX = new(uint64(rc.MxPreference))
	}
	req.Value = new(val)
	req.TTL = new(uint64(rc.TTL))

	return req
}

func recordToModifyRequest(rc *models.RecordConfig, recordID uint64, previous *models.RecordConfig) *dnspod.ModifyRecordRequest {
	req := dnspod.NewModifyRecordRequest()
	req.RecordId = new(recordID)
	req.SubDomain = new(rc.GetLabel())
	req.RecordType = new(rc.Type)
	if rc.Type == "ALIAS" {
		req.RecordType = new("CNAME")
	}
	line, lineID := recordLineMetadata(rc)
	req.RecordLine = new(line)
	if lineID != "" {
		req.RecordLineId = new(lineID)
	}
	if weight, ok := recordWeightMetadata(rc); ok {
		req.Weight = new(weight)
	} else if comparableRecordWeight(previous) != "" {
		// DNSPod requires weight 0 to explicitly disable weighted routing.
		req.Weight = new(uint64(0))
	}

	val := rc.GetRDATA().String()
	if rc.Type == "TXT" {
		val = txtutil.EncodeQuoted(rc.GetTargetTXTJoined())
	}
	if rc.Type == "MX" {
		val = rc.GetTargetField()
		req.MX = new(uint64(rc.MxPreference))
	}
	req.Value = new(val)
	req.TTL = new(uint64(rc.TTL))

	return req
}
