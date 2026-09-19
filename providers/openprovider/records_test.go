package openprovider

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/DNSControl/dnscontrol/v5/models"
)

func TestGetZoneRecordsAndNameservers(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/auth/login":
			writeJSON(t, w, http.StatusOK, `{"code":0,"data":{"token":"token"},"desc":""}`)
		case "/dns/zones/example.com":
			writeJSON(t, w, http.StatusOK, `{"code":0,"data":{"id":42,"name":"example.com"},"desc":""}`)
		case "/dns/zones/example.com/records":
			if r.URL.Query().Get("zone_provider") != "openprovider" {
				t.Error("record listing did not select the standard OpenProvider DNS service")
			}
			writeJSON(t, w, http.StatusOK, `{"code":0,"data":{"total":5,"results":[
				{"name":"","type":"SOA","value":"ns1.openprovider.nl. hostmaster.example.com. 1 2 3 4 5","ttl":86400},
				{"name":"","type":"NS","value":"ns2.openprovider.be.","ttl":86400},
				{"name":"example.com","type":"NS","value":"ns1.openprovider.nl.","ttl":86400},
				{"name":"example.com.","type":"NS","value":"ns3.openprovider.eu.","ttl":86400},
				{"name":"www","type":"A","value":"192.0.2.10","ttl":600}
			]},"desc":""}`)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	provider := &openproviderProvider{client: testAPIClient(t, server.URL)}
	records, err := provider.GetZoneRecords(&models.DomainConfig{Name: testOrigin})
	if err != nil {
		t.Fatalf("GetZoneRecords: %v", err)
	}
	if len(records) != 1 || records[0].Type != "A" || records[0].GetLabel() != "www" {
		t.Fatalf("records = %#v", records)
	}

	nameservers, err := provider.GetNameservers(testOrigin)
	if err != nil {
		t.Fatalf("GetNameservers: %v", err)
	}
	want := []string{"ns1.openprovider.nl", "ns2.openprovider.be", "ns3.openprovider.eu"}
	if len(nameservers) != len(want) {
		t.Fatalf("nameservers = %v", nameservers)
	}
	for i := range want {
		if nameservers[i].Name != want[i] {
			t.Errorf("nameserver[%d] = %q, want %q", i, nameservers[i].Name, want[i])
		}
	}
}

