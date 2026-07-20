package hostingde

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	"github.com/DNSControl/dnscontrol/v4/models"
	"github.com/DNSControl/dnscontrol/v4/pkg/diff"
	"github.com/DNSControl/dnscontrol/v4/pkg/providers"
)

var defaultNameservers = []string{"ns1.hosting.de", "ns2.hosting.de", "ns3.hosting.de"}

var features = providers.DocumentationNotes{
	// The default for unlisted capabilities is 'Cannot'.
	// See providers/capabilities.go for the entire list of capabilities.
	providers.CanAutoDNSSEC:          providers.Can(),
	providers.CanConcur:              providers.Unimplemented(),
	providers.CanGetZones:            providers.Can(),
	providers.CanOnlyDiff1Features:   providers.Can(),
	providers.CanUseAlias:            providers.Can(),
	providers.CanUseCAA:              providers.Can(),
	providers.CanUseDS:               providers.Can(),
	providers.CanUseLOC:              providers.Cannot(),
	providers.CanUseNAPTR:            providers.Cannot(),
	providers.CanUsePTR:              providers.Can(),
	providers.CanUseSOA:              providers.Can(),
	providers.CanUseSRV:              providers.Can(),
	providers.CanUseSSHFP:            providers.Can(),
	providers.CanUseTLSA:             providers.Can(),
	providers.DocCreateDomains:       providers.Can(),
	providers.DocDualHost:            providers.Can(),
	providers.DocOfficiallySupported: providers.Cannot(),
}

func init() {
	const providerName = "HOSTINGDE"
	const providerMaintainer = "@juliusrickert"
	providers.RegisterRegistrarType(providerName, newHostingdeReg)
	fns := providers.DspFuncs{
		Initializer:   newHostingdeDsp,
		RecordAuditor: AuditRecords,
	}
	providers.RegisterDomainServiceProviderType(providerName, fns, features)
	providers.RegisterMaintainer(providerName, providerMaintainer)
	providers.RegisterCredsMetadata(providerName, providers.CredsMetadata{
		DisplayName: "hosting.de",
		Kind:        providers.KindDNS | providers.KindRegistrar,
		DocsURL:     "https://docs.dnscontrol.org/provider/hostingde",
		PortalURL:   "https://secure.hosting.de/",
		Fields: []providers.CredsField{
			{
				Key:      "authToken",
				Label:    "Auth token",
				Help:     "Your hosting.de API auth token.",
				Secret:   true,
				Required: true,
			},
		},
	})
}

type providerMeta struct {
	DefaultNS []string `json:"default_ns"`
}

func newHostingde(m map[string]string, providermeta json.RawMessage) (*hostingdeProvider, error) {
	authToken, ownerAccountID, filterAccountID, baseURL := m["authToken"], m["ownerAccountId"], m["filterAccountId"], m["baseURL"]

	if authToken == "" {
		return nil, errors.New("hosting.de: authtoken must be provided")
	}

	if baseURL == "" {
		baseURL = "https://secure.hosting.de"
	}
	baseURL = strings.TrimSuffix(baseURL, "/")

	hp := &hostingdeProvider{
		authToken:       authToken,
		ownerAccountID:  ownerAccountID,
		filterAccountID: filterAccountID,
		baseURL:         baseURL,
		nameservers:     defaultNameservers,
	}

	if len(providermeta) > 0 {
		var pm providerMeta
		if err := json.Unmarshal(providermeta, &pm); err != nil {
			return nil, fmt.Errorf("hosting.de: could not parse providermeta: %w", err)
		}

		if len(pm.DefaultNS) > 0 {
			hp.nameservers = pm.DefaultNS
		}
	}

	return hp, nil
}

func newHostingdeDsp(m map[string]string, providermeta json.RawMessage) (providers.DNSServiceProvider, error) {
	return newHostingde(m, providermeta)
}

func newHostingdeReg(m map[string]string) (providers.Registrar, error) {
	return newHostingde(m, json.RawMessage{})
}

func (hp *hostingdeProvider) GetNameservers(domain string) ([]*models.Nameserver, error) {
	return models.ToNameservers(hp.nameservers)
}

func (hp *hostingdeProvider) GetZoneRecords(dc *models.DomainConfig) (models.Records, error) {
	domain := dc.Name

	zone, err := hp.getZone(domain)
	if err != nil {
		return nil, err
	}
	return hp.APIRecordsToStandardRecordsModel(domain, zone.Records)
}

