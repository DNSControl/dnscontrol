package commands

import (
	"cmp"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"regexp"
	"runtime"
	"slices"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
	"unsafe"

	"github.com/DNSControl/dnscontrol/v4/models"
	"github.com/DNSControl/dnscontrol/v4/pkg/bindserial"
	"github.com/DNSControl/dnscontrol/v4/pkg/credsfile"
	"github.com/DNSControl/dnscontrol/v4/pkg/domaintags"
	"github.com/DNSControl/dnscontrol/v4/pkg/nameservers"
	"github.com/DNSControl/dnscontrol/v4/pkg/normalize"
	"github.com/DNSControl/dnscontrol/v4/pkg/notifications"
	"github.com/DNSControl/dnscontrol/v4/pkg/printer"
	"github.com/DNSControl/dnscontrol/v4/pkg/providers"
	"github.com/DNSControl/dnscontrol/v4/pkg/rfc4183"
	"github.com/DNSControl/dnscontrol/v4/pkg/zonerecs"
	"github.com/dustin/go-humanize"
	"github.com/nozzle/throttler"
	"github.com/urfave/cli/v3"
	"golang.org/x/net/idna"
)

// CmdZoneCache is a cache of zone lists for providers.  This is used to
// optimize the "populate" phase of preview/push, so that we don't have to make
// multiple calls to the provider to get the list of zones.
type CmdZoneCache struct {
	cache map[string]*[]string
	sync.Mutex
}

var _ = cmd(catMain, func() *cli.Command {
	var args PPreviewArgs
	return &cli.Command{
		Name:  "preview",
		Usage: "read live configuration and identify changes to be made, without applying them",
		Action: func(ctx context.Context, c *cli.Command) error {
			return exit(PPreview(args))
		},
		Flags: args.flags(),
	}
}())

// PPreviewArgs contains all data/flags needed to run preview, independently of CLI.
type PPreviewArgs struct {
	GetDNSConfigArgs
	GetCredentialsArgs
	FilterArgs
	Notify            bool
	WarnChanges       bool
	ConcurMode        string
	ConcurMax         int // Maximum number of concurrent connections
	NoPopulate        bool
	PopulateOnPreview bool
	Report            string
	Full              bool
}

// ReportItem is a record of corrections for a particular domain/provider/registrar.
// type ReportItem struct {
//	Domain      string `json:"domain"`
//	Corrections int    `json:"corrections"`
//	Provider    string `json:"provider,omitempty"`
//	Registrar   string `json:"registrar,omitempty"`
//}

func (args *PPreviewArgs) flags() []cli.Flag {
	flags := args.GetDNSConfigArgs.flags()
	flags = append(flags, args.GetCredentialsArgs.flags()...)
	flags = append(flags, args.FilterArgs.flags()...)
	flags = append(flags, &cli.BoolFlag{
		Name:        "notify",
		Destination: &args.Notify,
		Usage:       `set to true to send notifications to configured destinations`,
	})
	flags = append(flags, &cli.BoolFlag{
		Name:        "expect-no-changes",
		Destination: &args.WarnChanges,
		Usage:       `set to true for non-zero return code if there are changes`,
	})
	flags = append(flags, &cli.StringFlag{
		Name:        "cmode",
		Destination: &args.ConcurMode,
		Value:       "concurrent",
		Usage:       `Which providers to run concurrently: concurrent, none, all`,
		Action: func(ctx context.Context, c *cli.Command, s string) error {
			if !slices.Contains([]string{"concurrent", "none", "all"}, s) {
				fmt.Printf("%q is not a valid option for --cmode.  Values are: concurrent, none, all\n", s)
				os.Exit(1)
			}
			return nil
		},
	})
	flags = append(flags, &cli.IntFlag{
		Name:        "cmax",
		Destination: &args.ConcurMax,
		Value:       100,
		Usage:       `Maximum number of concurrent connections`,
		Action: func(ctx context.Context, c *cli.Command, v int) error {
			if v < 1 {
				fmt.Printf("%d is not a valid value for --cmax.  Values must be 1 or greater\n", v)
				os.Exit(1)
			}
			return nil
		},
	})
	flags = append(flags, &cli.BoolFlag{
		Name:        "no-populate",
		Destination: &args.NoPopulate,
		Usage:       `Do not auto-create zones at the provider`,
	})
	flags = append(flags, &cli.BoolFlag{
		Name:        "populate-on-preview",
		Destination: &args.PopulateOnPreview,
		Value:       false,
		Usage:       `Auto-create zones on preview`,
	})
	flags = append(flags, &cli.BoolFlag{
		Name:        "full",
		Destination: &args.Full,
		Usage:       `Add headings, providers names, notifications of no changes, etc`,
	})
	flags = append(flags, &cli.IntFlag{
		Name:   "reportmax",
		Hidden: false,
		Usage:  `Limit the IGNORE/NO_PURGE report to this many lines (Expermental. Will change in the future.)`,
		Action: func(ctx context.Context, c *cli.Command, maxreport int) error {
			printer.MaxReport = maxreport
			return nil
		},
	})
	flags = append(flags, &cli.Int64Flag{
		Name:        "bindserial",
		Destination: &bindserial.ForcedValue,
		Usage:       `Force BIND serial numbers to this value (for reproducibility)`,
	})
	flags = append(flags, &cli.StringFlag{
		Name:        "report",
		Destination: &args.Report,
		Usage:       `Generate a machine-parseable report of corrections.`,
	})
	return flags
}

