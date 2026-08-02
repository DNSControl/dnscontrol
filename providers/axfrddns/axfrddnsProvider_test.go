package axfrddns

import (
	"context"
	"fmt"
	"io"
	"strings"
	"sync"
	"testing"
	"time"

	dnsv2 "codeberg.org/miekg/dns"
	"codeberg.org/miekg/dns/dnstest"
	"codeberg.org/miekg/dns/dnsutil"
	"github.com/DNSControl/dnscontrol/v5/models"
)

const (
	testZone   = "example.com."
	testSecret = "c2VjcmV0Cg=="
	otherKey   = "b3RoZXJzZWNyZXQK"
)

var testZoneContents = []string{
	"example.com. 3600 IN SOA ns1.example.com. hostmaster.example.com. 1 7200 3600 1209600 3600",
	"example.com. 3600 IN NS ns1.example.com.",
	"www.example.com. 300 IN A 192.0.2.1",
	"www.example.com. 300 IN AAAA 2001:db8::1",
	"example.com. 3600 IN MX 10 mail.example.com.",
	"txt.example.com. 60 IN TXT \"hello world\"",
	"example.com. 3600 IN SOA ns1.example.com. hostmaster.example.com. 1 7200 3600 1209600 3600",
}

func mustRR(t *testing.T, s string) dnsv2.RR {
	t.Helper()
	rr, err := dnsv2.New(s)
	if err != nil {
		t.Fatalf("dnsv2.New(%q): %v", s, err)
	}
	return rr
}

func mustKey(t *testing.T, raw string) *Key {
	t.Helper()
	key, err := readKey(raw, "transfer-key")
	if err != nil {
		t.Fatalf("readKey(%q): %v", raw, err)
	}
	return key
}

// serveAXFR starts a nameserver on localhost that answers an AXFR for testZone
// with contents in a single message, signing it with signer when it is not nil,
// and closes the connection afterwards. It returns the "host:port" the server
// listens on.
func serveAXFR(t *testing.T, contents []string, signer dnsv2.TSIGSigner) string {
	t.Helper()
	return newAXFRServer(t, [][]string{contents}, signer, true)
}

// newAXFRServer starts a nameserver on localhost that answers an AXFR for
// testZone with one message per element of messages, signing each with signer
// when it is not nil and closing the connection afterwards when closeConn is
// set. It returns the "host:port" the server listens on.
func newAXFRServer(t *testing.T, messages [][]string, signer dnsv2.TSIGSigner, closeConn bool) string {
	t.Helper()

	answers := make([][]dnsv2.RR, 0, len(messages))
	for _, contents := range messages {
		answer := make([]dnsv2.RR, 0, len(contents))
		for _, s := range contents {
			answer = append(answer, mustRR(t, s))
		}
		answers = append(answers, answer)
	}

	mux := dnsv2.NewServeMux()
	mux.HandleFunc(testZone, func(_ context.Context, w dnsv2.ResponseWriter, r *dnsv2.Msg) {
		if err := r.Unpack(); err != nil {
			t.Errorf("server: unpack request: %v", err)
			return
		}
		w.Hijack()
		if signer != nil && !isSigned(r) {
			t.Error("server: AXFR request arrived without a TSIG")
			w.Close()
			return
		}

		client := dnsv2.NewClient()
		if signer != nil {
			client.Transfer = &dnsv2.Transfer{TSIGSigner: signer}
		}

		env := make(chan *dnsv2.Envelope)
		var wg sync.WaitGroup
		wg.Go(func() {
			if err := client.TransferOut(w, r, env); err != nil {
				t.Errorf("server: TransferOut: %v", err)
			}
			if closeConn {
				w.Close()
			}
		})
		for _, answer := range answers {
			env <- &dnsv2.Envelope{Answer: answer}
		}
		close(env)
		wg.Wait()
	})

	cancel, addr, err := dnstest.TCPServer("127.0.0.1:0", func(s *dnsv2.Server) { s.Handler = mux })
	if err != nil {
		t.Fatalf("starting AXFR server: %v", err)
	}
	t.Cleanup(cancel)
	return addr
}

