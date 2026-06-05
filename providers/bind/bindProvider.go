package bind

/*

bind -
  Generate zonefiles suitable for BIND.

	The zonefiles are read and written to the directory -bind_dir

	If the old zonefiles are readable, we read them to determine
	if an update is actually needed. The old zonefile is also used
	as the basis for generating the new SOA serial number.

*/

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	dnsv2 "codeberg.org/miekg/dns"
	"github.com/DNSControl/dnscontrol/v4/models"
	"github.com/DNSControl/dnscontrol/v4/pkg/bindserial"
	"github.com/DNSControl/dnscontrol/v4/pkg/diff2"
	"github.com/DNSControl/dnscontrol/v4/pkg/dnsrr"
	"github.com/DNSControl/dnscontrol/v4/pkg/domaintags"
	"github.com/DNSControl/dnscontrol/v4/pkg/prettyzone"
	"github.com/DNSControl/dnscontrol/v4/pkg/printer"
	"github.com/DNSControl/dnscontrol/v4/pkg/providers"
)

// defaultZonesDir is used when the BIND credentials do not specify a
// directory.
const defaultZonesDir = "zones"

var features = providers.DocumentationNotes{
	// The default for unlisted capabilities is 'Cannot'.
	// See providers/capabilities.go for the entire list of capabilities.
	providers.CanAutoDNSSEC:          providers.Can("Just writes out a comment indicating DNSSEC was requested"),
	providers.CanConcur:              providers.Can(),
	providers.CanGetZones:            providers.Can(),
	providers.CanUseCAA:              providers.Can(),
	providers.CanUseDHCID:            providers.Can(),
	providers.CanUseDNAME:            providers.Can(),
	providers.CanUseDNSKEY:           providers.Can(),
	providers.CanUseDS:               providers.Can(),
	providers.CanUseHTTPS:            providers.Can(),
	providers.CanUseLOC:              providers.Can(),
	providers.CanUseNAPTR:            providers.Can(),
	providers.CanUseOPENPGPKEY:       providers.Can(),
	providers.CanUsePTR:              providers.Can(),
	providers.CanUseRP:               providers.Can(),
	providers.CanUseSMIMEA:           providers.Can(),
	providers.CanUseSOA:              providers.Can(),
	providers.CanUseSRV:              providers.Can(),
	providers.CanUseSSHFP:            providers.Can(),
	providers.CanUseSVCB:             providers.Can(),
	providers.CanUseTLSA:             providers.Can(),
	providers.DocCreateDomains:       providers.Can("Driver just maintains list of zone files. It should automatically add missing ones."),
	providers.DocDualHost:            providers.Can(),
	providers.DocOfficiallySupported: providers.Can(),
}

func initBind(config map[string]string, providermeta json.RawMessage) (providers.DNSServiceProvider, error) {
	// config -- the key/values from creds.json
	// meta -- the json blob from NewReq('name', 'TYPE', meta)
	api := &bindProvider{
		directory:      config["directory"],
		filenameformat: config["filenameformat"],
	}
	if api.directory == "" {
		api.directory = "zones"
	}
	if api.filenameformat == "" {
		api.filenameformat = "%c.zone"
	}
	if len(providermeta) != 0 {
		err := json.Unmarshal(providermeta, api)
		if err != nil {
			return nil, err
		}
	}
	var nss []string
	for i, ns := range api.DefaultNS {
		if ns == "" {
			return nil, fmt.Errorf("empty string in default_ns[%d]", i)
		}
		// If it contains a ".", it must end in a ".".
		if strings.ContainsRune(ns, '.') && ns[len(ns)-1] != '.' {
			return nil, fmt.Errorf("default_ns (%v) must end with a (.) [https://docs.dnscontrol.org/language-reference/why-the-dot]", ns)
		}
		// This is one of the (increasingly rare) cases where we store a
		// name without the trailing dot to indicate a FQDN.
		nss = append(nss, strings.TrimSuffix(ns, "."))
	}
	var err error
	api.nameservers, err = models.ToNameservers(nss)
	return api, err
}

func init() {
	const providerName = "BIND"
	const providerMaintainer = "@TomOnTime"
	fns := providers.DspFuncs{
		Initializer:   initBind,
		RecordAuditor: AuditRecords,
	}
	providers.RegisterDomainServiceProviderType(providerName, fns, features)
	providers.RegisterMaintainer(providerName, providerMaintainer)
	providers.RegisterCredsMetadata(providerName, providers.CredsMetadata{
		DisplayName: "ISC BIND",
		Kind:        providers.KindDNS,
		DocsURL:     "https://docs.dnscontrol.org/provider/bind",
		Notes:       "BIND writes zone files to a local directory; no API credentials are needed.",
		Fields: []providers.CredsField{
			{
				Key:     "directory",
				Label:   "Zone files directory",
				Help:    "Directory where BIND zone files are written. Defaults to zones.",
				Default: defaultZonesDir,
			},
			{
				Key:     "filenameformat",
				Label:   "File name format",
				Help:    "Format used for zone file names. Defaults to %c.zone.",
				Default: "%c.zone",
			},
		},
		PostWrite: func(fields map[string]string) error {
			dir := fields["directory"]
			if dir == "" {
				dir = defaultZonesDir
			}
			return os.MkdirAll(dir, 0o755)
		},
	})
}