var _ = cmd(catMain, func() *cli.Command {
	var args PPushArgs
	return &cli.Command{
		Name:  "push",
		Usage: "identify changes to be made, and perform them",
		Action: func(ctx context.Context, c *cli.Command) error {
			return exit(PPush(args))
		},
		Flags: args.flags(),
	}
}())

// PPushArgs contains all data/flags needed to run push, independently of CLI.
type PPushArgs struct {
	PPreviewArgs
	Interactive bool
}

func (args *PPushArgs) flags() []cli.Flag {
	flags := args.PPreviewArgs.flags()
	flags = append(flags, &cli.BoolFlag{
		Name:        "i",
		Destination: &args.Interactive,
		Usage:       "Interactive. Confirm or Exclude each correction before they run",
	})
	return flags
}

// PPreview implements the preview subcommand.
func PPreview(args PPreviewArgs) error {
	return prun(args, false, false, printer.DefaultPrinter, args.Report)
}

// PPush implements the push subcommand.
func PPush(args PPushArgs) error {
	return prun(args.PPreviewArgs, true, args.Interactive, printer.DefaultPrinter, args.Report)
}

var pobsoleteDiff2FlagUsed = false

// run is the main routine common to preview/push.
func prun(args PPreviewArgs, push bool, interactive bool, out printer.CLI, report string) error {
	// This is a hack until we have the new printer replacement.
	printer.SkinnyReport = !args.Full
	fullMode := args.Full

	if pobsoleteDiff2FlagUsed {
		printer.Println("WARNING: Please remove obsolete --diff2 flag. This will be an error in v5 or later. See https://github.com/DNSControl/dnscontrol/issues/2262")
	}

	out.PrintfIf(fullMode, "Reading dnsconfig.js or equiv.\n")
	cfg, err := GetDNSConfig(args.GetDNSConfigArgs)
	if err != nil {
		return err
	}

	out.PrintfIf(fullMode, "Reading creds: %q\n", args.CredsFile)
	providerConfigs, err := credsfile.LoadProviderConfigs(args.CredsFile)
	if err != nil {
		return err
	}

	var notify = args.Notify

	// We want to notify if args.Notify OR notify_on_*
	if notifications, ok := providerConfigs["notifications"]; ok && notifications != nil {
		if push {
			if notifyOnPush, ok := notifications["notify_on_push"]; ok {
				if b, _ := strconv.ParseBool(notifyOnPush); b {
					notify = true
				}
			}
		} else {
			if notifyOnPreview, ok := notifications["notify_on_preview"]; ok {
				if b, _ := strconv.ParseBool(notifyOnPreview); b {
					notify = true
				}
			}
		}
	}
	if notify {
		out.PrintfIf(fullMode, "Notifications are enabled...\n")
	}

	out.PrintfIf(fullMode, "Creating an in-memory model of 'desired'...\n")
	notifier, err := PInitializeProviders(cfg, providerConfigs, notify)
	if err != nil {
		return err
	}

	out.PrintfIf(fullMode, "Normalizing and validating 'desired'..\n")
	errs := normalize.ValidateAndNormalizeConfig(cfg)
	if PrintValidationErrors(errs) {
		return errors.New("exiting due to validation errors")
	}

	zcache := NewCmdZoneCache()

	// Loop over all (or some) zones:
	zonesToProcess := whichZonesToProcess(cfg.Domains, args.Domains)
	zonesSerial, zonesConcurrent := splitConcurrent(zonesToProcess, args.ConcurMode)
	zonesConcurrent = optimizeOrder(zonesConcurrent)

	var totalCorrections int
	var reportItems []*ReportItem
	var anyErrors bool
	var concurrentErrors atomic.Bool

	// Populate the zones (if desired/needed/able):
	if !args.NoPopulate {
		out.PrintfIf(fullMode, "PHASE 1: CHECKING for missing zones\n")
		t := throttler.New(args.ConcurMax, len(zonesConcurrent))
		out.Printf("CONCURRENTLY checking for %d zone(s)\n", len(zonesConcurrent))
		for i, zone := range zonesConcurrent {
			out.PrintfIf(fullMode, "Concurrently checking for zone: %q\n", zone.UniqueName)
			go func(zone *models.DomainConfig) {
				start := time.Now()
				err := oneZonePopulate(zone, zcache)
				if err != nil {
					concurrentErrors.Store(true)
				}
				out.Debugf("...DONE: %q (%.1fs)\n", zone.Name, time.Since(start).Seconds())
				t.Done(err)
			}(zone)
			// Delay the last call to t.Throttle() until the serial processing is done.
			if i != ultimate(zonesConcurrent) {
				errorCount := t.Throttle()
				if errorCount > 0 {
					anyErrors = true
				}
			}
		}

		out.Printf("SERIALLY checking for %d zone(s)\n", len(zonesSerial))
		for _, zone := range zonesSerial {
			out.Printf("Serially checking for zone: %q\n", zone.UniqueName)
			if err := oneZonePopulate(zone, zcache); err != nil {
				anyErrors = true
			}
		}

		if len(zonesConcurrent) > 0 {
			if printer.DefaultPrinter.Verbose {
				out.PrintfIf(true, "Waiting for concurrent checking(s) to complete...\n")
			} else {
				out.PrintfIf(true, "Waiting for concurrent checking(s) to complete...")
			}
			errorCount := t.Throttle()
			if errorCount > 0 {
				anyErrors = true
			}
			out.PrintfIf(true, "DONE\n")
		}

		for _, zone := range zonesToProcess {
			started := false // Do not emit noise when no provider has corrections.
			providersToProcess := whichProvidersToProcess(zone.DNSProviderInstances, args.Providers)
			for _, provider := range zone.DNSProviderInstances {
				corrections := zone.GetPopulateCorrections(provider.Name)
				if len(corrections) == 0 {
					continue // Do not emit noise when zone exists
				}
				if !started {
					out.StartDomain(zone)
					started = true
				}
				skip := skipProvider(provider.Name, providersToProcess)
				out.StartDNSProvider(provider.Name, skip)
				if !skip {
					totalCorrections += len(corrections)
					out.EndProvider2(provider.Name, len(corrections))
					reportItems = append(reportItems, genReportItem(zone.Name, corrections, provider.Name, ""))
					anyErrors = cmp.Or(anyErrors, pprintOrRunCorrections(zone.Name, provider.Name, corrections, out, push || args.PopulateOnPreview, interactive, notifier, report))
				}
			}
		}
	}

	out.PrintfIf(fullMode, "PHASE 2: GATHERING data\n")
	t := throttler.New(args.ConcurMax, len(zonesConcurrent))
	out.Printf("CONCURRENTLY gathering records of %d zone(s)\n", len(zonesConcurrent))
	for i, zone := range zonesConcurrent {
		out.PrintfIf(fullMode, "Concurrently gathering: %q\n", zone.UniqueName)
		go func(zone *models.DomainConfig, args PPreviewArgs, zcache *CmdZoneCache) {
			start := time.Now()
			err := oneZone(zone, args)
			if err != nil {
				concurrentErrors.Store(true)
			}
			out.Debugf("...DONE: %q (%.1fs)\n", zone.Name, time.Since(start).Seconds())
			t.Done(err)
		}(zone, args, zcache)
		// Delay the last call to t.Throttle() until the serial processing is done.
		if i != ultimate(zonesConcurrent) {
			errorCount := t.Throttle()
			if errorCount > 0 {
				anyErrors = true
			}
		}
	}
	out.Printf("SERIALLY gathering records of %d zone(s)\n", len(zonesSerial))
	for _, zone := range zonesSerial {
		out.Printf("Serially Gathering: %q\n", zone.UniqueName)
		if err := oneZone(zone, args); err != nil {
			anyErrors = true
		}
	}

	if len(zonesConcurrent) > 0 {
		msg := "Waiting for concurrent gathering(s) to complete..."
		if printer.DefaultPrinter.Verbose {
			msg = "Waiting for concurrent gathering(s) to complete...\n"
		}
		out.PrintfIf(true, msg)
		errorCount := t.Throttle()
		if errorCount > 0 {
			anyErrors = true
		}
		out.PrintfIf(true, "DONE\n")
	}

	anyErrors = cmp.Or(anyErrors, concurrentErrors.Load())

	// Now we know what to do, print or do the tasks.
	out.PrintfIf(fullMode, "PHASE 3: CORRECTIONS\n")
	for _, zone := range zonesToProcess {
		out.StartDomain(zone)

		// Process DNS provider changes:
		providersToProcess := whichProvidersToProcess(zone.DNSProviderInstances, args.Providers)
		for _, provider := range zone.DNSProviderInstances {
			skip := skipProvider(provider.Name, providersToProcess)
			out.StartDNSProvider(provider.Name, skip)
			if !skip {
				corrections := zone.GetCorrections(provider.Name)
				numActions := zone.GetChangeCount(provider.Name)
				totalCorrections += numActions
				out.EndProvider2(provider.Name, numActions)
				reportItems = append(reportItems, genReportItem(zone.Name, corrections, provider.Name, ""))
				anyErrors = cmp.Or(anyErrors, pprintOrRunCorrections(zone.Name, provider.Name, corrections, out, push, interactive, notifier, report))
			}
		}

		// Process Registrar changes:
		skip := skipProvider(zone.RegistrarInstance.Name, providersToProcess)
		out.StartRegistrar(zone.RegistrarName, !skip)
		if skip {
			corrections := zone.GetCorrections(zone.RegistrarInstance.Name)
			numActions := zone.GetChangeCount(zone.RegistrarInstance.Name)
			out.EndProvider2(zone.RegistrarName, numActions)
			totalCorrections += numActions
			reportItems = append(reportItems, genReportItem(zone.Name, corrections, "", zone.RegistrarName))
			anyErrors = cmp.Or(anyErrors, pprintOrRunCorrections(zone.Name, zone.RegistrarInstance.Name, corrections, out, push, interactive, notifier, report))
		}
	}

	if os.Getenv("TEAMCITY_VERSION") != "" {
		fmt.Fprintf(os.Stderr, "##teamcity[buildStatus status='SUCCESS' text='%d corrections']", totalCorrections)
	}
	rfc4183.PrintWarning()
	out.PrintfIf(fullMode, "Inaccurate statistics: %s\n", stats(cfg))
	notifier.Done()
	out.Printf("Done. %d corrections.\n", totalCorrections)

	err = writeReport(report, reportItems)
	if err != nil {
		return errors.New("could not write report")
	}
	if anyErrors {
		return errors.New("completed with errors")
	}
	if totalCorrections != 0 && args.WarnChanges {
		return errors.New("there are pending changes")
	}
	return nil
}

