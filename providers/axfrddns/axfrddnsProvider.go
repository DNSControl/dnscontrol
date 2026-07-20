package axfrddns

/*

axfrddns -
  Fetch the zone with an AXFR request (RFC5936) to a given primary master, and
  push Dynamic DNS updates (RFC2136) to the same server.

  Both the AXFR request and the updates might be authentificated with
  a TSIG.

*/

import (
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net" // Verified not used for net.IP
	"strings"
	"sync"
	"time"

	"codeberg.org/miekg/dns/dnsutil"
	"github.com/DNSControl/dnscontrol/v4/models"
	"github.com/DNSControl/dnscontrol/v4/pkg/diff2"
	"github.com/DNSControl/dnscontrol/v4/pkg/dnsrr"
	"github.com/DNSControl/dnscontrol/v4/pkg/printer"
	"github.com/DNSControl/dnscontrol/v4/pkg/providers"
	dnsv1 "github.com/miekg/dns"
)

const (
	dnsTimeout       = 30 * time.Second
	dnssecDummyLabel = "__dnssec"
	dnssecDummyTxt   = "Domain has DNSSec records, not displayed here."
)

var features = providers.DocumentationNotes{
	// The default for unlisted capabilities is 'Cannot'.
	// See providers/capabilities.go for the entire list of capabilities.
	providers.CanAutoDNSSEC:          providers.Can("Just warn when DNSSEC is requested but no RRSIG is found in the AXFR or warn when DNSSEC is not requested but RRSIG are found in the AXFR."),
	providers.CanConcur:              providers.Can(),
	providers.CanUseCAA:              providers.Can(),
	providers.CanUseDHCID:            providers.Can(),
	providers.CanUseDNAME:            providers.Can(),
	providers.CanUseDS:               providers.Can(),
	providers.CanUseHTTPS:            providers.Can(),
	providers.CanUseLOC:              providers.Can(),
	providers.CanUseNAPTR:            providers.Can(),
	providers.CanUseOPENPGPKEY:       providers.Can(),
	providers.CanUsePTR:              providers.Can(),
	providers.CanUseSMIMEA:           providers.Can(),
	providers.CanUseSRV:              providers.Can(),
	providers.CanUseSSHFP:            providers.Can(),
	providers.CanUseSVCB:             providers.Can(),
	providers.CanUseTLSA:             providers.Can(),
	providers.DocDualHost:            providers.Cannot(),
	providers.DocOfficiallySupported: providers.Cannot(),
	// Possible to support via catalog zones (RFC 9432), but those are not
	// directly supported by DNSControl right now (although nothing is stopping
	// you from manually updating a catalog zone using DNSControl if you wish).
	providers.CanGetZones:      providers.Cannot(),
	providers.DocCreateDomains: providers.Cannot(),
	// Not a valid RR type, so impossible to encode in an RFC-compliant DNS
	// packet.
	providers.CanUseAlias: providers.Cannot(),
	// These are both supported by RFC 2136 (DDNS), but neither work with
	// DNSControl right now.
	providers.CanUseSOA:    providers.Cannot(),
	providers.CanUseDNSKEY: providers.Cannot(),
}

// axfrddnsProvider stores the client info for the provider.
type axfrddnsProvider struct {
	master         string
	updateMode     string
	transferServer string
	transferMode   string
	nameservers    []*models.Nameserver
	transferKey    *Key
	updateKey      *Key

	mu               sync.Mutex // protects hasDnssecRecords during concurrent collection.
	hasDnssecRecords map[string]bool
}

