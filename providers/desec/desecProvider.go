package desec

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/DNSControl/dnscontrol/v5/models"
	"github.com/DNSControl/dnscontrol/v5/pkg/diff2"
	"github.com/DNSControl/dnscontrol/v5/pkg/printer"
	"github.com/DNSControl/dnscontrol/v5/pkg/providers"
)

/*
desec API DNS provider:
Info required in `creds.json`:
   - auth-token
*/

// NewDeSec creates the provider.
func NewDeSec(m map[string]string, metadata json.RawMessage) (providers.DNSServiceProvider, error) {
	c := &desecProvider{}
	c.token = strings.TrimSpace(m["auth-token"])
	if c.token == "" {
		return nil, errors.New("missing deSEC auth-token")
	}
	return c, nil
}

var features = providers.DocumentationNotes{
	// The default for unlisted capabilities is 'Cannot'.
	// See providers/capabilities.go for the entire list of capabilities.
	providers.CanAutoDNSSEC:          providers.Can("deSEC always signs all records. When trying to disable, a notice is printed."),
	providers.CanConcur:              providers.Can(),
	providers.CanGetZones:            providers.Can(),
	providers.CanOnlyDiff1Features:   providers.Can(),
	providers.CanUseAlias:            providers.Unimplemented("Apex aliasing is supported via new SVCB and HTTPS record types. For details, check the deSEC docs."),
	providers.CanUseCAA:              providers.Can(),
	providers.CanUseDNSKEY:           providers.Can(),
	providers.CanUseDS:               providers.Cannot(),
	providers.CanUseDSForChildren:    providers.Can(),
	providers.CanUseHTTPS:            providers.Can(),
	providers.CanUseLOC:              providers.Can(),
	providers.CanUseNAPTR:            providers.Can(),
	providers.CanUsePTR:              providers.Can(),
	providers.CanUseSMIMEA:           providers.Can(),
	providers.CanUseSRV:              providers.Can(),
	providers.CanUseSSHFP:            providers.Can(),
	providers.CanUseSVCB:             providers.Can(),
	providers.CanUseTLSA:             providers.Can(),
	providers.CanUseOPENPGPKEY:       providers.Can(),
	providers.DocCreateDomains:       providers.Can(),
	providers.DocDualHost:            providers.Unimplemented(),
	providers.DocOfficiallySupported: providers.Cannot(),
}

var defaultNameServerNames = []string{
	"ns1.desec.io",
	"ns2.desec.org",
}

func init() {
	const providerName = "DESEC"
	const providerMaintainer = "@D3luxee"
	fns := providers.DspFuncs{
		Initializer:   NewDeSec,
		RecordAuditor: AuditRecords,
	}
	providers.RegisterDomainServiceProviderType(providerName, fns, features)
	providers.RegisterMaintainer(providerName, providerMaintainer)
	providers.RegisterCredsMetadata(providerName, providers.CredsMetadata{
		DisplayName: "deSEC",
		Kind:        providers.KindDNS,
		DocsURL:     "https://docs.dnscontrol.org/provider/desec",
		PortalURL:   "https://desec.io/tokens", // TODO: Verify
		Fields: []providers.CredsField{
			{
				Key:      "auth-token",
				Label:    "Auth token",
				Help:     "Your deSEC API auth token.",
				Secret:   true,
				Required: true,
			},
		},
	})
}

// GetNameservers returns the nameservers for a domain.
func (c *desecProvider) GetNameservers(domain string) ([]*models.Nameserver, error) {
	return models.ToNameservers(defaultNameServerNames)
}

// GetZoneRecords gets the records of a zone and returns them in RecordConfig format.
func (c *desecProvider) GetZoneRecords(dc *models.DomainConfig) (models.Records, error) {
	records, err := c.getRecords(dc.Name)
	if err != nil {
		return nil, err
	}

	// Convert them to DNScontrol's native format:
	existingRecords := models.Records{}
	// spew.Dump(records)
	for _, rr := range records {
		existingRecords = append(existingRecords, nativeToRecords(rr, dc)...)
	}

	return existingRecords, nil
}

// EnsureZoneExists creates a zone if it does not exist.
func (c *desecProvider) EnsureZoneExists(dc *models.DomainConfig) error {
	domain := dc.Name
	_, ok, err := c.searchDomainIndex(domain)
	if err != nil {
		return err
	}

	if ok {
		// Domain already exists
		return nil
	}
	return c.createDomain(domain)
}

