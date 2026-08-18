package nexdns

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/DNSControl/dnscontrol/v5/models"
)

func TestGetZoneRecords(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/zones":
			_, _ = w.Write([]byte(`{"status":"success","data":[{"id":"z1","name":"example.com","unicode_name":"example.com"}]}`))
		case "/zones/z1":
			_, _ = w.Write([]byte(`{"status":"success","data":{"id":"z1","name":"example.com","nameservers":["ns1.example.net","ns2.example.net"]}}`))
		case "/zones/z1/records":
			_, _ = w.Write([]byte(`{"status":"success","data":[
				{"id":"r1","name":"@","type":"SOA","content":"ns1.example.net. hostmaster.example.net. 1 2 3 4 5","ttl":3600},
				{"id":"r2","name":"@","type":"NS","content":"ns1.example.net.","ttl":3600},
				{"id":"r3","name":"sub","type":"NS","content":"ns1.example.org.","ttl":3600},
				{"id":"r4","name":"www","type":"A","content":"203.0.113.10","ttl":300}
			]}`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	p := testProvider(server.URL)
	dc := models.MustNewDomainConfig("example.com")
	recs, err := p.GetZoneRecords(dc)
	if err != nil {
		t.Fatalf("GetZoneRecords() error = %v", err)
	}

	// The SOA and the apex NS are maintained by the platform and must not be
	// reported; the NS record of a delegated child must be.
	if len(recs) != 2 {
		t.Fatalf("GetZoneRecords() returned %d records, want 2: %v", len(recs), recs)
	}
	if recs[0].Type != "NS" || recs[0].GetLabel() != "sub" {
		t.Errorf("first record = %s %s, want NS sub", recs[0].Type, recs[0].GetLabel())
	}
	if recs[1].Type != "A" || recs[1].AsA().Addr.String() != "203.0.113.10" {
		t.Errorf("second record = %s %s, want A 203.0.113.10", recs[1].Type, recs[1].AsA().Addr.String())
	}
	if recs[1].Original.(apiRecord).ID != "r4" {
		t.Errorf("record id was not carried over: %v", recs[1].Original)
	}
}

func TestEnsureZoneExists(t *testing.T) {
	var created []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/zones":
			var body map[string]string
			_ = json.NewDecoder(r.Body).Decode(&body)
			created = append(created, body["name"])
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{"status":"success","data":{"id":"z9","name":"new.example.com"}}`))
		case r.URL.Path == "/zones":
			// The search matches on a substring, so a lookup of "ample.com"
			// would see this zone too.
			_, _ = w.Write([]byte(`{"status":"success","data":[{"id":"z1","name":"example.com"}]}`))
		case r.URL.Path == "/zones/z1":
			_, _ = w.Write([]byte(`{"status":"success","data":{"id":"z1","name":"example.com"}}`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	p := testProvider(server.URL)
	dc := models.MustNewDomainConfig("example.com")
	if err := p.EnsureZoneExists(dc); err != nil {
		t.Fatalf("EnsureZoneExists() error = %v", err)
	}
	if len(created) != 0 {
		t.Errorf("an existing zone was created again: %v", created)
	}

	ndc := models.MustNewDomainConfig("new.example.com")
	if err := p.EnsureZoneExists(ndc); err != nil {
		t.Fatalf("EnsureZoneExists() error = %v", err)
	}
	if len(created) != 1 || created[0] != "new.example.com" {
		t.Errorf("created = %v, want [new.example.com]", created)
	}
}

func testProvider(baseURL string) *nexdnsProvider {
	return &nexdnsProvider{
		client: newAPIClient(baseURL, "nxd_notarealtoken"),
		zones:  map[string]*apiZone{},
	}
}
