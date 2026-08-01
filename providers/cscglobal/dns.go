package cscglobal

import (
	"fmt"
	"strings"

	"github.com/DNSControl/dnscontrol/v5/models"
	"github.com/DNSControl/dnscontrol/v5/pkg/diff2"
)

// GetZoneRecords gets the records of a zone and returns them in RecordConfig format.
func (client *providerClient) GetZoneRecords(dc *models.DomainConfig) (models.Records, error) {
	domain := dc.Name

	records, err := client.getZoneRecordsAll(domain)
	if err != nil {
		return nil, err
	}

	// Convert them to DNScontrol's native format:

	existingRecords := models.Records{}

	defaultTTL := records.Soa.TTL
	for _, rr := range records.A {
		existingRecords = append(existingRecords, nativeToRecordA(rr, dc, defaultTTL))
	}
	for _, rr := range records.Cname {
		existingRecords = append(existingRecords, nativeToRecordCNAME(rr, dc, defaultTTL))
	}
	for _, rr := range records.Aaaa {
		existingRecords = append(existingRecords, nativeToRecordAAAA(rr, dc, defaultTTL))
	}
	for _, rr := range records.Txt {
		existingRecords = append(existingRecords, nativeToRecordTXT(rr, dc, defaultTTL))
	}
	for _, rr := range records.Mx {
		existingRecords = append(existingRecords, nativeToRecordMX(rr, dc, defaultTTL))
	}
	for _, rr := range records.Ns {
		existingRecords = append(existingRecords, nativeToRecordNS(rr, dc, defaultTTL))
	}
	for _, rr := range records.Srv {
		existingRecords = append(existingRecords, nativeToRecordSRV(rr, dc, defaultTTL))
	}
	for _, rr := range records.Caa {
		existingRecords = append(existingRecords, nativeToRecordCAA(rr, dc, defaultTTL))
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
	changes, actualChangeCount, err := diff2.ByRecord(foundRecords, dc, nil)
	if err != nil {
		return nil, 0, err
	}

	var corrections []*models.Correction
	var creates, dels, modifications diff2.ChangeList
	for _, change := range changes {
		switch change.Type {
		case diff2.REPORT:
			corrections = append(corrections, &models.Correction{Msg: change.MsgsJoined})
		case diff2.CREATE:
			creates = append(creates, change)
		case diff2.DELETE:
			dels = append(dels, change)
		case diff2.CHANGE:
			modifications = append(modifications, change)
		default:
			panic(fmt.Sprintf("unhandled change.Type %s", change.Type))
		}
	}

	// CSCGlobal has a unique API.  A list of edits is sent in one API
	// call. Edits aren't permitted if an existing edit is being
	// processed. Therefore, before we do an edit we block until the
	// previous edit is done executing.

	var edits []zoneResourceRecordEdit
	var descriptions []string
	for _, del := range dels {
		edits = append(edits, makePurge(del.Old[0]))
		descriptions = append(descriptions, del.MsgsJoined)
	}
	for _, cre := range creates {
		edits = append(edits, makeAdd(cre.New[0]))
		descriptions = append(descriptions, cre.MsgsJoined)
	}
	for _, m := range modifications {
		edits = append(edits, makeEdit(m.Old[0], m.New[0]))
		descriptions = append(descriptions, m.MsgsJoined)
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

func makePurge(existing *models.RecordConfig) zoneResourceRecordEdit {
	var existingTarget string

	switch existing.Type {
	case "TXT":
		existingTarget = existing.GetTargetTXTJoined()
	default:
		existingTarget = existing.GetRDATA().String()
	}

	zer := zoneResourceRecordEdit{
		Action:       "PURGE",
		RecordType:   existing.Type,
		CurrentKey:   existing.Name,
		CurrentValue: existingTarget,
	}

	if existing.Type == "CAA" {
		tagValue := existing.AsCAA().Tag
		// printer.Printf("DEBUG: CAA TAG = %q\n", tagValue)
		zer.CurrentTag = &tagValue
	}

	return zer
}

func makeAdd(rec *models.RecordConfig) zoneResourceRecordEdit {

	zer := zoneResourceRecordEdit{
		Action:     "ADD",
		RecordType: rec.Type,
		NewKey:     rec.Name,
		NewTTL:     rec.TTL,
	}

	switch rec.Type {
	case "CAA":
		f := rec.AsCAA()
		tagValue := f.Tag
		flagValue := f.Flag
		zer.NewTag = &tagValue
		zer.NewFlag = &flagValue
		zer.NewValue = f.Value
	case "MX":
		f := rec.AsMX()
		zer.NewPriority = f.Preference
		zer.NewValue = f.Mx
	case "SRV":
		f := rec.AsSRV()
		zer.NewPriority = f.Priority
		zer.NewWeight = f.Weight
		zer.NewPort = f.Port
		zer.NewValue = f.Target
	case "TXT":
		zer.NewValue = rec.GetTargetTXTJoined()
	default:
		zer.NewValue = rec.GetRDATA().String()
	}

	return zer
}

func makeEdit(old, rec *models.RecordConfig) zoneResourceRecordEdit {
	if old.Type != rec.Type {
		panic(fmt.Sprintf("record type mismatch: %q != %q", old.Type, rec.Type))
	}
	if old.Name != rec.Name {
		panic(fmt.Sprintf("record name mismatch: %q != %q", old.Name, rec.Name))
	}

	var oldTarget, recTarget string
	switch old.Type {
	case "TXT":
		oldTarget = old.GetTargetTXTJoined()
		recTarget = rec.GetTargetTXTJoined()
	default:
		oldTarget = old.GetRDATA().String()
		recTarget = rec.GetRDATA().String()
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
		of := old.AsCAA()
		tagValue := of.Tag
		zer.CurrentTag = &tagValue
		if old.AsCAA().Tag != rec.AsCAA().Tag || old.AsCAA().Flag != rec.AsCAA().Flag || old.TTL != rec.TTL {
			// If anything changed, we need to update both tag and flag.
			zer.NewFlag = new(rec.AsCAA().Flag)
			zer.NewTag = new(rec.AsCAA().Tag)
			zer.NewFlag = new(rec.AsCAA().Flag)
		}
	case "MX":
		if old.AsMX().Preference != rec.AsMX().Preference {
			zer.NewPriority = rec.AsMX().Preference
		}
	case "SRV":
		f := rec.AsSRV()
		zer.NewWeight = f.Weight
		zer.NewPort = f.Port
		zer.NewPriority = f.Priority
	default:
		// Nothing to do.
	}

	return zer
}
