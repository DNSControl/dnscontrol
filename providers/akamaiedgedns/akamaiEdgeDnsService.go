package akamaiedgedns

/*
For information about Akamai's "Edge DNS Zone Management API", see:
https://developer.akamai.com/api/cloud_security/edge_dns_zone_management/v2.html

For information about "AkamaiOPEN-edgegrid-golang" library, see:
https://github.com/akamai/AkamaiOPEN-edgegrid-golang
*/

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/DNSControl/dnscontrol/v5/models"
	"github.com/DNSControl/dnscontrol/v5/pkg/printer"
	"github.com/DNSControl/dnscontrol/v5/pkg/privatetypes"
	"github.com/DNSControl/dnscontrol/v5/pkg/providers"
	"github.com/akamai/AkamaiOPEN-edgegrid-golang/v12/pkg/dns"
	"github.com/akamai/AkamaiOPEN-edgegrid-golang/v12/pkg/edgegrid"
	"github.com/akamai/AkamaiOPEN-edgegrid-golang/v12/pkg/session"
)

// initialize initializes the "Akamai OPEN EdgeGrid" library.
func newEdgegridConfig(clientSecret string, host string, accessToken string, clientToken string) *edgegrid.Config {
	return &edgegrid.Config{
		ClientSecret: clientSecret,
		Host:         host,
		AccessToken:  accessToken,
		ClientToken:  clientToken,
		MaxBody:      131072,
		Debug:        false,
	}
}

func initialize(clientSecret string, host string, accessToken string, clientToken string) (dns.DNS, error) {
	config := newEdgegridConfig(clientSecret, host, accessToken, clientToken)
	sess, err := session.New(
		session.WithSigner(config),
		session.WithHTTPTracing(true),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create EdgeGrid session: %w", err)
	}

	return dns.Client(sess), nil
}

// zoneDoesExist returns true if the zone exists, false otherwise.
func (a *edgeDNSProvider) zoneDoesExist(ctx context.Context, zonename string) bool {
	_, err := a.client.GetZone(ctx, dns.GetZoneRequest{Zone: zonename})
	return err == nil
}

// createZone create a new zone and creates SOA and NS records for the zone.
// Akamai assigns a unique set of authoritative nameservers for each contract. These authorities should be
// used as the NS records on all zones belonging to this contract.
func (a *edgeDNSProvider) createZone(ctx context.Context, zonename string, contractID string, groupID string) error {
	zone := &dns.ZoneCreate{
		Zone:                  zonename,
		Type:                  "PRIMARY",
		Comment:               "This zone created by DNSControl (http://dnscontrol.org)",
		SignAndServe:          false,
		SignAndServeAlgorithm: "RSA_SHA512",
		ContractID:            contractID,
	}

	queryArgs := dns.ZoneQueryString{
		Contract: contractID,
		Group:    groupID,
	}

	err := dns.ValidateZone(zone)
	if err != nil {
		return fmt.Errorf("invalid value provided for zone. error: %s", err.Error())
	}

	err = a.client.CreateZone(ctx, dns.CreateZoneRequest{
		CreateZone:      zone,
		ZoneQueryString: queryArgs,
	})
	if err != nil {
		return fmt.Errorf("zone create failed. error: %s", err.Error())
	}

	// Indirectly create NS and SOA records
	err = a.client.SaveChangeList(ctx, dns.SaveChangeListRequest{Zone: zonename})
	if err != nil {
		return errors.New("zone initialization failed. SOA and NS records need to be created")
	}
	err = a.client.SubmitChangeList(ctx, dns.SubmitChangeListRequest{Zone: zonename})
	if err != nil {
		return fmt.Errorf("zone create failed. error: %s", err.Error())
	}

	printer.Printf("Created zone: %s\n", zone.Zone)
	printer.Printf("  Type: %s\n", zone.Type)
	printer.Printf("  Comment: %s\n", zone.Comment)
	printer.Printf("  SignAndServe: %v\n", zone.SignAndServe)
	printer.Printf("  SignAndServeAlgorithm: %s\n", zone.SignAndServeAlgorithm)
	printer.Printf("  ContractId: %s\n", zone.ContractID)
	printer.Printf("  GroupId: %s\n", queryArgs.Group)

	return nil
}

// listZones lists all zones associated with this contract.
func (a *edgeDNSProvider) listZones(ctx context.Context, contractID string) ([]string, error) {
	queryArgs := dns.ListZonesRequest{
		ContractIDs: contractID,
		ShowAll:     true,
	}

	zoneListResp, err := a.client.ListZones(ctx, queryArgs)
	if err != nil {
		return nil, fmt.Errorf("zone list retrieval failed. error: %s", err.Error())
	}

	edgeDNSZones := zoneListResp.Zones // what we have
	var zones []string                 // what we return

	for _, edgeDNSZone := range edgeDNSZones {
		zones = append(zones, edgeDNSZone.Zone)
	}

	return zones, nil
}

