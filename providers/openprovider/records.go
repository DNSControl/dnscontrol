package openprovider

import (
	"fmt"
	"sort"
	"strings"

	"github.com/DNSControl/dnscontrol/v5/models"
	"github.com/DNSControl/dnscontrol/v5/pkg/diff2"
	"github.com/DNSControl/dnscontrol/v5/pkg/printer"
)

var acceptedTTLs = [...]uint32{900, 3600, 10800, 21600, 43200, 86400}

// GetZoneRecords returns the records in an OpenProvider zone. Provider-managed
// SOA and NS records are omitted because the API will not mutate them.
func (p *openproviderProvider) GetZoneRecords(dc *models.DomainConfig) (models.Records, error) {
	zone, err := p.client.getZone(dc.Name)
	if err != nil {
		return nil, err
	}
	apiRecords, err := p.client.listRecords(*zone)
	if err != nil {
		return nil, err
	}

	records := make(models.Records, 0, len(apiRecords))
	for _, apiRecord := range apiRecords {
		if apiRecord.Type == "SOA" || apiRecord.Type == "NS" {
			continue
		}
		record, err := toRecordConfig(apiRecord, dc.Name)
		if err != nil {
			return nil, err
		}
		records = append(records, record)
	}
	return records, nil
}

// GetZoneRecordsCorrections computes individual OpenProvider record mutations.
func (p *openproviderProvider) GetZoneRecordsCorrections(dc *models.DomainConfig, existing models.Records) ([]*models.Correction, int, error) {
	zone, err := p.client.getZone(dc.Name)
	if err != nil {
		return nil, 0, err
	}
	desired := providerDomainConfig(dc)
	filterApexNS(desired)
	normalizeTTLs(desired.Records)

	// OpenProvider's update endpoint accepts individual record operations. Using
	// ByRecord is important here: ByRecordSet can contain several old and new
	// records, and pairing those slices by position can update or remove the
	// wrong record when a set is changed.
	changes, actualChangeCount, err := diff2.ByRecord(existing, desired, nil)
	if err != nil {
		return nil, 0, err
	}

	corrections := make([]*models.Correction, 0, len(changes))
	for _, change := range changes {
		switch change.Type {
		case diff2.REPORT:
			corrections = append(corrections, change.CreateMessage())
		case diff2.CREATE:
			newRecord := fromRecordConfig(change.New[0])
			corrections = append(corrections, change.CreateCorrection(func() error {
				return p.client.updateZoneRecords(*zone, recordUpdates{Add: []apiRecord{newRecord}})
			}))
		case diff2.CHANGE:
			oldRecord, ok := change.Old[0].Original.(apiRecord)
			if !ok {
				return nil, 0, errorsMissingOriginal(change.Old[0])
			}
			update := recordUpdate{
				Original: normalizeAPIRecordName(oldRecord, zone.Name),
				Record:   fromRecordConfig(change.New[0]),
			}
			corrections = append(corrections, change.CreateCorrection(func() error {
				return p.client.updateZoneRecords(*zone, recordUpdates{Update: []recordUpdate{update}})
			}))
		case diff2.DELETE:
			oldRecord, ok := change.Old[0].Original.(apiRecord)
			if !ok {
				return nil, 0, errorsMissingOriginal(change.Old[0])
			}
			oldRecord = normalizeAPIRecordName(oldRecord, zone.Name)
			corrections = append(corrections, change.CreateCorrection(func() error {
				return p.client.updateZoneRecords(*zone, recordUpdates{Remove: []apiRecord{oldRecord}})
			}))
		default:
			panic(fmt.Sprintf("unhandled change type %s", change.Type))
		}
	}
	return corrections, actualChangeCount, nil
}

// updateZoneRecords sends one record modifier per request. Although the
// request shape contains arrays, OpenProvider processes only one item from a
// modifier reliably. Sending each operation separately is also necessary for
// multiple TXT records at the same owner name.
func (c *apiClient) updateZoneRecords(zone apiZone, updates recordUpdates) error {
	for _, record := range updates.Add {
		if err := c.updateZone(zone, recordUpdates{Add: []apiRecord{record}}); err != nil {
			return err
		}
	}
	for _, record := range updates.Remove {
		if err := c.updateZone(zone, recordUpdates{Remove: []apiRecord{record}}); err != nil {
			return err
		}
	}
	for _, update := range updates.Update {
		if err := c.updateZone(zone, recordUpdates{Update: []recordUpdate{update}}); err != nil {
			return err
		}
	}
	return nil
}

