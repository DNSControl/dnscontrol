package loopia

/*

Loopia XML_RPC API V1 DNS provider:

Documentation: https://www.loopia.com/api/
Endpoint: https://api.loopia.se/RPCSERV

Settings from `creds.json`:
   - username
   - password
   - debug
   - rate_limit_per

*/

import (
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/DNSControl/dnscontrol/v4/models"
	"github.com/DNSControl/dnscontrol/v4/pkg/diff"
	"github.com/DNSControl/dnscontrol/v4/pkg/printer"
	"github.com/DNSControl/dnscontrol/v4/pkg/providers"
	dnsutilv1 "github.com/miekg/dns/dnsutil"
)

// Section 1: Register this provider in the system.

// init registers the provider to dnscontrol.
func init() {
	const providerName = "LOOPIA"
	const providerMaintainer = "@systemcrash"
	fns := providers.DspFuncs{
		Initializer:   newDsp,
		RecordAuditor: AuditRecords,
	}
	providers.RegisterDomainServiceProviderType(providerName, fns, features)
	providers.RegisterRegistrarType(providerName, newReg)
	providers.RegisterMaintainer(providerName, providerMaintainer)
	providers.RegisterCredsMetadata(providerName, providers.CredsMetadata{
		DisplayName: "Loopia",
		Kind:        providers.KindDNS | providers.KindRegistrar,
		DocsURL:     "https://docs.dnscontrol.org/provider/loopia",
		PortalURL:   "https://support.loopia.com/wiki/loopiaapi/",
		Fields: []providers.CredsField{
			{
				Key:      "username",
				Label:    "API username",
				Help:     "Your Loopia API username.",
				Required: true,
			},
			{
				Key:      "password",
				Label:    "API password",
				Help:     "Your Loopia API password.",
				Secret:   true,
				Required: true,
			},
		},
	})
}

// features declares which features and options are available.
var features = providers.DocumentationNotes{
	// The default for unlisted capabilities is 'Cannot'.
	// See providers/capabilities.go for the entire list of capabilities.
	providers.CanAutoDNSSEC:          providers.Cannot(),
	providers.CanConcur:              providers.Unimplemented(),
	providers.CanGetZones:            providers.Can(),
	providers.CanOnlyDiff1Features:   providers.Can(),
	providers.CanUseAKAMAICDN:        providers.Cannot(),
	providers.CanUseAlias:            providers.Cannot(),
	providers.CanUseAzureAlias:       providers.Cannot(),
	providers.CanUseCAA:              providers.Can(),
	providers.CanUseDHCID:            providers.Cannot(), // Verified 2025-07-24
	providers.CanUseDNSKEY:           providers.Cannot(), // Verified 2025-07-24
	providers.CanUseDS:               providers.Cannot("Only supports DS records at the apex, only for .se and .nu domains; done automatically at back-end."),
	providers.CanUseDSForChildren:    providers.Cannot(),
	providers.CanUseHTTPS:            providers.Cannot(), // Verified 2025-07-24
	providers.CanUseLOC:              providers.Can(),
	providers.CanUseNAPTR:            providers.Can(),
	providers.CanUsePTR:              providers.Cannot(),
	providers.CanUseSOA:              providers.Cannot("💩"),
	providers.CanUseSRV:              providers.Can(),
	providers.CanUseSSHFP:            providers.Can(),
	providers.CanUseSVCB:             providers.Cannot(), // Verified 2025-07-24
	providers.CanUseTLSA:             providers.Can(),
	providers.DocCreateDomains:       providers.Cannot("Can only manage domains registered through their service"),
	providers.DocDualHost:            providers.Can(),
	providers.DocOfficiallySupported: providers.Cannot(),
}

// Section 2: Define the API client.

// See client.go

// newDsp generates a DNS Service Provider client handle.
func newDsp(conf map[string]string, metadata json.RawMessage) (providers.DNSServiceProvider, error) {
	return newHelper(conf, metadata)
}

