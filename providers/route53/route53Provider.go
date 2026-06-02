package route53

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/credentials/stscreds"
	r53 "github.com/aws/aws-sdk-go-v2/service/route53"
	r53Types "github.com/aws/aws-sdk-go-v2/service/route53/types"
	r53d "github.com/aws/aws-sdk-go-v2/service/route53domains"
	r53dTypes "github.com/aws/aws-sdk-go-v2/service/route53domains/types"
	"github.com/aws/aws-sdk-go-v2/service/sts"

	"github.com/DNSControl/dnscontrol/v4/models"
	"github.com/DNSControl/dnscontrol/v4/pkg/diff2"
	"github.com/DNSControl/dnscontrol/v4/pkg/printer"
	"github.com/DNSControl/dnscontrol/v4/pkg/providers"
	"github.com/DNSControl/dnscontrol/v4/pkg/txtutil"
)

type route53Provider struct {
	client        *r53.Client
	registrar     *r53d.Client
	delegationSet *string
	zonesMu       sync.Mutex
	zonesByID     map[string]r53Types.HostedZone
	zonesByDomain map[string]r53Types.HostedZone
}

func newRoute53Reg(conf map[string]string) (providers.Registrar, error) {
	// AWS European Sovereign Cloud (aws.eu) does not support registering domains, at least not yet.
	// Let us assume only the global AWS is capable of registering domains currently.
	if conf["Region"] != "" && conf["Region"] != "us-east-1" {
		return nil, errors.New("Error! Domain register endpoint is only supported on the global AWS region us-east-1")
	} else {
		return newRoute53(conf, nil)
	}
}

func newRoute53Dsp(conf map[string]string, metadata json.RawMessage) (providers.DNSServiceProvider, error) {
	return newRoute53(conf, metadata)
}

func newRoute53(m map[string]string, _ json.RawMessage) (*route53Provider, error) {
	optFns := []func(*config.LoadOptions) error{}

	if m["Region"] != "" {
		// AWS European Sovereign Cloud (aws.eu) region eusc-de-east-1 uses a separate Route 53 API endpoint from the global AWS
		optFns = append(optFns, config.WithRegion(m["Region"]))
	} else {
		// Route53 uses a global endpoint and route53domains
		// currently only has a single regional endpoint in us-east-1
		// https://docs.aws.amazon.com/general/latest/gr/rande.html#r53_region
		optFns = append(optFns, config.WithRegion("us-east-1"))
	}

	keyID, secretKey, tokenID, roleArn, externalID := m["KeyId"], m["SecretKey"], m["Token"], m["RoleArn"], m["ExternalId"]
	// Token is optional and left empty unless required
	if keyID != "" || secretKey != "" {
		optFns = append(optFns, config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(keyID, secretKey, tokenID)))
	}

	profile := m["Profile"]
	if profile != "" && (keyID != "" || secretKey != "") {
		return nil, fmt.Errorf("route53: cannot set both Profile and KeyId/SecretKey")
	}
	if profile != "" {
		optFns = append(optFns, config.WithSharedConfigProfile(profile))
	}

	config, err := config.LoadDefaultConfig(context.Background(), optFns...)
	if err != nil {
		return nil, err
	}

	if roleArn != "" {
		stsClient := sts.NewFromConfig(config)
		sessionName := fmt.Sprintf("dnscontrol-route53-%d", time.Now().Unix())

		var assumeOpts []func(*stscreds.AssumeRoleOptions)
		if externalID != "" {
			assumeOpts = append(assumeOpts, func(o *stscreds.AssumeRoleOptions) { o.ExternalID = aws.String(externalID) })
		}
		assumeOpts = append(assumeOpts, func(o *stscreds.AssumeRoleOptions) { o.RoleSessionName = sessionName })

		stsCredsProvider := stscreds.NewAssumeRoleProvider(stsClient, roleArn, assumeOpts...)
		config.Credentials = aws.NewCredentialsCache(stsCredsProvider)
	}

	var dls *string
	if val, ok := m["DelegationSet"]; ok {
		printer.Printf("ROUTE53 DelegationSet %s configured\n", val)
		dls = aws.String(val)
	}
	api := &route53Provider{
		client:        r53.NewFromConfig(config),
		registrar:     r53d.NewFromConfig(config),
		delegationSet: dls,
		zonesByDomain: make(map[string]r53Types.HostedZone),
		zonesByID:     make(map[string]r53Types.HostedZone),
	}
	err = api.getZones()
	if err != nil {
		return nil, err
	}
	return api, nil
}