// stats returns a JSON string with memory usage statistics.
// These stats are unofficial and subject to change without notice.
// "average_mem_per_record" is misleading because it includes all memory overhead.
func stats(cfg *models.DNSConfig) string {

	// https://www.datadoghq.com/blog/go-memory-metrics/
	// [T]he following expression accurately reflects the value the runtime attempts to maintain as the limit:
	// runtime.MemStats.Sys − runtime.MemStats.HeapReleased
	runtime.GC()
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	memoryInUse := m.Sys - m.HeapReleased

	numRecords := countRecords(cfg)
	memPerRecord := int64(float64(memoryInUse) / float64(max(1, numRecords)))
	memPerRecordStr := humanize.IBytes(uint64(memPerRecord)) + " bytes"

	statsInfo := struct {
		MemoryInUse    uint64 `json:"memory_in_use"`
		MemoryInUseStr string `json:"memory_in_use_str"`
		NumRecords     int    `json:"num_records"`
		NumZones       int    `json:"num_zones"`
		RCSize         int    `json:"rc_size"`
		Benchmark1     int64  `json:"benchmark1"`
		Benchmark1Str  string `json:"benchmark1str"`
	}{
		MemoryInUse:    memoryInUse,
		MemoryInUseStr: humanize.Bytes(memoryInUse),
		NumRecords:     numRecords,
		NumZones:       len(cfg.Domains),
		RCSize:         int(unsafe.Sizeof((models.RecordConfig{}))),
		Benchmark1:     memPerRecord,
		Benchmark1Str:  memPerRecordStr,
	}

	jsonBytes, err := json.Marshal(statsInfo)
	if err != nil {
		return fmt.Sprintf("error marshaling stats: %v", err)
	}
	return string(jsonBytes)
}

