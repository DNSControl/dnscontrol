package ns1

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/DNSControl/dnscontrol/v4/models"
	"github.com/DNSControl/dnscontrol/v4/pkg/diff2"
	"github.com/DNSControl/dnscontrol/v4/pkg/printer"
	"gopkg.in/ns1/ns1-go.v2/rest"
	"gopkg.in/ns1/ns1-go.v2/rest/model/dns"
	"gopkg.in/ns1/ns1-go.v2/rest/model/filter"
)

// GetZoneRecords gets the records of a zone and returns them in RecordConfig format.
func (n *nsone) GetZoneRecords(dc *models.DomainConfig) (models.Records, error) {
	domain := dc.Name

	z, _, err := n.Zones.Get(domain, true)
	if err != nil && errors.Is(err, rest.ErrZoneMissing) {
		// if we get here, zone wasn't created, but we ended up continuing regardless.
		// This should be revisited, but for now let's get out early with a relevant message
		// one case: preview --no-populate
		printer.Warnf("GetZonerecords: Zone %s not created in NS1. Either create manually or ensure dnscontrol can create it.\n", domain)
		return nil, err
	}
	if err != nil {
		return nil, err
	}

	found := models.Records{}
	for _, r := range z.Records {
		zrs, err := convert(r, domain)
		if err != nil {
			return nil, err
		}
		found = append(found, zrs...)
	}
	return found, nil
}

// GetZoneRecordsCorrections returns a list of corrections that will turn existing records into dc.Records.
func (n *nsone) GetZoneRecordsCorrections(dc *models.DomainConfig, existingRecords models.Records) ([]*models.Correction, int, error) {
	var corrections []*models.Correction
	domain := dc.Name

	// add DNSSEC-related corrections
	if dnssecCorrections := n.getDomainCorrectionsDNSSEC(domain, dc.AutoDNSSEC); dnssecCorrections != nil {
		corrections = append(corrections, dnssecCorrections)
	}

	changes, actualChangeCount, err := diff2.ByRecordSet(existingRecords, dc, nil)
	if err != nil {
		return nil, 0, err
	}

	for _, change := range changes {
		key := change.Key
		recs := change.New
		desc := strings.Join(change.Msgs, "\n")

		switch change.Type {
		case diff2.REPORT:
			corrections = append(corrections, &models.Correction{Msg: change.MsgsJoined})
		case diff2.CREATE:
			corrections = append(corrections, &models.Correction{
				Msg: desc,
				F:   func() error { return n.add(recs, dc.Name) },
			})
		case diff2.CHANGE:
			corrections = append(corrections, &models.Correction{
				Msg: desc,
				F:   func() error { return n.modify(recs, dc.Name) },
			})
		case diff2.DELETE:
			corrections = append(corrections, &models.Correction{
				Msg: desc,
				F:   func() error { return n.remove(key, dc.Name) },
			})
		default:
			panic(fmt.Sprintf("unhandled inst.Type %s", change.Type))
		}
	}
	return corrections, actualChangeCount, nil
}

func (n *nsone) add(recs models.Records, domain string) error {
	for rtr := 0; ; rtr++ {
		httpResp, err := n.Records.Create(buildRecord(recs, domain, ""))
		if httpResp.StatusCode == http.StatusTooManyRequests && rtr < clientRetries {
			continue
		}
		return err
	}
}

func (n *nsone) remove(key models.RecordKey, domain string) error {
	for rtr := 0; ; rtr++ {
		httpResp, err := n.Records.Delete(domain, key.NameFQDN, key.Type)
		if httpResp.StatusCode == http.StatusTooManyRequests && rtr < clientRetries {
			continue
		}
		return err
	}
}

func (n *nsone) modify(recs models.Records, domain string) error {
	for rtr := 0; ; rtr++ {
		httpResp, err := n.Records.Update(buildRecord(recs, domain, ""))
		if httpResp.StatusCode == http.StatusTooManyRequests && rtr < clientRetries {
			continue
		}
		return err
	}
}

