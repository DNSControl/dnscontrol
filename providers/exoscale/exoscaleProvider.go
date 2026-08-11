package exoscale

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"

	egoscale "github.com/exoscale/egoscale/v3"
	"github.com/exoscale/egoscale/v3/credentials"

	"github.com/DNSControl/dnscontrol/v4/models"
	"github.com/DNSControl/dnscontrol/v4/pkg/diff"
	"github.com/DNSControl/dnscontrol/v4/pkg/printer"
	"github.com/DNSControl/dnscontrol/v4/pkg/providers"
)

type exoscaleProvider struct {
	client *egoscale.Client
}

// NewExoscale creates a new Exoscale DNS provider.
func NewExoscale(m map[string]string, _ json.RawMessage) (providers.DNSServiceProvider, error) {
	apiKey, secretKey := m["apikey"], m["secretkey"]

	creds := credentials.NewStaticCredentials(apiKey, secretKey)
	client, err := egoscale.NewClient(creds)
	if err != nil {
		return nil, err
	}

	// Endpoint is only for internal use now, not for production.
	endpoint := os.Getenv("EXOSCALE_API_ENDPOINT")
	if endpoint != "" {
		client = client.WithEndpoint(egoscale.Endpoint(endpoint))
	}

	ctx := context.Background()
	if zone, ok := m["apizone"]; ok {
		endpoint, err := client.GetZoneAPIEndpoint(ctx, egoscale.ZoneName(zone))
		if err != nil {
			return nil, fmt.Errorf("switch client zone: %w", err)
		}
		client = client.WithEndpoint(endpoint)
	}

	return &exoscaleProvider{
		client: client,
	}, nil
}

var features = providers.DocumentationNotes{
	// The default for unlisted capabilities is 'Cannot'.
	// See providers/capabilities.go for the entire list of capabilities.
	providers.CanGetZones:            providers.Unimplemented(),
	providers.CanConcur:              providers.Cannot(),
	providers.CanUseAlias:            providers.Can(),
	providers.CanUseCAA:              providers.Can(),
	providers.CanUseLOC:              providers.Cannot(),
	providers.CanUsePTR:              providers.Can(),
	providers.CanUseSRV:              providers.Can("SRV records with empty targets are not supported"),
	providers.CanUseTLSA:             providers.Cannot(),
	providers.DocCreateDomains:       providers.Cannot(),
	providers.DocDualHost:            providers.Cannot("Exoscale does not allow sufficient control over the apex NS records"),
	providers.DocOfficiallySupported: providers.Cannot(),
}

func init() {
	const providerName = "EXOSCALE"
	const providerMaintainer = "@Giza"
	fns := providers.DspFuncs{
		Initializer:   NewExoscale,
		RecordAuditor: AuditRecords,
	}
	providers.RegisterDomainServiceProviderType(providerName, fns, features)
	providers.RegisterMaintainer(providerName, providerMaintainer)
}

// EnsureZoneExists creates a zone if it does not exist.
func (provider *exoscaleProvider) EnsureZoneExists(dc *models.DomainConfig) error {
	domain := dc.Name
	_, err := provider.findDomainByName(domain)
	if errors.Is(err, egoscale.ErrNotFound) {
		_, err = provider.client.CreateDNSDomain(context.Background(), egoscale.CreateDNSDomainRequest{
			UnicodeName: domain,
		})
	}

	return err
}

// GetNameservers returns the nameservers for domain.
func (provider *exoscaleProvider) GetNameservers(_ string) ([]*models.Nameserver, error) {
	return nil, nil
}

// GetZoneRecords gets the records of a zone and returns them in RecordConfig format.
func (provider *exoscaleProvider) GetZoneRecords(domainConfig *models.DomainConfig) (models.Records, error) {
	domainName := domainConfig.Name

	domain, err := provider.findDomainByName(domainName)
	if err != nil {
		return nil, err
	}
	domainID := domain.ID

	ctx := context.Background()
	records, err := provider.client.ListDNSDomainRecords(ctx, domainID)
	if err != nil {
		return nil, err
	}

	existingRecords := make([]*models.RecordConfig, 0, len(records.DNSDomainRecords))
	for i := range records.DNSDomainRecords {
		recordConfig, err := nativeToRecord(&records.DNSDomainRecords[i], domainName)
		if err != nil {
			return nil, err
		}
		if recordConfig != nil {
			existingRecords = append(existingRecords, recordConfig)
		}
	}

	return existingRecords, nil
}

