package namedotcom

import (
	"fmt"

	dnsv2 "codeberg.org/miekg/dns"
	"github.com/DNSControl/dnscontrol/v5/models"
	"github.com/DNSControl/dnscontrol/v5/pkg/diff2"
	"github.com/DNSControl/dnscontrol/v5/pkg/nrc"
	"github.com/DNSControl/dnscontrol/v5/pkg/privatetypes"
	"github.com/DNSControl/dnscontrol/v5/pkg/providers"
	"github.com/namedotcom/go/namecom"
)

// GetZoneRecords gets the records of a zone and returns them in RecordConfig format.
func (n *namedotcomProvider) GetZoneRecords(dc *models.DomainConfig) (models.Records, error) {
	domain := dc.Name

	records, err := n.getRecords(domain)
	if err != nil {
		return nil, err
	}

	actual := make([]*models.RecordConfig, len(records))
	for i, r := range records {
		before := providers.BeginToRC(n.observer, "toRecord", r)
		actual[i], err = toRecord(r, dc)
		providers.EndToRC(n.observer, "toRecord", before, r, models.Records{actual[i]}, err)
		if err != nil {
			return nil, err
		}
	}

	return actual, nil
}

// GetZoneRecordsCorrections returns a list of corrections that will turn existing records into dc.Records.
func (n *namedotcomProvider) GetZoneRecordsCorrections(dc *models.DomainConfig, actual models.Records) ([]*models.Correction, int, error) {
	checkNSModifications(dc)

	changes, actualChangeCount, err := diff2.ByRecord(actual, dc, nil)
	if err != nil {
		return nil, 0, err
	}

	var corrections []*models.Correction
	for _, change := range changes {
		switch change.Type {
		case diff2.REPORT:
			corrections = append(corrections, &models.Correction{Msg: change.MsgsJoined})
		case diff2.CREATE:
			newRec := change.New[0]
			corrections = append(corrections, &models.Correction{
				Msg: change.MsgsJoined,
				F:   func() error { return n.createRecord(newRec, dc.Name) },
			})
		case diff2.DELETE:
			oldRec := change.Old[0].Original.(*namecom.Record)
			corrections = append(corrections, &models.Correction{
				Msg: change.MsgsJoined,
				F:   func() error { return n.deleteRecord(oldRec.ID, dc.Name) },
			})
		case diff2.CHANGE:
			// name.com has no update API; modify by deleting then recreating.
			oldRec := change.Old[0].Original.(*namecom.Record)
			newRec := change.New[0]
			corrections = append(corrections, &models.Correction{
				Msg: change.MsgsJoined,
				F: func() error {
					if err := n.deleteRecord(oldRec.ID, dc.Name); err != nil {
						return err
					}
					return n.createRecord(newRec, dc.Name)
				},
			})
		default:
			panic(fmt.Sprintf("unhandled change.Type %s", change.Type))
		}
	}

	return corrections, actualChangeCount, nil
}

func checkNSModifications(dc *models.DomainConfig) {
	newList := make([]*models.RecordConfig, 0, len(dc.Records))
	for _, rec := range dc.Records {
		if rec.Type == "NS" && rec.GetLabel() == "@" {
			continue // Apex NS records are automatically created for the domain's nameservers and cannot be managed otherwise via the name.com API.
		}
		newList = append(newList, rec)
	}
	dc.Records = newList
}

func toRecord(r *namecom.Record, dc *models.DomainConfig) (*models.RecordConfig, error) {
	label := dc.LabelFromFQDNWithDot(r.Fqdn)

	var rc *models.RecordConfig
	var err error
	switch rtype := r.Type; rtype { // #rtype_variations
	case "ANAME":
		rc, err = dc.NewRecordConfig(label, r.TTL, privatetypes.TypeALIAS, r.Answer)
	case "MX":
		rc, err = dc.NewRecordConfig(label, r.TTL, dnsv2.TypeMX, r.Priority, r.Answer)
	case "SRV":
		rc, err = dc.NewRecordConfig(label, r.TTL, dnsv2.TypeSRV, r.Priority, r.Answer,
			nrc.Flags{SrvWeirdSplit: true, TargetIsFqdnNoDot: true})
	default:
		rc, err = dc.NewRecordConfigParse(label, r.TTL, rtype, r.Answer,
			nrc.Flags{TxtDontParse: true})
	}
	if err != nil {
		return nil, fmt.Errorf("unparsable record received from ndc: %w", err)
	}

	rc.Original = r

	return rc, nil
}

func (n *namedotcomProvider) getRecords(domain string) ([]*namecom.Record, error) {
	var (
		err      error
		records  []*namecom.Record
		response *namecom.ListRecordsResponse
	)

	request := &namecom.ListRecordsRequest{
		DomainName: domain,
		Page:       1,
	}

	for request.Page > 0 {
		response, err = n.client.ListRecords(request)
		if err != nil {
			return nil, err
		}

		records = append(records, response.Records...)
		request.Page = response.NextPage
	}

	for _, rc := range records {
		if rc.Type == "CNAME" || rc.Type == "ANAME" || rc.Type == "MX" || rc.Type == "NS" {
			rc.Answer = rc.Answer + "."
		}
	}
	return records, nil
}

func (n *namedotcomProvider) createRecord(rc *models.RecordConfig, domain string) error {
	record := toNative(rc, domain)
	_, err := n.client.CreateRecord(record)
	return err
}

func toNative(rc *models.RecordConfig, domain string) *namecom.Record {

	rtype := rc.Type
	var answer string
	var priority uint32

	switch rc.TypeNum {
	case privatetypes.TypeALIAS:
		// NDC uses "ANAME" for aliases. We switch .Type at the last chance.
		rtype = "ANAME"
		answer = rc.AsALIAS().Target
	case dnsv2.TypeTXT:
		answer = rc.GetTargetTXTJoined()
	case dnsv2.TypeMX:
		f := rc.AsMX()
		priority = uint32(f.Preference)
		answer = f.Mx
	case dnsv2.TypeSRV:
		// SRV records with empty targets are not supported (as of 2019-11-05, the API returns 'Parameter Value Error - Invalid Srv Format'
		f := rc.AsSRV()
		priority = uint32(f.Priority)
		answer = fmt.Sprintf("%d %d %v", f.Weight, f.Port, f.Target)
	default:
		answer = rc.GetRDATA().String()
	}

	return &namecom.Record{
		DomainName: domain,
		Host:       rc.GetLabel(),
		Type:       rtype,
		Answer:     answer,
		TTL:        rc.TTL,
		Priority:   priority,
	}
}

func (n *namedotcomProvider) deleteRecord(id int32, domain string) error {
	request := &namecom.DeleteRecordRequest{
		DomainName: domain,
		ID:         id,
	}

	_, err := n.client.DeleteRecord(request)
	return err
}