// serveUpdate starts a nameserver on localhost that answers every DDNS update
// for testZone with rcode. The request is verified with requestKey when it is
// not nil, and the reply is signed with replyKey when it is not nil. Received
// updates are sent to the returned channel.
func serveUpdate(t *testing.T, rcode uint16, requestKey, replyKey dnsv2.TSIGSigner) (string, <-chan *dnsv2.Msg) {
	t.Helper()

	received := make(chan *dnsv2.Msg, 8)

	mux := dnsv2.NewServeMux()
	mux.HandleFunc(testZone, func(_ context.Context, w dnsv2.ResponseWriter, r *dnsv2.Msg) {
		if err := r.Unpack(); err != nil {
			t.Errorf("server: unpack update: %v", err)
			return
		}

		option := dnsv2.TSIGOption{}
		var stub dnsv2.RR
		if requestKey != nil {
			if len(r.Pseudo) == 0 {
				t.Error("server: update arrived without a TSIG")
				return
			}
			stub = r.Pseudo[len(r.Pseudo)-1]
			if err := dnsv2.TSIGVerify(r, requestKey, &option); err != nil {
				t.Errorf("server: TSIGVerify: %v", err)
				return
			}
		}
		received <- r

		reply := new(dnsv2.Msg)
		dnsutil.SetReply(reply, r)
		reply.Rcode = rcode
		if replyKey != nil {
			reply.Pseudo = []dnsv2.RR{stub}
			if err := reply.Pack(); err != nil {
				t.Errorf("server: pack reply: %v", err)
				return
			}
			if err := dnsv2.TSIGSign(reply, replyKey, &option); err != nil {
				t.Errorf("server: TSIGSign: %v", err)
				return
			}
		}
		if _, err := io.Copy(w, reply); err != nil {
			t.Errorf("server: write reply: %v", err)
		}
	})

	cancel, addr, err := dnstest.TCPServer("127.0.0.1:0", func(s *dnsv2.Server) { s.Handler = mux })
	if err != nil {
		t.Fatalf("starting update server: %v", err)
	}
	t.Cleanup(cancel)
	return addr, received
}

func TestReadKey(t *testing.T) {
	tests := []struct {
		name     string
		raw      string
		wantAlgo string
		wantID   string
		wantErr  string
	}{
		{"hmac-md5", "hmac-md5:key:" + testSecret, dnsv2.HmacMD5, "key.", ""},
		{"md5", "md5:key:" + testSecret, dnsv2.HmacMD5, "key.", ""},
		{"hmac-sha1", "hmac-sha1:key:" + testSecret, dnsv2.HmacSHA1, "key.", ""},
		{"sha1", "sha1:key:" + testSecret, dnsv2.HmacSHA1, "key.", ""},
		{"hmac-sha224", "hmac-sha224:key:" + testSecret, dnsv2.HmacSHA224, "key.", ""},
		{"sha224", "sha224:key:" + testSecret, dnsv2.HmacSHA224, "key.", ""},
		{"hmac-sha256", "hmac-sha256:KEY.example.com:" + testSecret, dnsv2.HmacSHA256, "key.example.com.", ""},
		{"sha256", "sha256:key:" + testSecret, dnsv2.HmacSHA256, "key.", ""},
		{"hmac-sha384", "hmac-sha384:key:" + testSecret, dnsv2.HmacSHA384, "key.", ""},
		{"sha384", "sha384:key:" + testSecret, dnsv2.HmacSHA384, "key.", ""},
		{"hmac-sha512", "hmac-sha512:key:" + testSecret, dnsv2.HmacSHA512, "key.", ""},
		{"sha512", "sha512:key:" + testSecret, dnsv2.HmacSHA512, "key.", ""},
		{"empty", "", "", "", ""},
		{"too few fields", "hmac-sha256:key", "", "", "invalid key format"},
		{"unknown algorithm", "hmac-sha3:key:" + testSecret, "", "", "unknown algorithm"},
		{"bad base64", "hmac-sha256:key:!!!", "", "", "cannot decode Base64 secret"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			key, err := readKey(tt.raw, "transfer-key")
			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("readKey() error = %v, want it to contain %q", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("readKey() error = %v", err)
			}
			if tt.raw == "" {
				if key != nil {
					t.Fatalf("readKey(\"\") = %v, want nil", key)
				}
				return
			}
			if key.algo != tt.wantAlgo {
				t.Errorf("algo = %q, want %q", key.algo, tt.wantAlgo)
			}
			if key.id != tt.wantID {
				t.Errorf("id = %q, want %q", key.id, tt.wantID)
			}
			if string(key.secret) != "secret\n" {
				t.Errorf("secret = %q, want the decoded secret", key.secret)
			}
		})
	}
}