var features = providers.DocumentationNotes{
	// The default for unlisted capabilities is 'Cannot'.
	// See providers/capabilities.go for the entire list of capabilities.
	providers.CanGetZones:            providers.Can(),
	providers.CanConcur:              providers.Can(),
	providers.CanUseAlias:            providers.Cannot("R53 does not provide a generic ALIAS functionality. Use R53_ALIAS instead."),
	providers.CanUseCAA:              providers.Can(),
	providers.CanUseHTTPS:            providers.Can(),
	providers.CanUseLOC:              providers.Cannot(),
	providers.CanUsePTR:              providers.Can(),
	providers.CanUseRoute53Alias:     providers.Can(),
	providers.CanUseSRV:              providers.Can(),
	providers.CanUseSSHFP:            providers.Can(),
	providers.CanUseSVCB:             providers.Can(),
	providers.CanUseTLSA:             providers.Can(),
	providers.DocCreateDomains:       providers.Can(),
	providers.DocDualHost:            providers.Can(),
	providers.DocOfficiallySupported: providers.Can(),
}

func init() {
	const providerName = "ROUTE53"
	const providerMaintainer = "@tresni"
	fns := providers.DspFuncs{
		Initializer:   newRoute53Dsp,
		RecordAuditor: AuditRecords,
	}
	providers.RegisterDomainServiceProviderType(providerName, fns, features)
	providers.RegisterRegistrarType(providerName, newRoute53Reg)
	providers.RegisterCustomRecordType("R53_ALIAS", providerName, "")
	providers.RegisterMaintainer(providerName, providerMaintainer)
	providers.RegisterCredsMetadata(providerName, providers.CredsMetadata{
		DisplayName: "Amazon Route 53",
		Kind:        providers.KindDNS | providers.KindRegistrar,
		DocsURL:     "https://docs.dnscontrol.org/provider/route53",
		PortalURL:   "https://console.aws.amazon.com/route53/",
		Notes:       "Route53 supports several auth methods: a named profile from ~/.aws/config (including AWS IAM Identity Center / SSO), static access keys, or the SDK's default credential chain (environment variables, EC2 instance role, etc.). RoleArn can be layered on top of any of these.",
		Fields: []providers.CredsField{
			{
				Key:    "Region",
				Label:  "AWS Region to use for Route 53 control plane (optional)",
				Help:   "Leave blank to use default global Route 53 in us-east-1. Type \"eusc-de-east-1\" for AWS European Sovereign Cloud.",
				EnvVar: "AWS_DEFAULT_REGION",
			},
			{
				Key:      "_authMethod",
				Label:    "Which authentication method do you want to use?",
				Help:     "Named profile reads ~/.aws/config and supports SSO. Static access key uses KeyId/SecretKey. Default credential chain relies on the AWS SDK to discover credentials from the environment or instance role.",
				Choices:  []string{"Named profile (~/.aws/config, including SSO)", "Static access key", "Default credential chain"},
				Required: true,
				Internal: true,
			},
			{
				Key:      "Profile",
				Label:    "AWS profile name",
				Help:     "Name of the profile in ~/.aws/config to use. For SSO profiles, run `aws sso login` before invoking dnscontrol.",
				Required: true,
				ShowIf:   map[string]string{"_authMethod": "Named profile (~/.aws/config, including SSO)"},
			},
			{
				Key:      "KeyId",
				Label:    "AWS access key ID",
				Help:     "The AWS_ACCESS_KEY_ID for an IAM user or role with Route 53 permissions.",
				EnvVar:   "AWS_ACCESS_KEY_ID",
				Required: true,
				ShowIf:   map[string]string{"_authMethod": "Static access key"},
			},
			{
				Key:      "SecretKey",
				Label:    "AWS secret access key",
				Help:     "The AWS_SECRET_ACCESS_KEY paired with the access key ID.",
				EnvVar:   "AWS_SECRET_ACCESS_KEY",
				Secret:   true,
				Required: true,
				ShowIf:   map[string]string{"_authMethod": "Static access key"},
			},
			{
				Key:    "Token",
				Label:  "AWS session token (optional)",
				Help:   "STS session token. Leave blank unless you are using temporary credentials.",
				EnvVar: "AWS_SESSION_TOKEN",
				Secret: true,
				ShowIf: map[string]string{"_authMethod": "Static access key"},
			},
			{
				Key:   "RoleArn",
				Label: "Role ARN to assume (optional)",
				Help:  "If set, dnscontrol will call sts:AssumeRole on this ARN using the source credentials selected above. Leave blank to use the source credentials directly.",
			},
			{
				Key:   "ExternalId",
				Label: "External ID for AssumeRole (optional)",
				Help:  "External ID required by some trust policies. Only relevant when RoleArn is set.",
			},
			{
				Key:   "DelegationSet",
				Label: "Reusable delegation set ID (optional)",
				Help:  "Existing Route 53 reusable delegation set ID (the value after /delegationset/). Only applied when creating new domains.",
			},
		},
	})
}

