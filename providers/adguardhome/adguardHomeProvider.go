package adguardhome

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/netip"

	dnsv2 "codeberg.org/miekg/dns"
	"github.com/DNSControl/dnscontrol/v5/models"
	"github.com/DNSControl/dnscontrol/v5/pkg/diff2"
	"github.com/DNSControl/dnscontrol/v5/pkg/printer"
	"github.com/DNSControl/dnscontrol/v5/pkg/privatetypes"
	"github.com/DNSControl/dnscontrol/v5/pkg/providers"
)

func newDsp(conf map[string]string, metadata json.RawMessage) (providers.DNSServiceProvider, error) {
	return newAdguardHome(conf, metadata)
}

// newAdguardHome creates the provider.
func newAdguardHome(m map[string]string, _ json.RawMessage) (*adguardHomeProvider, error) {
	c := &adguardHomeProvider{}

	c.username, c.password, c.host = m["username"], m["password"], m["host"]

	if c.username == "" {
		return nil, errors.New("missing adguard home username")
	}
	if c.password == "" {
		return nil, errors.New("missing adguard home password")
	}
	if c.host == "" {
		return nil, errors.New("missing adguard home endpoint")
	}

	return c, nil
}

var features = providers.DocumentationNotes{
	providers.CanConcur:              providers.Unimplemented(),
	providers.CanUseAlias:            providers.Can(),
	providers.CanGetZones:            providers.Cannot(),
	providers.DocOfficiallySupported: providers.Cannot(),
}

func init() {
	const providerName = "ADGUARDHOME"
	const providerMaintainer = "@ishanjain28"
	fns := providers.DspFuncs{
		Initializer:   newDsp,
		RecordAuditor: AuditRecords,
	}
	providers.RegisterCustomRecordType("ADGUARDHOME_A_PASSTHROUGH", providerName, "")
	providers.RegisterCustomRecordType("ADGUARDHOME_AAAA_PASSTHROUGH", providerName, "")
	providers.RegisterDomainServiceProviderType(providerName, fns, features)
	providers.RegisterMaintainer(providerName, providerMaintainer)
}

// GetNameservers returns the nameservers for a domain.
func (c *adguardHomeProvider) GetNameservers(domain string) ([]*models.Nameserver, error) {
	return []*models.Nameserver{}, nil
}

// GetZoneRecordsCorrections returns a list of corrections that will turn existing records into dc.Records.
func (c *adguardHomeProvider) GetZoneRecordsCorrections(dc *models.DomainConfig, existingRecords models.Records) ([]*models.Correction, int, error) {
	// TTLs don't matter in ADGUARDHOME and
	// we use the default value of 300
	for _, record := range dc.Records {
		record.TTL = 300
	}

	var corrections []*models.Correction

	changes, actualChangeCount, err := diff2.ByRecord(existingRecords, dc,
		func(rec *models.RecordConfig) string { return "" },
	)
	if err != nil {
		return nil, 0, err
	}
	for _, change := range changes {
		var corr *models.Correction
		switch change.Type {
		case diff2.REPORT:
			printer.Warnf("diff2 report message\n")
			corr = &models.Correction{Msg: change.MsgsJoined}
		case diff2.CREATE:
			re, err := toRewriteEntry(dc, change.New[0])
			if err != nil {
				return nil, 0, err
			}
			corr = &models.Correction{
				Msg: change.Msgs[0],
				F: func() error {
					return c.createRecord(re)
				},
			}

		case diff2.CHANGE:
			oldRe, err := toRewriteEntry(dc, change.Old[0])
			if err != nil {
				return nil, 0, err
			}
			newRe, err := toRewriteEntry(dc, change.New[0])
			if err != nil {
				return nil, 0, err
			}
			corr = &models.Correction{
				Msg: change.Msgs[0],
				F: func() error {
					return c.modifyRecord(oldRe, newRe)
				},
			}

		case diff2.DELETE:
			re, err := toRewriteEntry(dc, change.Old[0])
			if err != nil {
				return nil, 0, err
			}

			corr = &models.Correction{
				Msg: change.Msgs[0],
				F: func() error {
					return c.deleteRecord(re)
				},
			}
		default:
			panic(fmt.Sprintf("unhandled change.Type %s", change.Type))
		}

		corrections = append(corrections, corr)
	}

	return corrections, actualChangeCount, nil
}

// GetZoneRecords gets the records of a zone and returns them in RecordConfig format.
func (c *adguardHomeProvider) GetZoneRecords(dc *models.DomainConfig) (models.Records, error) {
	domain := dc.Name

	records, err := c.getRecords(domain)
	if err != nil {
		return nil, err
	}

	existingRecords := make(models.Records, 0, len(records))
	for _, r := range records {
		newRec, err := toRc(dc, r)
		if err != nil {
			return nil, err
		}
		existingRecords = append(existingRecords, newRec)
	}

	return existingRecords, nil
}

func toRewriteEntry(dc *models.DomainConfig, rc *models.RecordConfig) (rewriteEntry, error) {
	re := rewriteEntry{
		Domain: rc.NameFQDN,
	}
	switch rc.TypeNum {
	case dnsv2.TypeA:
		re.Answer = rc.AsA().Addr.String()
	case dnsv2.TypeAAAA:
		re.Answer = rc.AsAAAA().Addr.String()

	case privatetypes.TypeALIAS:
		re.Answer = dc.ToShort(rc.AsALIAS().Target)

	case dnsv2.TypeCNAME:
		re.Answer = dc.ToShort(rc.AsCNAME().Target)

	case privatetypes.TypeADGUARDHOMEAPASSTHROUGH:
		re.Answer = "A"

	case privatetypes.TypeADGUARDHOMEAAAAPASSTHROUGH:
		re.Answer = "AAAA"

	default:
		return re, fmt.Errorf("rtype %s is not supported", rc.Type)
	}

	return re, nil
}

func toRc(dc *models.DomainConfig, r rewriteEntry) (*models.RecordConfig, error) {
	label := dc.LabelFromFQDNNoDot(r.Domain)
	var rc *models.RecordConfig
	var err error

	if addr, parseErr := netip.ParseAddr(r.Answer); parseErr == nil {
		rtype := dnsv2.TypeAAAA
		if addr.Is4() {
			rtype = dnsv2.TypeA
		}
		rc, err = dc.NewRecordConfig(label, 300, rtype, addr)
	} else {
		switch r.Answer {
		case "A":
			rc, err = dc.NewRecordConfig(label, 300, privatetypes.TypeADGUARDHOMEAPASSTHROUGH, "")
		case "AAAA":
			rc, err = dc.NewRecordConfig(label, 300, privatetypes.TypeADGUARDHOMEAAAAPASSTHROUGH, "")
		default:
			answer := dc.ToShort(r.Answer)
			if r.Domain == dc.Name {
				rc, err = dc.NewRecordConfig(label, 300, privatetypes.TypeALIAS, answer)
			} else {
				rc, err = dc.NewRecordConfig(label, 300, dnsv2.TypeCNAME, answer)
			}
		}
	}
	if err != nil {
		return nil, err
	}

	rc.Original = r
	return rc, nil
}
