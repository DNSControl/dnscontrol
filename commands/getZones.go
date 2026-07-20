package commands

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/DNSControl/dnscontrol/v4/models"
	"github.com/DNSControl/dnscontrol/v4/pkg/credsfile"
	"github.com/DNSControl/dnscontrol/v4/pkg/domaintags"
	"github.com/DNSControl/dnscontrol/v4/pkg/prettyzone"
	"github.com/DNSControl/dnscontrol/v4/pkg/providers"
	"github.com/DNSControl/dnscontrol/v4/pkg/rtypecontrol"

	"github.com/urfave/cli/v3"
)

var _ = cmd(catUtils, func() *cli.Command {
	var args GetZoneArgs
	return &cli.Command{
		Name:    "get-zones",
		Aliases: []string{"get-zone"},
		Usage:   "gets a zone from a provider (stand-alone)",
		Action: func(ctx context.Context, c *cli.Command) error {
			if c.NArg() < 2 {
				return cli.Exit("Arguments should be: credskey zone(s) (Ex: my_cloudflare example.com)", 1)
			}
			args.CredName = c.Args().Get(0)
			if c.NArg() == 2 || c.Args().Get(1) == "-" {
				args.ProviderName = ""
				if c.Args().Get(1) == "-" {
					args.ZoneNames = c.Args().Slice()[2:]
				} else {
					args.ZoneNames = c.Args().Slice()[1:]
				}
			} else {
				arg1 := c.Args().Get(1)
				if _, ok := providers.DNSProviderTypes[arg1]; ok {
					// Deprecated form: credkey provider zone [...]
					args.ProviderName = arg1
					args.ZoneNames = c.Args().Slice()[2:]
					fmt.Fprintf(os.Stderr, "WARNING: The provider name argument is deprecated. Please use \"dnscontrol get-zones %s %s\" instead. See %q\n",
						args.CredName, strings.Join(args.ZoneNames, " "),
						"https://docs.dnscontrol.org/commands/get-zones",
					)
				} else {
					// New form with multiple zones: credkey zone1 zone2 [...]
					args.ProviderName = ""
					args.ZoneNames = c.Args().Slice()[1:]
				}
			}

			return exit(GetZone(args))
		},
		Flags:     args.flags(),
		UsageText: "dnscontrol get-zones [command options] credkey zone [...]",
		Description: `Download a zone from a provider.  This is a stand-alone utility.

ARGUMENTS:
   credkey:  The name used in creds.json
   zone:     One or more zones (domains) to download; or "all".

EXAMPLES:
   dnscontrol get-zones my_route53 example.com
   dnscontrol get-zones my_gandi example.com other.com
   dnscontrol get-zones my_cloudflare all
   dnscontrol get-zones --format=tsv my_bind example.com
   dnscontrol get-zones --format=djs --out=draft.js my_gcloud example.com

Documentation: https://docs.dnscontrol.org/commands/get-zones`,
	}
}())

// check-creds foo bar
// is the same as
// get-zones --format=nameonly foo bar all.
var _ = cmd(catUtils, func() *cli.Command {
	var args GetZoneArgs
	return &cli.Command{
		Name:  "check-creds",
		Usage: "Do a small operation to verify credentials (stand-alone)",
		Action: func(ctx context.Context, c *cli.Command) error {
			var arg0, arg1 string
			// This takes one or two command-line args.
			// Starting in v3.16: Using it with 2 args will generate a warning.
			// Starting in v4.0: Using it with 2 args might be an error.
			if c.NArg() == 1 {
				arg0 = c.Args().Get(0)
				arg1 = ""
			} else if c.NArg() == 2 {
				arg0 = c.Args().Get(0)
				arg1 = c.Args().Get(1)
			} else {
				return cli.Exit("Arguments should be: credskey [providername] (Ex: r53 ROUTE53)", 1)
			}
			args.CredName = arg0
			args.ProviderName = arg1
			args.ZoneNames = []string{"all"}
			args.OutputFormat = "nameonly"
			return exit(GetZone(args))
		},
		Flags:     args.flags(),
		UsageText: "dnscontrol check-creds [command options] credkey provider",
		Description: `Do a trivia operation to verify credentials.  This is a stand-alone utility.

If successful, a list of zones will be output. If not, hopefully you
see verbose error messages.

ARGUMENTS:
   credkey:  The name used in creds.json (first parameter to NewDnsProvider() in dnsconfig.js)
   provider: The name of the provider (second parameter to NewDnsProvider() in dnsconfig.js)

EXAMPLES:
   dnscontrol check-creds myr53 ROUTE53      # Pre v3.16, or pre-v4.0 for backwards-compatibility
   dnscontrol check-creds myr53
   dnscontrol check-creds --out=/dev/null myr53 && echo Success

Documentation: https://docs.dnscontrol.org/commands/check-creds`,
	}
}())

