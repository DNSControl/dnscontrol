package ns1

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	dnsv2 "codeberg.org/miekg/dns"
	"github.com/DNSControl/dnscontrol/v5/models"
	"github.com/DNSControl/dnscontrol/v5/pkg/diff2"
	"github.com/DNSControl/dnscontrol/v5/pkg/nrc"
	"github.com/DNSControl/dnscontrol/v5/pkg/printer"
	"github.com/DNSControl/dnscontrol/v5/pkg/providers"
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
		before := providers.BeginToRC(n.observer, "convert", r)
		zrs, err := convert(r, dc)
		providers.EndToRC(n.observer, "convert", before, r, zrs, err)
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
		switch r.TypeNum {
		case dnsv2.TypeMX:
			f := r.AsMX()
			rec.AddAnswer(&dns.Answer{Rdata: []string{strconv.FormatInt(int64(f.Preference), 10), f.Mx}})
		case dnsv2.TypeTXT:
			rec.AddAnswer(&dns.Answer{Rdata: []string{r.GetTargetTXTJoined()}})
		case dnsv2.TypeCAA:
			f := r.AsCAA()
			rec.AddAnswer(&dns.Answer{
				Rdata: []string{
					strconv.FormatUint(uint64(f.Flag), 10),
					f.Tag,
					f.Value,
				},
			})
		case dnsv2.TypeSRV:
			f := r.AsSRV()
			rec.AddAnswer(&dns.Answer{Rdata: []string{
				strconv.FormatInt(int64(f.Priority), 10),
				strconv.FormatInt(int64(f.Weight), 10),
				strconv.FormatInt(int64(f.Port), 10),
				f.Target}})
		case dnsv2.TypeNAPTR:
			f := r.AsNAPTR()
			rec.AddAnswer(&dns.Answer{Rdata: []string{
				strconv.Itoa(int(f.Order)),
				strconv.Itoa(int(f.Preference)),
				f.Flags,
				f.Service,
				f.Regexp,
				f.Replacement,
			}})
		case dnsv2.TypeDS:
			f := r.AsDS()
			rec.AddAnswer(&dns.Answer{Rdata: []string{
				strconv.Itoa(int(f.KeyTag)),
				strconv.Itoa(int(f.Algorithm)),
				strconv.Itoa(int(f.DigestType)),
				f.Digest,
			}})
		case dnsv2.TypeSVCB:
			f := r.AsSVCB()
			rec.AddAnswer(&dns.Answer{Rdata: []string{
				strconv.Itoa(int(f.Priority)),
				f.Target,
				models.Svcbv2ValueToString(f.Value),
			}})
		case dnsv2.TypeHTTPS:
			f := r.AsHTTPS()
			rec.AddAnswer(&dns.Answer{Rdata: []string{
				strconv.Itoa(int(f.Priority)),
				f.Target,
				models.Svcbv2ValueToString(f.Value),
			}})
		case dnsv2.TypeTLSA:
			f := r.AsTLSA()
			rec.AddAnswer(&dns.Answer{Rdata: []string{
				strconv.Itoa(int(f.Usage)),
				strconv.Itoa(int(f.Selector)),
				strconv.Itoa(int(f.MatchingType)),
				f.Certificate,
			}})
		default:
			rec.AddAnswer(&dns.Answer{Rdata: strings.Fields(r.GetRDATA().String())})
		}
	}
	return rec
}

func convert(zr *dns.ZoneRecord, dc *models.DomainConfig) (models.Records, error) {
	found := models.Records{}
	for _, ans := range zr.ShortAns {

		label := dc.LabelFromFQDNNoDot(zr.Domain)
		ttl := uint32(zr.TTL)

		var rec *models.RecordConfig
		var err error
		switch rtype := zr.Type; rtype {
		case "DNSKEY", "RRSIG":
			// if a zone is enabled for DNSSEC, NS1 autoconfigures DNSKEY & RRSIG records.
			// these entries are not modifiable via the API though, so we have to ignore them while converting.
			// 	ie. API returns "405 Operation on DNSSEC record is not allowed" on such operations
			continue
		// case "ALIAS":
		// 	rec, err = dc.NewRecordConfig(label, ttl, rtype, ans)
		case "CAA":
			// dnscontrol expects quotes around multivalue CAA entries, API doesn't add them
			xAns := strings.SplitN(ans, " ", 3)
			rec, err = dc.NewRecordConfig(label, ttl, rtype, xAns[0], xAns[1], xAns[2])
		case "NAPTR":
			ans, err = ns1NAPTRAnswer(ans)
			if err != nil {
				break
			}
			rec, err = dc.NewRecordConfigParse(label, ttl, rtype, ans)
		case "REDIRECT":
			// NS1 returns REDIRECTs as records, but there is only one and dummy answer:
			// "NS1 MANAGED RECORD"
			// Redirects are managed via a different API endpoint https://api.nsone.net/v1/redirect
			// It also involves cert management
			// We may simpply ignore REDIRECTs for now until we support it
			printer.Warnf("NS1_REDIRECT is NOT supported by dnscontrol and all existing redirects are ignored.\n")
			continue
		default:
			// NS1 returns TXT values as plain strings, not RFC1035 quoted presentation.
			rec, err = dc.NewRecordConfigParse(label, ttl, rtype, ans,
				nrc.Flags{TxtDontParse: true})
		}
		if err != nil {
			return nil, fmt.Errorf("unparsable %s record received from ns1: %w", zr.Type, err)
		}

		rec.Original = zr

		found = append(found, rec)
	}
	return found, nil
}
