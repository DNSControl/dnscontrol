package azuredns

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	aauth "github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	adns "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/dns/armdns"

	"github.com/DNSControl/dnscontrol/v4/models"
	"github.com/DNSControl/dnscontrol/v4/pkg/diff2"
	"github.com/DNSControl/dnscontrol/v4/pkg/printer"
	"github.com/DNSControl/dnscontrol/v4/pkg/providers"
)

const azurePendingOperationConflictMessage = "Another operation is pending for requested object"

type azurednsProvider struct {
	zonesClient    *adns.ZonesClient
	recordsClient  *adns.RecordSetsClient
	zones          map[string]*adns.Zone
	resourceGroup  *string
	subscriptionID *string
}

// Modified `newAzureDNSDsp` to maintain backward compatibility with the new OIDC support.
func newAzureDNSDsp(conf map[string]string, metadata json.RawMessage) (providers.DNSServiceProvider, error) {
	return newAzureDNS(conf, metadata)
}

// Updated function to prioritize DefaultAzureCredential and fallback to OIDC or client-secret-based methods.
func newAzureDNS(m map[string]string, _ json.RawMessage) (*azurednsProvider, error) {
	subID, rg := m["SubscriptionID"], m["ResourceGroup"]
	clientID, clientSecret, tenantID := m["ClientID"], m["ClientSecret"], m["TenantID"]
	useOIDC := m["UseOIDC"] == "true"

	var credential azcore.TokenCredential
	var authErr error

	// Authentication Logic
	if useOIDC {
		// OIDC Authentication with `InteractiveBrowserCredential`
		oidcCredentialOpts := aauth.InteractiveBrowserCredentialOptions{
			TenantID: tenantID,
		}
		credential, authErr = aauth.NewInteractiveBrowserCredential(&oidcCredentialOpts)
		if authErr != nil {
			return nil, fmt.Errorf("failed to create OIDC credential: %w", authErr)
		}
	} else if clientID != "" && clientSecret != "" {
		// Client ID and Secret-based Authentication
		credential, authErr = aauth.NewClientSecretCredential(tenantID, clientID, clientSecret, nil)
		if authErr != nil {
			return nil, fmt.Errorf("failed to create Client Secret credential: %w", authErr)
		}
	} else {
		// Default Azure Credential as the default mechanism
		credential, authErr = aauth.NewDefaultAzureCredential(nil)
		if authErr != nil {
			return nil, fmt.Errorf("failed to create Default Azure credential: %w", authErr)
		}
	}

	// Create DNS clients using the selected credential
	zonesClient, err := adns.NewZonesClient(subID, credential, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create zones client: %w", err)
	}

	recordsClient, err := adns.NewRecordSetsClient(subID, credential, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create records client: %w", err)
	}

	api := &azurednsProvider{
		zonesClient:    zonesClient,
		recordsClient:  recordsClient,
		resourceGroup:  new(rg),
		subscriptionID: new(subID),
	}

	// Initialize zones
	if err := api.getZones(); err != nil {
		return nil, err
	}

	return api, nil
}

var features = providers.DocumentationNotes{
	// The default for unlisted capabilities is 'Cannot'.
	// See providers/capabilities.go for the entire list of capabilities.
	providers.CanGetZones:            providers.Can(),
	providers.CanConcur:              providers.Can(),
	providers.CanUseAlias:            providers.Cannot("Azure DNS does not provide a generic ALIAS functionality. Use AZURE_ALIAS instead."),
	providers.CanUseAzureAlias:       providers.Can(),
	providers.CanUseCAA:              providers.Can(),
	providers.CanUseLOC:              providers.Cannot(),
	providers.CanUseNAPTR:            providers.Cannot(),
	providers.CanUsePTR:              providers.Can(),
	providers.CanUseSRV:              providers.Can(),
	providers.CanUseSSHFP:            providers.Cannot(),
	providers.CanUseTLSA:             providers.Cannot(),
	providers.DocCreateDomains:       providers.Can(),
	providers.DocDualHost:            providers.Can("Azure does not permit modifying the existing NS records, only adding/removing additional records."),
	providers.DocOfficiallySupported: providers.Can(),
}

