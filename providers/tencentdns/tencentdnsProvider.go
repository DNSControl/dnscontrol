package tencentdns

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/DNSControl/dnscontrol/v4/models"
	"github.com/DNSControl/dnscontrol/v4/pkg/diff2"
	"github.com/DNSControl/dnscontrol/v4/pkg/providers"
	dnspod "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/dnspod/v20210323"
)

const (
	defaultTTL          = uint32(600)
	defaultRecordLine   = "默认"
	defaultRecordLineID = "0"
	metaRecordLine      = "tencentdns_line"
	metaRecordLineID    = "tencentdns_line_id"
	metaRecordWeight    = "tencentdns_weight"
)

var features = providers.DocumentationNotes{
	providers.CanUseAlias:            providers.Can("DNSPod doesn't natively support the ALIAS record type."),
	providers.CanGetZones:            providers.Can(),
	providers.CanUseCAA:              providers.Can(),
	providers.CanUsePTR:              providers.Can(),
	providers.CanUseSRV:              providers.Can(),
	providers.DocCreateDomains:       providers.Can(),
	providers.DocDualHost:            providers.Can("Tencent Cloud allows full management of apex NS records"),
	providers.DocOfficiallySupported: providers.Cannot(),
}

func init() {
	const providerName = "TENCENTDNS"
	const providerMaintainer = "@cylonchau"
	fns := providers.DspFuncs{
		Initializer:   newTencentDNSDsp,
		RecordAuditor: AuditRecords,
	}
	providers.RegisterDomainServiceProviderType(providerName, fns, features)
	providers.RegisterRegistrarType(providerName, newTencentDNSReg)
	providers.RegisterMaintainer(providerName, providerMaintainer)
	// Default TTL for Tencent Cloud DNSPod is 600 for free plan.
	providers.RegisterDefaultTTL(providerName, defaultTTL)
	providers.RegisterCredsMetadata(providerName, providers.CredsMetadata{
		DisplayName: "Tencent Cloud DNS",
		Kind:        providers.KindDNS | providers.KindRegistrar,
		DocsURL:     "https://docs.dnscontrol.org/provider/tencentdns",
		PortalURL:   "https://console.intl.cloud.tencent.com/cam/capi",
		Fields: []providers.CredsField{
			{
				Key:      "secret_id",
				Label:    "Secret ID",
				Help:     "Tencent Cloud SecretId.",
				Required: true,
				Secret:   true,
			},
			{
				Key:      "secret_key",
				Label:    "Secret Key",
				Help:     "Tencent Cloud SecretKey.",
				Required: true,
				Secret:   true,
			},
			{
				Key:     "region",
				Label:   "Region",
				Help:    "The region value does not affect DNS management (DNS is global).",
				Default: "ap-guangzhou",
			},
			{
				Key:     "site",
				Label:   "Site",
				Help:    "Tencent Cloud site. Use cn for mainland China or intl for international APIs.",
				Default: "cn",
			},
		},
	})
}

type tencentdnsProvider struct {
	client *tencentCloudClient
}

func newTencentDNSDsp(config map[string]string, metadata json.RawMessage) (providers.DNSServiceProvider, error) {
	return newTencentDNS(config)
}

func newTencentDNSReg(config map[string]string) (providers.Registrar, error) {
	return newTencentDNS(config)
}

func newTencentDNS(config map[string]string) (*tencentdnsProvider, error) {
	secretID := config["secret_id"]
	secretKey := config["secret_key"]
	if secretID == "" || secretKey == "" {
		return nil, fmt.Errorf("missing tencent cloud credentials (secret_id, secret_key)")
	}

	region := config["region"]
	if region == "" {
		region = "ap-guangzhou"
	}

	siteConfig, err := siteConfigForSite(config["site"])
	if err != nil {
		return nil, err
	}

	client, err := newClient(secretID, secretKey, region, siteConfig.dnspodEndpoint, siteConfig.useIntlDomainClient)
	if err != nil {
		return nil, err
	}

	return &tencentdnsProvider{
		client: client,
	}, nil
}

type tencentSiteConfig struct {
	dnspodEndpoint      string
	useIntlDomainClient bool
}

func siteConfigForSite(site string) (tencentSiteConfig, error) {
	switch strings.ToLower(site) {
	case "", "cn", "china":
		return tencentSiteConfig{}, nil
	case "intl", "international":
		return tencentSiteConfig{
			dnspodEndpoint:      intlDNSPodEndpoint,
			useIntlDomainClient: true,
		}, nil
	default:
		return tencentSiteConfig{}, fmt.Errorf("unsupported tencent cloud site %q: expected cn or intl", site)
	}
}

func (p *tencentdnsProvider) ListZones() ([]string, error) {
	// For simplicity, we just use the API to list all domains.
	// In a real implementation, we might want to handle pagination better.
	request := dnspod.NewDescribeDomainListRequest()
	response, err := p.client.dnspodClient.DescribeDomainList(request)
	if err != nil {
		return nil, err
	}

	var zones []string
	for _, domain := range response.Response.DomainList {
		zones = append(zones, *domain.Name)
	}
	return zones, nil
}

func (p *tencentdnsProvider) GetNameservers(domainName string) ([]*models.Nameserver, error) {
	nss, err := p.client.getNameservers(domainName)
	if err != nil {
		if strings.Contains(err.Error(), "DomainNotExists") || strings.Contains(err.Error(), "域名有误") {
			return nil, nil
		}
		return nil, err
	}
	return models.ToNameservers(nss)
}