func initAxfrDdns(config map[string]string, providermeta json.RawMessage) (providers.DNSServiceProvider, error) {
	// config -- the key/values from creds.json
	// providermeta -- the json blob from NewReq('name', 'TYPE', providermeta)
	var err error
	api := &axfrddnsProvider{
		hasDnssecRecords: map[string]bool{},
	}
	param := &Param{}
	if len(providermeta) != 0 {
		err := json.Unmarshal(providermeta, param)
		if err != nil {
			return nil, err
		}
	}
	var nss []string
	if config["nameservers"] != "" {
		nss = strings.Split(config["nameservers"], ",")
	}
	for _, ns := range param.DefaultNS {
		nss = append(nss, ns[0:len(ns)-1])
	}
	api.nameservers, err = models.ToNameservers(nss)
	if err != nil {
		return nil, err
	}
	if config["update-mode"] != "" {
		switch config["update-mode"] {
		case "tcp", "tcp-tls", "unix":
			api.updateMode = config["update-mode"]
		case "udp":
			api.updateMode = ""
		default:
			printer.Printf("[Warning] AXFRDDNS: Unknown update-mode in `creds.json` (%s)\n", config["update-mode"])
		}
	} else {
		api.updateMode = "tcp"
	}
	if config["transfer-mode"] != "" {
		switch config["transfer-mode"] {
		case "tcp", "tcp-tls", "unix":
			api.transferMode = config["transfer-mode"]
		default:
			printer.Printf("[Warning] AXFRDDNS: Unknown transfer-mode in `creds.json` (%s)\n", config["transfer-mode"])
		}
	} else {
		api.transferMode = "tcp"
	}
	if config["master"] != "" {
		api.master = config["master"]
		if api.updateMode != "unix" && !strings.Contains(api.master, ":") {
			api.master = api.master + ":53"
		}
	} else if len(api.nameservers) != 0 {
		api.master = api.nameservers[0].Name + ":53"
	} else {
		return nil, errors.New("nameservers list is empty: creds.json needs a default `nameservers` or an explicit `master`")
	}
	if config["transfer-server"] != "" {
		api.transferServer = config["transfer-server"]
		if api.transferMode != "unix" && !strings.Contains(api.transferServer, ":") {
			api.transferServer = api.transferServer + ":53"
		}
	} else {
		api.transferServer = api.master
	}
	api.updateKey, err = readKey(config["update-key"], "update-key")
	if err != nil {
		return nil, err
	}
	api.transferKey, err = readKey(config["transfer-key"], "transfer-key")
	if err != nil {
		return nil, err
	}
	switch strings.ToLower(strings.TrimSpace(config["buggy-cname"])) {
	case "yes", "true":
		printer.Warnf("'buggy-cname' is deprecated as it is no longer necessary.\n")
	}
	for key := range config {
		switch key {
		case "master",
			"nameservers",
			"update-key",
			"transfer-key",
			"transfer-server",
			"update-mode",
			"transfer-mode",
			"buggy-cname",
			"domain",
			"TYPE":
			continue
		default:
			printer.Printf("[Warning] AXFRDDNS: unknown key in `creds.json` (%s)\n", key)
		}
	}
	return api, err
}

func init() {
	const providerName = "AXFRDDNS"
	const providerMaintainer = "@hnrgrgr"
	fns := providers.DspFuncs{
		Initializer:   initAxfrDdns,
		RecordAuditor: AuditRecords,
	}
	providers.RegisterDomainServiceProviderType(providerName, fns, features)
	providers.RegisterMaintainer(providerName, providerMaintainer)
}

// Param is used to decode extra parameters sent to provider.
type Param struct {
	DefaultNS []string `json:"default_ns"`
}

// Key stores the individual parts of a TSIG key.
type Key struct {
	algo   string
	id     string
	secret string
}

func readKey(raw string, kind string) (*Key, error) {
	if raw == "" {
		return nil, nil
	}
	arr := strings.Split(raw, ":")
	if len(arr) != 3 {
		return nil, fmt.Errorf("invalid key format (%s) in AXFRDDNS.TSIG", kind)
	}
	var algo string
	switch arr[0] {
	case "hmac-md5", "md5":
		algo = dnsv1.HmacMD5
	case "hmac-sha1", "sha1":
		algo = dnsv1.HmacSHA1
	case "hmac-sha224", "sha224":
		algo = dnsv1.HmacSHA224
	case "hmac-sha256", "sha256":
		algo = dnsv1.HmacSHA256
	case "hmac-sha384", "sha384":
		algo = dnsv1.HmacSHA384
	case "hmac-sha512", "sha512":
		algo = dnsv1.HmacSHA512
	default:
		return nil, fmt.Errorf("unknown algorithm (%s) in AXFRDDNS.TSIG", kind)
	}
	_, err := base64.StdEncoding.DecodeString(arr[2])
	if err != nil {
		return nil, fmt.Errorf("cannot decode Base64 secret (%s) in AXFRDDNS.TSIG", kind)
	}
	id := dnsutil.Canonical(arr[1])
	return &Key{algo: algo, id: id, secret: arr[2]}, nil
}

