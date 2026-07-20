package desec

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/DNSControl/dnscontrol/v4/models"
	"github.com/DNSControl/dnscontrol/v4/pkg/diff"
	"github.com/DNSControl/dnscontrol/v4/pkg/printer"
	"github.com/DNSControl/dnscontrol/v4/pkg/providers"
	dnsutilv1 "github.com/miekg/dns/dnsutil"
	"golang.org/x/net/idna"
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
	providers.CanUseDS:               providers.Can(),
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
	domain := dc.Name

	punycodeDomain, err := idna.ToASCII(domain)
	if err != nil {
		return nil, err
	}

	records, err := c.getRecords(punycodeDomain)
	if err != nil {
		return nil, err
	}

	// Convert them to DNScontrol's native format:
	existingRecords := []*models.RecordConfig{}
	// spew.Dump(records)
	for _, rr := range records {
		existingRecords = append(existingRecords, nativeToRecords(rr, punycodeDomain)...)
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

	// dc.Punycode()
	recordsToKeep := make([]*models.RecordConfig, 0, len(dc.Records))
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
	punycodeName, err := idna.ToASCII(dc.Name)
	if err != nil {
		return nil, 0, err
	}

	minTTL, ok, err := c.searchDomainIndex(punycodeName)
	if err != nil {
		return nil, 0, err
	}
	if !ok {
		minTTL = 3600
	}

	PrepDesiredRecords(dc, minTTL)

	keysToUpdate, toReport, actualChangeCount, err := diff.NewCompat(dc).ChangedGroups(existing)
	if err != nil {
		return nil, 0, err
	}
	// Start corrections with the reports
	corrections := diff.GenerateMessageCorrections(toReport)

	if len(corrections) == 0 && len(keysToUpdate) == 0 {
		return nil, 0, nil
	}

	desiredRecords := dc.Records.GroupedByKey()
	var rrs []resourceRecord
	buf := &bytes.Buffer{}
	// For any key with an update, delete or replace those records.
	for label := range keysToUpdate {
		if _, ok := desiredRecords[label]; !ok {
			// we could not find this RecordKey in the desiredRecords
			// this means it must be deleted
			for i, msg := range keysToUpdate[label] {
				if i == 0 {
					rc := resourceRecord{}
					rc.Type = label.Type
					rc.Records = make([]string, 0) // empty array of records should delete this rrset
					rc.TTL = 3600
					shortname := dnsutilv1.TrimDomainName(label.NameFQDN, punycodeName)
					if shortname == "@" {
						shortname = ""
					}
					rc.Subname = shortname
					fmt.Fprintln(buf, msg)
					rrs = append(rrs, rc)
				} else {
					// just add the message
					fmt.Fprintln(buf, msg)
				}
			}
		} else {
			// it must be an update or create, both can be done with the same api call.
			ns := recordsToNative(desiredRecords[label], punycodeName)
			if len(ns) > 1 {
				panic("we got more than one resource record to create / modify")
			}
			for i, msg := range keysToUpdate[label] {
				if i == 0 {
					rrs = append(rrs, ns[0])
					fmt.Fprintln(buf, msg)
				} else {
					// noop just for printing the additional messages
					fmt.Fprintln(buf, msg)
				}
			}
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
					err := c.upsertRR(rc, punycodeName)
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
