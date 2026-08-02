package axfrddns

/*

axfrddns -
  Fetch the zone with an AXFR request (RFC5936) to a given primary master, and
  push Dynamic DNS updates (RFC2136) to the same server.

  Both the AXFR request and the updates might be authentificated with
  a TSIG.

*/

import (
	"context"
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net" // Verified not used for net.IP
	"strings"
	"sync"
	"time"

	dnsv2 "codeberg.org/miekg/dns"
	"codeberg.org/miekg/dns/dnsutil"
	"github.com/DNSControl/dnscontrol/v5/models"
	"github.com/DNSControl/dnscontrol/v5/pkg/diff2"
	"github.com/DNSControl/dnscontrol/v5/pkg/dnsrr"
	"github.com/DNSControl/dnscontrol/v5/pkg/printer"
	"github.com/DNSControl/dnscontrol/v5/pkg/providers"
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
	secret []byte
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
		algo = dnsv2.HmacMD5
	case "hmac-sha1", "sha1":
		algo = dnsv2.HmacSHA1
	case "hmac-sha224", "sha224":
		algo = dnsv2.HmacSHA224
	case "hmac-sha256", "sha256":
		algo = dnsv2.HmacSHA256
	case "hmac-sha384", "sha384":
		algo = dnsv2.HmacSHA384
	case "hmac-sha512", "sha512":
		algo = dnsv2.HmacSHA512
	default:
		return nil, fmt.Errorf("unknown algorithm (%s) in AXFRDDNS.TSIG", kind)
	}
	secret, err := base64.StdEncoding.DecodeString(arr[2])
	if err != nil {
		return nil, fmt.Errorf("cannot decode Base64 secret (%s) in AXFRDDNS.TSIG", kind)
	}
	id := dnsutil.Canonical(arr[1])
	return &Key{algo: algo, id: id, secret: secret}, nil
}

// signer returns the TSIG signer for the key.
func (k *Key) signer() dnsv2.TSIGSigner {
	if k.algo == dnsv2.HmacMD5 {
		return md5Provider(k.secret)
	}
	return dnsv2.HmacTSIG{Secret: k.secret}
}

// stub returns the unsigned TSIG record to attach to a message signed with the key.
func (k *Key) stub() dnsv2.RR {
	return dnsv2.NewTSIG(k.id, k.algo, 300)
}

// GetNameservers returns the nameservers for a domain.
func (c *axfrddnsProvider) GetNameservers(domain string) ([]*models.Nameserver, error) {
	return c.nameservers, nil
}

func (c *axfrddnsProvider) getAxfrConnection() (net.Conn, error) {
	switch c.transferMode {
	case "tcp-tls":
		// RFC 9103 "DNS Zone Transfer over TLS" section 7.1 requires "dot"
		return tls.Dial("tcp", c.transferServer, &tls.Config{NextProtos: []string{"dot"}})
	case "unix":
		return net.Dial("unix", c.transferServer)
	default:
		return net.Dial("tcp", c.transferServer)
	}
}