// GetNameservers returns the nameservers for a domain.
func (c *axfrddnsProvider) GetNameservers(domain string) ([]*models.Nameserver, error) {
	return c.nameservers, nil
}

func (c *axfrddnsProvider) getAxfrConnection() (*dnsv1.Transfer, error) {
	var con net.Conn
	var err error
	switch c.transferMode {
	case "tcp-tls":
		// RFC 9103 "DNS Zone Transfer over TLS" section 7.1 requires "dot"
		con, err = tls.Dial("tcp", c.transferServer, &tls.Config{NextProtos: []string{"dot"}})
	case "unix":
		con, err = net.Dial("unix", c.transferServer)
	default:
		con, err = net.Dial("tcp", c.transferServer)
	}
	if err != nil {
		return nil, err
	}
	dnscon := &dnsv1.Conn{Conn: con}
	transfer := &dnsv1.Transfer{Conn: dnscon}
	return transfer, nil
}

// FetchZoneRecords gets the records of a zone and returns them in dns.RR format.
func (c *axfrddnsProvider) FetchZoneRecords(domain string) ([]dnsv1.RR, error) {
	transfer, err := c.getAxfrConnection()
	if err != nil {
		return nil, err
	}
	transfer.DialTimeout = dnsTimeout
	transfer.ReadTimeout = dnsTimeout

	request := new(dnsv1.Msg)
	request.SetAxfr(domain + ".")

	if c.transferKey != nil {
		transfer.TsigSecret = map[string]string{c.transferKey.id: c.transferKey.secret}
		request.SetTsig(c.transferKey.id, c.transferKey.algo, 300, time.Now().Unix())
		if c.transferKey.algo == dnsv1.HmacMD5 {
			transfer.TsigProvider = md5Provider(c.transferKey.secret)
		}
	}

	envelope, err := transfer.In(request, c.transferServer)
	if err != nil {
		return nil, err
	}

	var rawRecords []dnsv1.RR
	for msg := range envelope {
		if msg.Error != nil {
			// Fragile but more "user-friendly" error-handling
			err := msg.Error.Error()
			if err == "dns: bad xfr rcode: 9" {
				err = "NOT AUTH (9)"
			}
			return nil, fmt.Errorf("[Error] AXFRDDNS: nameserver refused to transfer the zone %s: %s", domain, err)
		}
		rawRecords = append(rawRecords, msg.RR...)
	}
	return rawRecords, nil
}

// GetZoneRecords gets the records of a zone and returns them in RecordConfig format.
func (c *axfrddnsProvider) GetZoneRecords(dc *models.DomainConfig) (models.Records, error) {
	domain := dc.Name

	rawRecords, err := c.FetchZoneRecords(domain)
	if err != nil {
		return nil, err
	}

	var foundDNSSecRecords *models.RecordConfig
	foundRecords := models.Records{}
	for _, rr := range rawRecords {
		switch rr.Header().Rrtype {
		case dnsv1.TypeRRSIG,
			dnsv1.TypeDNSKEY,
			dnsv1.TypeCDNSKEY,
			dnsv1.TypeCDS,
			dnsv1.TypeNSEC,
			dnsv1.TypeNSEC3,
			dnsv1.TypeNSEC3PARAM,
			dnsv1.TypeZONEMD,
			65281,
			65534:
			// Ignoring DNSSec RRs, but replacing it with a single
			// "TXT" placeholder
			// Ignoring TYPE65281 Technitium Conditional Forwarder Zone Record
			// Also ignoring spurious TYPE65534, see:
			// https://bind9-users.isc.narkive.com/zX29ay0j/rndc-signing-list-not-working#post2
			if foundDNSSecRecords == nil {
				foundDNSSecRecords = new(models.RecordConfig)
				foundDNSSecRecords.Type = "TXT"
				foundDNSSecRecords.SetLabel(dnssecDummyLabel, domain)
				err = foundDNSSecRecords.SetTargetTXT(dnssecDummyTxt)
				if err != nil {
					return nil, err
				}
			}
			continue
		default:
			rec, err := dnsrr.RRtoRC(rr, domain)
			if err != nil {
				return nil, err
			}
			foundRecords = append(foundRecords, &rec)
		}
	}

	if len(foundRecords) >= 1 && foundRecords[len(foundRecords)-1].Type == "SOA" {
		// The SOA is sent two times: as the first and the last record
		// See section 2.2 of RFC5936. We remove the later one.
		foundRecords = foundRecords[:len(foundRecords)-1]
	}

	if foundDNSSecRecords != nil {
		foundRecords = append(foundRecords, foundDNSSecRecords)
	}

	if len(foundRecords) >= 1 {
		last := foundRecords[len(foundRecords)-1]
		if last.Type == "TXT" &&
			last.Name == dnssecDummyLabel &&
			last.GetTargetTXTSegmentCount() == 1 &&
			last.GetTargetTXTSegmented()[0] == dnssecDummyTxt {
			c.mu.Lock()
			c.hasDnssecRecords[domain] = true
			c.mu.Unlock()
			foundRecords = foundRecords[0:(len(foundRecords) - 1)]
		}
	}

	return foundRecords, nil
}