func init() {
	const providerName = "AZURE_DNS"
	const providerMaintainer = "@vatsalyagoel"
	fns := providers.DspFuncs{
		Initializer:   newAzureDNSDsp,
		RecordAuditor: AuditRecords,
	}
	providers.RegisterDomainServiceProviderType(providerName, fns, features)
	providers.RegisterCustomRecordType("AZURE_ALIAS", providerName, "")
	providers.RegisterMaintainer(providerName, providerMaintainer)
	providers.RegisterCredsMetadata(providerName, providers.CredsMetadata{
		DisplayName: "Azure DNS",
		Kind:        providers.KindDNS,
		DocsURL:     "https://docs.dnscontrol.org/provider/azuredns",
		PortalURL:   "https://portal.azure.com/#view/Microsoft_AAD_IAM/ActiveDirectoryMenuBlade/~/RegisteredApps",
		Fields: []providers.CredsField{
			{
				Key:      "SubscriptionID",
				Label:    "Subscription ID",
				Help:     "Azure subscription ID that contains the DNS zones.",
				Required: true,
			},
			{
				Key:      "ResourceGroup",
				Label:    "Resource group",
				Help:     "Azure resource group that contains the DNS zones.",
				Required: true,
			},
			{
				Key:      "TenantID",
				Label:    "Tenant ID",
				Help:     "Azure AD tenant ID for the service principal.",
				Required: true,
			},
			{
				Key:      "ClientID",
				Label:    "Client ID",
				Help:     "Service principal client (application) ID.",
				Required: true,
			},
			{
				Key:      "ClientSecret",
				Label:    "Client secret",
				Help:     "Service principal client secret.",
				Secret:   true,
				Required: true,
			},
		},
	})
}

func (a *azurednsProvider) getExistingZones() ([]*adns.Zone, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 6000*time.Second)
	defer cancel()
	zonesPager := a.zonesClient.NewListByResourceGroupPager(*a.resourceGroup, nil)
	var zones []*adns.Zone
	for zonesPager.More() {
		nextResult, zonesErr := zonesPager.NextPage(ctx)
		if zonesErr != nil {
			return nil, zonesErr
		}
		zones = append(zones, nextResult.Value...)
	}
	return zones, nil
}

func (a *azurednsProvider) getZones() error {
	a.zones = make(map[string]*adns.Zone)

	zones, err := a.getExistingZones()
	if err != nil {
		return err
	}

	for _, z := range zones {
		zone := z
		domain := strings.TrimSuffix(*z.Name, ".")
		a.zones[domain] = zone
	}

	return nil
}

type errNoExist struct {
	domain string
}

func (e errNoExist) Error() string {
	return fmt.Sprintf("Domain %s not found in you Azure account", e.domain)
}

func (a *azurednsProvider) GetNameservers(domain string) ([]*models.Nameserver, error) {
	zone, ok := a.zones[domain]
	if !ok {
		return nil, errNoExist{domain}
	}

	var nss []string
	if zone.Properties != nil {
		for _, ns := range zone.Properties.NameServers {
			nss = append(nss, *ns)
		}

		nonDefaultNss, err := a.getNameNonDefaultNameServers(domain, nss)
		if err != nil {
			return nil, err
		}

		nss = append(nss, nonDefaultNss...)
	}

	return models.ToNameserversStripTD(nss)
}

func (a *azurednsProvider) ListZones() ([]string, error) {
	zonesResult, err := a.getExistingZones()
	if err != nil {
		return nil, err
	}
	var zones []string

	for _, z := range zonesResult {
		domain := strings.TrimSuffix(*z.Name, ".")
		zones = append(zones, domain)
	}

	return zones, nil
}

