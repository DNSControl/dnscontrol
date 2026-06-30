package gandiv5

// Convert the provider's native record description to models.RecordConfig.

import (
	"fmt"
	"strings"

	dnsrdatav2 "codeberg.org/miekg/dns/rdata"
	"github.com/DNSControl/dnscontrol/v4/models"
	"github.com/DNSControl/dnscontrol/v4/pkg/printer"
	"github.com/DNSControl/dnscontrol/v4/pkg/privatetypes"
	"github.com/DNSControl/dnscontrol/v4/pkg/txtutil"
	"github.com/go-gandi/go-gandi/livedns"
)

// nativeToRecord takes a DNS record from Gandi and returns a native RecordConfig struct.
func nativeToRecords(dc *models.DomainConfig, n livedns.DomainRecord) (rcs []*models.RecordConfig, err error) {
	// Gandi returns all the values for a given label/rtype pair in each
	// livedns.DomainRecord.  In other words, if there are multiple A
	// records for a label, all the IP addresses are listed in
	// n.RrsetValues rather than having many livedns.DomainRecord's.
	// We must split them out into individual records, one for each value.

	// dcn := domaintags.MakeDomainNameVarieties(origin)
	origin := dc.Name

	for _, value := range n.RrsetValues {
		var rc *models.RecordConfig
		var err error

		rtype := n.RrsetType

		if rtype == "TXT" {
			fmt.Printf("DEBUG: gandi rcv3 txt get plain %+v\n", value)
		}

		if privatetypes.IsModernType(rtype) && rtype != "ALIAS" {
			rc, err = dc.NewRecordConfigParse(n.RrsetName, uint32(n.RrsetTTL), rtype, value)
			if err != nil {
				return nil, fmt.Errorf("unparsable record received from gandi1: %w", err)
			}
			rc.Original = n
		} else {
			rc = &models.RecordConfig{
				TTL:      uint32(n.RrsetTTL),
				Original: n,
			}
			rc.SetLabel(n.RrsetName, origin)

			switch rtype := n.RrsetType; rtype {
			case "ALIAS":
				rc.Type = "ALIAS"
				err = rc.SetTarget(value)
			default:
				err = rc.PopulateFromStringFunc(rtype, value, origin, txtutil.ParseQuoted)
			}
			if err != nil {
				return nil, fmt.Errorf("unparsable record received from gandi2: %w", err)
			}
		}
		if rtype == "TXT" {
			fmt.Printf("DEBUG: gandi rcv3 txt get decoded %+v\n", rc.GetTargetDebug())
		}
		rcs = append(rcs, rc)

	}

	return rcs, nil
}

func recordsToNative(rcs []*models.RecordConfig, origin string) []livedns.DomainRecord {
	// Take a list of RecordConfig and return an equivalent list of ZoneRecords.
	// Gandi requires one ZoneRecord for each label:key tuple, therefore we
	// might collapse many RecordConfig into one ZoneRecord.

	keys := map[models.RecordKey]*livedns.DomainRecord{}
	var zrs []livedns.DomainRecord

	for _, r := range rcs {
		label := r.GetLabel()
		if label == "@" {
			label = origin
		}
		key := r.Key()

		if zr, ok := keys[key]; !ok {
			// Allocate a new ZoneRecord:
			zr := livedns.DomainRecord{
				RrsetType: r.Type,
				RrsetTTL:  int(r.TTL),
				RrsetName: label,
			}
			if r.Type == "TXT" {
				rdtxt := r.GetRDATA().(dnsrdatav2.TXT)
				fmt.Printf("DEBUG: gandi rcv3 txt create plain %q\n", rdtxt)
				zr.RrsetValues = []string{txtutil.EncodeQuoted(strings.Join(rdtxt.Txt, ""))}
				fmt.Printf("DEBUG: gandi rcv3 txt create encoded %+v\n", zr.RrsetValues)
			} else {
				zr.RrsetValues = []string{r.GetTargetCombinedFunc(txtutil.EncodeQuoted)}
			}
			keys[key] = &zr
		} else {
			zr.RrsetValues = append(zr.RrsetValues, r.GetTargetCombinedFunc(txtutil.EncodeQuoted))

			if r.TTL != uint32(zr.RrsetTTL) {
				printer.Warnf("All TTLs for a rrset (%v) must be the same. Using smaller of %v and %v.\n", key, r.TTL, zr.RrsetTTL)
				if r.TTL < uint32(zr.RrsetTTL) {
					zr.RrsetTTL = int(r.TTL)
				}
			}
		}
	}

	for _, zr := range keys {
		zrs = append(zrs, *zr)
	}
	return zrs
}
