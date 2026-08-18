package softlayer

import (
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"

	dnsv2 "codeberg.org/miekg/dns"
	"github.com/DNSControl/dnscontrol/v5/models"
	"github.com/DNSControl/dnscontrol/v5/pkg/diff2"
	"github.com/DNSControl/dnscontrol/v5/pkg/printer"
	"github.com/DNSControl/dnscontrol/v5/pkg/providers"
	"github.com/softlayer/softlayer-go/datatypes"
	"github.com/softlayer/softlayer-go/filter"
	"github.com/softlayer/softlayer-go/services"
	"github.com/softlayer/softlayer-go/session"
)

// softlayerProvider is the protocol handle for this provider.
type softlayerProvider struct {
	Session *session.Session
}

var features = providers.DocumentationNotes{
	// The default for unlisted capabilities is 'Cannot'.
	// See providers/capabilities.go for the entire list of capabilities.
	providers.CanConcur:            providers.Unimplemented(),
	providers.CanGetZones:          providers.Unimplemented(),
	providers.CanOnlyDiff1Features: providers.Can(),
	providers.CanUseLOC:            providers.Cannot(),
	providers.CanUseSRV:            providers.Can(),
}

func init() {
	const providerName = "SOFTLAYER"
	const providerMaintainer = "NEEDS VOLUNTEER"
	fns := providers.DspFuncs{
		Initializer:   newReg,
		RecordAuditor: AuditRecords,
	}
	providers.RegisterDomainServiceProviderType(providerName, fns, features)
	providers.RegisterMaintainer(providerName, providerMaintainer)
}

func newReg(conf map[string]string, _ json.RawMessage) (providers.DNSServiceProvider, error) {
	printer.Warnf("The SOFTLAYER provider is unmaintained: https://github.com/DNSControl/dnscontrol/issues/1079")
	s := session.New(conf["username"], conf["api_key"], conf["endpoint_url"], conf["timeout"])

	if len(s.UserName) == 0 || len(s.APIKey) == 0 {
		return nil, errors.New("SoftLayer UserName and APIKey must be provided")
	}

	// s.Debug = true

	api := &softlayerProvider{
		Session: s,
	}

	return api, nil
}

// GetNameservers returns the nameservers for a domain.
func (s *softlayerProvider) GetNameservers(domain string) ([]*models.Nameserver, error) {
	// Always use the same nameservers for softlayer
	return models.ToNameservers([]string{"ns1.softlayer.com", "ns2.softlayer.com"})
}

// GetZoneRecords gets all the records for domainName and converts
// them to model.RecordConfig.
func (s *softlayerProvider) GetZoneRecords(dc *models.DomainConfig) (models.Records, error) {
	domainName := dc.Name

	domain, err := s.getDomain(&domainName)
	if err != nil {
		return nil, err
	}

	actual, err := s.getExistingRecords(dc, domain.ResourceRecords)
	if err != nil {
		return nil, err
	}

	return actual, nil
}

// GetZoneRecordsCorrections returns a list of corrections that will turn existing records into dc.Records.
func (s *softlayerProvider) GetZoneRecordsCorrections(dc *models.DomainConfig, actual models.Records) ([]*models.Correction, int, error) {
	domain, err := s.getDomain(&dc.Name)
	if err != nil {
		return nil, 0, err
	}

	changes, actualChangeCount, err := diff2.ByRecord(actual, dc, nil)
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
				F:   s.createRecordFunc(change.New[0], domain),
			})

		case diff2.DELETE:
			existing := change.Old[0].Original.(datatypes.Dns_Domain_ResourceRecord)
			corrections = append(corrections, &models.Correction{
				Msg: change.Msgs[0],
				F:   s.deleteRecordFunc(*existing.Id),
			})

		case diff2.CHANGE:
			existing := change.Old[0].Original.(datatypes.Dns_Domain_ResourceRecord)
			corrections = append(corrections, &models.Correction{
				Msg: change.Msgs[0],
				F:   s.updateRecordFunc(&existing, change.New[0]),
			})

		default:
			panic(fmt.Sprintf("unhandled change.Type %s", change.Type))
		}
	}

	return corrections, actualChangeCount, nil
}