func (a *azurednsProvider) getNameNonDefaultNameServers(domain string, nss []string) ([]string, error) {
	zone, ok := a.zones[domain]
	if !ok {
		return nil, errNoExist{domain}
	}
	zoneName := *zone.Name
	var nameServers []string
	ctx, cancel := context.WithTimeout(context.Background(), 6000*time.Second)
	defer cancel()
	recordsPager := a.recordsClient.NewListByTypePager(*a.resourceGroup, zoneName, "NS", nil)

	for recordsPager.More() {
		waitTime := 1
	retry:
		nextResult, recordsErr := recordsPager.NextPage(ctx)

		if recordsErr != nil {
			err := recordsErr
			var e *azcore.ResponseError
			if errors.As(err, &e) {
				if e.StatusCode == http.StatusTooManyRequests {
					waitTime = waitTime * 2
					if waitTime > 300 {
						return nil, err
					}
					printer.Printf("AZURE_DNS: rate-limit paused for %v.\n", waitTime)
					time.Sleep(time.Duration(waitTime+1) * time.Second)
					goto retry
				}
			}
		}

		for _, record := range nextResult.Value {
			if record.Properties != nil && domain == removeTrailingDot(*record.Properties.Fqdn) {
				for _, ns := range record.Properties.NsRecords {
					if !slices.Contains(nss, *ns.Nsdname) {
						nameServers = append(nameServers, *ns.Nsdname)
					}
				}
			}
		}
	}

	return nameServers, nil
}

// GetZoneRecords gets the records of a zone and returns them in RecordConfig format.
func (a *azurednsProvider) GetZoneRecords(dc *models.DomainConfig) (models.Records, error) {
	domain := dc.Name

	existingRecords, _, _, err := a.getExistingRecords(domain)
	if err != nil {
		return nil, err
	}

	return existingRecords, nil
}

func (a *azurednsProvider) getExistingRecords(domain string) (models.Records, []*adns.RecordSet, string, error) {
	zone, ok := a.zones[domain]
	if !ok {
		return nil, nil, "", errNoExist{domain}
	}
	zoneName := *zone.Name
	rawRecords, err := a.fetchRecordSets(zoneName)
	if err != nil {
		return nil, nil, "", err
	}

	var existingRecords models.Records
	for _, set := range rawRecords {
		existingRecords = append(existingRecords, nativeToRecords(set, zoneName)...)
	}

	return existingRecords, rawRecords, zoneName, nil
}

// GetZoneRecordsCorrections returns a list of corrections that will turn existing records into dc.Records.
func (a *azurednsProvider) GetZoneRecordsCorrections(dc *models.DomainConfig, existingRecords models.Records) ([]*models.Correction, int, error) {
	var corrections []*models.Correction

	// Azure is a "ByRecordSet" API.

	changes, actualChangeCount, err := diff2.ByRecordSet(existingRecords, dc, nil)
	if err != nil {
		return nil, 0, err
	}

	for _, change := range changes {
		// Copy all param values to local variables to avoid overwrites
		msgs := change.MsgsJoined
		dcn := dc.Name
		chaKey := change.Key

		if change.Type == diff2.CHANGE || change.Type == diff2.CREATE {
			if chaKey.Type == "NS" && dcn == removeTrailingDot(change.Key.NameFQDN) {
				change.New = deduplicateNameServerTargets(change.New)
			}
		}

		switch change.Type {
		case diff2.REPORT:
			corrections = append(corrections, &models.Correction{Msg: change.MsgsJoined})
		case diff2.CHANGE, diff2.CREATE:
			changeNew := change.New
			corrections = append(corrections, &models.Correction{
				Msg: msgs,
				F: func() error {
					return a.recordCreate(dcn, chaKey, changeNew)
				},
			})
		case diff2.DELETE:
			corrections = append(corrections, &models.Correction{
				Msg: msgs,
				F: func() error {
					return a.recordDelete(dcn, chaKey)
				},
			})
		default:
			panic(fmt.Sprintf("unhandled change.Type %s", change.Type))
		}
	}

	return corrections, actualChangeCount, nil
}

