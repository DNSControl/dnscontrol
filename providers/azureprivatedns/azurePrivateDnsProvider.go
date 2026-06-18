package azureprivatedns

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	aauth "github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	adns "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/privatedns/armprivatedns"
	"github.com/DNSControl/dnscontrol/v4/models"
	"github.com/DNSControl/dnscontrol/v4/pkg/diff2"
	"github.com/DNSControl/dnscontrol/v4/pkg/printer"
	"github.com/DNSControl/dnscontrol/v4/pkg/providers"
)

const azurePendingOperationConflictMessage = "Another operation is pending for requested object"

type azurednsProvider struct {
	zonesClient    *adns.PrivateZonesClient
	recordsClient  *adns.RecordSetsClient
	zones          map[string]*adns.PrivateZone
	resourceGroup  *string
	subscriptionID *string
	rawRecords     map[string][]*adns.RecordSet
	zoneName       map[string]string
}

func newAzureDNSDsp(conf map[string]string, metadata json.RawMessage) (providers.DNSServiceProvider, error) {
	return newAzureDNS(conf, metadata)
}

func newAzureDNS(m map[string]string, _ json.RawMessage) (*azurednsProvider, error) {
	subID, rg := m["SubscriptionID"], m["ResourceGroup"]
	clientID, clientSecret, tenantID := m["ClientID"], m["ClientSecret"], m["TenantID"]
	credential, authErr := aauth.NewClientSecretCredential(tenantID, clientID, clientSecret, nil)
	if authErr != nil {
		return nil, authErr
	}
	zonesClient, zoneErr := adns.NewPrivateZonesClient(subID, credential, nil)
	if zoneErr != nil {
		return nil, zoneErr
	}
	recordsClient, recordErr := adns.NewRecordSetsClient(subID, credential, nil)
	if recordErr != nil {
		return nil, recordErr
	}

	api := &azurednsProvider{
		zonesClient:    zonesClient,
		recordsClient:  recordsClient,
		resourceGroup:  new(rg),
		subscriptionID: new(subID),
		rawRecords:     map[string][]*adns.RecordSet{},
		zoneName:       map[string]string{},
	}
	err := api.getZones()
	if err != nil {
		return nil, err
	}
	return api, nil
}

var features = providers.DocumentationNotes{
	// The default for unlisted capabilities is 'Cannot'.
	// See providers/capabilities.go for the entire list of capabilities.
	providers.CanGetZones:            providers.Can(),
	providers.CanConcur:              providers.Unimplemented(),
	providers.CanUseAlias:            providers.Cannot("Azure DNS does not provide a generic ALIAS functionality. Use AZURE_ALIAS instead."),
	providers.CanUseAzureAlias:       providers.Cannot(),
	providers.CanUseCAA:              providers.Cannot("Azure Private DNS does not support CAA records"),
	providers.CanUseLOC:              providers.Cannot(),
	providers.CanUseNAPTR:            providers.Cannot(),
	providers.CanUsePTR:              providers.Can(),
	providers.CanUseSRV:              providers.Can(),
	providers.CanUseSSHFP:            providers.Cannot(),
	providers.CanUseTLSA:             providers.Cannot(),
	providers.DocCreateDomains:       providers.Can(),
	providers.DocDualHost:            providers.Cannot("Private zones can not change NS records"),
	providers.DocOfficiallySupported: providers.Can(),
}