// newReg generates a Registrar Provider client handle.
func newReg(conf map[string]string) (providers.Registrar, error) {
	return newHelper(conf, nil)
}

// newHelper generates a handle.
func newHelper(m map[string]string, _ json.RawMessage) (*APIClient, error) {
	if m["username"] == "" {
		return nil, errors.New("missing Loopia API username")
	}
	if m["password"] == "" {
		return nil, errors.New("missing Loopia API password")
	}

	const booleanStringWarn = " setting as a 'string': 't', 'true', 'True' etc"
	var err error

	modifyNameServers := false
	if m["modify_name_servers"] != "" { // optional
		modifyNameServers, err = strconv.ParseBool(m["modify_name_servers"])
		if err != nil {
			return nil, errors.New("creds.json requires the modify_name_servers" + booleanStringWarn)
		}
	}

	fetchApexNSEntries := false
	if m["fetch_apex_ns_entries"] != "" { // optional
		fetchApexNSEntries, err = strconv.ParseBool(m["fetch_apex_ns_entries"])
		if err != nil {
			return nil, errors.New("creds.json requires the fetch_apex_ns_entries" + booleanStringWarn)
		}
	}

	dbg := false
	if m["debug"] != "" { // debug is optional
		dbg, err = strconv.ParseBool(m["debug"])
		if err != nil {
			return nil, errors.New("creds.json requires the debug" + booleanStringWarn)
		}
	}

	api := NewClient(m["username"], m["password"], strings.ToLower(m["region"]), modifyNameServers, fetchApexNSEntries, dbg)

	quota := m["rate_limit_per"]
	err = api.requestRateLimiter.setRateLimitPer(quota)
	if err != nil {
		return nil, fmt.Errorf("unexpected value for rate_limit_per: %w", err)
	}
	return api, nil
}

// Section 3: Domain Service Provider (DSP) related functions

// ListZones lists the zones on this account.
func (c *APIClient) ListZones() ([]string, error) {
	listResp, err := c.getDomains()
	if err != nil {
		return nil, err
	}

	zones := make([]string, len(listResp))
	if c.Debug {
		fmt.Printf("DEBUG: DOMAIN LIST START\n")
	}
	for i, zone := range listResp {
		for _, prop := range zone.Properties {
			if prop.Name() == "domain" { // the zones name is stored in property 'domain'
				if c.Debug {
					fmt.Printf("DEBUG: DOMAIN LIST %d: %v\n", i, prop.String())
				}
				// zone := zone
				zones[i] = prop.String()
			}
		}
	}
	if c.Debug {
		fmt.Printf("DEBUG: DOMAIN LIST END\n")
	}
	return zones, nil
}

// GetZoneRecords gathers the DNS records and converts them to
// dnscontrol's format.
func (c *APIClient) GetZoneRecords(dc *models.DomainConfig) (models.Records, error) {
	domain := dc.Name

	// Two approaches. One: get all SubDomains, and get their respective records
	// simultaneously, or first get subdomains then fill each subdomain with its
	// respective records on a subsequent pass.

	// step 1: subdomains
	// Get existing subdomains for a domain:
	subdomains, err := c.GetSubDomains(domain)
	if err != nil {
		return nil, err
	}

	if c.Debug {
		fmt.Printf("Amount of subdomains: %d\n", len(subdomains))
	}

	// Convert them to DNScontrol's native format:
	existingRecords := []*models.RecordConfig{}
	for _, subdomain := range subdomains {
		// here seems like a good place to get the records for a subdomain.
		// fukn ballz tho: each subdomain requires one API call. 💩
		if c.Debug {
			fmt.Printf("%s\n", subdomain)
		}
		// step 2: records for subdomains
		// Get subdomain records:
		subdomainrecords, err := c.getDomainRecords(domain, subdomain)
		if err != nil {
			return nil, err
		}

		for _, subdRr := range subdomainrecords {
			// Note: subdomain cannot be any of [.-_ ]
			record, err := nativeToRecord(subdRr, domain, subdomain)
			if err != nil {
				return nil, err
			}
			existingRecords = append(existingRecords, record)
		}
	}

	if c.Debug {
		fmt.Printf("length of existingRecords: %d\n", len(existingRecords))
	}

	return existingRecords, nil
}

