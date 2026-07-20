package gcore

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/DNSControl/dnscontrol/v4/models"
	"github.com/DNSControl/dnscontrol/v4/pkg/diff2"
	"github.com/DNSControl/dnscontrol/v4/pkg/printer"
	"github.com/DNSControl/dnscontrol/v4/pkg/providers"
	dnssdk "github.com/G-Core/gcore-dns-sdk-go"
)

/*
G-Core API DNS provider:
Info required in `creds.json`:
   - api-key
*/

type gcoreProvider struct {
	provider *dnssdk.Client
	ctx      context.Context
	apiKey   string
}

// NewGCore creates the provider.
func NewGCore(m map[string]string, metadata json.RawMessage) (providers.DNSServiceProvider, error) {
	if m["api-key"] == "" {
		return nil, errors.New("missing G-Core API key")
	}

	c := &gcoreProvider{
		provider: dnssdk.NewClient(dnssdk.PermanentAPIKeyAuth(m["api-key"])),
		ctx:      context.TODO(),
		apiKey:   m["api-key"],
	}

	return c, nil
}

var features = providers.DocumentationNotes{
	// The default for unlisted capabilities is 'Cannot'.
	// See providers/capabilities.go for the entire list of capabilities.
	providers.CanAutoDNSSEC:          providers.Can(),
	providers.CanGetZones:            providers.Can(),
	providers.CanConcur:              providers.Can(),
	providers.CanUseAlias:            providers.Can(),
	providers.CanUseCAA:              providers.Can(),
	providers.CanUseDS:               providers.Cannot(),
	providers.CanUseLOC:              providers.Cannot(),
	providers.CanUseNAPTR:            providers.Cannot(),
	providers.CanUsePTR:              providers.Can("G-Core supports PTR records only in rDNS zones"),
	providers.CanUseSRV:              providers.Can("G-Core doesn't support SRV records with empty targets"),
	providers.CanUseSSHFP:            providers.Cannot(),
	providers.CanUseTLSA:             providers.Cannot(),
	providers.CanUseHTTPS:            providers.Can(),
	providers.CanUseSVCB:             providers.Can(),
	providers.DocCreateDomains:       providers.Can(),
	providers.DocDualHost:            providers.Can(),
	providers.DocOfficiallySupported: providers.Cannot(),
}

var defaultNameServerNames = []string{
	"ns1.gcorelabs.net",
	"ns2.gcdn.services",
}

func init() {
	const providerName = "GCORE"
	const providerMaintainer = "@xddxdd"
	fns := providers.DspFuncs{
		Initializer:   NewGCore,
		RecordAuditor: AuditRecords,
	}
	providers.RegisterDomainServiceProviderType(providerName, fns, features)
	providers.RegisterMaintainer(providerName, providerMaintainer)
	providers.RegisterCredsMetadata(providerName, providers.CredsMetadata{
		DisplayName: "G-Core Labs",
		Kind:        providers.KindDNS,
		DocsURL:     "https://docs.dnscontrol.org/provider/gcore",
		PortalURL:   "https://accounts.gcore.com/profile/api-tokens", // TODO: Verify
		Fields: []providers.CredsField{
			{
				Key:      "api-key",
				Label:    "API key",
				Help:     "G-Core Labs permanent API key.",
				Secret:   true,
				Required: true,
			},
		},
	})
}

// GetNameservers returns the nameservers for a domain.
func (c *gcoreProvider) GetNameservers(domain string) ([]*models.Nameserver, error) {
	return models.ToNameservers(defaultNameServerNames)
}

// GetZoneRecords gets the records of a zone and returns them in RecordConfig format.
func (c *gcoreProvider) GetZoneRecords(dc *models.DomainConfig) (models.Records, error) {
	domain := dc.Name

	zone, err := c.provider.Zone(c.ctx, domain)
	if err != nil {
		return nil, err
	}

	// Convert RRsets to DNSControl format on the fly
	existingRecords := []*models.RecordConfig{}

	// We cannot directly use Zone's ShortAnswers, they aren't complete for CAA & SRV

	rrsets, err := c.dnssdkRRSets(domain)
	if err != nil {
		return nil, err
	}

	for _, rec := range rrsets.RRSets {
		nativeRecords, err := nativeToRecords(rec, zone.Name)
		if err != nil {
			return nil, err
		}
		existingRecords = append(existingRecords, nativeRecords...)
	}

	return existingRecords, nil
}