func TestKeySigner(t *testing.T) {
	if _, ok := mustKey(t, "hmac-md5:key:"+testSecret).signer().(md5Provider); !ok {
		t.Error("hmac-md5 key is not signed by md5Provider")
	}
	if _, ok := mustKey(t, "hmac-sha256:key:"+testSecret).signer().(dnsv2.HmacTSIG); !ok {
		t.Error("hmac-sha256 key is not signed by dnsv2.HmacTSIG")
	}
}

func TestFetchZoneRecords(t *testing.T) {
	tests := []struct {
		name       string
		serverKey  string
		clientKey  string
		wantErrSub string
	}{
		{name: "unauthenticated"},
		{name: "hmac-sha256", serverKey: "hmac-sha256:key:" + testSecret, clientKey: "hmac-sha256:key:" + testSecret},
		{name: "hmac-sha512", serverKey: "hmac-sha512:key:" + testSecret, clientKey: "hmac-sha512:key:" + testSecret},
		{name: "hmac-md5", serverKey: "hmac-md5:key:" + testSecret, clientKey: "hmac-md5:key:" + testSecret},
		{
			name:       "wrong secret is rejected",
			serverKey:  "hmac-sha256:key:" + testSecret,
			clientKey:  "hmac-sha256:key:" + otherKey,
			wantErrSub: "bad signature",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var serverSigner dnsv2.TSIGSigner
			if tt.serverKey != "" {
				serverSigner = mustKey(t, tt.serverKey).signer()
			}
			addr := serveAXFR(t, testZoneContents, serverSigner)

			api := &axfrddnsProvider{transferServer: addr, transferMode: "tcp"}
			if tt.clientKey != "" {
				api.transferKey = mustKey(t, tt.clientKey)
			}

			got, err := api.FetchZoneRecords("example.com")
			if tt.wantErrSub != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErrSub) {
					t.Fatalf("FetchZoneRecords() error = %v, want it to contain %q", err, tt.wantErrSub)
				}
				return
			}
			if err != nil {
				t.Fatalf("FetchZoneRecords() error = %v", err)
			}
			if len(got) != len(testZoneContents) {
				t.Fatalf("FetchZoneRecords() returned %d records, want %d", len(got), len(testZoneContents))
			}
			for i, rr := range got {
				if want := mustRR(t, testZoneContents[i]).String(); rr.String() != want {
					t.Errorf("record %d = %q, want %q", i, rr.String(), want)
				}
			}
		})
	}
}