func countRecords(cfg *models.DNSConfig) int {
	total := 0
	for _, domain := range cfg.Domains {
		total += len(domain.Records)
	}
	return total
}

// whichZonesToProcess takes a list of DomainConfigs and a filter string and
// returns a list of DomainConfigs whose Domain.UniqueName matched the
// filter. The filter string is a comma-separated list of domain names. If the
// filter string is empty or "all", all domains are returned.
func whichZonesToProcess(domains []*models.DomainConfig, filter string) []*models.DomainConfig {
	fh := domaintags.CompilePermitList(filter)

	var picked []*models.DomainConfig
	for _, domain := range domains {
		if fh.Permitted(domain.GetUniqueName()) {
			picked = append(picked, domain)
		}
	}
	return picked
}

// splitConcurrent takes a list of DomainConfigs and returns two lists. The
// first list is the items that do NOT support concurrency.  The second is list
// the items that DO support concurrency.
func splitConcurrent(domains []*models.DomainConfig, filter string) (serial []*models.DomainConfig, concurrent []*models.DomainConfig) {
	if filter == "none" {
		return domains, nil
	}
	if filter == "all" {
		return nil, domains
	}
	for _, dc := range domains {
		if allConcur(dc) {
			concurrent = append(concurrent, dc)
		} else {
			serial = append(serial, dc)
		}
	}
	return
}

// allConcur returns true if its registrar and all DNS providers support
// concurrency.  Otherwise false is returned.
func allConcur(dc *models.DomainConfig) bool {
	if !providers.ProviderHasCapability(dc.RegistrarInstance.ProviderType, providers.CanConcur) {
		// fmt.Printf("WHY? %q: %+v\n", dc.Name, dc.RegistrarInstance)
		return false
	}
	for _, p := range dc.DNSProviderInstances {
		if !providers.ProviderHasCapability(p.ProviderType, providers.CanConcur) {
			// fmt.Printf("WHY? %q: %+v\n", dc.Name, p)
			return false
		}
	}
	return true
}

