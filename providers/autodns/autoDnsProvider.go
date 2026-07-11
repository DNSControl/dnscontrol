package autodns

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/DNSControl/dnscontrol/v4/models"
	"github.com/DNSControl/dnscontrol/v4/pkg/diff2"
	"github.com/DNSControl/dnscontrol/v4/pkg/providers"
	"github.com/DNSControl/dnscontrol/v4/providers/bind"
)

var features = providers.DocumentationNotes{
	// The default for unlisted capabilities is 'Cannot'.
	// See providers/capabilities.go for the entire list of capabilities.
	providers.CanGetZones:            providers.Can(),
	providers.CanConcur:              providers.Can(),
	providers.CanUseAlias:            providers.Can(),
	providers.CanUseCAA:              providers.Can(),
	providers.CanUseDS:               providers.Cannot(),
	providers.CanUsePTR:              providers.Can(),
	providers.CanUseSRV:              providers.Can(),
	providers.CanUseSSHFP:            providers.Cannot(),
	providers.CanUseTLSA:             providers.Cannot(),
	providers.DocCreateDomains:       providers.Cannot(),
	providers.DocDualHost:            providers.Cannot(),
	providers.DocOfficiallySupported: providers.Cannot(),
}

type autoDNSProvider struct {
	baseURL         url.URL
	defaultHeaders  http.Header
	includeChildren bool
}

func init() {
	const providerName = "AUTODNS"
	const providerMaintainer = "@arnoschoon"
	fns := providers.DspFuncs{
		Initializer: func(settings map[string]string, _ json.RawMessage) (providers.DNSServiceProvider, error) {
			return newAutoDNSProvider(settings), nil
		},
		RecordAuditor: AuditRecords,
	}
	providers.RegisterRegistrarType(providerName, func(settings map[string]string) (providers.Registrar, error) {
		return newAutoDNSProvider(settings), nil
	}, features)
	providers.RegisterDomainServiceProviderType(providerName, fns, features)
	providers.RegisterMaintainer(providerName, providerMaintainer)
	providers.RegisterCredsMetadata(providerName, providers.CredsMetadata{
		DisplayName: "AutoDNS",
		Kind:        providers.KindDNS | providers.KindRegistrar,
		DocsURL:     "https://docs.dnscontrol.org/provider/autodns",
		PortalURL:   "https://login.autodns.com/", // TODO: Verify
		Fields: []providers.CredsField{
			{
				Key:      "username",
				Label:    "Username",
				Help:     "AutoDNS / Domainrobot username.",
				Required: true,
			},
			{
				Key:      "password",
				Label:    "Password",
				Help:     "AutoDNS / Domainrobot password.",
				Secret:   true,
				Required: true,
			},
			{
				Key:      "context",
				Label:    "Context",
				Help:     "Value for the X-Domainrobot-Context header.",
				Required: true,
			},
			{
				Key:   "children",
				Label: "Include sub-user zones",
				Help:  "Set to \"true\" so get-zones also lists zones owned by sub-users (master/admin accounts). Optional; defaults to off.",
			},
		},
	})
}

func newAutoDNSProvider(settings map[string]string) *autoDNSProvider {
	api := &autoDNSProvider{}

	api.baseURL = url.URL{
		Scheme: "https",
		User: url.UserPassword(
			settings["username"],
			settings["password"],
		),
		Host: "api.autodns.com",
		Path: "/v1/",
	}

	api.defaultHeaders = http.Header{
		"Accept":                []string{"application/json; charset=UTF-8"},
		"Content-Type":          []string{"application/json; charset=UTF-8"},
		"X-Domainrobot-Context": []string{settings["context"]},
	}

	// AutoDNS hides zones owned by sub-users unless "children" is requested
	// (the same optional toggle the web UI offers). Opt-in via creds.json.
	api.includeChildren = settings["children"] == "true"

	return api
}