// PrepFoundRecords munges any records to make them compatible with
// this provider. Usually this is a no-op.
// func PrepFoundRecords(recs models.Records) models.Records {
// If there are records that need to be modified, removed, etc. we
// do it here.  Usually this is a no-op.
//return recs
//}

// PrepDesiredRecords munges any records to best suit this provider.
func PrepDesiredRecords(dc *models.DomainConfig) {
	// Sort through the dc.Records, eliminate any that can't be
	// supported; modify any that need adjustments to work with the
	// provider.  We try to do minimal changes otherwise it gets
	// confusing.

	recordsToKeep := make([]*models.RecordConfig, 0, len(dc.Records))
	for _, rec := range dc.Records {
		if rec.Type == "ALIAS" {
			// Loopia does not support ALIAS.
			// Therefore, we change this to a CNAME.
			rec.ChangeType("CNAME", dc.Name)
		}
		if rec.TTL < 300 {
			/* you can submit TTL lower than 300 but the dig results are normalized to 300 */
			printer.Warnf("Loopia does not support TTL < 300. Setting %s from %d to 300\n", rec.GetLabelFQDN(), rec.TTL)
			rec.TTL = 300
		} else if rec.TTL > 2147483647 {
			/* you can submit a TTL higher than 4294967296 but Loopia shortens it to 2147483647. 68 year timeout tho. */
			printer.Warnf("Loopia does not support TTL > 68 years. Setting %s from %d to 2147483647\n", rec.GetLabelFQDN(), rec.TTL)
			rec.TTL = 2147483647
		}
		// if rec.Type == "NS" && rec.GetLabel() == "@" {
		// 	if !strings.HasSuffix(rec.GetTargetField(), ".loopia.se.") {
		// 		printer.Warnf("Loopia does not support changing apex NS records. Ignoring %s\n", rec.GetTargetField())
		// 	}
		// 	continue
		// }
		recordsToKeep = append(recordsToKeep, rec)
	}
	dc.Records = recordsToKeep
}

// gatherAffectedLabels takes the output of diff.ChangedGroups and
// regroups it by FQDN of the label, not by Key. It also returns
// a list of all the FQDNs.
func gatherAffectedLabels(groups map[models.RecordKey][]string) (labels map[string]bool, msgs map[string][]string) {
	labels = map[string]bool{}
	msgs = map[string][]string{}
	for k, v := range groups {
		labels[k.NameFQDN] = true
		msgs[k.NameFQDN] = append(msgs[k.NameFQDN], v...)
	}
	return labels, msgs
}