func TestFetchZoneRecordsDoesNotStopOnTheOpeningSOA(t *testing.T) {
	addr := newAXFRServer(t, [][]string{testZoneContents[:1], testZoneContents[1:]}, nil, true)
	api := &axfrddnsProvider{transferServer: addr, transferMode: "tcp"}

	got, err := api.FetchZoneRecords("example.com")
	if err != nil {
		t.Fatalf("FetchZoneRecords() error = %v", err)
	}
	if len(got) != len(testZoneContents) {
		t.Fatalf("FetchZoneRecords() returned %d records, want %d", len(got), len(testZoneContents))
	}
	for i, rr := range got {
		if want := mustRR(t, testZoneContents[i]).String(); rr.String() != want {
			t.Errorf("record %d = %q, want %q", i, rr.String(), want)
		}
	}
}

func TestFetchZoneRecordsReturnsOnTheClosingSOA(t *testing.T) {
	addr := newAXFRServer(t, [][]string{testZoneContents}, nil, false)
	api := &axfrddnsProvider{transferServer: addr, transferMode: "tcp"}

	type result struct {
		records []dnsv2.RR
		err     error
	}
	done := make(chan result, 1)
	go func() {
		records, err := api.FetchZoneRecords("example.com")
		done <- result{records: records, err: err}
	}()

	select {
	case got := <-done:
		if got.err != nil {
			t.Fatalf("FetchZoneRecords() error = %v", got.err)
		}
		if len(got.records) != len(testZoneContents) {
			t.Fatalf("FetchZoneRecords() returned %d records, want %d", len(got.records), len(testZoneContents))
		}
	case <-time.After(5 * time.Second):
		t.Fatal("FetchZoneRecords() did not return while the nameserver held the connection open")
	}
}

func TestFetchZoneRecordsReportsRefusedTransfer(t *testing.T) {
	mux := dnsv2.NewServeMux()
	mux.HandleFunc(testZone, func(_ context.Context, w dnsv2.ResponseWriter, r *dnsv2.Msg) {
		if err := r.Unpack(); err != nil {
			t.Errorf("server: unpack request: %v", err)
			return
		}
		reply := new(dnsv2.Msg)
		dnsutil.SetReply(reply, r)
		reply.Rcode = dnsv2.RcodeNotAuth
		if _, err := io.Copy(w, reply); err != nil {
			t.Errorf("server: write reply: %v", err)
		}
	})
	cancel, addr, err := dnstest.TCPServer("127.0.0.1:0", func(s *dnsv2.Server) { s.Handler = mux })
	if err != nil {
		t.Fatalf("starting AXFR server: %v", err)
	}
	t.Cleanup(cancel)

	api := &axfrddnsProvider{transferServer: addr, transferMode: "tcp"}
	_, err = api.FetchZoneRecords("example.com")

	want := "[Error] AXFRDDNS: nameserver refused to transfer the zone example.com: dns: bad rcode: NOTAUTH"
	if err == nil || err.Error() != want {
		t.Fatalf("FetchZoneRecords() error = %v, want %q", err, want)
	}
}

func TestFetchZoneRecordsRejectsTruncatedTransfer(t *testing.T) {
	addr := serveAXFR(t, testZoneContents[:len(testZoneContents)-1], nil)
	api := &axfrddnsProvider{transferServer: addr, transferMode: "tcp"}

	_, err := api.FetchZoneRecords("example.com")
	if err == nil || !strings.Contains(err.Error(), "does not end with the SOA record") {
		t.Fatalf("FetchZoneRecords() error = %v, want it to report an incomplete transfer", err)
	}
}

func TestGetZoneRecordsDropsTrailingSOA(t *testing.T) {
	addr := serveAXFR(t, testZoneContents, nil)
	api := &axfrddnsProvider{
		transferServer:   addr,
		transferMode:     "tcp",
		hasDnssecRecords: map[string]bool{},
	}

	got, err := api.GetZoneRecords(models.MustNewDomainConfig("example.com"))
	if err != nil {
		t.Fatalf("GetZoneRecords() error = %v", err)
	}

	want := []string{
		"@ 3600 IN SOA ns1.example.com. hostmaster.example.com. 1 7200 3600 1209600 3600",
		"@ 3600 IN NS ns1.example.com.",
		"www 300 IN A 192.0.2.1",
		"www 300 IN AAAA 2001:db8::1",
		"@ 3600 IN MX 10 mail.example.com.",
		"txt 60 IN TXT \"hello world\"",
	}
	if len(got) != len(want) {
		t.Fatalf("GetZoneRecords() returned %d records, want %d", len(got), len(want))
	}
	for i, rc := range got {
		if rc.LineString() != want[i] {
			t.Errorf("record %d = %q, want %q", i, rc.LineString(), want[i])
		}
	}
}