func (hp *hostingdeProvider) APIRecordsToStandardRecordsModel(domain string, src []record) (models.Records, error) {
	records := []*models.RecordConfig{}
	for _, r := range src {
		if r.Type == "SOA" {
			continue
		}
		newr, err := r.nativeToRecord(domain)
		if err != nil {
			return nil, err
		}
		records = append(records, newr)
	}

	return records, nil
}

func soaToString(s soaValues) string {
	return fmt.Sprintf("refresh=%d retry=%d expire=%d negativettl=%d ttl=%d", s.Refresh, s.Retry, s.Expire, s.NegativeTTL, s.TTL)
}

// GetZoneRecordsCorrections returns a list of corrections that will turn existing records into dc.Records.
func (hp *hostingdeProvider) GetZoneRecordsCorrections(dc *models.DomainConfig, records models.Records) ([]*models.Correction, int, error) {
	var err error

	// TTL must be between (inclusive) 1m and 1y (in fact, a little bit more)
	for _, r := range dc.Records {
		if r.TTL < 60 {
			r.TTL = 60
		}
		if r.TTL > 31556926 {
			r.TTL = 31556926
		}
	}

	zoneChanged := false

	zone, err := hp.getZone(dc.Name)
	if err != nil {
		return nil, 0, err
	}

	toReport, create, del, mod, actualChangeCount, err := diff.NewCompat(dc).IncrementalDiff(records)
	if err != nil {
		return nil, 0, err
	}
	// Start corrections with the reports
	corrections := diff.GenerateMessageCorrections(toReport)

	// NOPURGE
	if dc.KeepUnknown {
		del = nil
	}

	// remove SOA record from corrections as it is handled separately
	for i, r := range create {
		if r.Desired.Type == "SOA" {
			create = append(create[:i], create[i+1:]...)
			break
		}
	}

	if len(create) != 0 || len(del) != 0 || len(mod) != 0 {
		zoneChanged = true
	}

	msg := []string{}
	for _, c := range append(del, append(create, mod...)...) {
		msg = append(msg, c.String())
	}

	var desiredSoa *models.RecordConfig
	for _, r := range dc.Records {
		if r.Type == "SOA" && r.Name == "@" {
			desiredSoa = r
			break
		}
	}
	if desiredSoa == nil {
		desiredSoa = &models.RecordConfig{}
	}

	defaultSoa := &hp.defaultSoa
	// Commented out because this can not happen:
	// if defaultSoa == nil {
	// 	defaultSoa = &soaValues{}
	// }

	newSOA := soaValues{
		Refresh:     firstNonZero(desiredSoa.SoaRefresh, defaultSoa.Refresh, 86400),
		Retry:       firstNonZero(desiredSoa.SoaRetry, defaultSoa.Retry, 7200),
		Expire:      firstNonZero(desiredSoa.SoaExpire, defaultSoa.Expire, 3600000),
		NegativeTTL: firstNonZero(desiredSoa.SoaMinttl, defaultSoa.NegativeTTL, 900),
		TTL:         firstNonZero(desiredSoa.TTL, defaultSoa.TTL, 86400),
	}

	if zone.ZoneConfig.SOAValues != newSOA {
		msg = append(msg, fmt.Sprintf("Updating SOARecord from (%s) to (%s)", soaToString(zone.ZoneConfig.SOAValues), soaToString(newSOA)))
		zone.ZoneConfig.SOAValues = newSOA
		zoneChanged = true
	}

	if desiredSoa.SoaMbox != "" {
		desiredMail := ""
		if desiredSoa.SoaMbox[len(desiredSoa.SoaMbox)-1] != '.' {
			desiredMail = desiredSoa.SoaMbox + "@" + dc.Name
		}
		if desiredMail != "" && zone.ZoneConfig.EmailAddress != desiredMail {
			msg = append(msg, fmt.Sprintf("Changing SOA Mail from %s to %s", zone.ZoneConfig.EmailAddress, desiredMail))
			zone.ZoneConfig.EmailAddress = desiredMail
			zoneChanged = true
		}
	}

	existingAutoDNSSecEnabled := zone.ZoneConfig.DNSSECMode == "automatic"
	desiredAutoDNSSecEnabled := dc.AutoDNSSEC == "on"

	var DNSSecOptions *dnsSecOptions
	var removeDNSSecEntries []dnsSecEntry

	// ensure that publishKsk is set for domains with AutoDNSSec
	if existingAutoDNSSecEnabled && desiredAutoDNSSecEnabled {
		currentDNSSecOptions, err := hp.getDNSSECOptions(zone.ZoneConfig.ID)
		if err != nil {
			return nil, 0, err
		}
		if !currentDNSSecOptions.PublishKSK {
			msg = append(msg, "Enabling publishKsk for AutoDNSSec")
			DNSSecOptions = currentDNSSecOptions
			DNSSecOptions.PublishKSK = true
			zoneChanged = true
		}
	}

	if !existingAutoDNSSecEnabled && desiredAutoDNSSecEnabled {
		msg = append(msg, "Enable AutoDNSSEC")
		DNSSecOptions = &dnsSecOptions{
			NSECMode:   "nsec3",
			PublishKSK: true,
		}
		zone.ZoneConfig.DNSSECMode = "automatic"
		zoneChanged = true
	} else if existingAutoDNSSecEnabled && !desiredAutoDNSSecEnabled {
		currentDNSSecOptions, err := hp.getDNSSECOptions(zone.ZoneConfig.ID)
		if err != nil {
			return nil, 0, err
		}
		msg = append(msg, "Disable AutoDNSSEC")
		zone.ZoneConfig.DNSSECMode = "off"

		// Remove auto dnssec keys from domain
		DomainConfig, err := hp.getDomainConfig(dc.Name)
		if err != nil {
			return nil, 0, err
		}
		for _, entry := range DomainConfig.DNSSecEntries {
			for _, autoDNSKey := range currentDNSSecOptions.Keys {
				if entry.KeyData.PublicKey == autoDNSKey.KeyData.PublicKey {
					removeDNSSecEntries = append(removeDNSSecEntries, entry)
				}
			}
		}
		zoneChanged = true
	}

	if !zoneChanged {
		return nil, 0, nil
	}

	corrections = append(corrections, &models.Correction{
		Msg: "\n" + strings.Join(msg, "\n"),
		F: func() error {
			for i := range 10 {
				err := hp.updateZone(&zone.ZoneConfig, DNSSecOptions, create, del, mod)
				if err == nil {
					return nil
				}
				// Code:10205 indicates the zone is currently blocked due to a running zone update.
				if !strings.Contains(err.Error(), "Code:10205") {
					return err
				}

				// Exponential back-off retry.
				// Base of 1.8 seemed like a good trade-off, retrying for approximately 45 seconds.
				time.Sleep(time.Duration(math.Pow(1.8, float64(i))) * 100 * time.Millisecond)
			}
			return errors.New("retry exhaustion: zone blocked for 10 attempts")
		},
	},
	)

	if removeDNSSecEntries != nil {
		correction := &models.Correction{
			Msg: "Removing AutoDNSSEC Keys from Domain",
			F: func() error {
				err := hp.dnsSecKeyModify(dc.Name, nil, removeDNSSecEntries)
				if err != nil {
					return err
				}
				return nil
			},
		}
		corrections = append(corrections, correction)
	}

	return corrections, actualChangeCount, nil
}

