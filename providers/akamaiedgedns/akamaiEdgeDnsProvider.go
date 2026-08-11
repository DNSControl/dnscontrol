package akamaiedgedns

/*
Akamai Edge DNS provider

For information about Akamai Edge DNS, see:
https://www.akamai.com/us/en/products/security/edge-dns.jsp
https://learn.akamai.com/en-us/products/cloud_security/edge_dns.html
https://www.akamai.com/us/en/multimedia/documents/product-brief/edge-dns-product-brief.pdf
*/

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	"github.com/DNSControl/dnscontrol/v4/models"
	"github.com/DNSControl/dnscontrol/v4/pkg/diff"
	"github.com/DNSControl/dnscontrol/v4/pkg/printer"
	"github.com/DNSControl/dnscontrol/v4/pkg/providers"
	"github.com/akamai/AkamaiOPEN-edgegrid-golang/v12/pkg/dns"
)

var features = providers.DocumentationNotes{
	// The default for unlisted capabilities is 'Cannot'.
	// See providers/capabilities.go for the entire list of capabilities.
	providers.CanAutoDNSSEC:          providers.Can(),
	providers.CanConcur:              providers.Unimplemented(),
	providers.CanGetZones:            providers.Can(),
	providers.CanOnlyDiff1Features:   providers.Can(),
	providers.CanUseAKAMAICDN:        providers.Can(),
	providers.CanUseAKAMAITLC:        providers.Can(),
	providers.CanUseAlias:            providers.Can("Akamai Edge DNS does not directly support ALIAS. Apex record will be converted to AKAMAITLC, any others to CNAME."),
	providers.CanUseCAA:              providers.Can(),
	providers.CanUseDS:               providers.Cannot(),
	providers.CanUseDSForChildren:    providers.Can(),
	providers.CanUseLOC:              providers.Can(),
	providers.CanUseNAPTR:            providers.Can(),
	providers.CanUsePTR:              providers.Can(),
	providers.CanUseSOA:              providers.Cannot(),
	providers.CanUseSRV:              providers.Can(),
	providers.CanUseSSHFP:            providers.Can(),
	providers.CanUseTLSA:             providers.Can(),
	providers.DocCreateDomains:       providers.Can(),
	providers.DocDualHost:            providers.Can(),
	providers.DocOfficiallySupported: providers.Cannot(),
}

type edgeDNSProvider struct {
	contractID string
	groupID    string
	client     dns.DNS
}

func init() {
	const providerName = "AKAMAIEDGEDNS"
	const providerMaintainer = "@edglynes"
	fns := providers.DspFuncs{
		Initializer:   newEdgeDNSDSP,
		RecordAuditor: AuditRecords,
	}
	providers.RegisterDomainServiceProviderType(providerName, fns, features)
	providers.RegisterCustomRecordType("AKAMAICDN", providerName, "")
	providers.RegisterCustomRecordType("AKAMAITLC", providerName, "")
	providers.RegisterMaintainer(providerName, providerMaintainer)
	providers.RegisterCredsMetadata(providerName, providers.CredsMetadata{
		DisplayName: "Akamai Edge DNS",
		Kind:        providers.KindDNS,
		DocsURL:     "https://docs.dnscontrol.org/provider/akamaiedgedns",
		PortalURL:   "https://control.akamai.com/apps/identity-management/", // TODO: Verify
		Fields: []providers.CredsField{
			{
				Key:      "host",
				Label:    "EdgeGrid host",
				Help:     "EdgeGrid API host, e.g. akaa-xxxx.xxxx.akamaiapis.net.",
				Required: true,
			},
			{
				Key:      "client_token",
				Label:    "Client token",
				Help:     "EdgeGrid client_token from your API client credentials.",
				Required: true,
			},
			{
				Key:      "client_secret",
				Label:    "Client secret",
				Help:     "EdgeGrid client_secret from your API client credentials.",
				Secret:   true,
				Required: true,
			},
			{
				Key:      "access_token",
				Label:    "Access token",
				Help:     "EdgeGrid access_token from your API client credentials.",
				Secret:   true,
				Required: true,
			},
			{
				Key:      "contract_id",
				Label:    "Contract ID",
				Help:     "Akamai contract ID used when creating zones (e.g. X-XXXX).",
				Required: true,
			},
			{
				Key:      "group_id",
				Label:    "Group ID",
				Help:     "Akamai group ID used when creating zones (numeric).",
				Required: true,
			},
		},
	})
}