func (a *azurednsProvider) recordCreate(zoneName string, reckey models.RecordKey, recs models.Records) error {
	rrset, azRecType, err := a.recordToNativeDiff2(reckey, recs)
	if err != nil {
		return err
	}

	var recordName string
	var i int64
	for _, r := range recs {
		i = int64(r.TTL)
		recordName = r.Name
	}
	rrset.Properties.TTL = &i

	waitTime := 1

retry:
	ctx, cancel := context.WithTimeout(context.Background(), 6000*time.Second)
	defer cancel()
	_, err = a.recordsClient.CreateOrUpdate(ctx, *a.resourceGroup, zoneName, recordName, azRecType, *rrset, nil)

	if retryReason := retryableRecordSetMutation(err); retryReason != "" {
		waitTime = waitTime * 2
		if waitTime > 300 {
			return err
		}
		printer.Printf("AZURE_DNS: %s paused for %v.\n", retryReason, waitTime)
		time.Sleep(time.Duration(waitTime+1) * time.Second)
		goto retry
	}

	return err
}

func (a *azurednsProvider) recordDelete(zoneName string, reckey models.RecordKey) error {
	shortName := strings.TrimSuffix(reckey.NameFQDN, "."+zoneName)
	if shortName == zoneName {
		shortName = "@"
	}

	azRecType, err := nativeToRecordTypeDiff2(new(reckey.Type))
	if err != nil {
		return nil
	}

	waitTime := 1
retry:

	ctx, cancel := context.WithTimeout(context.Background(), 6000*time.Second)
	defer cancel()
	_, err = a.recordsClient.Delete(ctx, *a.resourceGroup, zoneName, shortName, azRecType, nil)

	if retryReason := retryableRecordSetMutation(err); retryReason != "" {
		waitTime = waitTime * 2
		if waitTime > 300 {
			return err
		}
		printer.Printf("AZURE_DNS: %s paused for %v.\n", retryReason, waitTime)
		time.Sleep(time.Duration(waitTime+1) * time.Second)
		goto retry
	}

	return err
}

func retryableRecordSetMutation(err error) string {
	var e *azcore.ResponseError
	if !errors.As(err, &e) {
		return ""
	}

	if e.StatusCode == http.StatusTooManyRequests {
		return "rate-limit"
	}

	if e.StatusCode == http.StatusConflict && e.ErrorCode == "Conflict" && strings.Contains(e.Error(), azurePendingOperationConflictMessage) {
		return "pending operation"
	}

	return ""
}

func nativeToRecordTypeDiff2(recordType *string) (adns.RecordType, error) {
	recordTypeStripped := strings.TrimPrefix(*recordType, "Microsoft.Network/dnszones/")
	switch recordTypeStripped {
	case "A", "AZURE_ALIAS_A":
		return adns.RecordTypeA, nil
	case "AAAA", "AZURE_ALIAS_AAAA":
		return adns.RecordTypeAAAA, nil
	case "CAA":
		return adns.RecordTypeCAA, nil
	case "CNAME", "AZURE_ALIAS_CNAME":
		return adns.RecordTypeCNAME, nil
	case "MX":
		return adns.RecordTypeMX, nil
	case "NS":
		return adns.RecordTypeNS, nil
	case "PTR":
		return adns.RecordTypePTR, nil
	case "SRV":
		return adns.RecordTypeSRV, nil
	case "TXT":
		return adns.RecordTypeTXT, nil
	case "SOA":
		return adns.RecordTypeSOA, nil
	default:
		// Unimplemented type. Return adns.A as a decoy, but send an error.
		return adns.RecordTypeA, fmt.Errorf("nativeToRecordTypeDiff2 RTYPE %v UNIMPLEMENTED", *recordType)
	}
}

func safeTarget(t *string) string {
	if t == nil {
		return "foundnil"
	}
	return *t
}