// SoaDefaults contains the parts of the default SOA settings.
type SoaDefaults struct {
	Ns      string `json:"master"`
	Mbox    string `json:"mbox"`
	Serial  uint32 `json:"serial"`
	Refresh uint32 `json:"refresh"`
	Retry   uint32 `json:"retry"`
	Expire  uint32 `json:"expire"`
	Minttl  uint32 `json:"minttl"`
	// TTL isn't part of the SOA RDATA.  It is a default applied to the SOA record's TTL itself.
	TTL uint32 `json:"ttl,omitempty"`
}

func (s SoaDefaults) String() string {
	return fmt.Sprintf("%s %s %d %d %d %d %d %d", s.Ns, s.Mbox, s.Serial, s.Refresh, s.Retry, s.Expire, s.Minttl, s.TTL)
}

// bindProvider is the provider handle for the bindProvider driver.
type bindProvider struct {
	DefaultNS      []string    `json:"default_ns"`
	DefaultSoa     SoaDefaults `json:"default_soa"`
	nameservers    []*models.Nameserver
	directory      string
	filenameformat string
}

// GetNameservers returns the nameservers for a domain.
func (c *bindProvider) GetNameservers(string) ([]*models.Nameserver, error) {
	var r []string
	for _, j := range c.nameservers {
		r = append(r, j.Name)
	}
	return models.ToNameservers(r)
}

// ListZones returns all the zones in an account.
func (c *bindProvider) ListZones() ([]string, error) {
	if _, err := os.Stat(c.directory); os.IsNotExist(err) {
		return nil, fmt.Errorf("directory %q does not exist", c.directory)
	}

	var files []string
	f, err := os.Open(c.directory)
	if err != nil {
		return files, fmt.Errorf("bind ListZones open dir %q: %w",
			c.directory, err)
	}
	filenames, err := f.Readdirnames(-1)
	if err != nil {
		return files, fmt.Errorf("bind ListZones readdir %q: %w",
			c.directory, err)
	}

	return extractZonesFromFilenames(c.filenameformat, filenames), nil
}

// GetZoneRecords gets the records of a zone and returns them in RecordConfig format.
func (c *bindProvider) GetZoneRecords(dc *models.DomainConfig) (models.Records, error) {
	domain := dc.Name
	meta := dc.Metadata

	var zonefile string

	if _, err := os.Stat(c.directory); os.IsNotExist(err) {
		printer.Printf("\nWARNING: BIND directory %q does not exist! (will create)\n", c.directory)
	}
	ff := domaintags.DomainNameVarieties{
		Tag:         meta[models.DomainTag],
		NameRaw:     meta[models.DomainNameRaw],
		NameASCII:   domain,
		NameUnicode: meta[models.DomainNameUnicode],
		UniqueName:  meta[models.DomainUniqueName],
		// NB(tlim): When "get-zones" is called, these values are populated
		// directly by commands/getZones.go near where provider.GetZoneRecords()
		// is called. Changes here may need to be reflected there too.
	}
	zonefile = filepath.Join(c.directory,
		makeFileName(
			c.filenameformat,
			ff,
		),
	)
	//fmt.Printf("DEBUG: Reading zonefile %q\n", zonefile)
	//fmt.Printf("DEBUG: Meta %+v\n", meta)
	//fmt.Printf("DEBUG: Domain Names %+v\n", ff)

	content, err := os.ReadFile(zonefile)
	if os.IsNotExist(err) {
		// If the file doesn't exist, that's not an error. Just informational.
		fmt.Fprintf(os.Stderr, "INFO: File does not (yet) exist: %q\n", zonefile)
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("can't open %s: %w", zonefile, err)
	}

	return ParseZoneContents(string(content), domain, zonefile)
}

func findSoaRecord(recs models.Records) *models.RecordConfig {
	for _, rec := range recs {
		if rec.Type == "SOA" {
			return rec
		}
	}
	return nil
}

func updateSerialNumber(origin string, recs models.Records, forcedSerial uint32) {

	recToUpdate := findSoaRecord(recs)
	if recToUpdate == nil {
		return
	}

	if forcedSerial != 0 {
		recToUpdate.SoaSerial = forcedSerial
	} else {
		recToUpdate.SoaSerial = generateSerial(recToUpdate.SoaSerial)
	}

	recToUpdate.RDATA = nil
	recToUpdate.FixUp(origin)
}

