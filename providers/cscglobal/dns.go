package cscglobal

import (
	"strings"

	"github.com/DNSControl/dnscontrol/v4/models"
	"github.com/DNSControl/dnscontrol/v4/pkg/diff"
)

// GetZoneRecords gets the records of a zone and returns them in RecordConfig format.
func (client *providerClient) GetZoneRecords(dc *models.DomainConfig) (models.Records, error) {
	domain := dc.Name

	records, err := client.getZoneRecordsAll(domain)
	if err != nil {
		return nil, err
	}

	// Convert them to DNScontrol's native format:

	existingRecords := []*models.RecordConfig{}

	// Option 1: One long list.  If your provider returns one long list,
	// convert each one to RecordType like this:
	// for _, rr := range records {
	// 	existingRecords = append(existingRecords, nativeToRecord(rr, domain))
	//}

	// Option 2: Grouped records. Sometimes the provider returns one item per
	// label. Each item contains a list of all the records at that label.
	// You'll need to split them out into one RecordConfig for each record.  An
	// example of this is the ROUTE53 provider.
	// for _, rg := range records {
	// 	for _, rr := range rg {
	// 		existingRecords = append(existingRecords, nativeToRecords(rg, rr, domain)...)
	// 	}
	// }

	// Option 3: Something else.  In this case, we get a big massive structure
	// which needs to be broken up.  Still, we're generating a list of
	// RecordConfig structures.
	defaultTTL := records.Soa.TTL
	for _, rr := range records.A {
		existingRecords = append(existingRecords, nativeToRecordA(rr, domain, defaultTTL))
	}
	for _, rr := range records.Cname {
		existingRecords = append(existingRecords, nativeToRecordCNAME(rr, domain, defaultTTL))
	}
	for _, rr := range records.Aaaa {
		existingRecords = append(existingRecords, nativeToRecordAAAA(rr, domain, defaultTTL))
	}
	for _, rr := range records.Txt {
		existingRecords = append(existingRecords, nativeToRecordTXT(rr, domain, defaultTTL))
	}
	for _, rr := range records.Mx {
		existingRecords = append(existingRecords, nativeToRecordMX(rr, domain, defaultTTL))
	}
	for _, rr := range records.Ns {
		existingRecords = append(existingRecords, nativeToRecordNS(rr, domain, defaultTTL))
	}
	for _, rr := range records.Srv {
		existingRecords = append(existingRecords, nativeToRecordSRV(rr, domain, defaultTTL))
	}
	for _, rr := range records.Caa {
		existingRecords = append(existingRecords, nativeToRecordCAA(rr, domain, defaultTTL))
	}

	return existingRecords, nil
}

func (client *providerClient) GetNameservers(domain string) ([]*models.Nameserver, error) {
	nss, err := client.getNameservers(domain)
	if err != nil {
		return nil, err
	}
	return models.ToNameservers(nss)
}

// GetZoneRecordsCorrections returns a list of corrections that will turn existing records into dc.Records.
func (client *providerClient) GetZoneRecordsCorrections(dc *models.DomainConfig, foundRecords models.Records) ([]*models.Correction, int, error) {
	toReport, creates, dels, modifications, actualChangeCount, err := diff.NewCompat(dc).IncrementalDiff(foundRecords)
	if err != nil {
		return nil, 0, err
	}
	// Start corrections with the reports
	corrections := diff.GenerateMessageCorrections(toReport)

	// CSCGlobal has a unique API.  A list of edits is sent in one API
	// call. Edits aren't permitted if an existing edit is being
	// processed. Therefore, before we do an edit we block until the
	// previous edit is done executing.

	var edits []zoneResourceRecordEdit
	var descriptions []string
	for _, del := range dels {
		edits = append(edits, makePurge(del))
		descriptions = append(descriptions, del.String())
	}
	for _, cre := range creates {
		edits = append(edits, makeAdd(cre))
		descriptions = append(descriptions, cre.String())
	}
	for _, m := range modifications {
		edits = append(edits, makeEdit(m))
		descriptions = append(descriptions, m.String())
	}
	if len(edits) > 0 {
		c := &models.Correction{
			Msg: "\t" + strings.Join(descriptions, "\n\t"),
			F: func() error {
				// CSCGlobal's API only permits one pending update at a time.
				// Therefore we block until any outstanding updates are done.
				// We also clear out any failures, since (and I can't believe
				// I'm writing this) any time something fails, the failure has
				// to be cleared out with an additional API call.

				err := client.clearRequests(dc.Name)
				if err != nil {
					return err
				}
				return client.sendZoneEditRequest(dc.Name, edits)
			},
		}
		corrections = append(corrections, c)
	}

	return corrections, actualChangeCount, nil
}