func nativeToRecords(set *adns.RecordSet, origin string) []*models.RecordConfig {
	var results []*models.RecordConfig
	switch rtype := *set.Type; rtype {
	case "Microsoft.Network/dnszones/A":
		if set.Properties.ARecords != nil {
			// This is an A recordset. Process all the targets there.
			for _, rec := range set.Properties.ARecords {
				rc := &models.RecordConfig{TTL: uint32(*set.Properties.TTL), Original: set}
				rc.SetLabelFromFQDN(*set.Properties.Fqdn, origin)
				rc.Type = "A"
				_ = rc.SetTarget(*rec.IPv4Address)
				results = append(results, rc)
			}
		} else {
			// This is an AZURE_ALIAS of an "A" record.
			rc := &models.RecordConfig{
				Type: "AZURE_ALIAS",
				TTL:  uint32(*set.Properties.TTL),
				AzureAlias: map[string]string{
					"type": "A",
				},
				Original: set,
			}
			rc.SetLabelFromFQDN(*set.Properties.Fqdn, origin)
			_ = rc.SetTarget(safeTarget(set.Properties.TargetResource.ID))
			results = append(results, rc)
		}
	case "Microsoft.Network/dnszones/AAAA":
		if set.Properties.AaaaRecords != nil {
			// This is an AAAA recordset. Process all the targets there.
			for _, rec := range set.Properties.AaaaRecords {
				rc := &models.RecordConfig{TTL: uint32(*set.Properties.TTL), Original: set}
				rc.SetLabelFromFQDN(*set.Properties.Fqdn, origin)
				rc.Type = "AAAA"
				_ = rc.SetTarget(*rec.IPv6Address)
				results = append(results, rc)
			}
		} else {
			// This is an AZURE_ALIAS of an "AAAA" record.
			rc := &models.RecordConfig{
				Type: "AZURE_ALIAS",
				TTL:  uint32(*set.Properties.TTL),
				AzureAlias: map[string]string{
					"type": "AAAA",
				},
				Original: set,
			}
			rc.SetLabelFromFQDN(*set.Properties.Fqdn, origin)
			_ = rc.SetTarget(safeTarget(set.Properties.TargetResource.ID))
			results = append(results, rc)
		}
	case "Microsoft.Network/dnszones/CNAME":
		if set.Properties.CnameRecord != nil {
			// This is a CNAME recordset. Process the targets. (there can only be one)
			rc := &models.RecordConfig{TTL: uint32(*set.Properties.TTL), Original: set}
			rc.SetLabelFromFQDN(*set.Properties.Fqdn, origin)
			rc.Type = "CNAME"
			_ = rc.SetTarget(*set.Properties.CnameRecord.Cname)
			results = append(results, rc)
		} else {
			// This is an AZURE_ALIAS of a "CNAME" record.
			rc := &models.RecordConfig{
				Type: "AZURE_ALIAS",
				TTL:  uint32(*set.Properties.TTL),
				AzureAlias: map[string]string{
					"type": "CNAME",
				},
				Original: set,
			}
			rc.SetLabelFromFQDN(*set.Properties.Fqdn, origin)
			_ = rc.SetTarget(safeTarget(set.Properties.TargetResource.ID))
			results = append(results, rc)
		}
	case "Microsoft.Network/dnszones/NS":
		for _, rec := range set.Properties.NsRecords {
			rc := &models.RecordConfig{TTL: uint32(*set.Properties.TTL), Original: set}
			rc.SetLabelFromFQDN(*set.Properties.Fqdn, origin)
			rc.Type = "NS"
			_ = rc.SetTarget(*rec.Nsdname)
			results = append(results, rc)
		}
	case "Microsoft.Network/dnszones/PTR":
		for _, rec := range set.Properties.PtrRecords {
			rc := &models.RecordConfig{TTL: uint32(*set.Properties.TTL), Original: set}
			rc.SetLabelFromFQDN(*set.Properties.Fqdn, origin)
			rc.Type = "PTR"
			_ = rc.SetTarget(*rec.Ptrdname)
			results = append(results, rc)
		}
	case "Microsoft.Network/dnszones/TXT":
		if len(set.Properties.TxtRecords) == 0 { // Empty String Record Parsing
			// This is a null TXT record.
			rc := &models.RecordConfig{TTL: uint32(*set.Properties.TTL), Original: set}
			rc.SetLabelFromFQDN(*set.Properties.Fqdn, origin)
			rc.Type = "TXT"
			_ = rc.SetTargetTXT("")
			results = append(results, rc)
		} else {
			// This is a normal TXT record. Collect all its segments.
			for _, rec := range set.Properties.TxtRecords {
				rc := &models.RecordConfig{TTL: uint32(*set.Properties.TTL), Original: set}
				rc.SetLabelFromFQDN(*set.Properties.Fqdn, origin)
				rc.Type = "TXT"
				var txts []string
				for _, txt := range rec.Value {
					txts = append(txts, *txt)
				}
				_ = rc.SetTargetTXTs(txts)
				results = append(results, rc)
			}
		}
	case "Microsoft.Network/dnszones/MX":
		for _, rec := range set.Properties.MxRecords {
			rc := &models.RecordConfig{TTL: uint32(*set.Properties.TTL), Original: set}
			rc.SetLabelFromFQDN(*set.Properties.Fqdn, origin)
			rc.Type = "MX"
			_ = rc.SetTargetMX(uint16(*rec.Preference), *rec.Exchange)
			results = append(results, rc)
		}
	case "Microsoft.Network/dnszones/SRV":
		for _, rec := range set.Properties.SrvRecords {
			rc := &models.RecordConfig{TTL: uint32(*set.Properties.TTL), Original: set}
			rc.SetLabelFromFQDN(*set.Properties.Fqdn, origin)
			rc.Type = "SRV"
			_ = rc.SetTargetSRV(uint16(*rec.Priority), uint16(*rec.Weight), uint16(*rec.Port), *rec.Target)
			results = append(results, rc)
		}
	case "Microsoft.Network/dnszones/CAA":
		for _, rec := range set.Properties.CaaRecords {
			rc := &models.RecordConfig{TTL: uint32(*set.Properties.TTL), Original: set}
			rc.SetLabelFromFQDN(*set.Properties.Fqdn, origin)
			rc.Type = "CAA"
			_ = rc.SetTargetCAA(uint8(*rec.Flags), *rec.Tag, *rec.Value)
			results = append(results, rc)
		}
	case "Microsoft.Network/dnszones/SOA":
	default:
		panic(fmt.Errorf("nativeToRecords rtype %v unimplemented", *set.Type))
	}
	return results
}