func withRetry(f func() error) {
	const maxRetries = 23
	// TODO: exponential backoff
	const sleepTime = 5 * time.Second
	var currentRetry int
	for {
		err := f()
		if err == nil {
			return
		}
		if strings.Contains(err.Error(), "Rate exceeded") {
			currentRetry++
			if currentRetry >= maxRetries {
				return
			}
			printer.Printf("============ Route53 rate limit exceeded. Waiting %s to retry.\n", sleepTime)
			time.Sleep(sleepTime)
		} else {
			return
		}
	}
}

// ListZones lists the zones on this account.
func (r *route53Provider) ListZones() ([]string, error) {
	r.zonesMu.Lock()
	defer r.zonesMu.Unlock()
	var zones []string
	for i := range r.zonesByDomain {
		zones = append(zones, i)
	}
	return zones, nil
}

func (r *route53Provider) getZoneByDomain(domain string) (r53Types.HostedZone, bool) {
	r.zonesMu.Lock()
	defer r.zonesMu.Unlock()
	zone, ok := r.zonesByDomain[domain]
	return zone, ok
}

func (r *route53Provider) getZoneByID(id string) (r53Types.HostedZone, bool) {
	r.zonesMu.Lock()
	defer r.zonesMu.Unlock()
	zone, ok := r.zonesByID[id]
	return zone, ok
}

func (r *route53Provider) getZones() error {
	r.zonesMu.Lock()
	defer r.zonesMu.Unlock()
	var nextMarker *string
	for {
		var out *r53.ListHostedZonesOutput
		var err error
		withRetry(func() error {
			inp := &r53.ListHostedZonesInput{Marker: nextMarker}
			out, err = r.client.ListHostedZones(context.Background(), inp)
			return err
		})
		if err != nil && strings.Contains(err.Error(), "is not authorized") {
			return errors.New("check your credentials, you're not authorized to perform actions on Route 53 AWS Service")
		} else if err != nil {
			return err
		}
		for _, z := range out.HostedZones {
			r.addZoneToCacheLocked(z)
		}
		if out.NextMarker != nil {
			nextMarker = out.NextMarker
		} else {
			break
		}
	}
	return nil
}

// addZoneToCacheLocked adds the given zone to the cache.
// The caller must hold zonesMu.
func (r *route53Provider) addZoneToCacheLocked(z r53Types.HostedZone) {
	domain := strings.TrimSuffix(aws.ToString(z.Name), ".")
	r.zonesByDomain[domain] = z
	r.zonesByID[parseZoneID(aws.ToString(z.Id))] = z
}

type errDomainNoExist struct {
	domain string
}

type errZoneNoExist struct {
	zoneID string
}

func (e errDomainNoExist) Error() string {
	return fmt.Sprintf("Domain %s not found in your route 53 account", e.domain)
}

func (e errZoneNoExist) Error() string {
	return fmt.Sprintf("Zone with id %s not found in your route 53 account", e.zoneID)
}