func buildRecord(recs models.Records, domain string, id string) *dns.Record {
	r := recs[0]
	rec := &dns.Record{
		Domain:  r.GetLabelFQDN(),
		Type:    r.Type,
		ID:      id,
		TTL:     int(r.TTL),
		Zone:    domain,
		Filters: []*filter.Filter{}, // Work through a bug in the NS1 API library that causes 400 Input validation failed (Value None for field '<obj>.filters' is not of type array)
	}
	for _, r := range recs {
		switch r.Type {
		case "MX":
			rec.AddAnswer(&dns.Answer{Rdata: strings.Fields(fmt.Sprintf("%d %v", r.MxPreference, r.GetTargetField()))})
		case "TXT":
			rec.AddAnswer(&dns.Answer{Rdata: []string{r.GetTargetTXTJoined()}})
		case "CAA":
			rec.AddAnswer(&dns.Answer{
				Rdata: []string{
					strconv.FormatUint(uint64(r.CaaFlag), 10),
					r.CaaTag,
					r.GetTargetField(),
				},
			})
		case "SRV":
			rec.AddAnswer(&dns.Answer{Rdata: strings.Fields(fmt.Sprintf("%d %d %d %v", r.SrvPriority, r.SrvWeight, r.SrvPort, r.GetTargetField()))})
		case "NAPTR":
			rec.AddAnswer(&dns.Answer{Rdata: []string{
				strconv.Itoa(int(r.NaptrOrder)),
				strconv.Itoa(int(r.NaptrPreference)),
				r.NaptrFlags,
				r.NaptrService,
				r.NaptrRegexp,
				r.GetTargetField(),
			}})
		case "DS":
			rec.AddAnswer(&dns.Answer{Rdata: []string{
				strconv.Itoa(int(r.DsKeyTag)),
				strconv.Itoa(int(r.DsAlgorithm)),
				strconv.Itoa(int(r.DsDigestType)),
				r.DsDigest,
			}})
		case "SVCB", "HTTPS":
			rec.AddAnswer(&dns.Answer{Rdata: []string{
				strconv.Itoa(int(r.SvcPriority)),
				r.GetTargetField(),
				r.SvcParams,
			}})
		case "TLSA":
			rec.AddAnswer(&dns.Answer{Rdata: []string{
				strconv.Itoa(int(r.TlsaUsage)),
				strconv.Itoa(int(r.TlsaSelector)),
				strconv.Itoa(int(r.TlsaMatchingType)),
				r.GetTargetField(),
			}})
		default:
			rec.AddAnswer(&dns.Answer{Rdata: strings.Fields(r.GetTargetField())})
		}
	}
	return rec
}

func convert(zr *dns.ZoneRecord, domain string) ([]*models.RecordConfig, error) {
	found := []*models.RecordConfig{}
	for _, ans := range zr.ShortAns {
		rec := &models.RecordConfig{
			TTL:      uint32(zr.TTL),
			Original: zr,
		}
		rec.SetLabelFromFQDN(zr.Domain, domain)
		switch rtype := zr.Type; rtype {
		case "DNSKEY", "RRSIG":
			// if a zone is enabled for DNSSEC, NS1 autoconfigures DNSKEY & RRSIG records.
			// these entries are not modifiable via the API though, so we have to ignore them while converting.
			// 	ie. API returns "405 Operation on DNSSEC record is not allowed" on such operations
			continue
		case "ALIAS":
			rec.Type = rtype
			if err := rec.SetTarget(ans); err != nil {
				return nil, fmt.Errorf("unparsable %s record received from ns1: %w", rtype, err)
			}
		case "CAA":
			// dnscontrol expects quotes around multivalue CAA entries, API doesn't add them
			xAns := strings.SplitN(ans, " ", 3)
			if err := rec.SetTargetCAAStrings(xAns[0], xAns[1], xAns[2]); err != nil {
				return nil, fmt.Errorf("unparsable %s record received from ns1: %w", rtype, err)
			}
		case "NAPTR":
			// NB(tlim): This is a stupid hack.  NS1 doesn't quote a missing
			// parameter properly. Therefore we look for 2 spaces and assume there is
			// a missing item.
			ans = strings.ReplaceAll(ans, "  ", ` "" `)
			if err := rec.PopulateFromString(rtype, ans, domain); err != nil {
				return nil, fmt.Errorf("unparsable record received from ns1: %w", err)
			}
		case "REDIRECT":
			// NS1 returns REDIRECTs as records, but there is only one and dummy answer:
			// "NS1 MANAGED RECORD"
			// Redirects are managed via a different API endpoint https://api.nsone.net/v1/redirect
			// It also involves cert management
			// We may simpply ignore REDIRECTs for now until we support it
			printer.Warnf("NS1_REDIRECT is NOT supported by dnscontrol and all existing redirects are ignored.\n")
			continue
		default:
			if err := rec.PopulateFromString(rtype, ans, domain); err != nil {
				return nil, fmt.Errorf("unparsable record received from ns1: %w", err)
			}
		}
		found = append(found, rec)
	}
	return found, nil
}