// GetZoneRecordsCorrections returns a list of corrections that will turn existing records into dc.Records.
func (api *autoDNSProvider) GetZoneRecordsCorrections(dc *models.DomainConfig, existingRecords models.Records) ([]*models.Correction, int, error) {
	domain := dc.Name

	var corrections []*models.Correction

	result, err := diff2.ByZone(existingRecords, dc, nil)
	if err != nil {
		return nil, 0, err
	}
	msgs, changed, actualChangeCount := result.Msgs, result.HasChanges, result.ActualChangeCount

	if changed {
		msgs = append(msgs, "Zone update for "+domain)
		msg := strings.Join(msgs, "\n")

		nameServers, zoneTTL, resourceRecords := recordsToNative(result.DesiredPlus)

		corrections = append(corrections,
			&models.Correction{
				Msg: msg,
				F: func() error {
					nameServers := nameServers
					zoneTTL := zoneTTL
					resourceRecords := resourceRecords

					err := api.updateZone(domain, resourceRecords, nameServers, zoneTTL)
					if err != nil {
						return errors.New(err.Error())
					}

					return nil
				},
			})
	}

	return corrections, actualChangeCount, nil
}

func recordsToNative(recs models.Records) ([]*models.Nameserver, uint32, []*ResourceRecord) {
	var nameServers []*models.Nameserver
	var zoneTTL uint32
	var resourceRecords []*ResourceRecord

	for _, record := range recs {
		if record.Type == "NS" && record.Name == "@" {
			// NS records for the APEX should be handled differently
			nameServers = append(nameServers, &models.Nameserver{
				Name: strings.TrimSuffix(record.GetTargetField(), "."),
			})

			zoneTTL = record.TTL
		} else {
			resourceRecord := &ResourceRecord{
				Name:  record.Name,
				TTL:   int64(record.TTL),
				Type:  record.Type,
				Value: record.GetTargetField(),
			}

			if resourceRecord.Name == "@" {
				resourceRecord.Name = ""
			}

			if record.Type == "MX" {
				resourceRecord.Pref = int32(record.MxPreference)
			}

			if record.Type == "SRV" {
				resourceRecord.Value = fmt.Sprintf("%d %d %d %s",
					record.SrvPriority,
					record.SrvWeight,
					record.SrvPort,
					record.GetTargetField(),
				)
			}

			if record.Type == "CAA" {
				resourceRecord.Value = fmt.Sprintf("%d %s \"%s\"",
					record.CaaFlag,
					record.CaaTag,
					record.GetTargetField(),
				)
			}

			resourceRecords = append(resourceRecords, resourceRecord)
		}
	}
	return nameServers, zoneTTL, resourceRecords
}

// GetNameservers returns the nameservers for a domain.
func (api *autoDNSProvider) GetNameservers(domain string) ([]*models.Nameserver, error) {
	zone, err := api.getZone(domain)
	if err != nil {
		return nil, err
	}

	return zone.NameServers, nil
}

// GetZoneRecords gets the records of a zone and returns them in RecordConfig format.
func (api *autoDNSProvider) GetZoneRecords(dc *models.DomainConfig) (models.Records, error) {
	domain := dc.Name

	zone, err := api.getZone(domain)
	if err != nil {
		return nil, err
	}

	existingRecords := make([]*models.RecordConfig, len(zone.ResourceRecords))
	for i, resourceRecord := range zone.ResourceRecords {
		var err error
		existingRecords[i], err = toRecordConfig(domain, resourceRecord)
		if err != nil {
			return nil, err
		}
		// If TTL is not set for an individual RR AutoDNS defaults to the zone TTL defined in SOA
		if existingRecords[i].TTL == 0 {
			existingRecords[i].TTL = zone.Soa.TTL
		}
	}

	// AutoDNS doesn't respond with APEX nameserver records as regular RR but rather as a zone property
	for _, nameServer := range zone.NameServers {
		nameServerRecord := &models.RecordConfig{
			TTL: zone.Soa.TTL,
		}

		nameServerRecord.SetLabel("", domain)

		// make sure the value for this NS record is suffixed with a dot at the end
		_ = nameServerRecord.PopulateFromString("NS", strings.TrimSuffix(nameServer.Name, ".")+".", domain)

		existingRecords = append(existingRecords, nameServerRecord)
	}

	if zone.MainRecord != nil && zone.MainRecord.Value != "" {
		addressRecord := &models.RecordConfig{
			TTL: uint32(zone.MainRecord.TTL),
		}

		// If TTL is not set for an individual RR AutoDNS defaults to the zone TTL defined in SOA
		if addressRecord.TTL == 0 {
			addressRecord.TTL = zone.Soa.TTL
		}

		addressRecord.SetLabel("", domain)

		_ = addressRecord.PopulateFromString("A", zone.MainRecord.Value, domain)

		existingRecords = append(existingRecords, addressRecord)

		if zone.IncludeWwwForMain {
			prefixedAddressRecord := &models.RecordConfig{
				TTL: uint32(zone.MainRecord.TTL),
			}

			// If TTL is not set for an individual RR AutoDNS defaults to the zone TTL defined in SOA
			if prefixedAddressRecord.TTL == 0 {
				prefixedAddressRecord.TTL = zone.Soa.TTL
			}

			prefixedAddressRecord.SetLabel("www", domain)

			_ = prefixedAddressRecord.PopulateFromString("A", zone.MainRecord.Value, domain)

			existingRecords = append(existingRecords, prefixedAddressRecord)
		}
	}

	return existingRecords, nil
}