// optimizeOrder returns a list of DomainConfigs so that they gather fastest.
//
// The current algorithm is based on the heuistic that larger zones (zones with
// the most records) need the most time to be processed.  Therefore, the largest
// zones are moved to the front of the list.
// This isn't perfect but it is good enough.
func optimizeOrder(zones []*models.DomainConfig) []*models.DomainConfig {
	slices.SortFunc(zones, func(a, b *models.DomainConfig) int {
		return len(b.Records) - len(a.Records) // Biggest to smallest.
	})

	// // For benchmarking. Randomize the list. If you aren't better
	// // than random, you might as well not play.
	// rand.Shuffle(len(zones), func(i, j int) {
	// 	zones[i], zones[j] = zones[j], zones[i]
	// })

	return zones
}

func oneZonePopulate(zone *models.DomainConfig, zc *CmdZoneCache) error {
	var errs []error
	// Loop over all the providers configured for that zone:
	for _, provider := range zone.DNSProviderInstances {
		populateCorrections, err := generatePopulateCorrections(provider, zone, zc)
		if err != nil {
			errs = append(errs, err)
		}
		zone.StorePopulateCorrections(provider.Name, populateCorrections)
	}
	return errors.Join(errs...)
}

func oneZone(zone *models.DomainConfig, args PPreviewArgs) error {
	var errs []error
	// Fix the parent zone's delegation: (if able/needed)
	delegationCorrections, dcCount, err := generateDelegationCorrections(zone, zone.DNSProviderInstances, zone.RegistrarInstance)
	if err != nil {
		errs = append(errs, err)
	}

	// Loop over the (selected) providers configured for that zone:
	providersToProcess := whichProvidersToProcess(zone.DNSProviderInstances, args.Providers)
	for _, provider := range providersToProcess {
		// Update the zone's records at the provider:
		zoneCor, rep, actualChangeCount, err := generateZoneCorrections(zone, provider)
		zone.StoreCorrections(provider.Name, rep)
		zone.StoreCorrections(provider.Name, zoneCor)
		zone.IncrementChangeCount(provider.Name, actualChangeCount)
		if err != nil {
			errs = append(errs, err)
		}
	}

	// Do the delegation corrections after the zones are updated.
	zone.StoreCorrections(zone.RegistrarInstance.Name, delegationCorrections)
	zone.IncrementChangeCount(zone.RegistrarInstance.Name, dcCount)
	return errors.Join(errs...)
}

func whichProvidersToProcess(providers []*models.DNSProviderInstance, filter string) []*models.DNSProviderInstance {
	if filter == "all" { // all
		return providers
	}

	permitList := strings.Split(filter, ",")
	var picked []*models.DNSProviderInstance

	// Just the default providers:
	if filter == "" {
		for _, provider := range providers {
			if provider.IsDefault {
				picked = append(picked, provider)
			}
		}
		return picked
	}

	// Just the exact matches:
	for _, provider := range providers {
		for _, filterItem := range permitList {
			if provider.Name == filterItem {
				picked = append(picked, provider)
			}
		}
	}
	return picked
}

func skipProvider(name string, providers []*models.DNSProviderInstance) bool {
	return !slices.ContainsFunc(providers, func(p *models.DNSProviderInstance) bool {
		return p.Name == name
	})
}

func parseCorrectionMsg(s string) []string {
	// Regex to remove the terminal styled formatting
	ansiRe := regexp.MustCompile(`\x1b\[[0-9;]*[a-zA-Z]`)
	s = ansiRe.ReplaceAllString(s, "")
	// Create a slice(array) of correction/actions/changes from Msg
	corrections := strings.Split(s, "\n")
	// Clean up the slice, precaution remove any empty entries.
	clean := make([]string, 0, len(corrections))
	for _, l := range corrections {
		l = strings.TrimSpace(l)
		if l != "" {
			clean = append(clean, l)
		}
	}
	return clean
}

func genReportItem(zoneName string, corrections []*models.Correction, providerName string, registrarName string) *ReportItem {
	correctionDetails := make([]string, 0)
	for _, cor := range corrections {
		if cor.F != nil {
			// `corrections` is a list that contains "informational" messages
			// (where `.F = nil`) and "actions" to be taken (where `.F != nil`).
			// When `.F = nil`, the contents of `.Msg` can either be a concatenation of all
			// actions(all changes done in a single API call) or a single
			// action(one API call per change), depending on the provider's implementation.
			//
			// We are parsing `cor.Msg` to remove terminal styled formatting and create
			// a comprehensive list of actions (changes), as well as get an accurate
			// number of corrections (`len(correctionDetails)`).

			correctionDetails = append(correctionDetails, parseCorrectionMsg(cor.Msg)...)
		}
	}

	r := ReportItem{
		Domain:            zoneName,
		Corrections:       len(correctionDetails),
		CorrectionDetails: correctionDetails,
	}
	if providerName != "" {
		r.Provider = providerName
	}
	if registrarName != "" {
		r.Registrar = registrarName
	}
	return &r
}