// nativeToRecord converts an Exoscale DNS record to a RecordConfig.
// Returns nil, nil for record types that should be silently skipped (SOA, NS, TXT ALIAS mirrors).
func nativeToRecord(record *egoscale.DNSDomainRecord, domainName string) (*models.RecordConfig, error) {
	recordContent := record.Content

	if record.Type == "SOA" || record.Type == "NS" {
		return nil, nil
	}
	if record.Name == "" {
		record.Name = "@"
	}
	if record.Type == "CNAME" || record.Type == "MX" || record.Type == "ALIAS" || record.Type == "SRV" {
		if !strings.HasSuffix(recordContent, ".") {
			recordContent += "."
		}
		// for SRV records we need to additionally prefix target with priority, which API handles as separate field.
		if record.Type == "SRV" && record.Priority != 0 {
			recordContent = fmt.Sprintf("%d %s", record.Priority, recordContent)
		}
	}
	// Based on tests, exoscale adds these odd txt records that mirror the alias records.
	if record.Type == "TXT" && strings.HasPrefix(recordContent, "ALIAS for ") {
		return nil, nil
	}

	recordConfig := &models.RecordConfig{
		Original: record,
	}
	if record.Ttl != 0 {
		recordConfig.TTL = uint32(record.Ttl)
	}
	recordConfig.SetLabel(record.Name, domainName)

	var err error
	switch record.Type {
	case "ALIAS", "URL":
		recordConfig.Type = string(record.Type)
		_ = recordConfig.SetTarget(recordContent)
	case "MX":
		var priority uint16
		if record.Priority != 0 {
			priority = uint16(record.Priority)
		}
		err = recordConfig.SetTargetMX(priority, recordContent)
	default:
		err = recordConfig.PopulateFromString(string(record.Type), recordContent, domainName)
	}
	if err != nil {
		return nil, fmt.Errorf("unparsable record received from exoscale: %w", err)
	}

	return recordConfig, nil
}

// GetZoneRecordsCorrections returns a list of corrections that will turn existing records into dc.Records.
func (provider *exoscaleProvider) GetZoneRecordsCorrections(
	domainConfig *models.DomainConfig,
	existingRecords models.Records) ([]*models.Correction, int, error) {
	removeOtherNS(domainConfig)
	domain, err := provider.findDomainByName(domainConfig.Name)
	if err != nil {
		return nil, 0, err
	}

	toReport, toCreate, toDelete, toUpdate, actualChangeCount, err := diff.NewCompat(domainConfig).IncrementalDiff(existingRecords)
	if err != nil {
		return nil, 0, err
	}
	// Start corrections with the reports
	corrections := diff.GenerateMessageCorrections(toReport)

	for _, deletionCorrelation := range toDelete {
		record := deletionCorrelation.Existing.Original.(*egoscale.DNSDomainRecord)
		corrections = append(corrections, &models.Correction{
			Msg: deletionCorrelation.String(),
			F:   provider.deleteRecordFunc(record.ID, domain.ID),
		})
	}

	for _, creationCorrelation := range toCreate {
		recordConfig := creationCorrelation.Desired
		corrections = append(corrections, &models.Correction{
			Msg: creationCorrelation.String(),
			F:   provider.createRecordFunc(recordConfig, domain.ID),
		})
	}

	for _, updateCorrelation := range toUpdate {
		oldc := updateCorrelation.Existing.Original.(*egoscale.DNSDomainRecord)
		newc := updateCorrelation.Desired
		corrections = append(corrections, &models.Correction{
			Msg: updateCorrelation.String(),
			F:   provider.updateRecordFunc(oldc, newc, domain.ID),
		})
	}

	return corrections, actualChangeCount, nil
}