func TestGetZoneRecordsReplacesDNSSECRecordsWithPlaceholder(t *testing.T) {
	contents := []string{
		"example.com. 3600 IN SOA ns1.example.com. hostmaster.example.com. 1 7200 3600 1209600 3600",
		"example.com. 3600 IN NS ns1.example.com.",
		"example.com. 3600 IN NSEC3PARAM 1 0 0 -",
		"example.com. 3600 IN SOA ns1.example.com. hostmaster.example.com. 1 7200 3600 1209600 3600",
	}
	addr := serveAXFR(t, contents, nil)
	api := &axfrddnsProvider{
		transferServer:   addr,
		transferMode:     "tcp",
		hasDnssecRecords: map[string]bool{},
	}

	got, err := api.GetZoneRecords(models.MustNewDomainConfig("example.com"))
	if err != nil {
		t.Fatalf("GetZoneRecords() error = %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("GetZoneRecords() returned %d records, want 2", len(got))
	}
	if !api.hasDnssecRecords["example.com"] {
		t.Error("GetZoneRecords() did not record that the zone is signed")
	}
}

func TestNewUpdateMessage(t *testing.T) {
	update := newUpdate(testZone)

	if update.Opcode != dnsv2.OpcodeUpdate {
		t.Errorf("Opcode = %d, want %d", update.Opcode, dnsv2.OpcodeUpdate)
	}
	if update.Response {
		t.Error("Response is set")
	}
	if update.RecursionDesired {
		t.Error("RecursionDesired is set")
	}
	if len(update.Question) != 1 {
		t.Fatalf("zone section holds %d entries, want 1", len(update.Question))
	}
	zone := update.Question[0]
	if zone.Header().Name != testZone {
		t.Errorf("zone name = %q, want %q", zone.Header().Name, testZone)
	}
	if got := dnsv2.RRToType(zone); got != dnsv2.TypeSOA {
		t.Errorf("zone type = %d, want %d", got, dnsv2.TypeSOA)
	}
	if zone.Header().Class != dnsv2.ClassINET {
		t.Errorf("zone class = %d, want %d", zone.Header().Class, dnsv2.ClassINET)
	}
}

func TestUpdateSectionEncoding(t *testing.T) {
	rr := "www.example.com. 300 IN A 192.0.2.1"

	tests := []struct {
		name      string
		build     func(*dnsv2.Msg, dnsv2.RR)
		wantClass uint16
		wantTTL   uint32
		wantType  uint16
	}{
		{"insert", insert, dnsv2.ClassINET, 300, dnsv2.TypeA},
		{"remove", remove, dnsv2.ClassNONE, 0, dnsv2.TypeA},
		{"removeName", removeName, dnsv2.ClassANY, 0, dnsv2.TypeANY},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			update := newUpdate(testZone)
			tt.build(update, mustRR(t, rr))

			if len(update.Ns) != 1 {
				t.Fatalf("update section holds %d entries, want 1", len(update.Ns))
			}
			got := update.Ns[0]
			if got.Header().Name != "www.example.com." {
				t.Errorf("name = %q, want %q", got.Header().Name, "www.example.com.")
			}
			if got.Header().Class != tt.wantClass {
				t.Errorf("class = %d, want %d", got.Header().Class, tt.wantClass)
			}
			if got.Header().TTL != tt.wantTTL {
				t.Errorf("ttl = %d, want %d", got.Header().TTL, tt.wantTTL)
			}
			if n := dnsv2.RRToType(got); n != tt.wantType {
				t.Errorf("type = %d, want %d", n, tt.wantType)
			}
			if err := update.Pack(); err != nil {
				t.Fatalf("Pack() error = %v", err)
			}
		})
	}
}