// ParseZoneContents parses a string as a BIND zone and returns the records.
func ParseZoneContents(content string, zoneName string, zonefileName string) (models.Records, error) {
	//zp := dnsv1.NewZoneParser(strings.NewReader(content), zoneName, zonefileName)
	zp := dnsv2.NewZoneParser(strings.NewReader(content), zoneName, zonefileName)

	foundRecords := models.Records{}
	for rr, ok := zp.Next(); ok; rr, ok = zp.Next() {
		var rec models.RecordConfig
		// var prec *models.RecordConfig
		var err error

		// rtype := rr.Header().Rrtype
		// rtypeStr := dnsv1.TypeToString[rtype]

		// Set rtypeStr to the string version of the type, but if it's a fake type, use the fake type name instead.
		// rtype := dnsv2.RRToType(rr)
		// rtypeStr := dnsv2.TypeToString[rtype]

		// if rtypeinfo.IsModernType(rtypeStr) {
		// 	// Modern types:
		// 	name := rr.Header().Name
		// 	prec, err = rtypecontrol.NewRecordConfigFromStruct(name, rr.Header().TTL, rtypeStr, rr.Data(), domaintags.MakeDomainNameVarieties(zoneName))
		// 	if err != nil {
		// 		return nil, err
		// 	}
		// 	rec = *prec
		// 	rec.TTL = rr.Header().TTL
		// } else {
		// 	// Legacy types:
		rec, err = dnsrr.RRtoRCV2(rr, zoneName)
		if err != nil {
			return nil, err
		}
		// }

		rec.FixUp(zoneName) // hack.  Maybe not needed.

		foundRecords = append(foundRecords, &rec)
	}

	if err := zp.Err(); err != nil {
		return nil, fmt.Errorf("error while parsing '%v': %w", zonefileName, err)
	}
	return foundRecords, nil
}

func (c *bindProvider) EnsureZoneExists(_ string, _ map[string]string) error {
	return nil
}

// GetZoneRecordsCorrections returns a list of corrections that will turn existing records into dc.Records.
func (c *bindProvider) GetZoneRecordsCorrections(dc *models.DomainConfig, foundRecords models.Records) ([]*models.Correction, int, error) {
	var corrections []*models.Correction
	var zonefile string

	changes := false
	var msg string

	AddSoaIfMissing(dc, c.DefaultSoa)

	var msgs []string
	var actualChangeCount int
	result, err := diff2.ByZone(foundRecords, dc, nil)
	if err != nil {
		return nil, 0, err
	}
	msgs, changes, actualChangeCount = result.Msgs, result.HasChanges, result.ActualChangeCount
	if !changes {
		return nil, 0, nil
	}
	msg = strings.Join(msgs, "\n")

	comments := make([]string, 0, 5)
	comments = append(comments,
		"generated with dnscontrol "+time.Now().Format(time.RFC3339),
	)
	if dc.AutoDNSSEC == "on" {
		// This does nothing but reminds the user to add the correct
		// auto-dnssecc zone statement to named.conf.
		// While it is a no-op, it is useful for situations where a zone
		// has multiple providers.
		comments = append(comments, "Automatic DNSSEC signing requested")
	}

	zonefile = filepath.Join(c.directory,
		makeFileName(
			c.filenameformat,
			domaintags.DomainNameVarieties{
				Tag:         dc.Tag,
				NameRaw:     dc.NameRaw,
				NameASCII:   dc.Name,
				NameUnicode: dc.NameUnicode,
				UniqueName:  dc.UniqueName,
			},
		),
	)

	// We know there are changes. Update the SOA record's serial number.
	updateSerialNumber(dc.Name, result.DesiredPlus, uint32(bindserial.ForcedValue&0xFFFF))

	corrections = append(corrections,
		&models.Correction{
			Msg: msg,
			F: func() error {
				printer.Printf("WRITING ZONEFILE: %v\n", zonefile)
				fname, err := preprocessFilename(zonefile)
				if err != nil {
					return fmt.Errorf("could not create zonefile: %w", err)
				}
				zf, err := os.Create(fname)
				if err != nil {
					return fmt.Errorf("could not create zonefile: %w", err)
				}
				// Beware that if there are any fake types, then they will
				// be commented out on write, but we don't reverse that when
				// reading, so there will be a diff on every invocation.
				err = prettyzone.WriteZoneFileRC(zf, result.DesiredPlus, dc.Name, 0, comments)
				if err != nil {
					return fmt.Errorf("failed WriteZoneFile: %w", err)
				}
				err = zf.Close()
				if err != nil {
					return fmt.Errorf("closing: %w", err)
				}
				return nil
			},
		})

	return corrections, actualChangeCount, nil
}

// preprocessFilename pre-processes a filename we're about to os.Create()
// * On Windows systems, it translates the separator.
// * It attempts to mkdir the directories leading up to the filename.
// * If running on Linux as root, it does not attempt to create directories.
func preprocessFilename(name string) (string, error) {
	universalName := filepath.FromSlash(name)
	// Running as root? Don't create the parent directories. It is unsafe.
	if os.Getuid() != 0 {
		// Create the parent directories
		dir := filepath.Dir(name)
		universalDir := filepath.FromSlash(dir)
		if err := os.MkdirAll(universalDir, 0o750); err != nil {
			return "", err
		}
	}
	return universalName, nil
}
