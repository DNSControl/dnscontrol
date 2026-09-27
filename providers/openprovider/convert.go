package openprovider

import (
	"fmt"
	"strings"

	dnsv2 "codeberg.org/miekg/dns"
	"github.com/DNSControl/dnscontrol/v5/models"
	"github.com/DNSControl/dnscontrol/v5/pkg/nrc"
	"github.com/DNSControl/dnscontrol/v5/pkg/txtutil"
)

const apexLabel = "@"

func toRecordConfig(record apiRecord, dc *models.DomainConfig) (*models.RecordConfig, error) {
	rtype := strings.ToUpper(record.Type)
	label := labelFromAPIRecordName(record.Name, dc)

	value := record.Value
	var rc *models.RecordConfig
	var err error
	switch rtype {
	case "MX":
		rc, err = dc.NewRecordConfig(label, uint32(record.TTL), dnsv2.TypeMX, uint16(record.Prio), value, nrc.Flags{TargetIsFqdnNoDot: true})
	case "SRV":
		rc, err = dc.NewRecordConfig(label, uint32(record.TTL), dnsv2.TypeSRV, uint16(record.Prio), value,
			nrc.Flags{SrvWeirdSplit: true, TargetIsFqdnNoDot: true})
	case "TXT", "SPF":
		// SPF (RR99) is a legacy API type. DNSControl represents it as TXT,
		// which is also the form users can declare in dnsconfig.js.
		rc, err = dc.NewRecordConfigParse(label, uint32(record.TTL), dnsv2.TypeTXT, value)
	case "CNAME":
		rc, err = dc.NewRecordConfig(label, uint32(record.TTL), dnsv2.TypeCNAME, value,
			nrc.Flags{TargetIsFqdnNoDot: true})
	default:
		rc, err = dc.NewRecordConfigParse(label, uint32(record.TTL), rtype, value)
	}
	if err != nil {
		return nil, fmt.Errorf("OPENPROVIDER: parse %s record %q: %w", rtype, record.Name, err)
	}
	rc.Original = record
	return rc, nil
}

// labelFromAPIRecordName converts the owner-name formats returned by
// Openprovider into a RecordConfig label using the DomainConfig helpers.
func labelFromAPIRecordName(name string, dc *models.DomainConfig) string {
	origin := strings.TrimSuffix(dc.Name, ".")
	trimmed := strings.TrimSuffix(name, ".")

	switch {
	case name == "" || name == apexLabel:
		return dc.LabelFromShort(name)
	case strings.HasSuffix(name, "."):
		return dc.LabelFromFQDNWithDot(name)
	case strings.EqualFold(trimmed, origin) || strings.HasSuffix(strings.ToLower(trimmed), "."+strings.ToLower(origin)):
		return dc.LabelFromFQDNNoDot(name)
	default:
		return dc.LabelFromShort(name)
	}
}

func fromRecordConfig(rc *models.RecordConfig) apiRecord {
	record := apiRecord{
		Type: rc.Type,
		TTL:  int(rc.TTL),
	}
	if rc.GetLabel() != apexLabel {
		record.Name = rc.GetLabel()
	}

	switch rc.Type {
	case "MX":
		mx := rc.AsMX()
		record.Prio = int(mx.Preference)
		record.Value = strings.TrimSuffix(mx.Mx, ".")
	case "SRV":
		srv := rc.AsSRV()
		record.Prio = int(srv.Priority)
		record.Value = fmt.Sprintf("%d %d %s", srv.Weight, srv.Port, strings.TrimSuffix(srv.Target, "."))
	case "TXT":
		record.Value = txtutil.EncodeQuoted(rc.GetTargetTXTJoined())
	case "CNAME":
		record.Value = strings.TrimSuffix(rc.AsCNAME().Target, ".")
	case "CAA":
		caa := rc.AsCAA()
		record.Value = fmt.Sprintf("%d %s %q", caa.Flag, caa.Tag, caa.Value)
	case "TLSA":
		tlsa := rc.AsTLSA()
		record.Value = fmt.Sprintf("%d %d %d %s", tlsa.Usage, tlsa.Selector, tlsa.MatchingType, strings.ToLower(tlsa.Certificate))
	default:
		record.Value = rc.GetTargetField()
	}
	return record
}

func isApexRecordName(name, origin string) bool {
	name = strings.TrimSuffix(name, ".")
	return name == "" || name == apexLabel || strings.EqualFold(name, origin)
}