func (r *route53Provider) GetNameservers(domain string) ([]*models.Nameserver, error) {
	zone, ok := r.getZoneByDomain(domain)
	if !ok {
		return nil, errDomainNoExist{domain}
	}
	var z *r53.GetHostedZoneOutput
	var err error
	withRetry(func() error {
		z, err = r.client.GetHostedZone(context.Background(), &r53.GetHostedZoneInput{Id: zone.Id})
		return err
	})
	if err != nil {
		return nil, err
	}

	var nss []string
	if z.DelegationSet != nil {
		nss = z.DelegationSet.NameServers
	}
	return models.ToNameservers(nss)
}

func (r *route53Provider) GetZoneRecords(dc *models.DomainConfig) (models.Records, error) {
	domain := dc.Name
	meta := dc.Metadata

	// If the zone_id is specified in meta, use it.
	if zoneID, ok := meta["zone_id"]; ok {
		zone, found := r.getZoneByID(zoneID)
		if !found {
			return nil, errZoneNoExist{zoneID}
		}
		return r.getZoneRecords(zone)
	}

	//	fmt.Printf("DEBUG: ROUTE53 zones:\n")
	//	for i, j := range r.zonesByDomain {
	//		fmt.Printf("       %s: %v\n", i, aws.ToString(j.Id))
	//	}

	// Otherwise, use the domain name to look up the zone.
	if zone, ok := r.getZoneByDomain(domain); ok {
		return r.getZoneRecords(zone)
	}

	// Not found there?  Error.
	return nil, errDomainNoExist{domain}
}

func (r *route53Provider) getZone(dc *models.DomainConfig) (r53Types.HostedZone, error) {
	if zoneID, ok := dc.Metadata["zone_id"]; ok {
		zone, ok := r.getZoneByID(zoneID)
		if !ok {
			return r53Types.HostedZone{}, errZoneNoExist{zoneID}
		}
		return zone, nil
	}

	if zone, ok := r.getZoneByDomain(dc.Name); ok {
		return zone, nil
	}

	return r53Types.HostedZone{}, errDomainNoExist{dc.Name}
}

func (r *route53Provider) getZoneRecords(zone r53Types.HostedZone) (models.Records, error) {
	records, err := r.fetchRecordSets(zone.Id)
	if err != nil {
		return nil, err
	}

	existingRecords := []*models.RecordConfig{}
	for _, set := range records {
		rts, err := nativeToRecords(set, unescape(zone.Name))
		if err != nil {
			return nil, err
		}
		existingRecords = append(existingRecords, rts...)
	}
	return existingRecords, nil
}

