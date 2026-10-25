package constellix

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/DNSControl/dnscontrol/v5/models"
	"github.com/DNSControl/dnscontrol/v5/pkg/diff2"
	"github.com/DNSControl/dnscontrol/v5/pkg/providers"
)

// supportedTypes are the Constellix record types DNSControl manages.
var supportedTypes = map[string]bool{
	"A":     true,
	"AAAA":  true,
	"ANAME": true,
	"CAA":   true,
	"CNAME": true,
	"MX":    true,
	"NS":    true,
	"PTR":   true,
	"SRV":   true,
	"TXT":   true,
}

// GetZoneRecords gets the records of a zone and returns them in RecordConfig format.
//
// A record that DNSControl could not write back unchanged makes the read fail;
// see checkRecord.
func (p *constellixProvider) GetZoneRecords(dc *models.DomainConfig) (models.Records, error) {
	z, err := p.getZoneForRecords(dc.Name)
	if err != nil {
		return nil, err
	}
	if z.template {
		return nil, fmt.Errorf("CONSTELLIX: zone %s is linked to a template, which DNSControl does not manage; unlink it in the Constellix portal", dc.Name)
	}

	records, err := p.client.listRecords(z.id)
	if err != nil {
		return nil, err
	}

	recs := make(models.Records, 0, len(records))
	for _, rec := range records {
		if err := checkRecord(rec); err != nil {
			return nil, err
		}

		before := providers.BeginToRC(p.observer, "toRC", rec)
		rcs, err := toRC(dc, rec)
		providers.EndToRC(p.observer, "toRC", before, rec, rcs, err)
		if err != nil {
			return nil, err
		}
		recs = append(recs, rcs...)
	}

	return recs, nil
}

// checkRecord refuses a record that uses something DNSControl does not
// model. Reading it as plain records would let the next change to it erase
// the feature, and skipping it would make DNSControl try to create a record
// that already exists, so the user has to resolve it in the Constellix portal
// first. Absent fields count as their defaults.
func checkRecord(rec apiRecord) error {
	fail := func(format string, args ...any) error {
		return fmt.Errorf("CONSTELLIX: record %q/%s %s; remove or convert it in the Constellix portal",
			rec.Name, rec.Type, fmt.Sprintf(format, args...))
	}

	if !supportedTypes[rec.Type] {
		return fail("has a type DNSControl does not manage with this provider")
	}
	if rec.Mode != "" && rec.Mode != "standard" {
		return fail("uses %s mode, which DNSControl does not manage", rec.Mode)
	}
	if rec.Region != "" && rec.Region != "default" {
		return fail("is in the GTD region %q, which DNSControl does not manage", rec.Region)
	}
	if isSet(rec.IPFilter) {
		return fail("uses an IP filter, which DNSControl does not manage")
	}
	if isSet(rec.GeoProximity) {
		return fail("uses GeoProximity, which DNSControl does not manage")
	}
	if rec.Enabled != nil && !*rec.Enabled {
		return fail("is disabled, which DNSControl does not manage")
	}

	values, err := decodeValues(rec)
	if err != nil {
		return fail("has a value that is not a list of standard values (%v)", err)
	}
	if len(values) == 0 {
		return fail("has no values")
	}
	for _, v := range values {
		if v.Enabled != nil && !*v.Enabled {
			return fail("has a disabled value, which DNSControl does not manage")
		}
	}
	return nil
}

// isSet reports whether a raw JSON field is present and not null.
func isSet(raw []byte) bool {
	raw = bytes.TrimSpace(raw)
	return len(raw) > 0 && !bytes.Equal(raw, []byte("null"))
}

// GetZoneRecordsCorrections returns a list of corrections that will turn existing records into dc.Records.
func (p *constellixProvider) GetZoneRecordsCorrections(dc *models.DomainConfig, existing models.Records) ([]*models.Correction, int, error) {
	z, err := p.getZoneForRecords(dc.Name)
	if err != nil {
		return nil, 0, err
	}

	filterApexNS(dc, z.nameservers)

	// Lowered before the diff, not when the request is built: otherwise the
	// stored TTL would never match the desired one and every run would offer
	// the same change again.
	for _, rec := range dc.Records {
		rec.TTL = fixTTL(rec.TTL)
	}

	instructions, actualChangeCount, err := diff2.ByRecordSet(existing, dc, nil)
	if err != nil {
		return nil, 0, err
	}

	var corrections []*models.Correction
	for _, inst := range instructions {
		switch inst.Type {
		case diff2.REPORT:
			corrections = append(corrections, inst.CreateMessage())

		case diff2.CREATE:
			req, err := p.nativeRecord(inst.New)
			if err != nil {
				return nil, 0, err
			}
			corrections = append(corrections, inst.CreateCorrection(func() error {
				return p.client.createRecord(z.id, req)
			}))

		case diff2.CHANGE:
			req, err := p.nativeRecord(inst.New)
			if err != nil {
				return nil, 0, err
			}
			update := recordUpdate{TTL: req.TTL, Value: req.Value, SkipLookup: req.SkipLookup}
			id := recordID(inst.Old)
			corrections = append(corrections, inst.CreateCorrection(func() error {
				return p.client.updateRecord(z.id, id, update)
			}))

		case diff2.DELETE:
			id := recordID(inst.Old)
			corrections = append(corrections, inst.CreateCorrection(func() error {
				return p.client.deleteRecord(z.id, id)
			}))

		default:
			panic(fmt.Sprintf("unhandled inst.Type %s", inst.Type))
		}
	}

	return corrections, actualChangeCount, nil
}

// nativeRecord is toNative reported to the conversion observer.
func (p *constellixProvider) nativeRecord(records models.Records) (recordRequest, error) {
	before := providers.BeginToNative(p.observer, "toNative", records)
	req, err := toNative(records)
	providers.EndToNative(p.observer, "toNative", before, records, req, err)
	return req, err
}

// recordID returns the ID of the existing record object of a record set.
func recordID(existing models.Records) flexID {
	return existing[0].Original.(apiRecord).ID
}

// filterApexNS removes from dc.Records the apex NS records that point at one
// of the zone's own nameservers: Constellix serves those anyway and does not
// list them as records. Only the desired side is filtered. An apex NS record
// set stored in the zone keeps its identity, so adding a foreign nameserver
// to it is an update, and a stored value that repeats a Constellix nameserver
// is removed once.
func filterApexNS(dc *models.DomainConfig, nameservers []string) {
	own := make(map[string]bool, len(nameservers))
	for _, ns := range nameservers {
		own[normalizeHost(ns)] = true
	}

	kept := make(models.Records, 0, len(dc.Records))
	for _, rec := range dc.Records {
		if rec.Type == "NS" && rec.GetLabel() == apexLabel && own[normalizeHost(rec.AsNS().Ns)] {
			continue
		}
		kept = append(kept, rec)
	}
	dc.Records = kept
}

func normalizeHost(name string) string {
	return strings.ToLower(strings.TrimSuffix(name, "."))
}