// FetchZoneRecords gets the records of a zone and returns them in dns.RR format.
func (c *axfrddnsProvider) FetchZoneRecords(domain string) ([]dnsv2.RR, error) {
	con, err := c.getAxfrConnection()
	if err != nil {
		return nil, err
	}

	client := dnsv2.NewClient()
	client.ReadTimeout = dnsTimeout

	request := dnsv2.NewMsg(domain+".", dnsv2.TypeAXFR)
	request.RecursionDesired = false

	if c.transferKey != nil {
		request.Pseudo = []dnsv2.RR{c.transferKey.stub()}
		client.Transfer = &dnsv2.Transfer{TSIGSigner: c.transferKey.signer()}
	}

	envelope, err := client.TransferInWithConn(context.Background(), request, con)
	if err != nil {
		return nil, err
	}

	// RFC 5936 section 2.2: a complete AXFR answer ends with the SOA it began
	// with. The connection is closed on that SOA and the remaining envelopes
	// are drained.
	var rawRecords []dnsv2.RR
	var complete bool
	for msg := range envelope {
		if complete {
			continue
		}
		if msg.Error != nil {
			return nil, fmt.Errorf("[Error] AXFRDDNS: nameserver refused to transfer the zone %s: %s", domain, msg.Error)
		}
		rawRecords = append(rawRecords, msg.Answer...)
		if len(rawRecords) >= 2 && dnsv2.RRToType(rawRecords[len(rawRecords)-1]) == dnsv2.TypeSOA {
			complete = true
			con.Close()
		}
	}

	if !complete {
		return nil, fmt.Errorf("[Error] AXFRDDNS: incomplete transfer of the zone %s: the answer does not end with the SOA record", domain)
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
		switch dnsv2.RRToType(rr) {
		case dnsv2.TypeRRSIG,
			dnsv2.TypeDNSKEY,
			dnsv2.TypeCDNSKEY,
			dnsv2.TypeCDS,
			dnsv2.TypeNSEC,
			dnsv2.TypeNSEC3,
			dnsv2.TypeNSEC3PARAM,
			dnsv2.TypeZONEMD,
			65281,
			65534:
			// Ignoring DNSSec RRs, but replacing it with a single
			// "TXT" placeholder
			// Ignoring TYPE65281 Technitium Conditional Forwarder Zone Record
			// Also ignoring spurious TYPE65534, see:
			// https://bind9-users.isc.narkive.com/zX29ay0j/rndc-signing-list-not-working#post2
			if foundDNSSecRecords == nil {
				foundDNSSecRecords, err = dc.NewRecordConfig(dc.LabelFromShort(dnssecDummyLabel), 0, dnsv2.TypeTXT, dnssecDummyTxt)
				if err != nil {
					return nil, err
				}
			}
			continue
		default:
			rec, err := dnsrr.RRv2toRC(dc, rr)
			if err != nil {
				return nil, err
			}
			foundRecords = append(foundRecords, rec)
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
func (c *axfrddnsProvider) BuildCorrection(dc *models.DomainConfig, msgs []string, updates []*dnsv2.Msg) *models.Correction {
	if updates == nil {
		return &models.Correction{
			Msg: fmt.Sprintf("DDNS UPDATES to '%s' (primary master: '%s'). Changes:\n%s", dc.Name, c.master, strings.Join(msgs, "\n")),
		}
	}
	return &models.Correction{
		Msg: fmt.Sprintf("DDNS UPDATES to '%s' (primary master: '%s'). Changes:\n%s", dc.Name, c.master, strings.Join(msgs, "\n")),
		F: func() error {
			client := dnsv2.NewClient()
			client.Dialer = &net.Dialer{Timeout: dnsTimeout}
			client.ReadTimeout = dnsTimeout
			network := c.updateMode
			switch network {
			case "":
				network = "udp"
			case "tcp-tls":
				network = "tcp"
				client.TLSConfig = &tls.Config{}
			}

			var signer dnsv2.TSIGSigner
			if c.updateKey != nil {
				signer = c.updateKey.signer()
			}

			for _, update := range updates {
				option := dnsv2.TSIGOption{}
				if signer != nil {
					update.Pseudo = []dnsv2.RR{c.updateKey.stub()}
					if err := dnsv2.TSIGSign(update, signer, &option); err != nil {
						return err
					}
				}

				msg, _, err := client.Exchange(context.Background(), update, network, c.master)
				if err != nil {
					return err
				}
				if signer != nil && isSigned(msg) {
					if err := dnsv2.TSIGVerify(msg, signer, &option); err != nil {
						return err
					}
				}
				if msg.Rcode != 0 {
					return fmt.Errorf("[Error] AXFRDDNS: nameserver refused to update the zone: %s (%d)",
						dnsv2.RcodeToString[msg.Rcode],
						msg.Rcode)
				}
			}

			return nil
		},
	}
}

// isSigned reports whether msg carries a TSIG record.
func isSigned(msg *dnsv2.Msg) bool {
	for _, rr := range msg.Pseudo {
		if _, ok := rr.(*dnsv2.TSIG); ok {
			return true
		}
	}
	return false
}

// newUpdate returns an empty DDNS update whose zone section is zone, see RFC 2136 section 2.3.
func newUpdate(zone string) *dnsv2.Msg {
	update := dnsv2.NewMsg(zone, dnsv2.TypeSOA)
	update.Opcode = dnsv2.OpcodeUpdate
	update.RecursionDesired = false
	return update
}

// insert adds rr to the update section as an addition, see RFC 2136 section 2.5.1.
func insert(update *dnsv2.Msg, rr dnsv2.RR) {
	rr.Header().Class = dnsv2.ClassINET
	update.Ns = append(update.Ns, rr)
}

// remove adds rr to the update section as the deletion of that single RR, see
// RFC 2136 section 2.5.4.
func remove(update *dnsv2.Msg, rr dnsv2.RR) {
	hdr := rr.Header()
	hdr.Class = dnsv2.ClassNONE
	hdr.TTL = 0
	update.Ns = append(update.Ns, rr)
}

// removeName adds rr to the update section as the deletion of every RRset owned
// by its name, see RFC 2136 section 2.5.3.
func removeName(update *dnsv2.Msg, rr dnsv2.RR) {
	update.Ns = append(update.Ns, &dnsv2.ANY{Hdr: dnsv2.Header{Name: rr.Header().Name, Class: dnsv2.ClassANY}})
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
	updates := []*dnsv2.Msg{}

	dummyNs1, err := dnsv2.New(dc.Name + ". IN NS dnscontrol.invalid.")
	if err != nil {
		return nil, 0, err
	}
	dummyNs2, err := dnsv2.New(dc.Name + ". IN NS dnscontrol.invalid.")
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

	update := newUpdate(dc.Name + ".")

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
		insert(update, dummyNs1)
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
				removeName(update, change.Old[0].ToRRv2())
			} else {
				remove(update, change.Old[0].ToRRv2())
			}
		case diff2.CREATE:
			msgs = append(msgs, change.Msgs[0])
			// It's semantically invalid for any RRs to exist alongside a
			// CNAME RR
			if change.New[0].Type == "CNAME" {
				removeName(update, change.New[0].ToRRv2())
			}
			insert(update, change.New[0].ToRRv2())
		case diff2.CHANGE:
			msgs = append(msgs, change.Msgs[0])
			// It's semantically invalid for any RRs to exist alongside a
			// CNAME RR
			if (change.New[0].Type == "CNAME") || (change.Old[0].Type == "CNAME") {
				removeName(update, change.Old[0].ToRRv2())
			} else {
				remove(update, change.Old[0].ToRRv2())
			}
			insert(update, change.New[0].ToRRv2())
		case diff2.REPORT:
			reports = append(reports, change.Msgs...)
		}

		// Chunk packets that exceed 2^14 = 16 KiB.
		// A single DNS RR can theoretically reach 64 KiB, the total packet limit.
		// This is a compromise, succeeding whenever RRs are not bigger than about 64 KiB - 16 KiB = 48 KiB.
		if update.Len() >= 2<<13 {
			updates = append(updates, update)
			update = newUpdate(dc.Name + ".")
			appendFinalUpdate = false
			i = 1
		} else {
			appendFinalUpdate = true
			i++
		}
	}

	if hasNSDeletion {
		remove(update, dummyNs2)
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