func firstNonZero(items ...uint32) uint32 {
	for _, item := range items {
		if item != 0 {
			return item
		}
	}
	return 999
}

func (hp *hostingdeProvider) GetRegistrarCorrections(dc *models.DomainConfig) ([]*models.Correction, error) {
	found, err := hp.getNameservers(dc.Name)
	if err != nil {
		return nil, fmt.Errorf("error getting nameservers: %w", err)
	}
	sort.Strings(found)
	foundNameservers := strings.Join(found, ",")

	expected := []string{}
	for _, ns := range dc.Nameservers {
		expected = append(expected, ns.Name)
	}
	sort.Strings(expected)
	expectedNameservers := strings.Join(expected, ",")

	// We don't care about glued records because we disallowed them
	if foundNameservers != expectedNameservers {
		return []*models.Correction{
			{
				Msg: fmt.Sprintf("Update nameservers %s -> %s", foundNameservers, expectedNameservers),
				F:   hp.updateNameservers(expected, dc.Name),
			},
		}, nil
	}

	return nil, nil
}

func (hp *hostingdeProvider) EnsureZoneExists(dc *models.DomainConfig) error {
	domain := dc.Name
	_, err := hp.getZoneConfig(domain)
	if errors.Is(err, errZoneNotFound) {
		if err := hp.createZone(domain); err != nil {
			return err
		}
	}
	return nil
}

func (hp *hostingdeProvider) ListZones() ([]string, error) {
	zcs, err := hp.getAllZoneConfigs()
	if err != nil {
		return nil, err
	}
	zones := make([]string, 0, len(zcs))
	for _, zoneConfig := range zcs {
		zones = append(zones, zoneConfig.Name)
	}
	return zones, nil
}