// NOTE recordToNativeDiff2 is really "convert []RecordConfig to rrset".

func (a *azurednsProvider) recordToNativeDiff2(recordKey models.RecordKey, recordConfig []*models.RecordConfig) (*adns.RecordSet, adns.RecordType, error) {
	recordKeyType := recordKey.Type
	//	if recordKeyType == "AZURE_ALIAS" {
	//		fmt.Fprintf(os.Stderr, "DEBUG: XXXXXXXXXXXXXXXXXXXXXXX %v\n", recordKeyType)
	//	}

	recordSet := &adns.RecordSet{Type: new(recordKeyType), Properties: &adns.RecordSetProperties{}}
	for _, rec := range recordConfig {
		switch recordKeyType {
		case "A":
			recordSet.Properties.ARecords = append(recordSet.Properties.ARecords, &adns.ARecord{IPv4Address: new(rec.GetTargetField())})
		case "AAAA":
			recordSet.Properties.AaaaRecords = append(recordSet.Properties.AaaaRecords, &adns.AaaaRecord{IPv6Address: new(rec.GetTargetField())})
		case "CNAME":
			recordSet.Properties.CnameRecord = &adns.CnameRecord{Cname: new(rec.GetTargetField())}
		case "NS":
			recordSet.Properties.NsRecords = append(recordSet.Properties.NsRecords, &adns.NsRecord{Nsdname: new(rec.GetTargetField())})
		case "PTR":
			recordSet.Properties.PtrRecords = append(recordSet.Properties.PtrRecords, &adns.PtrRecord{Ptrdname: new(rec.GetTargetField())})
		case "TXT":
			// When a TXT record is empty, Azure requires that the .Properties.TxtRecords have no value, not "". Therefore we skip this.
			if rec.GetTargetTXTSegmentCount() != 1 || rec.GetTargetTXTSegmented()[0] != "" {
				var txts []*string
				for _, txt := range rec.GetTargetTXTSegmented() {
					txts = append(txts, new(txt))
				}
				recordSet.Properties.TxtRecords = append(recordSet.Properties.TxtRecords, &adns.TxtRecord{Value: txts})
			}
		case "MX":
			recordSet.Properties.MxRecords = append(recordSet.Properties.MxRecords, &adns.MxRecord{Exchange: new(rec.GetTargetField()), Preference: new(int32(rec.MxPreference))})
		case "SRV":
			recordSet.Properties.SrvRecords = append(recordSet.Properties.SrvRecords, &adns.SrvRecord{Target: new(rec.GetTargetField()), Port: new(int32(rec.SrvPort)), Weight: new(int32(rec.SrvWeight)), Priority: new(int32(rec.SrvPriority))})
		case "CAA":
			recordSet.Properties.CaaRecords = append(recordSet.Properties.CaaRecords, &adns.CaaRecord{Value: new(rec.GetTargetField()), Tag: new(rec.CaaTag), Flags: new(int32(rec.CaaFlag))})
		case "AZURE_ALIAS_A", "AZURE_ALIAS_AAAA", "AZURE_ALIAS_CNAME":
			recordSet.Type = new(rec.AzureAlias["type"])
			recordSet.Properties.TargetResource = new(adns.SubResource{ID: new(rec.GetTargetField())})

		default:
			return nil, adns.RecordTypeA, fmt.Errorf("recordToNativeDiff2 RTYPE %v UNIMPLEMENTED", recordKeyType) // ands.A is a placeholder
		}
	}

	rt, err := nativeToRecordTypeDiff2(new(*recordSet.Type))
	if err != nil {
		return nil, adns.RecordTypeA, err // adns.A is a placeholder
	}
	return recordSet, rt, nil
}