func init() {
	const providerName = "AZURE_PRIVATE_DNS"
	const providerMaintainer = "@matthewmgamble"
	fns := providers.DspFuncs{
		Initializer:   newAzureDNSDsp,
		RecordAuditor: AuditRecords,
	}
	providers.RegisterDomainServiceProviderType(providerName, fns, features)
	providers.RegisterMaintainer(providerName, providerMaintainer)
	providers.RegisterCredsMetadata(providerName, providers.CredsMetadata{
		DisplayName: "Azure Private DNS",
		Kind:        providers.KindDNS,
		DocsURL:     "https://docs.dnscontrol.org/provider/azureprivatedns",
		PortalURL:   "https://portal.azure.com/",
		Fields: []providers.CredsField{
			{
				Key:      "SubscriptionID",
				Label:    "Subscription ID",
				Help:     "Azure subscription ID that contains the private DNS zones.",
				Required: true,
			},
			{
				Key:      "ResourceGroup",
				Label:    "Resource group",
				Help:     "Azure resource group that contains the private DNS zones.",
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

func (a *azurednsProvider) getExistingZones() ([]*adns.PrivateZone, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 6000*time.Second)
	defer cancel()
	zonesPager := a.zonesClient.NewListByResourceGroupPager(*a.resourceGroup, nil)
	var zones []*adns.PrivateZone
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
	a.zones = make(map[string]*adns.PrivateZone)

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
	return fmt.Sprintf("Private 	Domain %s not found in you Azure account", e.domain)
}

func (a *azurednsProvider) GetNameservers(domain string) ([]*models.Nameserver, error) {
	// Azure Private DNS does not have the concept of "Name Servers" since these are local, private views of zones unique to the Azure environment
	var nss []string
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

	a.rawRecords[domain] = rawRecords
	a.zoneName[domain] = zoneName

	return existingRecords, rawRecords, zoneName, nil
}

// GetZoneRecordsCorrections returns a list of corrections that will turn existing records into dc.Records.
func (a *azurednsProvider) GetZoneRecordsCorrections(dc *models.DomainConfig, existingRecords models.Records) ([]*models.Correction, int, error) {
	var corrections []*models.Correction

	changes, actualChangeCount, err := diff2.ByRecordSet(existingRecords, dc, nil)
	if err != nil {
		return nil, 0, err
	}

	for _, change := range changes {
		// Copy all param values to local variables to avoid overwrites
		msgs := change.MsgsJoined
		dcn := dc.Name
		chaKey := change.Key

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
	_, err = a.recordsClient.CreateOrUpdate(ctx, *a.resourceGroup, zoneName, azRecType, recordName, *rrset, nil)

	if retryReason := retryableRecordSetMutation(err); retryReason != "" {
		waitTime = waitTime * 2
		if waitTime > 300 {
			return err
		}
		printer.Printf("AZURE_PRIVATE_DNS: %s paused for %v.\n", retryReason, waitTime)
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

	azRecType, err := nativeToRecordTypeDiff(new(reckey.Type))
	if err != nil {
		return nil
	}

	waitTime := 1
retry:

	ctx, cancel := context.WithTimeout(context.Background(), 6000*time.Second)
	defer cancel()
	_, err = a.recordsClient.Delete(ctx, *a.resourceGroup, zoneName, azRecType, shortName, nil)

	if retryReason := retryableRecordSetMutation(err); retryReason != "" {
		waitTime = waitTime * 2
		if waitTime > 300 {
			return err
		}
		printer.Printf("AZURE_PRIVATE_DNS: %s paused for %v.\n", retryReason, waitTime)
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

func nativeToRecordTypeDiff(recordType *string) (adns.RecordType, error) {
	recordTypeStripped := strings.TrimPrefix(*recordType, "Microsoft.Network/dnszones/")
	switch recordTypeStripped {
	case "A", "AZURE_ALIAS_A":
		return adns.RecordTypeA, nil
	case "AAAA", "AZURE_ALIAS_AAAA":
		return adns.RecordTypeAAAA, nil
	case "CAA":
		// CAA doesn't make any senese in a private dns zone in azure
		return adns.RecordTypeA, fmt.Errorf("nativeToRecordTypeDiff RTYPE %v UNIMPLEMENTED", *recordType)
	case "CNAME", "AZURE_ALIAS_CNAME":
		return adns.RecordTypeCNAME, nil
	case "MX":
		return adns.RecordTypeMX, nil
	case "NS":
		// NS record types don't make any sense in a private azure dns zone
		return adns.RecordTypeA, fmt.Errorf("nativeToRecordTypeDiff RTYPE %v UNIMPLEMENTED", *recordType)
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
		return adns.RecordTypeA, fmt.Errorf("nativeToRecordTypeDiff RTYPE %v UNIMPLEMENTED", *recordType)
	}
}

func nativeToRecords(set *adns.RecordSet, origin string) []*models.RecordConfig {
	var results []*models.RecordConfig
	switch rtype := *set.Type; rtype {
	case "Microsoft.Network/privateDnsZones/A":
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
			panic(fmt.Errorf("nativeToRecords rtype %v unimplemented", *set.Type))
		}
	case "Microsoft.Network/privateDnsZones/AAAA":
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
			panic(fmt.Errorf("nativeToRecords rtype %v unimplemented", *set.Type))
		}
	case "Microsoft.Network/privateDnsZones/CNAME":
		if set.Properties.CnameRecord != nil {
			// This is a CNAME recordset. Process the targets. (there can only be one)
			rc := &models.RecordConfig{TTL: uint32(*set.Properties.TTL), Original: set}
			rc.SetLabelFromFQDN(*set.Properties.Fqdn, origin)
			rc.Type = "CNAME"
			_ = rc.SetTarget(*set.Properties.CnameRecord.Cname)
			results = append(results, rc)
		} else {
			panic(fmt.Errorf("nativeToRecords rtype %v unimplemented", *set.Type))
		}
	case "Microsoft.Network/privateDnsZones/PTR":
		for _, rec := range set.Properties.PtrRecords {
			rc := &models.RecordConfig{TTL: uint32(*set.Properties.TTL), Original: set}
			rc.SetLabelFromFQDN(*set.Properties.Fqdn, origin)
			rc.Type = "PTR"
			_ = rc.SetTarget(*rec.Ptrdname)
			results = append(results, rc)
		}
	case "Microsoft.Network/privateDnsZones/TXT":
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
	case "Microsoft.Network/privateDnsZones/MX":
		for _, rec := range set.Properties.MxRecords {
			rc := &models.RecordConfig{TTL: uint32(*set.Properties.TTL), Original: set}
			rc.SetLabelFromFQDN(*set.Properties.Fqdn, origin)
			rc.Type = "MX"
			_ = rc.SetTargetMX(uint16(*rec.Preference), *rec.Exchange)
			results = append(results, rc)
		}
	case "Microsoft.Network/privateDnsZones/SRV":
		for _, rec := range set.Properties.SrvRecords {
			rc := &models.RecordConfig{TTL: uint32(*set.Properties.TTL), Original: set}
			rc.SetLabelFromFQDN(*set.Properties.Fqdn, origin)
			rc.Type = "SRV"
			_ = rc.SetTargetSRV(uint16(*rec.Priority), uint16(*rec.Weight), uint16(*rec.Port), *rec.Target)
			results = append(results, rc)
		}
	case "Microsoft.Network/privateDnsZones/SOA":
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
		case "PTR":
			recordSet.Properties.PtrRecords = append(recordSet.Properties.PtrRecords, &adns.PtrRecord{Ptrdname: new(rec.GetTargetField())})
		case "TXT":
			// When a TXT record is empty, Azure requires that the .Properties.TxtRecords have no value, not "".
			if rec.GetTargetTXTJoined() != "" {
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
			/* CAA records don't work in a private zone */
		case "AZURE_ALIAS_A", "AZURE_ALIAS_AAAA", "AZURE_ALIAS_CNAME":
			return nil, adns.RecordTypeA, fmt.Errorf("recordToNativeDiff2 RTYPE %v UNIMPLEMENTED", recordKeyType) // ands.A is a placeholder
		default:
			return nil, adns.RecordTypeA, fmt.Errorf("recordToNativeDiff2 RTYPE %v UNIMPLEMENTED", recordKeyType) // ands.A is a placeholder
		}
	}

	rt, err := nativeToRecordTypeDiff(new(*recordSet.Type))
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

	recordsPager := a.recordsClient.NewListPager(*a.resourceGroup, zoneName, nil)

	for recordsPager.More() {
		waitTime := 1
	retry:

		nextResult, recordsErr := recordsPager.NextPage(ctx)

		if recordsErr != nil {
			err := recordsErr
			if e, ok := err.(*azcore.ResponseError); ok {
				if e.StatusCode == http.StatusTooManyRequests {
					waitTime = waitTime * 2
					if waitTime > 300 {
						return nil, err
					}
					printer.Printf("AZURE_PRIVATE_DNS: rate-limit paused for %v.\n", waitTime)
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
	return nil
}
