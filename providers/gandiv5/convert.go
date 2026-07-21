package gandiv5

// Convert the provider's native record description to models.RecordConfig.

import (
	"fmt"

	dnsv2 "codeberg.org/miekg/dns"
	"github.com/DNSControl/dnscontrol/v5/models"
	"github.com/DNSControl/dnscontrol/v5/pkg/printer"
	"github.com/DNSControl/dnscontrol/v5/pkg/txtutil"
	"github.com/go-gandi/go-gandi/livedns"
)

// nativeToRecord takes a DNS record from Gandi and returns a native RecordConfig struct.
func nativeToRecords(dc *models.DomainConfig, n livedns.DomainRecord) (rcs []*models.RecordConfig, err error) {
	// Gandi returns all the values for a given label/rtype pair in each
	// livedns.DomainRecord.  In other words, if there are multiple A
	// records for a label, all the IP addresses are listed in
	// n.RrsetValues rather than having many livedns.DomainRecord's.
	// We must split them out into individual records, one for each value.

	// origin := dc.Name

	for _, value := range n.RrsetValues {
		var rc *models.RecordConfig
		var err error

		rtype := n.RrsetType

		switch rtype {
		case "TXT":
			// Gandi stores/returns TXT values in RFC1035 quoted+escaped
			// presentation form. Decode them with the same scheme used to
			// encode on the way out (txtutil, see recordsToNative), then build
			// the record via the normal maker path so its RDATA/ComparableV3
			// match a TXT created from dnsconfig.js. The generic dnsv2
			// presentation parser escapes backslashes differently, which made
			// TXT records containing backslashes fail to round-trip.
			var decoded string
			decoded, err = txtutil.ParseQuoted(value)
			if err != nil {
				return nil, fmt.Errorf("unparsable TXT received from gandi: %w", err)
			}
			rc, err = dc.NewRecordConfig(dc.LabelFromShort(n.RrsetName), uint32(n.RrsetTTL), dnsv2.TypeTXT, decoded)
			if err != nil {
				return nil, fmt.Errorf("unparsable record received from gandi (txt): %w", err)
			}
		case "ALIAS":
			rc, err = dc.NewRecordConfigParse(dc.LabelFromShort(n.RrsetName), uint32(n.RrsetTTL), rtype, value)
			if err != nil {
				return nil, fmt.Errorf("unparsable record received from gandi (alias): %w", err)
			}

		default:
			rc, err = dc.NewRecordConfigParse(dc.LabelFromShort(n.RrsetName), uint32(n.RrsetTTL), rtype, value)
			if err != nil {
				return nil, fmt.Errorf("unparsable record received from gandi (%s): %w", rtype, err)
			}
		}
		rc.Original = n
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
				RrsetType:   r.Type,
				RrsetTTL:    int(r.TTL),
				RrsetName:   label,
				RrsetValues: []string{r.GetTargetCombinedFunc(txtutil.EncodeQuoted)},
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