func (a *azurednsProvider) fetchRecordSets(zoneName string) ([]*adns.RecordSet, error) {
	if zoneName == "" {
		return nil, nil
	}
	var records []*adns.RecordSet
	ctx, cancel := context.WithTimeout(context.Background(), 6000*time.Second)
	defer cancel()
	recordsPager := a.recordsClient.NewListAllByDNSZonePager(*a.resourceGroup, zoneName, nil)

	for recordsPager.More() {
		waitTime := 1
	retry:

		nextResult, recordsErr := recordsPager.NextPage(ctx)

		if recordsErr != nil {
			err := recordsErr
			var e *azcore.ResponseError
			if errors.As(err, &e) {
				if e.StatusCode == http.StatusTooManyRequests {
					waitTime = waitTime * 2
					if waitTime > 300 {
						return nil, err
					}
					printer.Printf("AZURE_DNS: rate-limit paused for %v.\n", waitTime)
					time.Sleep(time.Duration(waitTime+1) * time.Second)
					goto retry
				}
			}
		}

		records = append(records, nextResult.Value...)
	}

	return records, nil
}

func (a *azurednsProvider) EnsureZoneExists(domain string, metadata map[string]string) error {
	if _, ok := a.zones[domain]; ok {
		return nil
	}
	printer.Printf("Adding zone for %s to Azure dns account\n", domain)

	ctx, cancel := context.WithTimeout(context.Background(), 6000*time.Second)
	defer cancel()

	z, err := a.zonesClient.CreateOrUpdate(ctx, *a.resourceGroup, domain, adns.Zone{Location: new("global")}, nil)
	if err != nil {
		return err
	}
	a.zones[domain] = &z.Zone
	return nil
}

func removeTrailingDot(record string) string {
	return strings.TrimSuffix(record, ".")
}

func deduplicateNameServerTargets(newRecs models.Records) models.Records {
	dedupedMap := make(map[string]bool)
	var deduped models.Records
	for _, rec := range newRecs {
		if !dedupedMap[rec.GetTargetField()] {
			dedupedMap[rec.GetTargetField()] = true
			deduped = append(deduped, rec)
		}
	}
	return deduped
}