// BuildCorrection return a Correction for a given set of DDNS update and the corresponding message.
func (c *axfrddnsProvider) BuildCorrection(dc *models.DomainConfig, msgs []string, updates []*dnsv1.Msg) *models.Correction {
	if updates == nil {
		return &models.Correction{
			Msg: fmt.Sprintf("DDNS UPDATES to '%s' (primary master: '%s'). Changes:\n%s", dc.Name, c.master, strings.Join(msgs, "\n")),
		}
	}
	return &models.Correction{
		Msg: fmt.Sprintf("DDNS UPDATES to '%s' (primary master: '%s'). Changes:\n%s", dc.Name, c.master, strings.Join(msgs, "\n")),
		F: func() error {
			for _, update := range updates {
				update.Compress = true
				client := new(dnsv1.Client)
				client.Net = c.updateMode
				client.Timeout = dnsTimeout
				if c.updateKey != nil {
					client.TsigSecret = map[string]string{c.updateKey.id: c.updateKey.secret}
					update.SetTsig(c.updateKey.id, c.updateKey.algo, 300, time.Now().Unix())
					if c.updateKey.algo == dnsv1.HmacMD5 {
						client.TsigProvider = md5Provider(c.updateKey.secret)
					}
				}

				msg, _, err := client.Exchange(update, c.master)
				if err != nil {
					return err
				}
				if msg.Rcode != 0 {
					return fmt.Errorf("[Error] AXFRDDNS: nameserver refused to update the zone: %s (%d)",
						dnsv1.RcodeToString[msg.Rcode],
						msg.Rcode)
				}
			}

			return nil
		},
	}
}

// hasNSDeletion returns true if there exist a correction that deletes or changes an NS record.
func hasNSDeletion(changes diff2.ChangeList) bool {
	for _, change := range changes {
		switch change.Type {
		case diff2.CHANGE:
			if change.Old[0].Type == "NS" && change.Old[0].Name == "@" {
				return true
			}
		case diff2.DELETE:
			if change.Old[0].Type == "NS" && change.Old[0].Name == "@" {
				return true
			}
		case diff2.CREATE:
		case diff2.REPORT:
		}
	}
	return false
}