// GetZoneArgs args required for the create-domain subcommand.
type GetZoneArgs struct {
	GetCredentialsArgs          // Args related to creds.json
	CredName           string   // key in creds.json
	ProviderName       string   // provider type: BIND, GANDI_V5, etc or "-"  (NB(tlim): In 4.0, this field goes away.)
	ZoneNames          []string // The zones to get
	OutputFormat       string   // Output format
	OutputFile         string   // Filename to send output ("" means stdout)
	DefaultTTL         int      // default TTL for providers where it is unknown
}

func (args *GetZoneArgs) flags() []cli.Flag {
	flags := args.GetCredentialsArgs.flags()
	flags = append(flags, &cli.StringFlag{
		Name:        "format",
		Destination: &args.OutputFormat,
		Value:       "zone",
		Usage:       `Output format: js djs zone tsv nameonly`,
	})
	flags = append(flags, &cli.StringFlag{
		Name:        "out",
		Destination: &args.OutputFile,
		Usage:       `Instead of stdout, write to this file`,
	})
	flags = append(flags, &cli.IntFlag{
		Name:        "ttl",
		Destination: &args.DefaultTTL,
		Usage:       `Default TTL (0 picks the most common TTL)`,
	})
	return flags
}

// GetZone contains all data/flags needed to run get-zones, independently of CLI.
func GetZone(args GetZoneArgs) error {
	var providerConfigs map[string]map[string]string
	var err error

	// Read it in:
	providerConfigs, err = credsfile.LoadProviderConfigs(args.CredsFile)
	if err != nil {
		return fmt.Errorf("failed GetZone LoadProviderConfigs(%q): %w", args.CredsFile, err)
	}
	provider, err := providers.CreateDNSProvider(args.ProviderName, providerConfigs[args.CredName], nil)
	if err != nil {
		return fmt.Errorf("failed GetZone CDP: %w", err)
	}

	// Get the actual provider type name from creds.json or args
	providerType := args.ProviderName
	if providerType == "" || providerType == "-" {
		providerType = providerConfigs[args.CredName][pproviderTypeFieldName]
	}

	// decide which zones we need to convert
	zones := args.ZoneNames
	if len(args.ZoneNames) == 1 && args.ZoneNames[0] == "all" {
		lister, ok := provider.(providers.ZoneLister)
		if !ok {
			return fmt.Errorf("provider type %s:%s cannot list zones to use the 'all' feature", args.CredName, args.ProviderName)
		}
		zones, err = lister.ListZones()
		if err != nil {
			return fmt.Errorf("failed GetZone LZ: %w", err)
		}
	}

	// first open output stream and print initial header (if applicable)
	w := os.Stdout
	if args.OutputFile != "" {
		w, err = os.Create(args.OutputFile)
	}
	if err != nil {
		return fmt.Errorf("failed GetZone Create(%q): %w", args.OutputFile, err)
	}
	defer w.Close()

	if args.OutputFormat == "nameonly" {
		for _, zone := range zones {
			fmt.Fprintln(w, zone)
		}
		return nil
	}

	// fetch all of the records
	zoneRecs := make([]models.Records, len(zones))
	for i, zone := range zones {
		ff := domaintags.MakeDomainNameVarieties(zone)
		recs, err := provider.GetZoneRecords(
			&models.DomainConfig{
				Name: ff.NameASCII,
				Metadata: map[string]string{
					models.DomainUniqueName:  ff.UniqueName,
					models.DomainNameRaw:     ff.NameRaw,
					models.DomainNameUnicode: ff.NameUnicode,
				},
			})
		if err != nil {
			return fmt.Errorf("failed GetZone gzr: %w", err)
		}
		rtypecontrol.FixLegacyRecords(&recs) // Call this after GetZoneRecords() to fix providers that haven't been updated for RecordConfigV2.
		zoneRecs[i] = recs
	}

	// Write the heading:

	dspVariableName := "DSP_" + strings.ToUpper(args.CredName)

	if args.OutputFormat == "js" || args.OutputFormat == "djs" {
		fmt.Fprintf(w, "// generated by get-zones. This is 'a decent first draft' and requires editing.\n")
		fmt.Fprintf(w, "\n")
		if args.ProviderName == "-" {
			fmt.Fprintf(w, `var %s = NewDnsProvider("%s");`+"\n",
				dspVariableName, args.CredName)
		} else {
			fmt.Fprintf(w, `var %s = NewDnsProvider("%s", "%s");`+"\n",
				dspVariableName, args.CredName, args.ProviderName)
		}
		fmt.Fprintf(w, `var REG_CHANGEME = NewRegistrar("none");`+"\n\n")
	}

	// print each zone
	for i, recs := range zoneRecs {
		zoneName := zones[i]

		z := prettyzone.PrettySort(recs, zoneName, 0, nil)
		switch args.OutputFormat {
		case "zone":
			fmt.Fprintf(w, "$ORIGIN %s.\n", zoneName)
			if err := prettyzone.WriteZoneFileRC(w, z.Records, zoneName, uint32(args.DefaultTTL), nil); err != nil {
				return err
			}
			fmt.Fprintln(w)

		case "js", "djs":
			sep := ",\n\t" // Commas at EOL
			if args.OutputFormat == "djs" {
				sep = "\n\t, " // Funky comma mode
			}

			fmt.Fprintf(w, `D("%s", REG_CHANGEME%s`, zoneName, sep)
			var o []string

			// If the provider returns no nameservers, emit {no_ns: "true"}
			// so that preview/push won't skip the domain.
			if ns, nsErr := provider.GetNameservers(zoneName); nsErr == nil && len(ns) == 0 {
				o = append(o, `{no_ns: "true"}`)
			}

			o = append(o, fmt.Sprintf("DnsProvider(%s)", dspVariableName))
			defaultTTL := uint32(args.DefaultTTL)
			if defaultTTL == 0 {
				defaultTTL = prettyzone.MostCommonTTL(recs)
			}
			// If provider has a registered default TTL and no records exist or MostCommonTTL returns 0,
			// use the provider's default TTL
			if defaultTTL == 0 || defaultTTL == models.DefaultTTL {
				if providerDefaultTTL := providers.GetDefaultTTL(providerType); providerDefaultTTL > 0 {
					defaultTTL = providerDefaultTTL
				}
			}
			if defaultTTL != models.DefaultTTL && defaultTTL != 0 {
				o = append(o, fmt.Sprintf("DefaultTTL(%d)", defaultTTL))
			}

			// Check if any records have comments or tags, and add management flags if so
			hasComments := false
			hasTags := false
			for _, rec := range recs {
				if rec.Metadata["cloudflare_comment"] != "" {
					hasComments = true
				}
				if rec.Metadata["cloudflare_tags"] != "" {
					hasTags = true
				}
				if hasComments && hasTags {
					break
				}
			}
			if hasComments {
				o = append(o, "CF_MANAGE_COMMENTS // opt into comments syncing")
			}
			if hasTags {
				o = append(o, "CF_MANAGE_TAGS // opt into tags syncing")
			}

			for _, rec := range recs {
				if (rec.Type == "CNAME") && (rec.Name == "@") {
					o = append(o, "// NOTE: CNAME at apex may require manual editing.")
				}
				o = append(o, formatDsl(rec, defaultTTL))
			}
			out := strings.Join(o, sep)

			// Joining with a comma between each item works great but
			// makes comments look terrible.  Here we clean them up
			// after the fact.
			if args.OutputFormat == "djs" {
				out = strings.ReplaceAll(out, "\n\t, //", "\n\t//, ") // Fix comments
				out = strings.ReplaceAll(out,
					"//,  NOTE: CNAME at apex may require manual editing.",
					"// NOTE: CNAME at apex may require manual editing.",
				)
				fmt.Fprint(w, out)
				fmt.Fprint(w, "\n)\n\n")
			} else {
				out = out + ","
				out = strings.ReplaceAll(out,
					"// NOTE: CNAME at apex may require manual editing.,",
					"// NOTE: CNAME at apex may require manual editing.",
				)
				fmt.Fprint(w, out)
				fmt.Fprint(w, "\n);\n\n")
			}

		case "tsv":
			for _, rec := range recs {
				providerMeta := ""
				if cp, ok := rec.Metadata["cloudflare_proxy"]; ok {
					if cp == "true" {
						providerMeta += ",cloudflare_proxy=true"
					}
				}
				if cf, ok := rec.Metadata["cloudflare_cname_flatten"]; ok {
					if cf == "on" {
						providerMeta += ",cloudflare_cname_flatten=on"
					}
				}
				if comment := rec.Metadata["cloudflare_comment"]; comment != "" {
					providerMeta += ",cloudflare_comment=" + comment
				}
				if tags := rec.Metadata["cloudflare_tags"]; tags != "" {
					providerMeta += ",cloudflare_tags=" + tags
				}
				// HEDNS metadata
				if dyn, ok := rec.Metadata["hedns_dynamic"]; ok && dyn == "on" {
					providerMeta += ",hedns_dynamic=on"
					if key := rec.Metadata["hedns_ddns_key"]; key != "" {
						providerMeta += ",hedns_ddns_key=" + key
					}
				}
				if providerMeta != "" {
					providerMeta = "\t" + providerMeta[1:] // Remove leading comma, add tab
				}

				ty := rec.Type
				if rec.Type == "UNKNOWN" {
					ty = rec.UnknownTypeName
				}
				fmt.Fprintf(w, "%s\t%s\t%d\tIN\t%s\t%s%s\n",
					rec.NameFQDN, rec.Name, rec.TTL, ty, rec.GetTargetCombinedFunc(nil), providerMeta)
			}

		default:
			return fmt.Errorf("format %q unknown", args.OutputFormat)
		}
	}
	return nil
}

