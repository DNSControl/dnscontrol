package tencentdns

import (
	"fmt"
	"strconv"

	dnsv2 "codeberg.org/miekg/dns"
	"github.com/DNSControl/dnscontrol/v5/models"
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

	// DNSPod does not have a native ALIAS record type. DNSControl uses
	// ALIAS("@") to model apex CNAME flattening, which DNSPod represents
	// as a CNAME record at "@".
	// See https://docs.dnspod.com/dns/faq-dns-resolution/?lang=en.
	// https://www.tencentcloud.com/document/product/1145/54764#2f681022-91ab-4a9e-ac3d-0a6c454d954e
	// https://docs.dnspod.com/dns/cname-flattening/
	// As a result, we can safely turn ALIAS records into CNAMEs.
	rtype := *r.Type
	if rtype == "ALIAS" {
		rtype = "CNAME"
	}

	var rc *models.RecordConfig
	var err error
	val := *r.Value
	switch rtype {
	case "MX":
		p := uint64(0)
		if r.MX != nil {
			p = *r.MX
		}
		fmt.Printf("DEBUG TENCENT: MX apip=%v p=%v v=%q\n", *r.MX, p, val)
		rc, err = dc.NewRecordConfig(label, ttl, dnsv2.TypeMX, p, val)
	case "TXT":
		// TODO(tlim): A few ways that might fix
		//--- FAIL: TestDNSProviders/oomkill.com/27:complex_TXT:a_256-byte_TXT (2.47s)
		//--- FAIL: TestDNSProviders/oomkill.com/28:TXT_backslashes:TXT_with_backslashs (4.47s)

		// Try this first:
		rc, err = dc.NewRecordConfigParse(label, ttl, rtype, val)

		// Try this if the other fails: (probably won't work)
		//rc, err = dc.NewRecordConfig(label, ttl, rtype, val)

		// Or this?
		//rc, err = dc.NewRecordConfig(label, ttl, rtype, txtutil.EncodeQuoted(val))
		// You'll need to add this to imports above:
		// "github.com/DNSControl/dnscontrol/v5/pkg/txtutil"

	// case "ALIAS":
	// 	rc, err = dc.NewRecordConfig(label, ttl, rtype, val)
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

	var val string
	switch rc.TypeNum {
	case dnsv2.TypeMX:
		f := rc.AsMX()
		val = f.Mx
		req.MX = new(uint64(f.Preference))
	case dnsv2.TypeTXT:
		f := rc.AsTXT()
		val = f.String()

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

	var val string
	switch rc.TypeNum {
	case dnsv2.TypeMX:
		f := rc.AsMX()
		val = f.Mx
		req.MX = new(uint64(f.Preference))

	// TODO(tlim): Try this if unicode required for MX targets.
	// You'll need to add this import: "golang.org/x/net/idna"
	// u, err := idna.ToUnicode(val)
	// if err == nil {
	// 	val = u
	// }

	// TODO(tlim): Try this if unicode is required for CNAME targets.
	// You'll need to add this import: "golang.org/x/net/idna"
	// case dnsv2.TypeCNAME:
	// 	f := rc.AsCNAME()
	// 	val := f.Target
	// 	u, err := idna.ToUnicode(val)
	// 	if err == nil {
	// 		val = u
	// 	}

	case dnsv2.TypeTXT:
		f := rc.AsTXT()
		val = f.String()
	default:
		val = rc.GetRDATA().String()
	}

	req.Value = new(val)
	req.TTL = new(uint64(rc.TTL))

	return req
}