func pprintOrRunCorrections(zoneName string, providerName string, corrections []*models.Correction, out printer.CLI, push bool, interactive bool, notifier notifications.Notifier, report string) bool {
	if len(corrections) == 0 {
		return false
	}
	var anyErrors bool
	cc := 0
	cn := 0
	for _, correction := range corrections {
		// Print what we're about to do.
		if correction.F == nil {
			out.PrintReport(cn, correction)
			cn++
		} else {
			out.PrintCorrection(cc, correction)
			cc++
		}

		// If interactive, ask "are you sure?" and skip if not.
		if push && interactive && !out.PromptToRun() {
			continue
		}

		// If it is an action (not an informational message), notify and execute.
		if correction.F != nil {
			var err error
			if push {
				err = correction.F()
				out.EndCorrection(err)
				if err != nil {
					anyErrors = true
				}
			}
			notifyErr := notifier.Notify(zoneName, providerName, correction.Msg, err, !push)
			if notifyErr != nil {
				out.Warnf("Error sending notification: %s\n", notifyErr)
			}
		}
	}

	_ = report // File name to write report to. (obsolete)
	return anyErrors
}

func writeReport(report string, reportItems []*ReportItem) error {
	// No filename? No report.
	if report == "" {
		return nil
	}

	f, err := os.OpenFile(report, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()

	// Disabling HTML encoding
	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	enc.SetEscapeHTML(false)

	if err := enc.Encode(reportItems); err != nil {
		return err
	}
	return nil
}

func generatePopulateCorrections(provider *models.DNSProviderInstance, dc *models.DomainConfig, zcache *CmdZoneCache) ([]*models.Correction, error) {
	lister, ok := provider.Driver.(providers.ZoneLister)
	if !ok {
		return nil, nil // We can't generate a list. No corrections are possible.
	}

	z, err := zcache.zoneList(provider.Name, lister)
	if err != nil {
		errMsg := fmt.Sprintf("zoneList failed for %q: %s", provider.Name, err)
		return []*models.Correction{{Msg: errMsg}}, errors.New(errMsg)
	}
	zones := *z

	// TODO(tlim): This conversion shouldn't be needed.  The provider should return the zone names in ACE format.
	// "aceZoneName := dc.Name" should be sufficient.
	aceZoneName, _ := idna.ToASCII(dc.Name)
	if slices.Contains(zones, aceZoneName) {
		return nil, nil // zone exists. Nothing to do.
	}

	creator, ok := provider.Driver.(providers.ZoneCreator)
	if !ok {
		errMsg := fmt.Sprintf("Zone %q does not exist. Can not create because %q does not implement ZoneCreator", aceZoneName, provider.Name)
		return []*models.Correction{{Msg: errMsg}}, errors.New(errMsg)
	}

	return []*models.Correction{{
		Msg: fmt.Sprintf("Ensuring zone %q exists in %q", aceZoneName, provider.Name),
		F:   func() error { return creator.EnsureZoneExists(dc) },
	}}, nil
}

func generateZoneCorrections(zone *models.DomainConfig, provider *models.DNSProviderInstance) ([]*models.Correction, []*models.Correction, int, error) {
	reports, zoneCorrections, actualChangeCount, err := zonerecs.CorrectZoneRecords(provider.Driver, zone)
	if err != nil {
		return []*models.Correction{{Msg: fmt.Sprintf("Domain %q provider %s Error: %s", zone.Name, provider.Name, err)}}, nil, 0, err
	}
	return zoneCorrections, reports, actualChangeCount, nil
}

func generateDelegationCorrections(zone *models.DomainConfig, providers []*models.DNSProviderInstance, _ *models.RegistrarInstance) ([]*models.Correction, int, error) {
	// fmt.Printf("DEBUG: generateDelegationCorrections start zone=%q nsList = %v\n", zone.Name, zone.Nameservers)
	nsList, err := nameservers.DetermineNameserversForProviders(zone, providers, true)
	if err != nil {
		return msg(fmt.Sprintf("DetermineNS: zone %q; Error: %s", zone.Name, err)), 0, err
	}
	zone.Nameservers = nsList
	nameservers.AddNSRecords(zone)

	if len(zone.Nameservers) == 0 && zone.Metadata["no_ns"] != "true" {
		return []*models.Correction{{Msg: fmt.Sprintf("Skipping registrar %q: No nameservers declared for domain %q. Add {no_ns: 'true'} to force",
			zone.RegistrarName,
			zone.Name,
		)}}, 0, nil
	}

	corrections, err := zone.RegistrarInstance.Driver.GetRegistrarCorrections(zone)
	if err != nil {
		return msg(fmt.Sprintf("zone %q; Rprovider %q; Error: %s", zone.Name, zone.RegistrarInstance.Name, err)), 0, err
	}
	return corrections, len(corrections), nil
}

func msg(s string) []*models.Correction {
	return []*models.Correction{{Msg: s}}
}

// PInitializeProviders takes (fully processed) configuration and instantiates all providers and returns them.
func PInitializeProviders(cfg *models.DNSConfig, providerConfigs map[string]map[string]string, notifyFlag bool) (notify notifications.Notifier, err error) {
	var notificationCfg map[string]string
	defer func() {
		notify = notifications.Init(notificationCfg)
	}()
	if notifyFlag {
		notificationCfg = providerConfigs["notifications"]
	}
	isNonDefault := map[string]bool{}
	for name, vals := range providerConfigs {
		// add "_exclude_from_defaults":"true" to a provider to exclude it from being run unless
		// -providers=all or -providers=name
		if vals["_exclude_from_defaults"] == "true" {
			isNonDefault[name] = true
		}
	}

	// Populate provider type ids based on values from creds.json:
	msgs, err := ppopulateProviderTypes(cfg, providerConfigs)
	if len(msgs) != 0 {
		fmt.Fprintln(os.Stderr, strings.Join(msgs, "\n"))
	}
	if err != nil {
		return notify, err
	}

	registrars := map[string]providers.Registrar{}
	dnsProviders := map[string]providers.DNSServiceProvider{}
	for _, d := range cfg.Domains {
		if registrars[d.RegistrarName] == nil {
			rCfg := cfg.RegistrarsByName[d.RegistrarName]
			r, err := providers.CreateRegistrar(rCfg.Type, providerConfigs[d.RegistrarName])
			if err != nil {
				return nil, err
			}
			registrars[d.RegistrarName] = r
		}
		d.RegistrarInstance.Driver = registrars[d.RegistrarName]
		d.RegistrarInstance.IsDefault = !isNonDefault[d.RegistrarName]
		for _, pInst := range d.DNSProviderInstances {
			if dnsProviders[pInst.Name] == nil {
				dCfg := cfg.DNSProvidersByName[pInst.Name]
				prov, err := providers.CreateDNSProvider(dCfg.Type, providerConfigs[dCfg.Name], dCfg.Metadata)
				if err != nil {
					return nil, err
				}
				dnsProviders[pInst.Name] = prov
			}
			pInst.Driver = dnsProviders[pInst.Name]
			pInst.IsDefault = !isNonDefault[pInst.Name]
		}
	}
	return notify, err
}

// pproviderTypeFieldName is the name of the field in creds.json that specifies the provider type id.
const pproviderTypeFieldName = "TYPE"

// ppurl is the documentation URL to list in the warnings related to missing provider type ids.
const purl = "https://docs.dnscontrol.org/commands/creds-json"

// ppopulateProviderTypes scans a DNSConfig for blank provider types and fills them in based on providerConfigs.
// That is, if the provider type is "-" or "", we take that as an flag
// that means this value should be replaced by the type found in creds.json.
func ppopulateProviderTypes(cfg *models.DNSConfig, providerConfigs map[string]map[string]string) ([]string, error) {
	var msgs []string

	for i := range cfg.Registrars {
		pType := cfg.Registrars[i].Type
		pName := cfg.Registrars[i].Name
		nt, warnMsg, err := prefineProviderType(pName, pType, providerConfigs[pName], "NewRegistrar")
		cfg.Registrars[i].Type = nt
		if warnMsg != "" {
			msgs = append(msgs, warnMsg)
		}
		if err != nil {
			return msgs, err
		}
	}

	for i := range cfg.DNSProviders {
		pName := cfg.DNSProviders[i].Name
		pType := cfg.DNSProviders[i].Type
		nt, warnMsg, err := prefineProviderType(pName, pType, providerConfigs[pName], "NewDnsProvider")
		cfg.DNSProviders[i].Type = nt
		if warnMsg != "" {
			msgs = append(msgs, warnMsg)
		}
		if err != nil {
			return msgs, err
		}
	}

	// Update these fields set by
	// commands/commands.go:preloadProviders().
	// This is probably a layering violation.  That said, the
	// fundamental problem here is that we're storing the provider
	// instances by string name, not by a pointer to a struct.  We
	// should clean that up someday.
	for _, domain := range cfg.Domains { // For each domain..
		for _, provider := range domain.DNSProviderInstances { // For each provider...
			pName := provider.Name
			pType := provider.ProviderType
			nt, warnMsg, err := prefineProviderType(pName, pType, providerConfigs[pName], "NewDnsProvider")
			provider.ProviderType = nt
			if warnMsg != "" {
				msgs = append(msgs, warnMsg)
			}
			if err != nil {
				return msgs, err
			}
		}
		p := domain.RegistrarInstance
		pName := p.Name
		pType := p.ProviderType
		nt, warnMsg, err := prefineProviderType(pName, pType, providerConfigs[pName], "NewRegistrar")
		p.ProviderType = nt
		if warnMsg != "" {
			msgs = append(msgs, warnMsg)
		}
		if err != nil {
			return msgs, err
		}
	}

	return puniqueStrings(msgs), nil
}

// puniqueStrings takes an unsorted slice of strings and returns the
// unique strings, in the order they first appeared in the list.
func puniqueStrings(stringSlice []string) []string {
	keys := make(map[string]bool)
	list := []string{}
	for _, entry := range stringSlice {
		if _, ok := keys[entry]; !ok {
			keys[entry] = true
			list = append(list, entry)
		}
	}
	return list
}

func prefineProviderType(credEntryName string, t string, credFields map[string]string, source string) (replacementType string, warnMsg string, err error) {
	// t="" and t="-" are processed the same. Standardize on "-" to reduce the number of cases to check.
	if t == "" {
		t = "-"
	}

	// Use cases:
	//
	// type     credsType
	// ----     ---------
	// - or ""  GANDI        lookup worked. Nothing to say.
	// - or ""  - or ""      ERROR "creds.json has invalid or missing data"
	// GANDI    ""           WARNING "Working but.... Please fix as follows..."
	// GANDI    INFO "working but unneeded: clean up as follows..."
	// GANDI    NAMEDOT      ERROR "error mismatched: please fix as follows..."

	// ERROR: Invalid.
	// WARNING: Required change to remain compatible with 4.0
	// INFO: Post-4.0 cleanups or other non-required changes.

	if t != "-" {
		// Old-style, dnsconfig.js specifies the type explicitly.
		// This is supported but we suggest updates for future compatibility.

		// If credFields is nil, that means there was no entry in creds.json:
		if credFields == nil {
			// Warn the user to update creds.json in preparation for 4.0:
			// In 4.0 this should be an error.  We could default to a
			// provider such as "NONE" but I suspect it would be confusing
			// to users to see references to a provider name that they did
			// not specify.
			return t, fmt.Sprintf(`WARNING: For future compatibility, add this entry creds.json: %q: { %q: %q }, (See %s#missing)`,
				credEntryName, pproviderTypeFieldName, t,
				purl,
			), nil
		}

		switch ct := credFields[pproviderTypeFieldName]; ct {
		case "":
			// Warn the user to update creds.json in preparation for 4.0:
			// In 4.0 this should be an error.
			return t, fmt.Sprintf(`WARNING: For future compatibility, update the %q entry in creds.json by adding: %q: %q, (See %s#missing)`,
				credEntryName,
				pproviderTypeFieldName, t,
				purl,
			), nil
		case "-":
			// This should never happen. The user is specifying "-" in a place that it shouldn't be used.
			return "-", "", fmt.Errorf(`ERROR: creds.json entry %q has invalid %q value %q (See %s#hyphen)`,
				credEntryName, pproviderTypeFieldName, ct,
				purl,
			)
		case t:
			// creds.json file is compatible with and dnsconfig.js can be updated.
			return ct, fmt.Sprintf(`INFO: In dnsconfig.js %s(%q, %q) can be simplified to %s(%q) (See %s#cleanup)`,
				source, credEntryName, t,
				source, credEntryName,
				purl,
			), nil
		default:
			// creds.json lists a TYPE but it doesn't match what's in dnsconfig.js!
			return t, "", fmt.Errorf(`ERROR: Mismatch found! creds.json entry %q has %q set to %q but dnsconfig.js specifies %s(%q, %q) (See %s#mismatch)`,
				credEntryName,
				pproviderTypeFieldName, ct,
				source, credEntryName, t,
				purl,
			)
		}
	}

	// t == "-"
	// New-style, dnsconfig.js does not specify the type (t == "") or a
	// command line tool accepted "-" as a positional argument for
	// backwards compatibility.

	// If credFields is nil, that means there was no entry in creds.json:
	if credFields == nil {
		return "", "", fmt.Errorf(`ERROR: creds.json is missing an entry called %q. Suggestion: %q: { %q: %q }, (See %s#missing)`,
			credEntryName,
			credEntryName, pproviderTypeFieldName, "FILL_IN_PROVIDER_TYPE",
			purl,
		)
	}

	// New-style, dnsconfig.js doesn't specifies the type. It will be
	// looked up in creds.json.
	switch ct := credFields[pproviderTypeFieldName]; ct {
	case "":
		return ct, "", fmt.Errorf(`ERROR: creds.json entry %q is missing: %q: %q, (See %s#fixcreds)`,
			credEntryName,
			pproviderTypeFieldName, "FILL_IN_PROVIDER_TYPE",
			purl,
		)
	case "-":
		// This should never happen. The user is confused and specified "-" in the wrong place!
		return "-", "", fmt.Errorf(`ERROR: creds.json entry %q has invalid %q value %q (See %s#hyphen)`,
			credEntryName,
			pproviderTypeFieldName, ct,
			purl,
		)
	default:
		// use the value in creds.json (this should be the normal case)
		return ct, "", nil
	}
}