func (s *softlayerProvider) getDomain(name *string) (*datatypes.Dns_Domain, error) {
	// FIXME(tlim) Memoize this

	domains, err := services.GetAccountService(s.Session).
		Filter(filter.Path("domains.name").Eq(name).Build()).
		Mask("resourceRecords").
		GetDomains()
	if err != nil {
		return nil, err
	}

	if len(domains) == 0 {
		return nil, fmt.Errorf("didn't find a domain matching %s", *name)
	} else if len(domains) > 1 {
		return nil, fmt.Errorf("found %d domains matching %s", len(domains), *name)
	}

	return &domains[0], nil
}

func (s *softlayerProvider) getExistingRecords(dc *models.DomainConfig, resourceRecords []datatypes.Dns_Domain_ResourceRecord) (models.Records, error) {
	actual := models.Records{}

	for _, record := range resourceRecords {
		recType := strings.ToUpper(*record.Type)

		if recType == "SOA" {
			continue
		}

		ttl := uint32(*record.Ttl)
		label := dc.LabelFromShort(*record.Host)
		var recConfig *models.RecordConfig
		var err error
		switch recType {
		case "SRV":
			service, protocol := "", "_tcp"
			weight, port, priority := 0, 0, 0
			if record.Weight != nil {
				weight = *record.Weight
			}
			if record.Port != nil {
				port = *record.Port
			}
			if record.Priority != nil {
				priority = *record.Priority
			}
			if record.Protocol != nil {
				protocol = *record.Protocol
			}
			if record.Service != nil {
				service = *record.Service
			}
			label = dc.LabelFromShort(fmt.Sprintf("%s.%s", service, strings.ToLower(protocol)))
			recConfig, err = dc.NewRecordConfig(label, ttl, dnsv2.TypeSRV, priority, weight, port, *record.Data)
		case "TXT":
			recConfig, err = dc.NewRecordConfig(label, ttl, dnsv2.TypeTXT, *record.Data)
		case "MX":
			preference := 0
			if record.MxPriority != nil {
				preference = *record.MxPriority
			}
			recConfig, err = dc.NewRecordConfig(label, ttl, dnsv2.TypeMX, preference, *record.Data)
		default:
			recConfig, err = dc.NewRecordConfig(label, ttl, recType, *record.Data)
		}
		if err != nil {
			return nil, err
		}
		recConfig.Original = record

		actual = append(actual, recConfig)
	}

	return actual, nil
}

func (s *softlayerProvider) createRecordFunc(desired *models.RecordConfig, domain *datatypes.Dns_Domain) func() error {

	ttl := verifyMinTTL(int(desired.TTL))
	domainID := *domain.Id

	host := desired.GetLabel()
	newType := desired.Type

	var err error

	srvRegexp := regexp.MustCompile(`^_(?P<Service>\w+)\.\_(?P<Protocol>\w+)$`)

	return func() error {
		newRecord := datatypes.Dns_Domain_ResourceRecord{
			DomainId: &domainID,
			Ttl:      &ttl,
			Type:     &newType,
			Host:     &host,
		}

		switch newType {
		case "MX":
			service := services.GetDnsDomainResourceRecordMxTypeService(s.Session)

			f := desired.AsMX()
			newRecord.MxPriority = new(int(f.Preference))
			newRecord.Data = new(f.Mx)

			newMx := datatypes.Dns_Domain_ResourceRecord_MxType{
				Dns_Domain_ResourceRecord: newRecord,
			}
			_, err = service.CreateObject(&newMx)

		case "SRV":
			service := services.GetDnsDomainResourceRecordSrvTypeService(s.Session)

			result := srvRegexp.FindStringSubmatch(host)
			if len(result) != 3 {
				return fmt.Errorf("SRV Record must match format \"_service._protocol\" not %s", host)
			}
			serviceName, protocol := result[1], strings.ToLower(result[2])

			f := desired.AsSRV()
			newRecord.Data = new(f.Target)
			newSrv := datatypes.Dns_Domain_ResourceRecord_SrvType{
				Dns_Domain_ResourceRecord: newRecord,
				Service:                   &serviceName,
				Port:                      new(int(f.Port)),
				Priority:                  new(int(f.Priority)),
				Protocol:                  &protocol,
				Weight:                    new(int(f.Weight)),
			}
			_, err = service.CreateObject(&newSrv)

		default:
			newRecord.Data = new(desired.GetRDATA().String())
			service := services.GetDnsDomainResourceRecordService(s.Session)
			_, err = service.CreateObject(&newRecord)
		}

		return err
	}
}