// DnsServiceProvider.
func newEdgeDNSDSP(config map[string]string, metadata json.RawMessage) (providers.DNSServiceProvider, error) {
	clientSecret := config["client_secret"]
	host := config["host"]
	accessToken := config["access_token"]
	clientToken := config["client_token"]
	contractID := config["contract_id"]
	groupID := config["group_id"]

	if clientSecret == "" {
		return nil, errors.New("creds.json: client_secret must not be empty")
	}
	if host == "" {
		return nil, errors.New("creds.json: host must not be empty")
	}
	if accessToken == "" {
		return nil, errors.New("creds.json: accessToken must not be empty")
	}
	if clientToken == "" {
		return nil, errors.New("creds.json: clientToken must not be empty")
	}
	if contractID == "" {
		return nil, errors.New("creds.json: contractID must not be empty")
	}
	if groupID == "" {
		return nil, errors.New("creds.json: groupID must not be empty")
	}

	dnsClient, err := initialize(clientSecret, host, accessToken, clientToken)
	if err != nil {
		return nil, err
	}

	api := &edgeDNSProvider{
		contractID: contractID,
		groupID:    groupID,
		client:     dnsClient,
	}
	return api, nil
}

// EnsureZoneExists creates a zone if it does not exist.
func (a *edgeDNSProvider) EnsureZoneExists(dc *models.DomainConfig) error {
	domain := dc.Name
	ctx := context.Background()
	if a.zoneDoesExist(ctx, domain) {
		printer.Debugf("Zone %s already exists\n", domain)
		return nil
	}
	return a.createZone(ctx, domain, a.contractID, a.groupID)
}