// normalizeAPIRecordName converts the fully-qualified owner names returned by
// OpenProvider into the relative names required by its zone modifier API.
func normalizeAPIRecordName(record apiRecord, origin string) apiRecord {
	record.Name = relativeRecordName(record.Name, origin)
	return record
}

func relativeRecordName(name, origin string) string {
	name = strings.TrimSuffix(name, ".")
	origin = strings.TrimSuffix(origin, ".")
	if isApexRecordName(name, origin) {
		return ""
	}
	suffix := "." + origin
	if strings.HasSuffix(strings.ToLower(name), strings.ToLower(suffix)) {
		return name[:len(name)-len(suffix)]
	}
	return name
}

// filterApexNS removes provider-managed apex NS records from the desired
// state. DNSControl injects the authoritative nameservers into every desired
// zone, but OpenProvider owns those records and does not accept their updates.
func filterApexNS(dc *models.DomainConfig) {
	declared := make(map[string]struct{}, len(dc.Nameservers))
	for _, ns := range dc.Nameservers {
		declared[strings.TrimSuffix(ns.Name, ".")] = struct{}{}
	}

	kept := make([]*models.RecordConfig, 0, len(dc.Records))
	for _, record := range dc.Records {
		if record.Type == "NS" && record.GetLabel() == apexLabel {
			target := strings.TrimSuffix(record.AsNS().Ns, ".")
			if _, ok := declared[target]; !ok {
				printer.Warnf("OpenProvider does not support changing apex NS records. %s will not be added.\n", record.AsNS().Ns)
			}
			continue
		}
		kept = append(kept, record)
	}
	dc.Records = kept
}

func errorsMissingOriginal(record *models.RecordConfig) error {
	return fmt.Errorf("OPENPROVIDER: %s record %q is missing its original API representation", record.Type, record.GetLabel())
}

func normalizeTTLs(records models.Records) {
	for _, record := range records {
		normalized := normalizeTTL(record.TTL)
		if record.TTL == normalized {
			continue
		}
		printer.Warnf("OpenProvider only accepts TTLs of 900, 3600, 10800, 21600, 43200, or 86400 seconds. Setting %s %s from %d to %d.\n", record.GetLabelFQDN(), record.Type, record.TTL, normalized)
		record.TTL = normalized
	}
}

func normalizeTTL(ttl uint32) uint32 {
	for _, accepted := range acceptedTTLs {
		if ttl <= accepted {
			return accepted
		}
	}
	return 86400
}

// providerDomainConfig prevents provider-specific filtering and TTL
// normalization from changing the desired state used by other providers.
func providerDomainConfig(dc *models.DomainConfig) *models.DomainConfig {
	desired := &models.DomainConfig{
		Name:              dc.Name,
		Nameservers:       dc.Nameservers,
		EnsureAbsent:      dc.EnsureAbsent,
		KeepUnknown:       dc.KeepUnknown,
		Unmanaged:         dc.Unmanaged,
		UnmanagedUnsafe:   dc.UnmanagedUnsafe,
		IgnoreExternalDNS: dc.IgnoreExternalDNS,
		ExternalDNSPrefix: dc.ExternalDNSPrefix,
	}
	desired.Records = make(models.Records, len(dc.Records))
	for i, record := range dc.Records {
		copy := *record
		desired.Records[i] = &copy
	}
	return desired
}

// GetNameservers returns the provider-managed authoritative NS records.
func (p *openproviderProvider) GetNameservers(domain string) ([]*models.Nameserver, error) {
	zone, err := p.client.getZone(domain)
	if err != nil {
		return nil, err
	}
	records, err := p.client.listRecords(*zone)
	if err != nil {
		return nil, err
	}

	unique := map[string]bool{}
	for _, record := range records {
		if record.Type == "NS" && isApexRecordName(record.Name, domain) {
			unique[strings.TrimSuffix(record.Value, ".")] = true
		}
	}
	names := make([]string, 0, len(unique))
	for name := range unique {
		names = append(names, name)
	}
	sort.Strings(names)
	return models.ToNameservers(names)
}

// ListZones returns all DNS zones visible to the OpenProvider account.
func (p *openproviderProvider) ListZones() ([]string, error) {
	zones, err := p.client.listZones()
	if err != nil {
		return nil, err
	}
	names := make([]string, 0, len(zones))
	for _, zone := range zones {
		if zone.Name != "" {
			names = append(names, strings.TrimSuffix(zone.Name, "."))
		}
	}
	sort.Strings(names)
	return names, nil
}