type statusCoder interface {
	StatusCode() int
}

// isAutoDNSSecEnabled returns true if AutoDNSSEC (SignAndServe) is enabled, false otherwise.
func (a *edgeDNSProvider) isAutoDNSSecEnabled(ctx context.Context, zonename string) (bool, error) {
	zone, err := a.client.GetZone(ctx, dns.GetZoneRequest{Zone: zonename})
	if err != nil {
		if sc, ok := err.(statusCoder); ok && sc.StatusCode() == 404 {
			return false, fmt.Errorf("zone %s does not exist. error: %s",
				zonename, err.Error())
		}
		return false, fmt.Errorf("error retrieving information for zone %s. error: %s",
			zonename, err.Error())
	}
	return zone.SignAndServe, nil
}

// autoDNSSecEnable enables or disables AutoDNSSEC (SignAndServe) for the zone.
func (a *edgeDNSProvider) autoDNSSecEnable(ctx context.Context, enable bool, zonename string) error {
	zone, err := a.client.GetZone(ctx, dns.GetZoneRequest{Zone: zonename})
	if err != nil {
		if sc, ok := err.(statusCoder); ok && sc.StatusCode() == 404 {
			return fmt.Errorf("zone %s does not exist. error: %s",
				zonename, err.Error())
		}
		return fmt.Errorf("error retrieving information for zone %s. error: %s",
			zonename, err.Error())
	}

	algorithm := "RSA_SHA512"
	if zone.SignAndServeAlgorithm != "" {
		algorithm = zone.SignAndServeAlgorithm
	}

	modifiedzone := &dns.ZoneCreate{
		Zone:                  zone.Zone,
		Type:                  zone.Type,
		Masters:               zone.Masters,
		Comment:               zone.Comment,
		SignAndServe:          enable, // AutoDNSSEC
		SignAndServeAlgorithm: algorithm,
		TSIGKey:               zone.TSIGKey,
		EndCustomerID:         zone.EndCustomerID,
		ContractID:            zone.ContractID,
	}

	err = a.client.UpdateZone(ctx, dns.UpdateZoneRequest{
		CreateZone: modifiedzone,
	})
	if err != nil {
		return fmt.Errorf("error updating zone %s. error: %s",
			zonename, err.Error())
	}

	return nil
}

// getAuthorities returns the list of authoritative nameservers for the contract.
// Akamai assigns a unique set of authoritative nameservers for each contract. These authorities should be
// used as the NS records on all zones belonging to this contract.
func (a *edgeDNSProvider) getAuthorities(ctx context.Context, contractID string) ([]string, error) {
	authorityResponse, err := a.client.GetAuthorities(ctx, dns.GetAuthoritiesRequest{ContractIDs: contractID})
	if err != nil {
		return nil, fmt.Errorf("getAuthorities - contractid %s: authorities retrieval failed. Error: %s",
			contractID, err.Error())
	}
	contracts := authorityResponse.Contracts
	if len(contracts) != 1 {
		return nil, fmt.Errorf("getAuthorities - contractid %s: Expected 1 element in array but got %d",
			contractID, len(contracts))
	}
	cid := contracts[0].ContractID
	if cid != contractID {
		return nil, fmt.Errorf("getAuthorities - contractID %s: got authorities for wrong contractID (%s)",
			contractID, cid)
	}
	authorities := contracts[0].Authorities
	return authorities, nil
}

// rcToRs converts DNSControl RecordConfig records to an AkamaiEdgeDNS recordset.
func (a *edgeDNSProvider) rcToRs(records models.Records) (*dns.RecordBody, error) {
	input := models.Records(records)
	before := providers.BeginToNative(a.observer, "rcToRs", input)
	if len(records) == 0 {
		err := errors.New("no records to replace")
		providers.EndToNative(a.observer, "rcToRs", before, input, nil, err)
		return nil, err
	}

	akaRecord := &dns.RecordBody{
		Name:       records[0].NameFQDN,
		RecordType: records[0].Type,
		TTL:        int(records[0].TTL),
	}

	for _, r := range records {
		if r.Type == "AKAMAITLC" {
			f := r.AsAKAMAITLC()
			akaRecord.Target = append(akaRecord.Target, f.AnswerType+" "+f.Target)
		} else {
			akaRecord.Target = append(akaRecord.Target, r.GetRDATA().String())
		}
	}

	providers.EndToNative(a.observer, "rcToRs", before, input, akaRecord, nil)
	return akaRecord, nil
}

// createRecordset creates a new AkamaiEdgeDNS recordset in the zone.
func (a *edgeDNSProvider) createRecordset(ctx context.Context, records models.Records, zonename string) error {
	akaRecord, err := a.rcToRs(records)
	if err != nil {
		return err
	}

	err = a.client.CreateRecord(ctx, dns.CreateRecordRequest{
		Zone:   zonename,
		Record: akaRecord,
	})
	if err != nil {
		return fmt.Errorf("recordset creation failed. error: %s", err.Error())
	}
	return nil
}