// GetZoneRecordsCorrections returns a list of corrections that will turn existing records into dc.Records.
func (r *route53Provider) GetZoneRecordsCorrections(dc *models.DomainConfig, existingRecords models.Records) ([]*models.Correction, int, error) {
	zone, err := r.getZone(dc)
	if err != nil {
		return nil, 0, err
	}

	// update zone_id to current zone.id if not specified by the user
	for _, want := range dc.Records {
		if want.Type == "R53_ALIAS" && want.R53Alias["zone_id"] == "" {
			want.R53Alias["zone_id"] = getZoneID(zone, want)
		}
	}

	var corrections []*models.Correction
	changes := []r53Types.Change{}
	changeDesc := []string{} // TODO(tlim): This should be a [][]string so that we aren't joining strings until the last moment.

	// Amazon Route53 is a "ByRecordSet" API.
	// At each label:rtype pair, we either delete all records or UPSERT the desired records.
	instructions, actualChangeCount, err := diff2.ByRecordSet(existingRecords, dc, r53ComparableFunc)
	if err != nil {
		return nil, 0, err
	}
	var reports []*models.Correction

	// wasReport := false
	for _, inst := range instructions {
		instNameFQDN := inst.Key.NameFQDN
		instType := inst.Key.Type

		// Strip set identifier suffix added by Key() for weighted routing.
		setIdentifier := ""
		if idx := strings.Index(instType, "!"); idx != -1 {
			setIdentifier = instType[idx+1:]
			instType = instType[:idx]
		}

		var chg r53Types.Change

		switch inst.Type {
		case diff2.REPORT:
			// REPORTs are held in a separate list so that they aren't part of the batching process.
			reports = append(reports,
				&models.Correction{
					Msg: inst.MsgsJoined,
				})
			continue

		case diff2.CREATE:
			fallthrough
		case diff2.CHANGE:
			// To CREATE/CHANGE, build a new record set from the desired state and UPSERT it.

			// Make the rrset to be UPSERTed:
			var rrset *r53Types.ResourceRecordSet
			if instType == "R53_ALIAS" || strings.HasPrefix(instType, "R53_ALIAS_") {
				// A R53_ALIAS_* requires ResourceRecordSet to a single item, not a list.
				if len(inst.New) != 1 {
					log.Fatal("Only one R53_ALIAS_ permitted on a label")
				}
				rrset = aliasToRRSet(zone, inst.New[0])
				rrset.Name = aws.String(instNameFQDN)
			} else {
				// Make a list of all the records to be installed at label:rtype
				rrset = &r53Types.ResourceRecordSet{
					Name: aws.String(instNameFQDN),
					Type: r53Types.RRType(instType),
				}

				for _, r := range inst.New {
					rr := r53Types.ResourceRecord{
						Value: aws.String(r.GetTargetCombinedFunc(txtutil.EncodeQuoted)),
					}
					rrset.ResourceRecords = append(rrset.ResourceRecords, rr)
					i := int64(r.TTL)
					rrset.TTL = &i
				}
			}

			// Apply weighted routing policy fields from record metadata.
			if setIdentifier != "" {
				rrset.SetIdentifier = aws.String(setIdentifier)
				applyR53RoutingFieldsToRRSet(rrset, inst.New[0])
			}

			chg = r53Types.Change{
				Action:            r53Types.ChangeActionUpsert,
				ResourceRecordSet: rrset,
			}

		case diff2.DELETE:
			rrset := inst.Old[0].Original.(r53Types.ResourceRecordSet) // The native record as downloaded via the API
			chg = r53Types.Change{
				Action:            r53Types.ChangeActionDelete,
				ResourceRecordSet: &rrset,
			}

		default:
			panic(fmt.Sprintf("unhandled inst.Type %s", inst.Type))
		}

		changes = append(changes, chg)
		changeDesc = append(changeDesc, inst.MsgsJoined)
	}

	addCorrection := func(msg string, req *r53.ChangeResourceRecordSetsInput) {
		corrections = append(corrections,
			&models.Correction{
				Msg: msg,
				F: func() error {
					var err error
					req.HostedZoneId = zone.Id
					withRetry(func() error {
						_, err = r.client.ChangeResourceRecordSets(context.Background(), req)
						return err
					})
					return err
				},
			})
	}

	// Send the changes in as few API calls as possible.
	batcher := newChangeBatcher(changes)
	for batcher.Next() {
		start, end := batcher.Batch()
		batch := changes[start:end]
		descBatchStr := strings.Join(changeDesc[start:end], "\n")
		req := &r53.ChangeResourceRecordSetsInput{
			ChangeBatch: &r53Types.ChangeBatch{Changes: batch},
		}
		addCorrection(descBatchStr, req)
	}
	if err := batcher.Err(); err != nil {
		return nil, 0, err
	}

	return append(reports, corrections...), actualChangeCount, nil
}