func (api *autoDNSProvider) EnsureZoneExists(dc *models.DomainConfig) error {
	domain := dc.Name

	// try to get zone
	_, err := api.getZone(domain)

	if !errors.Is(err, os.ErrNotExist) {
		return err
	}

	_, err = api.createZone(domain, &Zone{
		Origin: domain,
		NameServers: []*models.Nameserver{
			{Name: "a.ns14.net"}, {Name: "b.ns14.net"},
			{Name: "c.ns14.net"}, {Name: "d.ns14.net"},
		},
		Soa: &bind.SoaDefaults{
			Expire:  1209600,
			Refresh: 43200,
			Retry:   7200,
			TTL:     86400,
		},
	})

	return err
}

func (api *autoDNSProvider) ListZones() ([]string, error) {
	return api.getZones()
}

func (api *autoDNSProvider) GetRegistrarCorrections(dc *models.DomainConfig) ([]*models.Correction, error) {
	domain, err := api.getDomain(dc.Name)
	if err != nil {
		return nil, err
	}

	existingNs := make([]string, 0, len(domain.NameServers))
	for _, ns := range domain.NameServers {
		existingNs = append(existingNs, ns.Name)
	}
	sort.Strings(existingNs)
	existing := strings.Join(existingNs, ",")

	desiredNs := models.NameserversToStrings(dc.Nameservers)
	sort.Strings(desiredNs)
	desired := strings.Join(desiredNs, ",")

	if existing != desired {
		return []*models.Correction{
			{
				Msg: fmt.Sprintf("Change Nameservers from '%s' to '%s'", existing, desired),
				F: func() error {
					nameservers := make([]*NameServer, 0, len(desiredNs))
					for _, name := range desiredNs {
						nameservers = append(nameservers, &NameServer{
							Name: name,
						})
					}
					return api.updateDomain(dc.Name, &Domain{
						NameServers: nameservers,
					})
				},
			},
		}, nil
	}

	return nil, nil
}

func toRecordConfig(domain string, record *ResourceRecord) (*models.RecordConfig, error) {
	rc := &models.RecordConfig{
		Type:     record.Type,
		TTL:      uint32(record.TTL),
		Original: record,
	}
	rc.SetLabel(record.Name, domain)

	// special record types are handled below, skip the `rc.PopulateFromString` method
	if record.Type != "MX" && record.Type != "SRV" {
		if err := rc.PopulateFromString(record.Type, record.Value, domain); err != nil {
			return nil, err
		}
	}

	if record.Type == "MX" {
		rc.MxPreference = uint16(record.Pref)
		if err := rc.SetTarget(record.Value); err != nil {
			return nil, err
		}
	}

	if record.Type == "SRV" {
		rc.SrvPriority = uint16(record.Pref)

		re := regexp.MustCompile(`(\d+) (\d+) (.+)$`)
		found := re.FindStringSubmatch(record.Value)
		if len(found) != 4 {
			return nil, fmt.Errorf("invalid SRV record value: %s", record.Value)
		}

		weight, err := strconv.Atoi(found[1])
		if err != nil {
			return nil, err
		}
		if weight < 0 {
			rc.SrvWeight = 0
		} else if weight > 65535 {
			rc.SrvWeight = 65535
		} else {
			rc.SrvWeight = uint16(weight)
		}

		port, err := strconv.Atoi(found[2])
		if err != nil {
			return nil, err
		}
		if port < 0 {
			rc.SrvPort = 0
		} else if port > 65535 {
			rc.SrvPort = 65535
		} else {
			rc.SrvPort = uint16(port)
		}

		if err := rc.SetTarget(found[3]); err != nil {
			return nil, err
		}
	}

	return rc, nil
}
