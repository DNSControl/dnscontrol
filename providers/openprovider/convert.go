package openprovider

import (
	"fmt"
	"strings"

	dnsv2 "codeberg.org/miekg/dns"
	"github.com/DNSControl/dnscontrol/v5/models"
	"github.com/DNSControl/dnscontrol/v5/pkg/txtutil"
)

const apexLabel = "@"

func toRecordConfig(record apiRecord, origin string) (*models.RecordConfig, error) {
	rtype := strings.ToUpper(record.Type)
	dc := &models.DomainConfig{Name: origin}
	label := relativeRecordName(record.Name, origin)
	if label == "" {
		label = apexLabel
	}

	value := record.Value
	var rc *models.RecordConfig
	var err error
	switch rtype {
	case "MX":
		rc, err = dc.NewRecordConfig(label, uint32(record.TTL), dnsv2.TypeMX, uint16(record.Prio), absoluteTarget(value, origin))
	case "SRV":
		fields := strings.Fields(value)
		if len(fields) != 3 {
			return nil, fmt.Errorf("OPENPROVIDER: SRV record %q has invalid value", record.Name)
		}
		var weight, port uint16
		if _, err := fmt.Sscanf(strings.Join(fields[:2], " "), "%d %d", &weight, &port); err != nil {
			return nil, fmt.Errorf("OPENPROVIDER: SRV record %q has invalid value: %w", record.Name, err)
		}
		rc, err = dc.NewRecordConfig(label, uint32(record.TTL), dnsv2.TypeSRV, uint16(record.Prio), weight, port, absoluteTarget(fields[2], origin))
	case "TXT", "SPF":
		var decoded string
		decoded, err = txtutil.ParseQuoted(value)
		if err == nil {
			rc, err = dc.NewRecordConfig(label, uint32(record.TTL), dnsv2.TypeTXT, decoded)
			if rc != nil && rtype == "SPF" {
				rc.Type = rtype
			}
		}
	case "CNAME":
		rc, err = dc.NewRecordConfig(label, uint32(record.TTL), dnsv2.TypeCNAME, absoluteTarget(value, origin))
	default:
		rc, err = dc.NewRecordConfigParse(label, uint32(record.TTL), rtype, value)
	}
	if err != nil {
		return nil, fmt.Errorf("OPENPROVIDER: parse %s record %q: %w", rtype, record.Name, err)
	}
	rc.Original = record
	return rc, nil
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
	case "TXT", "SPF":
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

func absoluteTarget(target, origin string) string {
	if target == "" || target == "." || strings.HasSuffix(target, ".") {
		return target
	}
	if !strings.Contains(target, ".") {
		return target + "." + origin + "."
	}
	return target + "."
}

func isApexRecordName(name, origin string) bool {
	name = strings.TrimSuffix(name, ".")
	return name == "" || name == apexLabel || strings.EqualFold(name, origin)
}