func TestGetZoneRecordsCorrections(t *testing.T) {
	addr, received := serveUpdate(t, dnsv2.RcodeSuccess, nil, nil)

	dc := models.MustNewDomainConfig("example.com")
	dc.Records = models.Records{
		dc.MustNewRecordConfig("www", 300, dnsv2.TypeA, "192.0.2.9"),
		dc.MustNewRecordConfig("alias", 300, dnsv2.TypeCNAME, "www.example.com."),
	}
	foundRecords := models.Records{
		dc.MustNewRecordConfig("old", 300, dnsv2.TypeA, "192.0.2.8"),
	}

	api := &axfrddnsProvider{master: addr, updateMode: "tcp", hasDnssecRecords: map[string]bool{}}
	corrections, changeCount, err := api.GetZoneRecordsCorrections(dc, foundRecords)
	if err != nil {
		t.Fatalf("GetZoneRecordsCorrections() error = %v", err)
	}
	if changeCount != 3 {
		t.Errorf("changeCount = %d, want 3", changeCount)
	}
	if len(corrections) != 1 {
		t.Fatalf("GetZoneRecordsCorrections() returned %d corrections, want 1", len(corrections))
	}
	if err := corrections[0].F(); err != nil {
		t.Fatalf("correction error = %v", err)
	}

	got := <-received
	want := []struct {
		name     string
		ttl      uint32
		class    uint16
		rrtype   uint16
		rrstring string
	}{
		{"old.example.com.", 0, dnsv2.ClassNONE, dnsv2.TypeA, "old.example.com.\t0\tNONE\tA\t192.0.2.8"},
		{"www.example.com.", 300, dnsv2.ClassINET, dnsv2.TypeA, "www.example.com.\t300\tIN\tA\t192.0.2.9"},
		{"alias.example.com.", 0, dnsv2.ClassANY, dnsv2.TypeANY, "alias.example.com.\t0\tANY"},
		{"alias.example.com.", 300, dnsv2.ClassINET, dnsv2.TypeCNAME, "alias.example.com.\t300\tIN\tCNAME\twww.example.com."},
	}
	if len(got.Ns) != len(want) {
		t.Fatalf("update section holds %d entries, want %d", len(got.Ns), len(want))
	}
	for i, rr := range got.Ns {
		hdr := rr.Header()
		if hdr.Name != want[i].name || hdr.TTL != want[i].ttl || hdr.Class != want[i].class {
			t.Errorf("update section entry %d header = %s %d class %d, want %s %d class %d",
				i, hdr.Name, hdr.TTL, hdr.Class, want[i].name, want[i].ttl, want[i].class)
		}
		if n := dnsv2.RRToType(rr); n != want[i].rrtype {
			t.Errorf("update section entry %d type = %d, want %d", i, n, want[i].rrtype)
		}
		if rr.String() != want[i].rrstring {
			t.Errorf("update section entry %d = %q, want %q", i, rr.String(), want[i].rrstring)
		}
	}
}

