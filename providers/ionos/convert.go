package ionos

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/DNSControl/dnscontrol/v5/models"
)

// toRC converts a record returned by the API into a RecordConfig.
func toRC(dc *models.DomainConfig, r record) (*models.RecordConfig, error) {
	label := dc.LabelFromFQDNNoDot(strings.TrimSuffix(r.Name, "."))
	ttl := uint32(r.TTL)

	var rc *models.RecordConfig
	var err error
	switch r.Type {
	case "CNAME", "NS":
		rc, err = dc.NewRecordConfigParse(label, ttl, r.Type, withDot(r.Content))
	case "MX":
		rc, err = dc.NewRecordConfigParse(label, ttl, r.Type, strconv.Itoa(r.Prio)+" "+withDot(r.Content))
	case "SRV":
		rc, err = srvToRC(dc, label, ttl, r)
	case "TXT":
		// The API returns TXT values quoted, possibly split into several strings.
		if strings.HasPrefix(r.Content, `"`) {
			rc, err = dc.NewRecordConfigParse(label, ttl, r.Type, r.Content)
		} else {
			rc, err = dc.NewRecordConfig(label, ttl, r.Type, r.Content)
		}
	default:
		rc, err = dc.NewRecordConfigParse(label, ttl, r.Type, r.Content)
	}
	if err != nil {
		return nil, fmt.Errorf("converting %s record %q (%q): %w", r.Type, r.Name, r.Content, err)
	}
	rc.Original = r
	return rc, nil
}

// srvToRC accepts the priority in the prio field ("weight port target") and in
// the content ("priority weight port target").
func srvToRC(dc *models.DomainConfig, label string, ttl uint32, r record) (*models.RecordConfig, error) {
	fields := strings.Fields(r.Content)
	switch len(fields) {
	case 3:
		return dc.NewRecordConfigParse(label, ttl, r.Type, fmt.Sprintf("%d %s %s %s", r.Prio, fields[0], fields[1], withDot(fields[2])))
	case 4:
		return dc.NewRecordConfigParse(label, ttl, r.Type, fmt.Sprintf("%s %s %s %s", fields[0], fields[1], fields[2], withDot(fields[3])))
	default:
		return nil, fmt.Errorf("expected 3 or 4 fields, got %d", len(fields))
	}
}

// toNative converts a RecordConfig into the record sent to the API.
func toNative(rc *models.RecordConfig) record {
	r := record{
		Name: rc.GetLabelFQDN(),
		Type: rc.Type,
		TTL:  int(rc.TTL),
	}
	switch rc.Type {
	case "CNAME", "NS":
		r.Content = strings.TrimSuffix(rc.GetRDATA().String(), ".")
	case "MX":
		mx := rc.AsMX()
		r.Prio = int(mx.Preference)
		r.Content = strings.TrimSuffix(mx.Mx, ".")
	case "SRV":
		srv := rc.AsSRV()
		r.Prio = int(srv.Priority)
		r.Content = fmt.Sprintf("%d %d %s", srv.Weight, srv.Port, strings.TrimSuffix(srv.Target, "."))
	case "TXT":
		r.Content = rc.GetTargetTXTJoined()
	default:
		r.Content = rc.GetRDATA().String()
	}
	return r
}

func withDot(name string) string {
	if strings.HasSuffix(name, ".") {
		return name
	}
	return name + "."
}