func nativeToRecords(set r53Types.ResourceRecordSet, origin string) ([]*models.RecordConfig, error) {
	results := []*models.RecordConfig{}
	if set.AliasTarget != nil {
		rc := &models.RecordConfig{
			Type: "R53_ALIAS",
			TTL:  300,
			R53Alias: map[string]string{
				"type":                   string(set.Type),
				"zone_id":                aws.ToString(set.AliasTarget.HostedZoneId),
				"evaluate_target_health": strconv.FormatBool(set.AliasTarget.EvaluateTargetHealth),
			},
		}
		rc.SetLabelFromFQDN(unescape(set.Name), origin)
		if err := rc.SetTarget(aws.ToString(set.AliasTarget.DNSName)); err != nil {
			return nil, err
		}
		applyR53RoutingMeta(rc, set)
		// rc.Original stores a pointer to the original set for use by
		// r53Types.ChangeActionDelete and anything else that needs the
		// native record verbatim.
		rc.Original = set
		results = append(results, rc)
	} else if set.TrafficPolicyInstanceId != nil {
		// skip traffic policy records
	} else {
		for _, rec := range set.ResourceRecords {
			switch rtype := set.Type; rtype {
			case r53Types.RRTypeSoa:
				continue
			case r53Types.RRTypeSpf:
				// route53 uses a custom record type for SPF
				rtype = "TXT"
				fallthrough
			default:
				rtypeString := string(rtype)
				val := *rec.Value

				// AWS Route53 has a bug.  Sometimes it returns a target
				// without a trailing dot. In this case we add the dot.  It is
				// not risky to "just add the dot" because this field never
				// includes shortnames.  That said, we only do it for certain
				// record types where we can show the problem exists.
				// 2022-02-23: NS records do NOT have this bug.
				//
				// NOTE: The dot is missing when the record is added via the
				// AWS web console manually.
				//
				// The next "dnscontrol push" will update the record, even
				// though it doesn't seem to be broken. This only happens once
				// per record.  Sadly the updates only fix the first record.
				// So, if n records are affected by this bug, the next n
				// pushes will be required to clean up all the records.
				// Someone converting a new zone will see this issue for the
				// first n pushes. It will seem odd but this is AWS's bug.
				// The UPSERT command only fixes the first record, even if
				// the UPSET received a list of corrections.
				if rtypeString == "CNAME" || rtypeString == "MX" {
					if !strings.HasSuffix(val, ".") {
						val = val + "."
					}
				}

				rc := &models.RecordConfig{TTL: uint32(aws.ToInt64(set.TTL))}
				rc.SetLabelFromFQDN(unescape(set.Name), origin)
				rc.Original = set
				if err := rc.PopulateFromStringFunc(rtypeString, val, origin, txtutil.ParseQuoted); err != nil {
					return nil, fmt.Errorf("unparsable record type=%q received from ROUTE53: %w", rtypeString, err)
				}
				applyR53RoutingMeta(rc, set)

				results = append(results, rc)
			}
		}
	}
	return results, nil
}

// applyR53RoutingMeta populates RecordConfig metadata from native Route 53
// routing-policy fields (SetIdentifier, Weight, HealthCheckId).
func applyR53RoutingMeta(rc *models.RecordConfig, set r53Types.ResourceRecordSet) {
	if set.SetIdentifier == nil {
		return
	}
	if rc.Metadata == nil {
		rc.Metadata = map[string]string{}
	}
	rc.Metadata["r53_set_identifier"] = aws.ToString(set.SetIdentifier)
	if set.Weight != nil {
		rc.Metadata["r53_weight"] = strconv.FormatInt(*set.Weight, 10)
	}
	if set.HealthCheckId != nil {
		rc.Metadata["r53_health_check_id"] = aws.ToString(set.HealthCheckId)
	}
}

// r53ComparableFunc includes Route 53 routing-policy metadata in record
// comparison so that changes to weight or health check are detected by the diff.
func r53ComparableFunc(rc *models.RecordConfig) string {
	var parts []string
	if w, ok := rc.Metadata["r53_weight"]; ok && w != "" {
		parts = append(parts, "r53_weight="+w)
	}
	if hc, ok := rc.Metadata["r53_health_check_id"]; ok && hc != "" {
		parts = append(parts, "r53_health_check_id="+hc)
	}
	return strings.Join(parts, ",")
}

// applyR53RoutingFieldsToRRSet sets the Route 53 weighted routing fields on a
// ResourceRecordSet based on the RecordConfig metadata.
func applyR53RoutingFieldsToRRSet(rrset *r53Types.ResourceRecordSet, rc *models.RecordConfig) {
	if w, ok := rc.Metadata["r53_weight"]; ok && w != "" {
		weight, err := strconv.ParseInt(w, 10, 64)
		if err == nil {
			rrset.Weight = &weight
		}
	}
	if hc, ok := rc.Metadata["r53_health_check_id"]; ok && hc != "" {
		rrset.HealthCheckId = aws.String(hc)
	}
}