func (p *tencentdnsProvider) GetZoneRecords(dc *models.DomainConfig) (models.Records, error) {
	records, err := p.client.fetchRecords(dc.Name)
	if err != nil {
		if strings.Contains(err.Error(), "DomainNotExists") {
			return nil, nil
		}
		return nil, err
	}

	existingRecords := models.Records{}
	for _, r := range records {
		if *r.Status != "ENABLE" {
			continue
		}
		rc, err := nativeToRecord(r, dc.Name)
		if err != nil {
			return nil, err
		}
		existingRecords = append(existingRecords, rc)
	}
	return existingRecords, nil
}

func prepDesiredRecords(dc *models.DomainConfig, minTTL uint32) {
	for _, rec := range dc.Records {
		if rec.TTL != 0 && rec.TTL < minTTL {
			rec.TTL = minTTL
		}
	}
}

// recordLineIDsByName returns only unambiguous line name-to-ID mappings.
// The DNSPod default line always has ID 0.
func recordLineIDsByName(existingRecords models.Records) map[string]string {
	lineIDsByName := map[string]string{defaultRecordLine: defaultRecordLineID}
	ambiguous := make(map[string]bool)
	for _, rec := range existingRecords {
		line, lineID := recordLineMetadata(rec)
		if line == "" || lineID == "" || line == defaultRecordLine || ambiguous[line] {
			continue
		}
		if knownID, ok := lineIDsByName[line]; ok && knownID != lineID {
			delete(lineIDsByName, line)
			ambiguous[line] = true
			continue
		}
		lineIDsByName[line] = lineID
	}
	return lineIDsByName
}

// recordMetadataComparable returns a provider-specific comparison function
// that makes DNSPod's record line and weight part of a record's identity.
// DNSPod returns both a line name and an ID, while users may configure either
// one. Unambiguous line names are normalized to their IDs before diffing.
func recordMetadataComparable(existingRecords models.Records) diff2.ComparableFunc {
	lineIDsByName := recordLineIDsByName(existingRecords)

	return func(rec *models.RecordConfig) string {
		line, lineID := recordLineMetadata(rec)
		if lineID == "" {
			lineID = lineIDsByName[line]
		}
		lineComparable := "line=" + line
		if lineID != "" {
			lineComparable = "line_id=" + lineID
		}

		weight := comparableRecordWeight(rec)
		if weight == "" {
			return lineComparable
		}
		return lineComparable + " weight=" + weight
	}
}

func (p *tencentdnsProvider) GetZoneRecordsCorrections(dc *models.DomainConfig, existingRecords models.Records) ([]*models.Correction, int, error) {
	var corrections []*models.Correction

	minTTL, err := p.client.getMinTTL(dc.Name)
	if err != nil {
		return nil, 0, err
	}
	prepDesiredRecords(dc, minTTL)

	// Tencent Cloud is a "ByRecord" API.
	changes, actualChangeCount, err := diff2.ByRecord(existingRecords, dc, recordMetadataComparable(existingRecords))
	if err != nil {
		return nil, 0, err
	}

	for _, change := range changes {
		msgs := change.MsgsJoined
		domainName := dc.Name

		switch change.Type {
		case diff2.REPORT:
			corrections = append(corrections, &models.Correction{Msg: msgs})
		case diff2.CREATE:
			rc := change.New[0]
			corrections = append(corrections, &models.Correction{
				Msg: msgs,
				F: func() error {
					return p.client.createRecord(domainName, recordToCreateRequest(rc))
				},
			})
		case diff2.CHANGE:
			rc := change.New[0]
			previous := change.Old[0]
			recordID := *(previous.Original.(*dnspod.RecordListItem).RecordId)
			corrections = append(corrections, &models.Correction{
				Msg: msgs,
				F: func() error {
					return p.client.modifyRecord(domainName, recordToModifyRequest(rc, recordID, previous))
				},
			})
		case diff2.DELETE:
			recordID := *(change.Old[0].Original.(*dnspod.RecordListItem).RecordId)
			corrections = append(corrections, &models.Correction{
				Msg: msgs,
				F: func() error {
					return p.client.deleteRecord(domainName, recordID)
				},
			})
		}
	}

	return corrections, actualChangeCount, nil
}

func (p *tencentdnsProvider) GetRegistrarCorrections(dc *models.DomainConfig) ([]*models.Correction, error) {
	actualSet, err := p.client.getRegistrarNameservers(dc.Name)
	if err != nil {
		return nil, err
	}
	actualSet = normalizeNameserverSet(actualSet)
	actual := strings.Join(actualSet, ",")

	expectedSet := []string{}
	for _, ns := range dc.Nameservers {
		expectedSet = append(expectedSet, ns.Name)
	}
	expectedSet = normalizeNameserverSet(expectedSet)
	expected := strings.Join(expectedSet, ",")

	if actual != expected {
		return []*models.Correction{
			{
				Msg: fmt.Sprintf("Update nameservers %s -> %s", actual, expected),
				F: func() error {
					return p.client.updateRegistrarNameservers(dc.Name, expectedSet)
				},
			},
		}, nil
	}

	return nil, nil
}

func normalizeNameserverSet(nameservers []string) []string {
	normalized := make([]string, 0, len(nameservers))
	for _, ns := range nameservers {
		normalized = append(normalized, strings.ToLower(strings.TrimSuffix(ns, ".")))
	}
	sort.Strings(normalized)
	return normalized
}

func (p *tencentdnsProvider) EnsureZoneExists(dc *models.DomainConfig) error {
	domainName := dc.Name

	request := dnspod.NewCreateDomainRequest()
	request.Domain = &domainName
	_, err := p.client.dnspodClient.CreateDomain(request)
	if err != nil {
		if strings.Contains(err.Error(), "already exists") {
			return nil
		}
		return err
	}
	return nil
}
