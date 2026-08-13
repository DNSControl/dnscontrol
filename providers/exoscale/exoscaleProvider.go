package exoscale

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"

	dnsv2 "codeberg.org/miekg/dns"
	dnsrdatav2 "codeberg.org/miekg/dns/rdata"
	"github.com/DNSControl/dnscontrol/v5/models"
	"github.com/DNSControl/dnscontrol/v5/pkg/diff2"
	"github.com/DNSControl/dnscontrol/v5/pkg/printer"
	"github.com/DNSControl/dnscontrol/v5/pkg/providers"
	egoscale "github.com/exoscale/egoscale/v3"
	"github.com/exoscale/egoscale/v3/credentials"
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

	existingRecords := make(models.Records, 0, len(records.DNSDomainRecords))
	for i := range records.DNSDomainRecords {
		recordConfig, err := nativeToRecord(&records.DNSDomainRecords[i], domainConfig)
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
func nativeToRecord(record *egoscale.DNSDomainRecord, dc *models.DomainConfig) (*models.RecordConfig, error) {
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

	label := dc.LabelFromShort(record.Name)
	ttl := uint32(record.Ttl)

	var recordConfig *models.RecordConfig
	var err error
	switch record.Type {
	case "ALIAS", "URL":
		recordConfig, err = dc.NewRecordConfig(label, ttl, string(record.Type), recordContent)
	case "MX":
		recordConfig, err = dc.NewRecordConfig(label, ttl, string(record.Type), record.Priority, recordContent)
	case "TXT":
		recordConfig, err = dc.NewRecordConfig(label, ttl, string(record.Type), recordContent)
	default:
		recordConfig, err = dc.NewRecordConfigParse(label, ttl, string(record.Type), recordContent)
	}
	if err != nil {
		return nil, fmt.Errorf("unparsable record received from exoscale: %w", err)
	}
	recordConfig.Original = record

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

	changes, actualChangeCount, err := diff2.ByRecord(existingRecords, domainConfig, nil)
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
				Msg: change.Msgs[0],
				F:   provider.createRecordFunc(change.New[0], domain.ID),
			})

		case diff2.DELETE:
			record := change.Old[0].Original.(*egoscale.DNSDomainRecord)
			corrections = append(corrections, &models.Correction{
				Msg: change.Msgs[0],
				F:   provider.deleteRecordFunc(record.ID, domain.ID),
			})

		case diff2.CHANGE:
			oldc := change.Old[0].Original.(*egoscale.DNSDomainRecord)
			newc := change.New[0]
			corrections = append(corrections, &models.Correction{
				Msg: change.Msgs[0],
				F:   provider.updateRecordFunc(oldc, newc, domain.ID),
			})

		default:
			panic(fmt.Sprintf("unhandled change.Type %s", change.Type))
		}
	}

	return corrections, actualChangeCount, nil
}

// Returns a function that can be invoked to create a record in a zone.
func (provider *exoscaleProvider) createRecordFunc(
	recordConfig *models.RecordConfig,
	domainID egoscale.UUID) func() error {
	return func() error {
		name := recordConfig.GetLabel()
		var prio int64

		var target string
		switch recordConfig.TypeNum {
		case dnsv2.TypeMX:
			f := recordConfig.AsMX()
			target = f.Mx
			prio = int64(f.Preference)
		case dnsv2.TypeSRV:
			f := recordConfig.AsSRV()
			prio = int64(f.Priority)
			target = fmt.Sprintf("%d %d %s", f.Weight, f.Port, f.Target)
		default:
			target = recordConfig.GetRDATA().String()
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

		record.Ttl = int64(recordConfig.TTL)

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
	rc *models.RecordConfig,
	domainID egoscale.UUID) func() error {
	return func() error {
		name := rc.GetLabel()

		var target string
		switch rc.TypeNum {
		case dnsv2.TypeMX:
			mx := rc.GetRDATA().(dnsrdatav2.MX)
			target = mx.Mx
			record.Priority = int64(mx.Preference)
		case dnsv2.TypeSRV:
			// API wants priority as separate argument. The target contains the weight, port, target.
			srv := rc.GetRDATA().(dnsrdatav2.SRV)
			target = fmt.Sprintf("%d %d %s", srv.Weight, srv.Port, srv.Target)
			record.Priority = int64(srv.Priority)
		default:
			target = rc.GetRDATA().String()
		}

		if rc.Type == "NS" && (name == "@" || name == "") {
			name = "*"
		}

		record.Name = name
		record.Type = egoscale.DNSDomainRecordType(rc.Type)
		record.Content = target
		record.Ttl = int64(rc.TTL)

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
	recordConfigs := make(models.Records, 0, len(domainConfig.Records))
	for _, recordConfig := range domainConfig.Records {
		if recordConfig.Type == "NS" {
			// apex NS inside exoscale are expected.
			if recordConfig.GetLabelFQDN() == domainConfig.Name && defaultNSSUffix(recordConfig.AsNS().Ns) {
				continue
			}
			printer.Printf("Warning: exoscale.com(.io, .ch, .net) does not allow NS records to be modified. %s will not be added.\n", recordConfig.AsNS().Ns)
			continue
		}
		recordConfigs = append(recordConfigs, recordConfig)
	}
	domainConfig.Records = recordConfigs
}
