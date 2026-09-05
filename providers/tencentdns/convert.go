package tencentdns

import (
	"strconv"

	dnsv2 "codeberg.org/miekg/dns"
	dnsrdatav2 "codeberg.org/miekg/dns/rdata"
	"github.com/DNSControl/dnscontrol/v5/models"
	dnspod "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/dnspod/v20210323"
	"golang.org/x/net/idna"
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

	rtype := *r.Type

	var rc *models.RecordConfig
	var err error
	val := *r.Value
	switch rtype {
	case "MX":
		p := uint64(0)
		if r.MX != nil {
			p = *r.MX
		}
		//fmt.Printf("DEBUG TENCENT: MX apip=%v p=%v v=%q\n", *r.MX, p, val)
		rc, err = dc.NewRecordConfig(label, ttl, dnsv2.TypeMX, p, val)
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
	line, lineID := recordLineMetadata(rc)
	req.RecordLine = new(line)
	if lineID != "" {
		req.RecordLineId = new(lineID)
	}
	if weight, ok := recordWeightMetadata(rc); ok {
		req.Weight = new(weight)
	}

	var val string
	switch f := rc.GetRDATA().(type) {
	case dnsrdatav2.CNAME:
		var err error
		val, err = idna.ToUnicode(f.Target)
		if err != nil {
			// If there is a failure, use the original.
			val = f.Target
		}
	case dnsrdatav2.MX:
		req.MX = new(uint64(f.Preference))
		var err error
		val, err = idna.ToUnicode(f.Mx)
		if err != nil {
			// If there is a failure, use the original.
			val = f.Mx
		}
	case dnsrdatav2.TXT:
		val = rc.GetTargetTXTJoined()
	default:
		val = rc.GetRDATA().String()
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
	line, lineID := recordLineMetadata(rc)
	if lineID != "" {
		req.RecordLineId = new(lineID)
	}
	req.RecordLine = new(line)
	if weight, ok := recordWeightMetadata(rc); ok {
		req.Weight = new(weight)
	} else if comparableRecordWeight(previous) != "" {
		// DNSPod requires weight 0 to explicitly disable weighted routing.
		req.Weight = new(uint64(0))
	}

	var val string
	switch f := rc.GetRDATA().(type) {
	case dnsrdatav2.CNAME:
		var err error
		val, err = idna.ToUnicode(f.Target)
		if err != nil {
			// If there is a failure, use the original.
			val = f.Target
		}
	case dnsrdatav2.MX:
		req.MX = new(uint64(f.Preference))
		var err error
		val, err = idna.ToUnicode(f.Mx)
		if err != nil {
			val = f.Mx
		}
	case dnsrdatav2.TXT:
		val = rc.GetTargetTXTJoined()
	default:
		val = rc.GetRDATA().String()
	}

	req.Value = new(val)
	req.TTL = new(uint64(rc.TTL))

	return req
}