func TestCorrectionPayloads(t *testing.T) {
	tests := []struct {
		name        string
		existing    models.Records
		desired     models.Records
		wantAction  string
		wantRecords int
		wantCount   int
	}{
		{
			name:        "create",
			desired:     models.Records{makeRecord(t, "A", "new", "192.0.2.1")},
			wantAction:  "add",
			wantRecords: 1,
			wantCount:   1,
		},
		{
			name: "add record while changing existing TTL",
			existing: models.Records{mustNativeRecord(t, apiRecord{
				Name: "www.example.com", Type: "A", Value: "192.0.2.1", TTL: 600,
			})},
			desired: models.Records{
				func() *models.RecordConfig {
					record := makeRecord(t, "A", "www", "192.0.2.1")
					record.TTL = 700
					return record
				}(),
				func() *models.RecordConfig {
					record := makeRecord(t, "A", "www", "192.0.2.2")
					record.TTL = 700
					return record
				}(),
			},
			wantAction:  "mixed",
			wantRecords: 2,
			wantCount:   2,
		},
		{
			name: "create multiple TXT records at one name",
			desired: models.Records{
				makeRecord(t, "TXT", "_acme-challenge", "first"),
				makeRecord(t, "TXT", "_acme-challenge", "second"),
			},
			wantAction:  "add",
			wantRecords: 2,
			wantCount:   2,
		},
		{
			name: "update",
			existing: models.Records{mustNativeRecord(t, apiRecord{
				Name: "change.example.com", Type: "A", Value: "192.0.2.1", TTL: int(minimumTTL),
			})},
			desired:     models.Records{makeRecord(t, "A", "change", "192.0.2.2")},
			wantAction:  "update",
			wantRecords: 1,
			wantCount:   1,
		},
		{
			name: "delete",
			existing: models.Records{mustNativeRecord(t, apiRecord{
				Name: "old.example.com", Type: "A", Value: "192.0.2.1", TTL: int(minimumTTL),
			})},
			wantAction:  "remove",
			wantRecords: 1,
			wantCount:   1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var updates []updateZoneRequest
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				switch {
				case r.URL.Path == "/auth/login":
					writeJSON(t, w, http.StatusOK, `{"code":0,"data":{"token":"token"},"desc":""}`)
				case r.Method == http.MethodGet && r.URL.Path == "/dns/zones/example.com":
					writeJSON(t, w, http.StatusOK, `{"code":0,"data":{"id":42,"name":"example.com"},"desc":""}`)
				case r.Method == http.MethodPut && r.URL.Path == "/dns/zones/example.com":
					var update updateZoneRequest
					if err := json.NewDecoder(r.Body).Decode(&update); err != nil {
						t.Errorf("decode update: %v", err)
					}
					updates = append(updates, update)
					writeJSON(t, w, http.StatusOK, `{"code":0,"data":{"success":true},"desc":""}`)
				default:
					http.NotFound(w, r)
				}
			}))
			defer server.Close()

			provider := &openproviderProvider{client: testAPIClient(t, server.URL)}
			dc := &models.DomainConfig{Name: testOrigin, Records: tt.desired}
			corrections, count, err := provider.GetZoneRecordsCorrections(dc, tt.existing)
			if err != nil {
				t.Fatalf("GetZoneRecordsCorrections: %v", err)
			}
			if count != tt.wantCount {
				t.Fatalf("count=%d corrections=%d", count, len(corrections))
			}
			for _, correction := range corrections {
				if err := correction.F(); err != nil {
					t.Fatalf("apply correction: %v", err)
				}
			}
			for _, update := range updates {
				if update.ID != 42 || update.Name != testOrigin {
					t.Errorf("zone identity = %d %q", update.ID, update.Name)
				}
			}
			switch tt.wantAction {
			case "add":
				count := 0
				for _, update := range updates {
					count += len(update.Records.Add)
				}
				if count != tt.wantRecords {
					t.Errorf("add payload = %#v", updates)
				}
			case "update":
				if len(updates) != 1 || len(updates[0].Records.Update) != 1 || updates[0].Records.Update[0].Original.Value != "192.0.2.1" || updates[0].Records.Update[0].Record.Value != "192.0.2.2" {
					t.Errorf("update payload = %#v", updates)
				}
			case "remove":
				if len(updates) != 1 || len(updates[0].Records.Remove) != 1 || updates[0].Records.Remove[0].Name != "old" {
					t.Errorf("remove payload = %#v", updates)
				}
			case "mixed":
				if len(updates) != 2 || len(updates[0].Records.Update) != 1 || len(updates[1].Records.Add) != 1 {
					t.Errorf("mixed payload = %#v", updates)
				}
			}
		})
	}
}

func TestProviderSpecificNormalizationDoesNotMutateDomainConfig(t *testing.T) {
	ns := makeRecord(t, "NS", "@", "other.example.net.")
	low := makeRecord(t, "A", "www", "192.0.2.1")
	low.TTL = 300
	dc := &models.DomainConfig{Name: testOrigin, Records: models.Records{ns, low}}

	originalRecords := append(models.Records(nil), dc.Records...)
	providerDomainConfig(dc)
	if len(dc.Records) != len(originalRecords) || dc.Records[0] != originalRecords[0] || dc.Records[1].TTL != 300 {
		t.Fatalf("provider-specific copy changed the original domain config: %#v", dc.Records)
	}
}

func TestMXPriorityZeroIsSerialized(t *testing.T) {
	record := apiRecord{Type: "MX", Value: "mail.example.com", TTL: 900}
	payload, err := json.Marshal(record)
	if err != nil {
		t.Fatal(err)
	}
	if string(payload) != `{"type":"MX","value":"mail.example.com","prio":0,"ttl":900}` {
		t.Errorf("serialized MX record = %s", payload)
	}
}

func mustNativeRecord(t *testing.T, native apiRecord) *models.RecordConfig {
	t.Helper()
	record, err := toRecordConfig(native, testOrigin)
	if err != nil {
		t.Fatal(err)
	}
	return record
}