func makePurge(cor diff.Correlation) zoneResourceRecordEdit {
	var existingTarget string

	switch cor.Existing.Type {
	case "TXT":
		existingTarget = cor.Existing.GetTargetTXTJoined()
	default:
		existingTarget = cor.Existing.GetTargetField()
	}

	zer := zoneResourceRecordEdit{
		Action:       "PURGE",
		RecordType:   cor.Existing.Type,
		CurrentKey:   cor.Existing.Name,
		CurrentValue: existingTarget,
	}

	if cor.Existing.Type == "CAA" {
		tagValue := cor.Existing.CaaTag
		// printer.Printf("DEBUG: CAA TAG = %q\n", tagValue)
		zer.CurrentTag = &tagValue
	}

	return zer
}

func makeAdd(cre diff.Correlation) zoneResourceRecordEdit {
	rec := cre.Desired

	var recTarget string
	switch rec.Type {
	case "TXT":
		recTarget = rec.GetTargetTXTJoined()
	default:
		recTarget = rec.GetTargetField()
	}

	zer := zoneResourceRecordEdit{
		Action:     "ADD",
		RecordType: rec.Type,
		NewKey:     rec.Name,
		NewValue:   recTarget,
		NewTTL:     rec.TTL,
	}

	switch rec.Type {
	case "CAA":
		tagValue := rec.CaaTag
		flagValue := rec.CaaFlag
		zer.NewTag = &tagValue
		zer.NewFlag = &flagValue
	case "MX":
		zer.NewPriority = rec.MxPreference
	case "SRV":
		zer.NewPriority = rec.SrvPriority
		zer.NewWeight = rec.SrvWeight
		zer.NewPort = rec.SrvPort
	case "TXT":
		zer.NewValue = rec.GetTargetTXTJoined()
	default: // "A", "CNAME", "NS"
		// Nothing to do.
	}

	return zer
}

func makeEdit(m diff.Correlation) zoneResourceRecordEdit {
	old, rec := m.Existing, m.Desired
	// TODO: Assert that old.Type == rec.Type
	// TODO: Assert that old.Name == rec.Name

	var oldTarget, recTarget string
	switch old.Type {
	case "TXT":
		oldTarget = old.GetTargetTXTJoined()
		recTarget = rec.GetTargetTXTJoined()
	default:
		oldTarget = old.GetTargetField()
		recTarget = rec.GetTargetField()
	}

	zer := zoneResourceRecordEdit{
		Action:       "EDIT",
		RecordType:   old.Type,
		CurrentKey:   old.Name,
		CurrentValue: oldTarget,
	}
	if oldTarget != recTarget {
		zer.NewValue = recTarget
	}
	if old.TTL != rec.TTL {
		zer.NewTTL = rec.TTL
	}

	switch old.Type {
	case "CAA":
		tagValue := old.CaaTag
		zer.CurrentTag = &tagValue
		if old.CaaTag != rec.CaaTag || old.CaaFlag != rec.CaaFlag || old.TTL != rec.TTL {
			// If anything changed, we need to update both tag and flag.
			zer.NewTag = &(rec.CaaTag)
			zer.NewFlag = &(rec.CaaFlag)
		}
	case "MX":
		if old.MxPreference != rec.MxPreference {
			zer.NewPriority = rec.MxPreference
		}
	case "SRV":
		zer.NewWeight = rec.SrvWeight
		zer.NewPort = rec.SrvPort
		zer.NewPriority = rec.SrvPriority
	default: // "A", "CNAME", "NS", "TXT"
		// Nothing to do.
	}

	return zer
}