// Returns a function that can be invoked to create a record in a zone.
func (provider *exoscaleProvider) createRecordFunc(
	recordConfig *models.RecordConfig,
	domainID egoscale.UUID) func() error {
	return func() error {
		target := recordConfig.GetTargetCombined()
		name := recordConfig.GetLabel()
		var prio int64

		if recordConfig.Type == "MX" {
			target = recordConfig.GetTargetField()

			if recordConfig.MxPreference != 0 {
				prio = int64(recordConfig.MxPreference)
			}
		}

		if recordConfig.Type == "SRV" {
			// API wants priority as a separate argument, here we will strip it from combined target.
			sp := strings.Split(target, " ")
			target = strings.Join(sp[1:], " ")
			p, err := strconv.ParseInt(sp[0], 10, 64)
			if err != nil {
				return err
			}
			prio = p
		}

		if recordConfig.Type == "NS" && (name == "@" || name == "") {
			name = "*"
		}

		record := egoscale.CreateDNSDomainRecordRequest{
			Name:     name,
			Type:     egoscale.CreateDNSDomainRecordRequestType(recordConfig.Type),
			Content:  target,
			Priority: prio,
		}

		if recordConfig.TTL != 0 {
			record.Ttl = int64(recordConfig.TTL)
		}

		ctx := context.Background()
		op, err := provider.client.CreateDNSDomainRecord(ctx, domainID, record)
		if err != nil {
			return err

		}
		_, err = provider.client.Wait(ctx, op, egoscale.OperationStateSuccess)

		return err
	}
}

// Returns a function that can be invoked to delete a record in a zone.
func (provider *exoscaleProvider) deleteRecordFunc(recordID, domainID egoscale.UUID) func() error {
	return func() error {
		ctx := context.Background()
		op, err := provider.client.DeleteDNSDomainRecord(ctx, domainID, recordID)
		if err != nil {
			return err
		}

		_, err = provider.client.Wait(ctx, op, egoscale.OperationStateSuccess)
		return err
	}
}

// Returns a function that can be invoked to update a record in a zone.
func (provider *exoscaleProvider) updateRecordFunc(
	record *egoscale.DNSDomainRecord,
	recordConfig *models.RecordConfig,
	domainID egoscale.UUID) func() error {
	return func() error {
		target := recordConfig.GetTargetCombined()
		name := recordConfig.GetLabel()

		if recordConfig.Type == "MX" {
			target = recordConfig.GetTargetField()

			if recordConfig.MxPreference != 0 {
				record.Priority = int64(recordConfig.MxPreference)
			}
		}

		if recordConfig.Type == "SRV" {
			// API wants priority as separate argument, here we will strip it from combined target.
			sp := strings.Split(target, " ")
			target = strings.Join(sp[1:], " ")
			p, err := strconv.ParseInt(sp[0], 10, 64)
			if err != nil {
				return err
			}
			record.Priority = p
		}

		if recordConfig.Type == "NS" && (name == "@" || name == "") {
			name = "*"
		}

		record.Name = name
		record.Type = egoscale.DNSDomainRecordType(recordConfig.Type)
		record.Content = target
		if recordConfig.TTL != 0 {
			record.Ttl = int64(recordConfig.TTL)
		}

		ctx := context.Background()
		op, err := provider.client.UpdateDNSDomainRecord(ctx, domainID, record.ID, egoscale.UpdateDNSDomainRecordRequest{
			Name:     record.Name,
			Content:  record.Content,
			Priority: record.Priority,
			Ttl:      record.Ttl,
		})
		if err != nil {
			return err
		}
		_, err = provider.client.Wait(ctx, op, egoscale.OperationStateSuccess)
		return err
	}
}

func (provider *exoscaleProvider) findDomainByName(name string) (egoscale.DNSDomain, error) {
	domains, err := provider.client.ListDNSDomains(context.Background())
	if err != nil {
		return egoscale.DNSDomain{}, err
	}

	return domains.FindDNSDomain(name)
}

func defaultNSSUffix(defNS string) bool {
	return strings.HasSuffix(defNS, ".exoscale.io.") ||
		strings.HasSuffix(defNS, ".exoscale.com.") ||
		strings.HasSuffix(defNS, ".exoscale.ch.") ||
		strings.HasSuffix(defNS, ".exoscale.net.")
}

// remove all non-exoscale NS records from our desired state.
// if any are found, print a warning.
func removeOtherNS(domainConfig *models.DomainConfig) {
	recordConfigs := make([]*models.RecordConfig, 0, len(domainConfig.Records))
	for _, recordConfig := range domainConfig.Records {
		if recordConfig.Type == "NS" {
			// apex NS inside exoscale are expected.
			if recordConfig.GetLabelFQDN() == domainConfig.Name && defaultNSSUffix(recordConfig.GetTargetField()) {
				continue
			}
			printer.Printf("Warning: exoscale.com(.io, .ch, .net) does not allow NS records to be modified. %s will not be added.\n", recordConfig.GetTargetField())
			continue
		}
		recordConfigs = append(recordConfigs, recordConfig)
	}
	domainConfig.Records = recordConfigs
}