// GetZoneRecordsCorrections returns a list of corrections that will turn existing records into dc.Records.
func (c *axfrddnsProvider) GetZoneRecordsCorrections(dc *models.DomainConfig, foundRecords models.Records) ([]*models.Correction, int, error) {
	// Ignoring the SOA, others providers don't manage it either.
	if len(foundRecords) >= 1 && foundRecords[0].Type == "SOA" {
		foundRecords = foundRecords[1:]
	}

	// TODO(tlim): This check should be done on all providers. Move to the global validation code.
	c.mu.Lock()
	if dc.AutoDNSSEC == "on" && !c.hasDnssecRecords[dc.Name] {
		printer.Printf("Warning: AUTODNSSEC is enabled for %s, but no DNSKEY or RRSIG record was found in the AXFR answer!\n", dc.Name)
	}
	if dc.AutoDNSSEC == "off" && c.hasDnssecRecords[dc.Name] {
		printer.Printf("Warning: AUTODNSSEC is disabled for %s, but DNSKEY or RRSIG records were found in the AXFR answer!\n", dc.Name)
	}
	c.mu.Unlock()

	// An RFC2136-compliant server must silently ignore an
	// update that inserts a non-CNAME RRset when a CNAME RR
	// with the same name is present in the zone (and
	// vice-versa). Therefore we prefer to first remove records
	// and then insert new ones.
	//
	// Compliant servers must also silently ignore an update
	// that removes the last NS record of a zone. Therefore we
	// don't want to remove all NS records before inserting a
	// new one. Then, when an update want to change a NS record,
	// we first insert a dummy NS record that we will remove
	// at the end of the batched update.

	var msgs []string
	var reports []string
	updates := []*dnsv1.Msg{}

	dummyNs1, err := dnsv1.NewRR(dc.Name + ". IN NS dnscontrol.invalid.")
	if err != nil {
		return nil, 0, err
	}
	dummyNs2, err := dnsv1.NewRR(dc.Name + ". IN NS dnscontrol.invalid.")
	if err != nil {
		return nil, 0, err
	}

	changes, actualChangeCount, err := diff2.ByRecord(foundRecords, dc, nil)
	if err != nil {
		return nil, 0, err
	}
	if changes == nil {
		return nil, 0, nil
	}

	update := new(dnsv1.Msg)
	update.SetUpdate(dc.Name + ".")

	// A DNS server should silently ignore a DDNS update that removes
	// the last NS record of a zone. Since modifying a record is
	// implemented by successively a deletion of the old record and an
	// insertion of the new one, then modifying all the NS record of a
	// zone might will fail (even if the deletion and insertion
	// are grouped in a single batched update).
	//
	// To avoid this case, we will first insert a dummy NS record,
	// that will be removed at the end of the batched updates. This
	// record needs to inserted only when all NS records are touched
	// The current implementation insert this dummy record as soon as
	// a NS record is deleted or changed.
	hasNSDeletion := hasNSDeletion(changes)

	if hasNSDeletion {
		update.Insert([]dnsv1.RR{dummyNs1})
	}

	i := 1
	appendFinalUpdate := true

	for _, change := range changes {
		switch change.Type {
		case diff2.DELETE:
			msgs = append(msgs, change.Msgs[0])
			// It's semantically invalid for any RRs to exist alongside a
			// CNAME RR
			if change.Old[0].Type == "CNAME" {
				update.RemoveName([]dnsv1.RR{change.Old[0].ToRR()})
			} else {
				update.Remove([]dnsv1.RR{change.Old[0].ToRR()})
			}
		case diff2.CREATE:
			msgs = append(msgs, change.Msgs[0])
			// It's semantically invalid for any RRs to exist alongside a
			// CNAME RR
			if change.New[0].Type == "CNAME" {
				update.RemoveName([]dnsv1.RR{change.New[0].ToRR()})
			}
			update.Insert([]dnsv1.RR{change.New[0].ToRR()})
		case diff2.CHANGE:
			msgs = append(msgs, change.Msgs[0])
			// It's semantically invalid for any RRs to exist alongside a
			// CNAME RR
			if (change.New[0].Type == "CNAME") || (change.Old[0].Type == "CNAME") {
				update.RemoveName([]dnsv1.RR{change.Old[0].ToRR()})
			} else {
				update.Remove([]dnsv1.RR{change.Old[0].ToRR()})
			}
			update.Insert([]dnsv1.RR{change.New[0].ToRR()})
		case diff2.REPORT:
			reports = append(reports, change.Msgs...)
		}

		// Chunk packets that exceed 2^14 = 16 KiB.
		// A single DNS RR can theoretically reach 64 KiB, the total packet limit.
		// This is a compromise, succeeding whenever RRs are not bigger than about 64 KiB - 16 KiB = 48 KiB.
		if update.Len() >= 2<<13 {
			updates = append(updates, update)
			update = new(dnsv1.Msg)
			update.SetUpdate(dc.Name + ".")
			appendFinalUpdate = false
			i = 1
		} else {
			appendFinalUpdate = true
			i++
		}
	}

	if hasNSDeletion {
		update.Remove([]dnsv1.RR{dummyNs2})
		appendFinalUpdate = true
	}

	if appendFinalUpdate {
		updates = append(updates, update)
	}

	returnValue := []*models.Correction{}

	if len(msgs) > 0 {
		returnValue = append(returnValue, c.BuildCorrection(dc, msgs, updates))
	}
	if len(reports) > 0 {
		returnValue = append(returnValue, c.BuildCorrection(dc, reports, nil))
	}
	return returnValue, actualChangeCount, nil
}