func TestListZones(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/auth/login" {
			writeJSON(t, w, http.StatusOK, `{"code":0,"data":{"token":"token"},"desc":""}`)
			return
		}
		writeJSON(t, w, http.StatusOK, `{"code":0,"data":{"total":2,"results":[{"id":2,"name":"z.example."},{"id":1,"name":"a.example"}]},"desc":""}`)
	}))
	defer server.Close()

	provider := &openproviderProvider{client: testAPIClient(t, server.URL)}
	zones, err := provider.ListZones()
	if err != nil {
		t.Fatal(err)
	}
	if len(zones) != 2 || zones[0] != "a.example" || zones[1] != "z.example" {
		t.Errorf("zones = %v", zones)
	}
}

func TestEnsureZoneExists(t *testing.T) {
	created := []createZoneRequest{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/auth/login":
			writeJSON(t, w, http.StatusOK, `{"code":0,"data":{"token":"token"},"desc":""}`)
		case r.Method == http.MethodGet && r.URL.Path == "/dns/zones/existing.example":
			writeJSON(t, w, http.StatusOK, `{"code":0,"data":{"id":42,"name":"existing.example"},"desc":""}`)
		case r.Method == http.MethodGet && r.URL.Path == "/dns/zones/new.example":
			writeJSON(t, w, http.StatusNotFound, `{"code":800,"data":"","desc":"Unknown DNS zone"}`)
		case r.Method == http.MethodPost && r.URL.Path == "/dns/zones":
			var request createZoneRequest
			if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
				t.Errorf("decode create zone: %v", err)
			}
			created = append(created, request)
			writeJSON(t, w, http.StatusOK, `{"code":0,"data":{"success":true},"desc":""}`)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	provider := &openproviderProvider{client: testAPIClient(t, server.URL)}
	if err := provider.EnsureZoneExists(&models.DomainConfig{Name: "existing.example"}); err != nil {
		t.Fatalf("existing zone: %v", err)
	}
	if len(created) != 0 {
		t.Fatalf("existing zone was created again: %#v", created)
	}

	dc := &models.DomainConfig{Name: "new.example"}
	record := dc.MustNewRecordConfig("www", 300, "A", "192.0.2.1")
	if err := provider.EnsureZoneExists(&models.DomainConfig{Name: "new.example", Records: models.Records{record}}); err != nil {
		t.Fatalf("new zone: %v", err)
	}
	if len(created) != 1 {
		t.Fatalf("created = %#v", created)
	}
	if created[0].Domain.Name != "new" || created[0].Domain.Extension != "example" || created[0].Type != "master" || created[0].IsSpamExpertsEnabled != "off" || created[0].Secured {
		t.Errorf("create zone identity = %#v", created[0])
	}
	if len(created[0].Records) != 1 || created[0].Records[0].TTL != int(minimumTTL) {
		t.Errorf("create zone records = %#v", created[0].Records)
	}
}

func TestSplitZoneName(t *testing.T) {
	tests := map[string]zoneDomain{
		"example.com":   {Name: "example", Extension: "com"},
		"example.co.uk": {Name: "example", Extension: "co.uk"},
	}
	for input, want := range tests {
		got, err := splitZoneName(input)
		if err != nil {
			t.Fatalf("splitZoneName(%q): %v", input, err)
		}
		if got != want {
			t.Errorf("splitZoneName(%q) = %#v, want %#v", input, got, want)
		}
	}
}

func TestEnsureZoneExistsRequiresInitialRecord(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/auth/login" {
			writeJSON(t, w, http.StatusOK, `{"code":0,"data":{"token":"token"},"desc":""}`)
			return
		}
		writeJSON(t, w, http.StatusNotFound, `{"code":872,"data":null,"desc":"Zone specified is not found."}`)
	}))
	defer server.Close()

	provider := &openproviderProvider{client: testAPIClient(t, server.URL)}
	err := provider.EnsureZoneExists(&models.DomainConfig{Name: "new.example"})
	if err == nil {
		t.Fatal("EnsureZoneExists returned no error")
	}
}
