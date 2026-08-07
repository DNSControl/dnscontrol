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

	dnsrdatav2 "codeberg.org/miekg/dns/rdata"
	"github.com/DNSControl/dnscontrol/v5/models"
	"github.com/DNSControl/dnscontrol/v5/pkg/diff2"
	"github.com/DNSControl/dnscontrol/v5/pkg/printer"
	privatetypesrdata "github.com/DNSControl/dnscontrol/v5/pkg/privatetypes/rdata"
	"github.com/DNSControl/dnscontrol/v5/pkg/providers"
	"github.com/akamai/AkamaiOPEN-edgegrid-golang/v12/pkg/dns"
)

var features = providers.DocumentationNotes{
	// The default for unlisted capabilities is 'Cannot'.
	// See providers/capabilities.go for the entire list of capabilities.
	providers.CanAutoDNSSEC:          providers.Can(),
	providers.CanConcur:              providers.Unimplemented(),
	providers.CanGetZones:            providers.Can(),
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

	changes, actualChangeCount, err := diff2.ByRecordSet(existingRecords, dc, nil)
	if err != nil {
		return nil, 0, err
	}

	var corrections []*models.Correction

	for _, change := range changes {
		switch change.Type {
		case diff2.REPORT:
			corrections = append(corrections, &models.Correction{Msg: change.MsgsJoined})
		case diff2.CREATE:
			corrections = append(corrections, &models.Correction{
				Msg: change.MsgsJoined,
				F:   func() error { return a.createRecordset(ctx, change.New, dc.Name) },
			})
		case diff2.CHANGE:
			ttl := managedTTL(dc, change.New[0].NameFQDN, change.New[0].Type)
			for _, r := range change.New {
				if r.TTL != ttl {
					printer.Warnf("TTL mismatch in %s %s: using %d (managed), ignoring %d\n", change.New[0].NameFQDN, change.New[0].Type, ttl, r.TTL)
					break
				}
			}
			corrections = append(corrections, &models.Correction{
				Msg: change.MsgsJoined,
				F:   func() error { return a.replaceRecordset(ctx, change.New, ttl, dc.Name) },
			})
		case diff2.DELETE:
			corrections = append(corrections, &models.Correction{
				Msg: change.MsgsJoined,
				F:   func() error { return a.deleteRecordset(ctx, change.Old, dc.Name) },
			})
		}
	}

	// AutoDnsSec correction
	existingAutoDNSSecEnabled, err := a.isAutoDNSSecEnabled(ctx, dc.Name)
	if err != nil {
		return nil, 0, err
	}

	desiredAutoDNSSecEnabled := dc.AutoDNSSEC == "on"

	if !existingAutoDNSSecEnabled && desiredAutoDNSSecEnabled {
		corrections = append(corrections, &models.Correction{
			Msg: "Enable AutoDnsSec\n",
			F:   func() error { return a.autoDNSSecEnable(ctx, true, dc.Name) },
		})
	} else if existingAutoDNSSecEnabled && !desiredAutoDNSSecEnabled {
		corrections = append(corrections, &models.Correction{
			Msg: "Disable AutoDnsSec\n",
			F:   func() error { return a.autoDNSSecEnable(ctx, false, dc.Name) },
		})
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
	ctx := context.Background()
	records, err := a.getRecords(ctx, dc)
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

func managedTTL(dc *models.DomainConfig, fqdn, rtype string) uint32 {
	for _, r := range dc.Records {
		if r.NameFQDN == fqdn && r.Type == rtype {
			return r.TTL
		}
	}
	return 0
}

func (a *edgeDNSProvider) preprocessConfig(dc *models.DomainConfig) error {
	for _, rec := range dc.Records {
		// Convert ALIAS records to the Akamai equivalents. AKAMAITLC is only valid
		// at the apex, so any other ALIAS must be converted to CNAME.
		if rec.Type == "ALIAS" {
			target := rec.AsALIAS().Target
			if rec.Name == "@" {
				rec.ChangeType("AKAMAITLC", dc.Name)
				rec.SetRDATA(privatetypesrdata.AKAMAITLC{AnswerType: "DUAL", Target: target})
			} else {
				rec.ChangeType("CNAME", dc.Name)
				rec.SetRDATA(dnsrdatav2.CNAME{Target: target})
			}
		}
	}
	return nil
}