// GetZoneRecordsCorrections returns a list of corrections that will turn existing records into dc.Records.
func (a *edgeDNSProvider) GetZoneRecordsCorrections(dc *models.DomainConfig, existingRecords models.Records) ([]*models.Correction, int, error) {
	ctx := context.Background()

	if err := a.preprocessConfig(dc); err != nil {
		return nil, 0, err
	}

	keysToUpdate, toReport, actualChangeCount, err := diff.NewCompat(dc).ChangedGroups(existingRecords)
	if err != nil {
		return nil, 0, err
	}
	// Start corrections with the reports
	corrections := diff.GenerateMessageCorrections(toReport)

	existingRecordsMap := make(map[models.RecordKey][]*models.RecordConfig)
	for _, r := range existingRecords {
		key := models.RecordKey{NameFQDN: r.NameFQDN, Type: r.Type}
		existingRecordsMap[key] = append(existingRecordsMap[key], r)
	}

	desiredRecordsMap := dc.Records.GroupedByKey()

	// Deletes must occur first. For example, if replacing a existing CNAME with an A of the same name:
	//    DELETE CNAME foo.example.net
	// must occur before
	//    CREATE A foo.example.net
	// because both an A and a CNAME for the same name is not allowed.

	lastCorrections := []*models.Correction{} // creates and replaces last

	for key, msg := range keysToUpdate {
		existing, okExisting := existingRecordsMap[key]
		desired, okDesired := desiredRecordsMap[key]

		if okExisting && !okDesired {
			// In the existing map but not in the desired map: Delete
			corrections = append(corrections, &models.Correction{
				Msg: strings.Join(msg, "\n   "),
				F: func() error {
					return a.deleteRecordset(ctx, existing, dc.Name)
				},
			})
			printer.Debugf("deleteRecordset: %s %s\n", key.NameFQDN, key.Type)
			for _, rdata := range existing {
				printer.Debugf("  Rdata: %s\n", rdata.GetTargetCombined())
			}
		} else if !okExisting && okDesired {
			// Not in the existing map but in the desired map: Create
			lastCorrections = append(lastCorrections, &models.Correction{
				Msg: strings.Join(msg, "\n   "),
				F: func() error {
					return a.createRecordset(ctx, desired, dc.Name)
				},
			})
			printer.Debugf("createRecordset: %s %s\n", key.NameFQDN, key.Type)
			for _, rdata := range desired {
				printer.Debugf("  Rdata: %s\n", rdata.GetTargetCombined())
			}
		} else if okExisting && okDesired {
			// In the existing map and in the desired map: Replace
			lastCorrections = append(lastCorrections, &models.Correction{
				Msg: strings.Join(msg, "\n   "),
				F: func() error {
					return a.replaceRecordset(ctx, desired, dc.Name)
				},
			})
			printer.Debugf("replaceRecordset: %s %s\n", key.NameFQDN, key.Type)
			for _, rdata := range desired {
				printer.Debugf("  Rdata: %s\n", rdata.GetTargetCombined())
			}
		}
	}

	// Deletes first, then creates and replaces
	corrections = append(corrections, lastCorrections...)

	// AutoDnsSec correction
	existingAutoDNSSecEnabled, err := a.isAutoDNSSecEnabled(ctx, dc.Name)
	if err != nil {
		return nil, 0, err
	}

	desiredAutoDNSSecEnabled := dc.AutoDNSSEC == "on"

	if !existingAutoDNSSecEnabled && desiredAutoDNSSecEnabled {
		// Existing false (disabled), Desired true (enabled)
		corrections = append(corrections, &models.Correction{
			Msg: "Enable AutoDnsSec\n",
			F: func() error {
				return a.autoDNSSecEnable(ctx, true, dc.Name)
			},
		})
		printer.Debugf("autoDNSSecEnable: Enable AutoDnsSec for zone %s\n", dc.Name)
	} else if existingAutoDNSSecEnabled && !desiredAutoDNSSecEnabled {
		// Existing true (enabled), Desired false (disabled)
		corrections = append(corrections, &models.Correction{
			Msg: "Disable AutoDnsSec\n",
			F: func() error {
				return a.autoDNSSecEnable(ctx, false, dc.Name)
			},
		})
		printer.Debugf("autoDNSSecEnable: Disable AutoDnsSec for zone %s\n", dc.Name)
	}

	return corrections, actualChangeCount, nil
}

// GetNameservers returns the nameservers for a domain.
func (a *edgeDNSProvider) GetNameservers(domain string) ([]*models.Nameserver, error) {
	ctx := context.Background()
	authorities, err := a.getAuthorities(ctx, a.contractID)
	if err != nil {
		return nil, err
	}
	return models.ToNameserversStripTD(authorities)
}

// GetZoneRecords returns an array of RecordConfig structs for a zone.
func (a *edgeDNSProvider) GetZoneRecords(dc *models.DomainConfig) (models.Records, error) {
	domain := dc.Name

	ctx := context.Background()
	records, err := a.getRecords(ctx, domain)
	if err != nil {
		return nil, err
	}
	return records, nil
}

// ListZones returns all DNS zones managed by this provider.
func (a *edgeDNSProvider) ListZones() ([]string, error) {
	ctx := context.Background()
	zones, err := a.listZones(ctx, a.contractID)
	if err != nil {
		return nil, err
	}
	return zones, nil
}

func (a *edgeDNSProvider) preprocessConfig(dc *models.DomainConfig) error {
	for _, rec := range dc.Records {
		// Convert ALIAS records to the Akamai equivalents. AKAMAITLC is only valid
		// at the apex, so any other ALIAS must be converted to CNAME.
		if rec.Type == "ALIAS" {
			if rec.Name == "@" {
				rec.ChangeType("AKAMAITLC", dc.Name)
				rec.AnswerType = "DUAL"
			} else {
				rec.ChangeType("CNAME", dc.Name)
			}
		}
	}
	return nil
}