func (s *softlayerProvider) deleteRecordFunc(resID int) func() error {
	// seems to be no problem deleting MX and SRV records via common interface
	return func() error {
		_, err := services.GetDnsDomainResourceRecordService(s.Session).
			Id(resID).
			DeleteObject()

		return err
	}
}

func (s *softlayerProvider) updateRecordFunc(existing *datatypes.Dns_Domain_ResourceRecord, desired *models.RecordConfig) func() error {
	ttl := verifyMinTTL(int(desired.TTL))

	return func() error {
		changes := false
		var err error

		switch desired.TypeNum {
		case dnsv2.TypeMX:
			df := desired.AsMX()
			preference := int(df.Preference)
			service := services.GetDnsDomainResourceRecordMxTypeService(s.Session)
			updated := datatypes.Dns_Domain_ResourceRecord_MxType{}

			label := desired.GetLabel()
			if label != *existing.Host {
				updated.Host = &label
				changes = true
			}

			target := desired.AsMX().Mx
			if target != *existing.Data {
				updated.Data = &target
				changes = true
			}

			if ttl != *existing.Ttl {
				updated.Ttl = &ttl
				changes = true
			}

			if preference != *existing.MxPriority {
				updated.MxPriority = &preference
				changes = true
			}

			if !changes {
				return errors.New("didn't find changes when I expect some")
			}

			_, err = service.Id(*existing.Id).EditObject(&updated)

		case dnsv2.TypeSRV:
			df := desired.AsSRV()
			priority, weight, port := int(df.Priority), int(df.Weight), int(df.Port)
			service := services.GetDnsDomainResourceRecordSrvTypeService(s.Session)
			updated := datatypes.Dns_Domain_ResourceRecord_SrvType{}

			label := desired.GetLabel()
			if label != *existing.Host {
				updated.Host = &label
				changes = true
			}

			target := desired.AsSRV().Target
			if target != *existing.Data {
				updated.Data = &target
				changes = true
			}

			if ttl != *existing.Ttl {
				updated.Ttl = &ttl
				changes = true
			}

			if priority != *existing.Priority {
				updated.Priority = &priority
				changes = true
			}

			if weight != *existing.Weight {
				updated.Weight = &weight
				changes = true
			}

			if port != *existing.Port {
				updated.Port = &port
				changes = true
			}

			// TODO: handle service & protocol - or does that just result in a
			// delete and recreate?

			if !changes {
				return errors.New("didn't find changes when I expect some")
			}

			_, err = service.Id(*existing.Id).EditObject(&updated)

		default:
			service := services.GetDnsDomainResourceRecordService(s.Session)
			updated := datatypes.Dns_Domain_ResourceRecord{}

			label := desired.GetLabel()
			if label != *existing.Host {
				updated.Host = &label
				changes = true
			}

			target := desired.GetRDATA().String()
			if target != *existing.Data {
				updated.Data = &target
				changes = true
			}

			if ttl != *existing.Ttl {
				updated.Ttl = &ttl
				changes = true
			}

			if !changes {
				return errors.New("didn't find changes when I expect some")
			}

			_, err = service.Id(*existing.Id).EditObject(&updated)
		}

		return err
	}
}

func verifyMinTTL(ttl int) int {
	const minTTL = 60
	if ttl < minTTL {
		printer.Printf("\nMODIFY TTL to Min supported TTL value: (ttl=%d) -> (ttl=%d)\n", ttl, minTTL)
		return minTTL
	}
	return ttl
}