// replaceRecordset replaces an existing AkamaiEdgeDNS recordset in the zone.
func (a *edgeDNSProvider) replaceRecordset(ctx context.Context, records models.Records, ttl uint32, zonename string) error {
	akaRecord, err := a.rcToRs(records)
	if err != nil {
		return err
	}
	akaRecord.TTL = int(ttl)

	err = a.client.UpdateRecord(ctx, dns.UpdateRecordRequest{
		Zone:   zonename,
		Record: akaRecord,
	})
	if err != nil {
		return fmt.Errorf("recordset update failed. error: %s", err.Error())
	}
	return nil
}

// deleteRecordset deletes an existing AkamaiEdgeDNS recordset in the zone.
func (a *edgeDNSProvider) deleteRecordset(ctx context.Context, records models.Records, zonename string) error {
	akaRecord, err := a.rcToRs(records)
	if err != nil {
		return err
	}

	err = a.client.DeleteRecord(ctx, dns.DeleteRecordRequest{
		Zone:       zonename,
		Name:       akaRecord.Name,
		RecordType: akaRecord.RecordType,
	})
	if err != nil {
		if sc, ok := err.(statusCoder); ok && sc.StatusCode() == 404 {
			return errors.New("recordset not found")
		}
		return fmt.Errorf("failed to delete recordset. error: %s", err.Error())
	}
	return nil
}

/*
  Example AkamaiEdgeDNS Recordset (as JSON):
        {
            "name": "test.com",
            "rdata": [
                "a7.akafp.net.",
                "a4.akafp.net.",
                "a0.akafp.net."
            ],
            "ttl": 10000,
            "type": "NS"
        }
*/

// getRecords returns all RecordConfig records in the zone.
func (a *edgeDNSProvider) getRecords(ctx context.Context, dc *models.DomainConfig) (models.Records, error) {
	zonename := dc.Name
	queryArgs := dns.RecordSetQueryArgs{ShowAll: true}

	rsetResp, err := a.client.GetRecordSets(ctx, dns.GetRecordSetsRequest{
		Zone:      zonename,
		QueryArgs: &queryArgs,
	})
	if err != nil {
		return nil, fmt.Errorf("recordset list retrieval failed. error: %s", err.Error())
	}

	akaRecordsets := rsetResp.RecordSets // what we have
	var recordConfigs models.Records     // what we return

	// For each AkamaiEdgeDNS recordset...
	for _, akarecset := range akaRecordsets {
		before := providers.BeginToRC(a.observer, "nativeToRecords", akarecset)
		recs, err := nativeToRecords(dc, akarecset)
		providers.EndToRC(a.observer, "nativeToRecords", before, akarecset, recs, err)
		if err != nil {
			return nil, err
		}
		recordConfigs = append(recordConfigs, recs...)
	}

	return recordConfigs, nil
}

// nativeToRecords converts an AkamaiEdgeDNS recordset into 1 or more
// RecordConfig structs. It returns nothing for an SOA recordset, whose existence
// is not reported (because DnsControl will try to delete the SOA record).
func nativeToRecords(dc *models.DomainConfig, akarecset dns.RecordSet) (models.Records, error) {
	akaname := akarecset.Name
	akatype := akarecset.Type
	akattl := akarecset.TTL
	label := dc.LabelFromFQDNNoDot(akaname)

	if akatype == "SOA" {
		return nil, nil
	}

	// AKAMAITLC has 2 rdata entries that form 1 logical record: [answerType, target]
	if akatype == "AKAMAITLC" {
		combined := strings.Join(akarecset.Rdata, " ")
		parts := strings.Fields(combined)
		if len(parts) != 2 {
			return nil, fmt.Errorf("AKAMAITLC rdata must contain 2 fields, got: %v", akarecset.Rdata)
		}
		rc, err := dc.NewRecordConfig(label, uint32(akattl), privatetypes.TypeAKAMAITLC, parts[0], parts[1])
		if err != nil {
			return nil, err
		}
		rc.Metadata = map[string]string{"akamai_raw_rdata": combined}
		return models.Records{rc}, nil
	}

	var recordConfigs models.Records
	for _, r := range akarecset.Rdata {
		data := r
		if akatype == "LOC" {
			data = fixLocAltitude(r)
		}
		rc, err := dc.NewRecordConfigParse(label, uint32(akattl), akatype, data)
		if err != nil {
			return nil, err
		}

		rc.Metadata = map[string]string{"akamai_raw_rdata": r}
		recordConfigs = append(recordConfigs, rc)
	}

	return recordConfigs, nil
}

var reLocNegAlt = regexp.MustCompile(`(-\d+)\.-(\d+)(m\s)`)

func fixLocAltitude(rdata string) string {
	return strings.TrimSpace(reLocNegAlt.ReplaceAllString(rdata+" ", `$1.$2$3`))
}