// jsonQuoted returns a properly escaped JSON string (without quotes).
func jsonQuoted(i string) string {
	// https://stackoverflow.com/questions/51691901
	b, err := json.Marshal(i)
	if err != nil {
		panic(err)
	}
	return string(b)
}

func formatDsl(rec *models.RecordConfig, defaultTTL uint32) string {
	target := rec.GetTargetCombined()

	ttl := uint32(0)
	ttlop := ""
	if rec.TTL != defaultTTL && rec.TTL != 0 {
		ttl = rec.TTL
		ttlop = fmt.Sprintf(", TTL(%d)", ttl)
	}

	cfproxy := ""
	if cp, ok := rec.Metadata["cloudflare_proxy"]; ok {
		if cp == "true" {
			cfproxy = ", CF_PROXY_ON"
		}
	}

	cfflatten := ""
	if cf, ok := rec.Metadata["cloudflare_cname_flatten"]; ok {
		if cf == "on" {
			cfflatten = ", CF_CNAME_FLATTEN_ON"
		}
	}

	cfcomment := ""
	if comment := rec.Metadata["cloudflare_comment"]; comment != "" {
		cfcomment = fmt.Sprintf(", CF_COMMENT(%s)", jsonQuoted(comment))
	}

	cftags := ""
	if tags := rec.Metadata["cloudflare_tags"]; tags != "" {
		// Convert comma-separated tags to CF_TAGS("tag1", "tag2", ...)
		tagList := strings.Split(tags, ",")
		quotedTags := make([]string, len(tagList))
		for i, tag := range tagList {
			quotedTags[i] = fmt.Sprintf("%q", tag)
		}
		cftags = ", CF_TAGS(" + strings.Join(quotedTags, ", ") + ")"
	}

	// Mikrotik RouterOS metadata (match_subdomain, regexp, address_list, comment)
	mtmeta := ""
	if rec.Metadata != nil {
		var mtParts []string
		if v := rec.Metadata["match_subdomain"]; v == "true" {
			mtParts = append(mtParts, `match_subdomain: "true"`)
		}
		if v := rec.Metadata["regexp"]; v != "" {
			mtParts = append(mtParts, fmt.Sprintf("regexp: %s", jsonQuoted(v)))
		}
		if v := rec.Metadata["address_list"]; v != "" {
			mtParts = append(mtParts, fmt.Sprintf("address_list: %s", jsonQuoted(v)))
		}
		if v := rec.Metadata["comment"]; v != "" {
			mtParts = append(mtParts, fmt.Sprintf("comment: %s", jsonQuoted(v)))
		}
		if len(mtParts) > 0 {
			mtmeta = ", {" + strings.Join(mtParts, ", ") + "}"
		}
	}

	// HEDNS metadata
	hednsDynamic := ""
	if dyn, ok := rec.Metadata["hedns_dynamic"]; ok && dyn == "on" {
		hednsDynamic = ", HEDNS_DYNAMIC_ON"
		if key := rec.Metadata["hedns_ddns_key"]; key != "" {
			// HEDNS_DDNS_KEY implies dynamic, so use it instead.
			hednsDynamic = fmt.Sprintf(", HEDNS_DDNS_KEY(%s)", jsonQuoted(key))
		}
	}

	switch rec.Type { // #rtype_variations
	case "CAA":
		return makeCaa(rec, ttlop)
	case "DS":
		target = fmt.Sprintf(`%d, %d, %d, "%s"`, rec.DsKeyTag, rec.DsAlgorithm, rec.DsDigestType, rec.DsDigest)
	case "DNSKEY":
		target = fmt.Sprintf(`%d, %d, %d, "%s"`, rec.DnskeyFlags, rec.DnskeyProtocol, rec.DnskeyAlgorithm, rec.DnskeyPublicKey)
	case "MX":
		target = fmt.Sprintf(`%d, "%s"`, rec.MxPreference, rec.GetTargetField())
	case "NAPTR":
		target = fmt.Sprintf(`%d, %d, %s, %s, %s, %s`,
			rec.NaptrOrder,                   // 1
			rec.NaptrPreference,              // 10
			jsonQuoted(rec.NaptrFlags),       // U
			jsonQuoted(rec.NaptrService),     // E2U+sip
			jsonQuoted(rec.NaptrRegexp),      // regex
			jsonQuoted(rec.GetTargetField()), // .
		)
	case "SMIMEA":
		target = fmt.Sprintf(`%d, %d, %d, "%s"`, rec.SmimeaUsage, rec.SmimeaSelector, rec.SmimeaMatchingType, rec.GetTargetField())
	case "SSHFP":
		target = fmt.Sprintf(`%d, %d, "%s"`, rec.SshfpAlgorithm, rec.SshfpFingerprint, rec.GetTargetField())
	case "SOA":
		rec.Type = "//SOA"
		target = fmt.Sprintf(`"%s", "%s", %d, %d, %d, %d`, rec.GetTargetField(), rec.SoaMbox, rec.SoaRefresh, rec.SoaRetry, rec.SoaExpire, rec.SoaMinttl)
	case "SRV":
		target = fmt.Sprintf(`%d, %d, %d, "%s"`, rec.SrvPriority, rec.SrvWeight, rec.SrvPort, rec.GetTargetField())
	case "SVCB", "HTTPS":
		target = fmt.Sprintf(`%d, "%s", "%s"`, rec.SvcPriority, rec.GetTargetField(), rec.SvcParams)
	case "TLSA":
		target = fmt.Sprintf(`%d, %d, %d, "%s"`, rec.TlsaUsage, rec.TlsaSelector, rec.TlsaMatchingType, rec.GetTargetField())
	case "TXT":
		target = jsonQuoted(rec.GetTargetTXTJoined())
		// TODO(tlim): If this is an SPF record, generate a SPF_BUILDER().
	case "LUA":
		target = fmt.Sprintf("%q, %s", rec.LuaRType, jsonQuoted(rec.GetTargetTXTJoined()))
	case "NS":
		// NS records at the apex should be NAMESERVER() records.
		// DnsControl uses the API to get this info. NAMESERVER() is just
		// to override that when needed.
		if rec.Name == "@" {
			return fmt.Sprintf(`//NAMESERVER("%s")`, target)
		}
		target = `"` + target + `"`
	case "MIKROTIK_FWD":
		target = `"` + target + `"`
	case "MIKROTIK_NXDOMAIN":
		// NXDOMAIN has no target — emit only name + optional metadata + TTL
		return fmt.Sprintf(`MIKROTIK_NXDOMAIN("%s"%s%s)`, rec.Name, mtmeta, ttlop)
	case "MIKROTIK_FORWARDER":
		// Forwarder: target is dns-servers, metadata has doh_servers/verify_doh_cert
		target = `"` + target + `"`
		if rec.Metadata != nil {
			var fwdParts []string
			if v := rec.Metadata["doh_servers"]; v != "" {
				fwdParts = append(fwdParts, fmt.Sprintf("doh_servers: %s", jsonQuoted(v)))
			}
			if v := rec.Metadata["verify_doh_cert"]; v == "true" {
				fwdParts = append(fwdParts, `verify_doh_cert: "true"`)
			}
			if len(fwdParts) > 0 {
				target += ", {" + strings.Join(fwdParts, ", ") + "}"
			}
		}
	case "R53_ALIAS":
		return makeR53alias(rec, ttl)
	case "UNKNOWN":
		return makeUknown(rec, ttl)
	default:
		target = `"` + target + `"`
	}

	return fmt.Sprintf(`%s("%s", %s%s%s%s%s%s%s%s)`, rec.Type, rec.Name, target, cfproxy, cfflatten, cfcomment, cftags, mtmeta, hednsDynamic, ttlop)
}