// GetZoneRecordsCorrections returns a list of corrections that will turn existing records into dc.Records.
func (c *APIClient) GetZoneRecordsCorrections(dc *models.DomainConfig, existingRecords models.Records) ([]*models.Correction, int, error) {
	if c.Debug {
		debugRecords("GenerateZoneRecordsCorrections input:\n", existingRecords)
	}

	PrepDesiredRecords(dc)

	var keysToUpdate map[models.RecordKey][]string
	differ := diff.NewCompat(dc)
	toReport, create, del, modify, actualChangeCount, err := differ.IncrementalDiff(existingRecords)
	if err != nil {
		return nil, 0, err
	}
	// Start corrections with the reports
	corrections := diff.GenerateMessageCorrections(toReport)

	keysToUpdate, _, _, err = differ.ChangedGroups(existingRecords)
	if err != nil {
		return nil, 0, err
	}

	for _, d := range create {
		// fmt.Printf("a creation: subdomain: %+v, existingfqdn: %+v \n", d.Desired.Name, d.Desired.NameFQDN)
		des := d.Desired
		zrec := recordToNative(des)
		corrections = append(corrections, &models.Correction{
			Msg: d.String(),
			F: func() error {
				// return c.CreateRecordSimulate(dc.Name, des.Name, zrec)
				return c.CreateRecord(dc.Name, des.Name, zrec)
			},
		})
	}

	// Determine which subdomains become extinct. Delete them.
	affectedLabels, msgsForLabel := gatherAffectedLabels(keysToUpdate)
	_, desiredRecords := dc.Records.GroupedByFQDN()

	for fqdn := range affectedLabels {
		if len(desiredRecords[fqdn]) == 0 {
			msgs := strings.Join(msgsForLabel[fqdn], "\n")
			msgs = "records affected by deletion of subdomain " + fqdn + "\n" + msgs
			subdomain := dnsutilv1.TrimDomainName(fqdn, dc.Name)
			corrections = append(corrections, &models.Correction{
				Msg: msgs,
				F: func() error {
					return c.DeleteSubdomain(dc.Name, subdomain)
				},
			})
		}
	}

	for _, d := range del {
		skip := false
		for fqdn := range affectedLabels {
			if len(desiredRecords[fqdn]) == 0 {
				subdomain := dnsutilv1.TrimDomainName(fqdn, dc.Name)
				if d.Existing.NameFQDN == fqdn && d.Existing.Name == subdomain {
					// fmt.Printf("fqdn extinct wtf: %s\n", fqdn)
					// deletion is a member of fqdn. skip its deletion (otherwise extra API call and its error)
					skip = true
				}
			}
		}
		if !skip {
			// fmt.Printf("a deletion: subdomain: %+v, existingfqdn: %+v \n", d.Existing.Name, d.Existing.NameFQDN)
			existingRecord := d.Existing.Original.(zRec)
			corrections = append(corrections, &models.Correction{
				Msg: d.String(),
				F: func() error {
					// return c.DeleteRecordSimulate(dc.Name, d.Existing.Name, existingRecord.RecordID)
					return c.DeleteRecord(dc.Name, d.Existing.Name, existingRecord.RecordID)
				},
			})
		}
	}

	for _, d := range modify {
		subdomain := d.Existing.Name
		// fmt.Printf("a modification: subdomain: %+v, existingfqdn: %+v \n", d.Existing.Name, d.Existing.NameFQDN)
		rec := d.Desired
		existingID := d.Existing.Original.(zRec).RecordID
		zrec := recordToNative(rec, existingID)
		corrections = append(corrections, &models.Correction{
			Msg: d.String(),
			F: func() error {
				// weird BUG: if we provide d.Desired.Name, instead of 'subdomain',
				// all change records get assigned a single subdomain, common across all change records.
				// return c.UpdateRecordSimulate(dc.Name, subdomain, zrec)
				return c.UpdateRecord(dc.Name, subdomain, zrec)
			},
		})
	}

	return corrections, actualChangeCount, nil
}

// debugRecords prints a list of RecordConfig.
func debugRecords(note string, recs []*models.RecordConfig) {
	printer.Debugf("%s", note)
	for k, v := range recs {
		printer.Printf("   %v: %v %v %v %v\n", k, v.GetLabel(), v.Type, v.TTL, v.GetTargetCombined())
	}
}

// Section 3: Registrar-related functions

// GetNameservers returns a list of nameservers for domain.
func (c *APIClient) GetNameservers(domain string) ([]*models.Nameserver, error) {
	if c.ModifyNameServers {
		return nil, nil
	}
	nameservers, err := c.GetDomainNS(domain)
	if err != nil {
		return nil, err
	}
	return models.ToNameserversStripTD(nameservers)
}

// GetRegistrarCorrections returns a list of corrections for this registrar.
func (c *APIClient) GetRegistrarCorrections(dc *models.DomainConfig) ([]*models.Correction, error) {
	existingNs, err := c.GetDomainNS(dc.Name)
	if err != nil {
		return nil, err
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
				F: func() (err error) {
					// err = c.UpdateNameServers(dc.Name, desiredNs)
					return
				},
			},
		}, nil
	}
	return nil, nil
}