func aliasToRRSet(zone r53Types.HostedZone, r *models.RecordConfig) *r53Types.ResourceRecordSet {
	target := r.GetTargetField()
	zoneID := getZoneID(zone, r)
	evalTargetHealth, err := strconv.ParseBool(r.R53Alias["evaluate_target_health"])
	if err != nil {
		evalTargetHealth = false
	}
	rrset := &r53Types.ResourceRecordSet{
		Type: r53Types.RRType(r.R53Alias["type"]),
		AliasTarget: &r53Types.AliasTarget{
			DNSName:              &target,
			HostedZoneId:         aws.String(zoneID),
			EvaluateTargetHealth: evalTargetHealth,
		},
	}
	return rrset
}

func getZoneID(zone r53Types.HostedZone, r *models.RecordConfig) string {
	zoneID := r.R53Alias["zone_id"]
	if zoneID == "" {
		zoneID = aws.ToString(zone.Id)
	}
	return parseZoneID(zoneID)
}

/** Removes "/hostedzone/"" prefix from AWS ZoneId. */
func parseZoneID(zoneID string) string {
	return strings.TrimPrefix(zoneID, "/hostedzone/")
}

func (r *route53Provider) GetRegistrarCorrections(dc *models.DomainConfig) ([]*models.Correction, error) {
	corrections := []*models.Correction{}
	actualSet, err := r.getRegistrarNameservers(&dc.Name)
	if err != nil {
		return nil, err
	}
	sort.Strings(actualSet)
	actual := strings.Join(actualSet, ",")

	expectedSet := []string{}
	for _, ns := range dc.Nameservers {
		expectedSet = append(expectedSet, ns.Name)
	}
	sort.Strings(expectedSet)
	expected := strings.Join(expectedSet, ",")

	if actual != expected {
		return []*models.Correction{
			{
				Msg: fmt.Sprintf("Update nameservers %s -> %s", actual, expected),
				F: func() error {
					_, err := r.updateRegistrarNameservers(dc.Name, expectedSet)
					return err
				},
			},
		}, nil
	}

	return corrections, nil
}

func (r *route53Provider) getRegistrarNameservers(domainName *string) ([]string, error) {
	var domainDetail *r53d.GetDomainDetailOutput
	var err error
	withRetry(func() error {
		domainDetail, err = r.registrar.GetDomainDetail(context.Background(), &r53d.GetDomainDetailInput{DomainName: domainName})
		return err
	})
	if err != nil {
		return nil, err
	}

	nameservers := []string{}
	for _, ns := range domainDetail.Nameservers {
		nameservers = append(nameservers, aws.ToString(ns.Name))
	}

	return nameservers, nil
}

func (r *route53Provider) updateRegistrarNameservers(domainName string, nameservers []string) (*string, error) {
	servers := make([]r53dTypes.Nameserver, len(nameservers))
	for i := range nameservers {
		servers[i] = r53dTypes.Nameserver{Name: aws.String(nameservers[i])}
	}
	var domainUpdate *r53d.UpdateDomainNameserversOutput
	var err error
	withRetry(func() error {
		domainUpdate, err = r.registrar.UpdateDomainNameservers(context.Background(), &r53d.UpdateDomainNameserversInput{
			DomainName:  aws.String(domainName),
			Nameservers: servers,
		})
		return err
	})
	if err != nil {
		return nil, err
	}

	return domainUpdate.OperationId, nil
}

func (r *route53Provider) fetchRecordSets(zoneID *string) ([]r53Types.ResourceRecordSet, error) {
	if zoneID == nil || *zoneID == "" {
		return nil, nil
	}
	var next *string
	var nextType r53Types.RRType
	var nextIdentifier *string
	var records []r53Types.ResourceRecordSet
	for {
		listInput := &r53.ListResourceRecordSetsInput{
			HostedZoneId:          zoneID,
			StartRecordName:       next,
			StartRecordType:       nextType,
			StartRecordIdentifier: nextIdentifier,
			MaxItems:              aws.Int32(100),
		}
		var list *r53.ListResourceRecordSetsOutput
		var err error
		withRetry(func() error {
			list, err = r.client.ListResourceRecordSets(context.Background(), listInput)
			return err
		})
		if err != nil {
			return nil, err
		}

		records = append(records, list.ResourceRecordSets...)
		if list.NextRecordName != nil {
			next = list.NextRecordName
			nextType = list.NextRecordType
			nextIdentifier = list.NextRecordIdentifier
		} else {
			break
		}
	}
	return records, nil
}