func TestGetZoneRecordsCorrectionsChunksLargeUpdates(t *testing.T) {
	addr, received := serveUpdate(t, dnsv2.RcodeSuccess, nil, nil)

	const recordCount = 200
	dc := models.MustNewDomainConfig("example.com")
	for i := range recordCount {
		dc.Records = append(dc.Records,
			dc.MustNewRecordConfig(fmt.Sprintf("txt%d", i), 300, dnsv2.TypeTXT, strings.Repeat("x", 255)))
	}

	api := &axfrddnsProvider{master: addr, updateMode: "tcp", hasDnssecRecords: map[string]bool{}}
	corrections, _, err := api.GetZoneRecordsCorrections(dc, models.Records{})
	if err != nil {
		t.Fatalf("GetZoneRecordsCorrections() error = %v", err)
	}
	if len(corrections) != 1 {
		t.Fatalf("GetZoneRecordsCorrections() returned %d corrections, want 1", len(corrections))
	}
	if err := corrections[0].F(); err != nil {
		t.Fatalf("correction error = %v", err)
	}

	messages, inserted := 0, 0
	for inserted < recordCount {
		select {
		case update := <-received:
			messages++
			inserted += len(update.Ns)
		case <-time.After(5 * time.Second):
			t.Fatalf("received %d of %d records in %d update message(s)", inserted, recordCount, messages)
		}
	}
	if messages < 2 {
		t.Errorf("the update was sent as %d message(s), want it split over several", messages)
	}
}

func TestBuildCorrection(t *testing.T) {
	sha256Key := "hmac-sha256:key:" + testSecret

	tests := []struct {
		name       string
		key        string
		replyKey   string
		unsigned   bool
		rcode      uint16
		wantErrSub string
	}{
		{name: "unauthenticated"},
		{name: "hmac-sha256", key: sha256Key},
		{name: "hmac-md5", key: "hmac-md5:key:" + testSecret},
		{
			name:       "refused update is reported",
			rcode:      dnsv2.RcodeNotAuth,
			wantErrSub: "nameserver refused to update the zone: NOTAUTH (9)",
		},
		{
			name:       "reply signed with the wrong key is rejected",
			key:        sha256Key,
			replyKey:   "hmac-sha256:key:" + otherKey,
			wantErrSub: "bad signature",
		},
		{
			name:     "unsigned reply is accepted",
			key:      sha256Key,
			unsigned: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var requestKey, replyKey dnsv2.TSIGSigner
			if tt.key != "" {
				requestKey = mustKey(t, tt.key).signer()
				replyKey = requestKey
			}
			if tt.replyKey != "" {
				replyKey = mustKey(t, tt.replyKey).signer()
			}
			if tt.unsigned {
				replyKey = nil
			}
			addr, received := serveUpdate(t, tt.rcode, requestKey, replyKey)

			api := &axfrddnsProvider{master: addr, updateMode: "tcp"}
			if tt.key != "" {
				api.updateKey = mustKey(t, tt.key)
			}

			update := newUpdate(testZone)
			insert(update, mustRR(t, "www.example.com. 300 IN A 192.0.2.1"))
			remove(update, mustRR(t, "old.example.com. 300 IN A 192.0.2.2"))

			dc := models.MustNewDomainConfig("example.com")
			err := api.BuildCorrection(dc, []string{"+ A www"}, []*dnsv2.Msg{update}).F()

			if tt.wantErrSub != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErrSub) {
					t.Fatalf("correction error = %v, want it to contain %q", err, tt.wantErrSub)
				}
				return
			}
			if err != nil {
				t.Fatalf("correction error = %v", err)
			}

			got := <-received
			if got.Opcode != dnsv2.OpcodeUpdate {
				t.Errorf("Opcode = %d, want %d", got.Opcode, dnsv2.OpcodeUpdate)
			}
			if len(got.Question) != 1 || got.Question[0].Header().Name != testZone {
				t.Fatalf("zone section = %v, want a single %q entry", got.Question, testZone)
			}
			want := []string{
				"www.example.com.\t300\tIN\tA\t192.0.2.1",
				"old.example.com.\t0\tNONE\tA\t192.0.2.2",
			}
			if len(got.Ns) != len(want) {
				t.Fatalf("update section holds %d entries, want %d", len(got.Ns), len(want))
			}
			for i, rr := range got.Ns {
				if rr.String() != want[i] {
					t.Errorf("update section entry %d = %q, want %q", i, rr.String(), want[i])
				}
			}
		})
	}
}
