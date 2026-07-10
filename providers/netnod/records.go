package netnod

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/DNSControl/dnscontrol/v4/models"
	"github.com/DNSControl/dnscontrol/v4/pkg/diff2"
	"github.com/fatih/color"
	netnodPrimaryDNS "github.com/netnod/netnod-primary-dns-client"
)

func (dsp *netnodProvider) getDiff2DomainCorrections(dc *models.DomainConfig, existing models.Records) ([]*models.Correction, int, error) {
	changes, actualChangeCount, err := diff2.ByRecordSet(existing, dc, nil)
	if err != nil {
		return nil, 0, err
	}

	var corrections []*models.Correction
	var changeMsgs []string
	var rrChangeSets []netnodPrimaryDNS.RRset
	var deleteMsgs []string
	var rrDeleteSets []netnodPrimaryDNS.RRset

	// for pretty alignment, add an empty string
	changeMsgs = append(changeMsgs, color.YellowString("± BATCHED CHANGE/CREATEs for %s", dc.Name))
	deleteMsgs = append(deleteMsgs, color.RedString("- BATCHED DELETEs for %s", dc.Name))

	for _, change := range changes {
		labelName := canonical(change.Key.NameFQDN)
		labelType := change.Key.Type

		switch change.Type {
		case diff2.REPORT:
			corrections = append(corrections, &models.Correction{Msg: change.MsgsJoined})
		case diff2.CREATE, diff2.CHANGE:
			labelTTL := int64(change.New[0].TTL)
			records := buildRecordList(change)

			rrChangeSets = append(rrChangeSets, netnodPrimaryDNS.RRset{
				Name:       labelName,
				Type:       labelType,
				TTL:        &labelTTL,
				Records:    records,
				ChangeType: "REPLACE",
			})
			changeMsgs = append(changeMsgs, change.MsgsJoined)
		case diff2.DELETE:
			rrDeleteSets = append(rrDeleteSets, netnodPrimaryDNS.RRset{
				Name:       labelName,
				Type:       labelType,
				ChangeType: "DELETE",
			})
			deleteMsgs = append(deleteMsgs, change.MsgsJoined)
		default:
			panic(fmt.Sprintf("unhandled change.Type %s", change.Type))
		}
	}

	domainVariant := dc.Name + "."

	// only append a Correction if there are any, otherwise causes an error when sending an empty rrset
	if len(rrDeleteSets) > 0 {
		corrections = append(corrections, &models.Correction{
			Msg: strings.Join(deleteMsgs, "\n"),
			F: func() error {
				return dsp.client.PatchZoneRRsets(domainVariant, rrDeleteSets)
			},
		})
	}
	if len(rrChangeSets) > 0 {
		corrections = append(corrections, &models.Correction{
			Msg: strings.Join(changeMsgs, "\n"),
			F: func() error {
				return dsp.client.PatchZoneRRsets(domainVariant, rrChangeSets)
			},
		})
	}
	return corrections, actualChangeCount, nil
}

// httpsParamQuoteRe matches HTTPS SVCB parameter values that are quoted but
// don't contain characters requiring quoting (+ or /). These are stripped of
// their quotes before sending to the API (e.g. alpn="h2,h3" => alpn=h2,h3).
// Values containing + or / (e.g. ECH base64 data) retain their quotes.
var httpsParamQuoteRe = regexp.MustCompile(`="([^"+/ ]*)"`)

// buildRecordList returns a list of records for the resource record set from a change.
func buildRecordList(change diff2.Change) (records []netnodPrimaryDNS.Record) {
	for _, recordContent := range change.New {
		record := netnodPrimaryDNS.Record{
			Content: recordContent.GetTargetCombined(),
		}
		if recordContent.Type == "HTTPS" || recordContent.Type == "SVCB" {
			// The API rejects double-quoted simple param values (e.g. alpn="h2,h3")
			// but requires quotes around values containing + or / (e.g. ECH base64).
			// Strip quotes only from values that don't contain those characters.
			record.Content = httpsParamQuoteRe.ReplaceAllString(record.Content, `=$1`)
		}
		records = append(records, record)
	}
	return
}

func canonical(fqdn string) string {
	return fqdn + "."
}