// we have to process names from route53 to match what we expect and to remove their odd octal encoding.
func unescape(s *string) string {
	if s == nil {
		return ""
	}
	name := strings.TrimSuffix(*s, ".")
	name = strings.ReplaceAll(name, `\052`, "*") // TODO: escape all octal sequences
	return name
}

func (r *route53Provider) EnsureZoneExists(domain string, metadata map[string]string) error {
	if _, ok := r.getZoneByDomain(domain); ok {
		return nil
	}
	if r.delegationSet != nil {
		printer.Printf("Adding zone for %s to route 53 account with delegationSet %s\n", domain, *r.delegationSet)
	} else {
		printer.Printf("Adding zone for %s to route 53 account\n", domain)
	}
	in := &r53.CreateHostedZoneInput{
		Name:            &domain,
		DelegationSetId: r.delegationSet,
		CallerReference: aws.String(strconv.FormatInt(time.Now().UnixNano(), 10)),
	}

	var z *r53.CreateHostedZoneOutput
	var err error
	withRetry(func() error {
		z, err = r.client.CreateHostedZone(context.Background(), in)
		return err
	})
	if err != nil {
		return err
	}
	r.zonesMu.Lock()
	defer r.zonesMu.Unlock()
	r.addZoneToCacheLocked(*z.HostedZone)
	return nil
}

// changeBatcher takes a set of r53Types.Changes and turns them into a series of
// batches that meet the limits of the ChangeResourceRecordSets API.
//
// See also: https://docs.aws.amazon.com/Route53/latest/DeveloperGuide/DNSLimitations.html#limits-api-requests-changeresourcerecordsets
type changeBatcher struct {
	changes []r53Types.Change

	maxSize  int // Max records per request.
	maxChars int // Max record value characters per request.

	start, end int   // Cursors into changes.
	err        error // Populated by Next.
}

// newChangeBatcher returns a new changeBatcher.
func newChangeBatcher(changes []r53Types.Change) *changeBatcher {
	return &changeBatcher{
		changes:  changes,
		maxSize:  1000,  // "A request cannot contain more than 1,000 ResourceRecord elements."
		maxChars: 32000, // "The sum of the number of characters (including spaces) in all Value elements in a request cannot exceed 32,000 characters."
	}
}

// Next returns true if there is another batch of Changes.
// It returns false if there are no more batches or an error occurred.
func (b *changeBatcher) Next() bool {
	if b.end >= len(b.changes) || b.err != nil {
		return false
	}

	start, end := b.end, b.end
	var (
		reqSize  int
		reqChars int
	)
	for end < len(b.changes) {
		c := &b.changes[end]

		// Check that we won't exceed 1000 ResourceRecords in the request.
		if c.ResourceRecordSet == nil {
			end++
			continue
		}

		rrsetSize := len(c.ResourceRecordSet.ResourceRecords)
		if c.Action == r53Types.ChangeActionUpsert {
			// "When the value of the Action element is UPSERT, each ResourceRecord element is counted twice."
			rrsetSize *= 2
		}
		if newReqSize := reqSize + rrsetSize; newReqSize > b.maxSize {
			break
		} else {
			reqSize = newReqSize
		}

		// Check that we won't exceed 32000 Value characters in the request.
		var rrsetChars int
		for _, rr := range c.ResourceRecordSet.ResourceRecords {
			rrsetChars += utf8.RuneCountInString(aws.ToString(rr.Value))
		}
		if c.Action == r53Types.ChangeActionUpsert {
			// "When the value of the Action element is UPSERT, each character in a Value element is counted twice."
			rrsetChars *= 2
		}
		if newReqChars := reqChars + rrsetChars; newReqChars > b.maxChars {
			break
		} else {
			reqChars = newReqChars
		}

		end++
	}

	if start == end {
		b.err = errors.New("could not create ChangeResourceRecordSets request within AWS API limits")
		return false
	}

	b.start = start
	b.end = end

	return true
}

// Batch returns the current batch. It should only be called
// after Next returns true.
func (b *changeBatcher) Batch() (start, end int) {
	return b.start, b.end
}

// Err returns the error encountered during the previous call to Next.
func (b *changeBatcher) Err() error {
	return b.err
}