// EnsureZoneExists creates a zone if it does not exist.
func (c *gcoreProvider) EnsureZoneExists(dc *models.DomainConfig) error {
	domain := dc.Name
	zones, err := c.provider.Zones(c.ctx)
	if err != nil {
		return err
	}

	for _, zone := range zones {
		if zone.Name == domain {
			return nil
		}
	}

	_, err = c.provider.CreateZone(c.ctx, dnssdk.AddZone{
		Name: domain,
	})
	return err
}

func generateChangeMsg(updates []string) string {
	return strings.Join(updates, "\n")
}

// GenerateDomainCorrections takes the desired and existing records
// and produces a Correction list.  The correction list is simply
// a list of functions to call to actually make the desired
// correction, and a message to output to the user when the change is
// made.
func (c *gcoreProvider) GetZoneRecordsCorrections(dc *models.DomainConfig, existing models.Records) ([]*models.Correction, int, error) {
	// Make delete happen earlier than creates & updates.
	var corrections []*models.Correction
	var deletions []*models.Correction
	var reports []*models.Correction

	// Gcore auto uses ALIAS for apex zone CNAME records, just like CloudFlare
	for _, rec := range dc.Records {
		if rec.Type == "ALIAS" {
			rec.ChangeType("CNAME", dc.Name)
		}
	}

	changes, actualChangeCount, err := diff2.ByRecordSet(existing, dc, comparableFunc)
	if err != nil {
		return nil, 0, err
	}

	for _, change := range changes {
		record, err := recordsToNative(change.New, change.Key)
		if err != nil {
			return nil, 0, err
		}

		// Copy all params to avoid overwrites
		zone := dc.Name
		name := change.Key.NameFQDN
		typ := change.Key.Type
		msg := generateChangeMsg(change.Msgs)

		switch change.Type {
		case diff2.REPORT:
			corrections = append(corrections, &models.Correction{Msg: change.MsgsJoined})
		case diff2.CREATE:
			corrections = append(corrections, &models.Correction{
				Msg: msg,
				F: func() error {
					return c.provider.CreateRRSet(c.ctx, zone, name, typ, *record)
				},
			})
		case diff2.CHANGE:
			corrections = append(corrections, &models.Correction{
				Msg: msg,
				F: func() error {
					return c.provider.UpdateRRSet(c.ctx, zone, name, typ, *record)
				},
			})
		case diff2.DELETE:
			deletions = append(deletions, &models.Correction{
				Msg: msg,
				F: func() error {
					return c.provider.DeleteRRSet(c.ctx, zone, name, typ)
				},
			})
		default:
			panic(fmt.Sprintf("unhandled change.Type %s", change.Type))
		}
	}

	dnssecEnabled, err := c.dnssdkGetDNSSEC(dc.Name)
	if err != nil {
		return nil, 0, err
	}

	if !dnssecEnabled && dc.AutoDNSSEC == "on" {
		// Copy all params to avoid overwrites
		zone := dc.Name
		corrections = append(corrections, &models.Correction{
			Msg: "Enable DNSSEC",
			F: func() error {
				return c.dnssdkSetDNSSEC(zone, true)
			},
		})
	} else if dnssecEnabled && dc.AutoDNSSEC == "off" {
		// Copy all params to avoid overwrites
		zone := dc.Name
		corrections = append(corrections, &models.Correction{
			Msg: "Disable DNSSEC",
			F: func() error {
				return c.dnssdkSetDNSSEC(zone, false)
			},
		})
	}

	result := append(reports, deletions...)
	result = append(result, corrections...)
	return result, actualChangeCount, nil
}

// ListZones return all the zones in the account.
func (c *gcoreProvider) ListZones() ([]string, error) {
	zones, err := c.provider.Zones(c.ctx)
	if err != nil {
		return nil, err
	}

	var zoneList []string
	for _, zone := range zones {
		zoneList = append(zoneList, zone.Name)
	}

	return zoneList, nil
}

func comparableFunc(rec *models.RecordConfig) string {
	if len(rec.Metadata) == 0 {
		return ""
	}

	// json.Marshal always serialize all fields in alphabetical order,
	// so the metadata string will be consistent between runs
	result, err := json.Marshal(rec.Metadata)
	if err != nil {
		printer.Warnf("Cannot serialize metadata of record %s", rec)
		return ""
	}

	return string(result)
}
