package constellix

import (
	"encoding/json"
	"fmt"
	"strings"

	dnsv2 "codeberg.org/miekg/dns"
	"github.com/DNSControl/dnscontrol/v5/models"
	"github.com/DNSControl/dnscontrol/v5/pkg/privatetypes"
	"github.com/DNSControl/dnscontrol/v5/pkg/txtutil"
)

const (
	// apexLabel is how DNSControl names the zone apex; the API uses "".
	apexLabel = "@"

	// maxTTL is the largest TTL the API accepts.
	maxTTL = 2147483647
)

// decodeValues decodes the value list of a standard-mode record.
func decodeValues(rec apiRecord) ([]apiValue, error) {
	var values []apiValue
	if err := json.Unmarshal(rec.Value, &values); err != nil {
		return nil, err
	}
	return values, nil
}

// toRC converts a record object to one RecordConfig per value.
//
// Hostnames are stored as they were written: with a trailing dot they are
// fully qualified, without one Constellix serves them relative to the zone.
// The record factories read them the same way, so no TargetIsFqdnNoDot here.
// TXT values are stored in presentation format.
//
// Original is the record object itself; its ID addresses the set in a PUT or
// DELETE.
func toRC(dc *models.DomainConfig, rec apiRecord) (models.Records, error) {
	values, err := decodeValues(rec)
	if err != nil {
		return nil, fmt.Errorf("CONSTELLIX: record %q/%s: %w", rec.Name, rec.Type, err)
	}

	label := dc.LabelFromShort(rec.Name)
	recs := make(models.Records, 0, len(values))
	for _, v := range values {
		var rc *models.RecordConfig
		var err error
		switch rec.Type {
		case "A":
			rc, err = dc.NewRecordConfig(label, rec.TTL, dnsv2.TypeA, v.Value)
		case "AAAA":
			rc, err = dc.NewRecordConfig(label, rec.TTL, dnsv2.TypeAAAA, v.Value)
		case "CNAME":
			rc, err = dc.NewRecordConfig(label, rec.TTL, dnsv2.TypeCNAME, v.Value)
		case "ANAME":
			rc, err = dc.NewRecordConfig(label, rec.TTL, privatetypes.TypeALIAS, v.Value)
		case "TXT":
			rc, err = txtToRC(dc, label, rec.TTL, v.Value)
		case "MX":
			rc, err = dc.NewRecordConfig(label, rec.TTL, dnsv2.TypeMX, v.Priority, v.Server)
		case "SRV":
			rc, err = dc.NewRecordConfig(label, rec.TTL, dnsv2.TypeSRV, v.Priority, v.Weight, v.Port, v.Host)
		case "CAA":
			rc, err = dc.NewRecordConfig(label, rec.TTL, dnsv2.TypeCAA, v.Flags, v.Tag, v.Data)
		case "NS":
			rc, err = dc.NewRecordConfig(label, rec.TTL, dnsv2.TypeNS, v.Host)
		case "PTR":
			rc, err = dc.NewRecordConfig(label, rec.TTL, dnsv2.TypePTR, v.System)
		default:
			err = fmt.Errorf("unsupported type")
		}
		if err != nil {
			return nil, fmt.Errorf("CONSTELLIX: record %q/%s: %w", rec.Name, rec.Type, err)
		}
		rc.Original = rec
		recs = append(recs, rc)
	}
	return recs, nil
}

// toNative converts the records of one record set to the body of a create.
// Hostnames are sent fully qualified with the trailing dot, and every value
// is enabled. ALIAS is Constellix's ANAME; skipLookup lets the API accept a
// target that does not resolve yet, such as one created in the same push.
func toNative(records models.Records) (recordRequest, error) {
	first := records[0]
	req := recordRequest{
		Name:  first.GetLabel(),
		Type:  first.Type,
		TTL:   first.TTL,
		Value: make([]map[string]any, 0, len(records)),
	}
	if req.Name == apexLabel {
		req.Name = ""
	}
	if req.Type == "ALIAS" {
		req.Type = "ANAME"
		req.SkipLookup = true
	}

	for _, rc := range records {
		var v map[string]any
		switch rc.Type {
		case "A", "AAAA":
			v = map[string]any{"value": rc.GetRDATA().String()}
		case "TXT":
			v = map[string]any{"value": txtToNative(rc)}
		case "CNAME":
			v = map[string]any{"value": rc.AsCNAME().Target}
		case "ALIAS":
			v = map[string]any{"value": rc.AsALIAS().Target}
		case "MX":
			mx := rc.AsMX()
			v = map[string]any{"server": mx.Mx, "priority": mx.Preference}
		case "SRV":
			srv := rc.AsSRV()
			v = map[string]any{"priority": srv.Priority, "weight": srv.Weight, "port": srv.Port, "host": srv.Target}
		case "CAA":
			caa := rc.AsCAA()
			v = map[string]any{"flags": caa.Flag, "tag": caa.Tag, "data": caa.Value}
		case "NS":
			v = map[string]any{"host": rc.AsNS().Ns}
		case "PTR":
			v = map[string]any{"system": rc.AsPTR().Ptr}
		default:
			return recordRequest{}, fmt.Errorf("CONSTELLIX: record type %s is not supported", rc.Type)
		}
		v["enabled"] = true
		req.Value = append(req.Value, v)
	}
	return req, nil
}

// txtToRC reads a TXT value stored in presentation format. The API splits a
// long value into strings by its escaped length, so the strings can break
// elsewhere than DNSControl would break them; the record is therefore built
// from the joined text, which DNSControl splits the same way as a configured
// one.
func txtToRC(dc *models.DomainConfig, label string, ttl uint32, value string) (*models.RecordConfig, error) {
	parsed, err := dc.NewRecordConfigParse(label, ttl, dnsv2.TypeTXT, value)
	if err != nil {
		return nil, err
	}
	return dc.NewRecordConfig(label, ttl, dnsv2.TypeTXT, parsed.GetTargetTXTJoined())
}

// txtToNative returns a TXT value in presentation format with its strings
// written back to back, the way the API stores them. The API mangles a long
// value whose strings are separated by spaces ("a..." "b" is stored with the
// space inside the second string).
func txtToNative(rc *models.RecordConfig) string {
	var sb strings.Builder
	for _, s := range rc.GetTargetTXTSegmented() {
		sb.WriteString(txtutil.ZoneifyStringQuoted(s))
	}
	return sb.String()
}

// fixTTL lowers a TTL the API would refuse to the largest one it accepts.
func fixTTL(ttl uint32) uint32 {
	return min(ttl, maxTTL)
}