func makeCaa(rec *models.RecordConfig, ttlop string) string {
	var target string
	if rec.CaaFlag == 128 {
		target = fmt.Sprintf(`"%s", "%s", CAA_CRITICAL`, rec.CaaTag, rec.GetTargetField())
	} else {
		target = fmt.Sprintf(`"%s", "%s"`, rec.CaaTag, rec.GetTargetField())
	}
	return fmt.Sprintf(`%s("%s", %s%s)`, rec.Type, rec.Name, target, ttlop)

	// TODO(tlim): Generate a CAA_BUILDER() instead?
}

func makeR53alias(rec *models.RecordConfig, ttl uint32) string {
	items := []string{
		`"` + rec.Name + `"`,
		`"` + rec.R53Alias["type"] + `"`,
		`"` + rec.GetTargetField() + `"`,
	}
	if z, ok := rec.R53Alias["zone_id"]; ok {
		items = append(items, `R53_ZONE("`+z+`")`)
	}
	if e, ok := rec.R53Alias["evaluate_target_health"]; ok && e == "true" {
		items = append(items, "R53_EVALUATE_TARGET_HEALTH(true)")
	}
	if ttl != 0 {
		items = append(items, fmt.Sprintf("TTL(%d)", ttl))
	}
	return rec.Type + "(" + strings.Join(items, ", ") + ")"
}

func makeUknown(rc *models.RecordConfig, ttl uint32) string {
	return fmt.Sprintf(`// %s("%s", TTL(%d))`, rc.UnknownTypeName, rc.GetTargetField(), ttl)
}