// PrepDesiredRecords munges any records to best suit this provider.
func PrepDesiredRecords(dc *models.DomainConfig, minTTL uint32) {
	// Sort through the dc.Records, eliminate any that can't be
	// supported; modify any that need adjustments to work with the
	// provider.  We try to do minimal changes otherwise it gets
	// confusing.

	// TODO(tlim): We shouldn't modify rec's because if the same rec is used
	// for another provider, there will be confusion. Right now we use a
	// little memory by making a copy of every record for each provider, but
	// we'd like to not do that in the future.
	// If possible, it would be better to eliminate this function
	// and instead:
	// * ALIAS: Skip them in recordsToNative()
	// * minTTL: recordsToNative() should return records with fixed TTLs.

	recordsToKeep := make(models.Records, 0, len(dc.Records))
	for _, rec := range dc.Records {
		if rec.Type == "ALIAS" {
			// deSEC does not permit ALIAS records, just ignore it
			printer.Warnf("deSEC does not support alias records\n")
			continue
		}
		if rec.TTL < minTTL {
			if rec.Type != "NS" {
				printer.Warnf("Please contact support@desec.io if you need TTLs < %d. Setting TTL of %s type %s from %d to %d\n", minTTL, rec.GetLabelFQDN(), rec.Type, rec.TTL, minTTL)
			}
			rec.TTL = minTTL
		}
		recordsToKeep = append(recordsToKeep, rec)
	}
	dc.Records = recordsToKeep
}

// GetZoneRecordsCorrections returns a list of corrections that will turn existing records into dc.Records.
func (c *desecProvider) GetZoneRecordsCorrections(dc *models.DomainConfig, existing models.Records) ([]*models.Correction, int, error) {
	minTTL, ok, err := c.searchDomainIndex(dc.Name)
	if err != nil {
		return nil, 0, err
	}
	if !ok {
		minTTL = 3600
	}

	PrepDesiredRecords(dc, minTTL)

	changes, actualChangeCount, err := diff2.ByRecordSet(existing, dc, nil)
	if err != nil {
		return nil, 0, err
	}

	var corrections []*models.Correction
	var rrs []resourceRecord
	buf := &bytes.Buffer{}
	// For any rrset with an update, delete or replace those records. deSEC's
	// API is rrset-oriented: a single upsert call replaces the whole rrset (an
	// empty record list deletes it).
	for _, change := range changes {
		switch change.Type {
		case diff2.REPORT:
			corrections = append(corrections, &models.Correction{Msg: change.MsgsJoined})

		case diff2.DELETE:
			// An empty array of records deletes this rrset.
			rc := resourceRecord{}
			rc.Type = change.Key.Type
			rc.Records = make([]string, 0)
			rc.TTL = 3600
			shortname := dc.ToShort(change.Key.NameFQDN)
			if shortname == "@" {
				shortname = ""
			}
			rc.Subname = shortname
			rrs = append(rrs, rc)
			for _, msg := range change.Msgs {
				fmt.Fprintln(buf, msg)
			}

		case diff2.CREATE, diff2.CHANGE:
			// A create or update are both done with the same api call.
			ns := recordsToNative(change.New)
			if len(ns) > 1 {
				panic("we got more than one resource record to create / modify")
			}
			rrs = append(rrs, ns[0])
			for _, msg := range change.Msgs {
				fmt.Fprintln(buf, msg)
			}

		default:
			panic(fmt.Sprintf("unhandled change.Type %s", change.Type))
		}
	}

	// If there are changes, upsert them.
	if len(rrs) > 0 {
		msg := fmt.Sprintf("Changes:\n%s", buf)
		corrections = append(corrections,
			&models.Correction{
				Msg: msg,
				F: func() error {
					rc := rrs
					err := c.upsertRR(rc, dc.Name)
					if err != nil {
						return err
					}
					return nil
				},
			})
	}

	// NB(tlim): This sort is just to make updates look pretty. It is
	// cosmetic.  The risk here is that there may be some updates that
	// require a specific order (for example a delete before an add).
	// However the code doesn't seem to have such situation.  All tests
	// pass.  That said, if this breaks anything, the easiest fix might
	// be to just remove the sort.
	// sort.Slice(corrections, func(i, j int) bool { return diff.CorrectionLess(corrections, i, j) })

	return corrections, actualChangeCount, nil
}

// ListZones return all the zones in the account.
func (c *desecProvider) ListZones() ([]string, error) {
	return c.listDomainIndex()
}
