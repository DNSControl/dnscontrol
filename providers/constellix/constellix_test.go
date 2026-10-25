package constellix

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/DNSControl/dnscontrol/v5/models"
	"github.com/DNSControl/dnscontrol/v5/pkg/zonecache"
)

const testZone = "example.com"

var testNameservers = []string{"ns11.constellix.com.", "ns21.constellix.com.", "ns41.constellix.net."}

// fakeAPI is a small Constellix API: a list of zones, the records of each
// zone by id, and a log of every write.
type fakeAPI struct {
	t *testing.T

	mu      sync.Mutex
	zones   []map[string]any
	records map[string]string // zone id -> JSON list of record objects
	writes  []string          // "METHOD path body"

	// createStatus and readStatus, if set, are the answers to POST /domains
	// and to GET /domains/{id} of the created zone.
	createStatus int
	readStatus   int
	// rateLimitOnce lists paths that answer 429 the first time.
	rateLimitOnce map[string]bool
}

func newFakeAPI(t *testing.T) *fakeAPI {
	return &fakeAPI{
		t: t,
		zones: []map[string]any{
			{"id": 1, "name": testZone, "nameservers": testNameservers, "template": nil},
		},
		records: map[string]string{"1": `[]`},
	}
}

func (f *fakeAPI) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	f.mu.Lock()
	defer f.mu.Unlock()

	body, _ := io.ReadAll(r.Body)
	path := r.URL.Path
	if f.rateLimitOnce[path] {
		delete(f.rateLimitOnce, path)
		w.Header().Set("X-RateLimit-Remaining", "0")
		w.Header().Set("X-RateLimit-Reset", "3")
		w.WriteHeader(http.StatusTooManyRequests)
		_, _ = w.Write([]byte(`{"message":"You have made too many requests, please wait and try again later"}`))
		return
	}
	if r.Method != http.MethodGet {
		f.writes = append(f.writes, strings.TrimSpace(r.Method+" "+path+" "+string(body)))
	}

	list := func(items string) {
		var data []json.RawMessage
		if err := json.Unmarshal([]byte(items), &data); err != nil {
			f.t.Errorf("bad test data: %v", err)
		}
		resp, _ := json.Marshal(map[string]any{"data": data, "meta": map[string]any{"pagination": map[string]any{"total": len(data)}}})
		_, _ = w.Write(resp)
	}
	notFound := func() {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"message":"Not Found"}`))
	}

	parts := strings.Split(strings.Trim(path, "/"), "/")
	switch {
	case r.Method == http.MethodGet && path == "/domains":
		zones, _ := json.Marshal(f.zones)
		list(string(zones))

	case r.Method == http.MethodPost && path == "/domains":
		if f.createStatus != 0 {
			w.WriteHeader(f.createStatus)
			_, _ = w.Write([]byte(`{"message":"The given data was invalid.","errors":{"name":"The domain is not valid"}}`))
			return
		}
		var req struct{ Name string }
		_ = json.Unmarshal(body, &req)
		f.zones = append(f.zones, map[string]any{"id": 99, "name": req.Name, "nameservers": testNameservers, "template": nil})
		f.records["99"] = `[]`
		w.WriteHeader(http.StatusAccepted)
		_, _ = w.Write([]byte(`{"data": {"id": 99, "name": "` + req.Name + `"}}`))

	case r.Method == http.MethodGet && len(parts) == 2 && parts[0] == "domains":
		if parts[1] == "99" && f.readStatus != 0 {
			w.WriteHeader(f.readStatus)
			_, _ = w.Write([]byte(`{"message":"Not Found"}`))
			return
		}
		for _, z := range f.zones {
			if fmt.Sprint(z["id"]) == parts[1] {
				// The single GET sends the id as a string.
				z = map[string]any{"id": parts[1], "name": z["name"], "nameservers": z["nameservers"], "template": z["template"]}
				resp, _ := json.Marshal(map[string]any{"data": z})
				_, _ = w.Write(resp)
				return
			}
		}
		notFound()

	case r.Method == http.MethodGet && len(parts) == 3 && parts[2] == "records":
		recs, ok := f.records[parts[1]]
		if !ok {
			notFound()
			return
		}
		list(recs)

	case r.Method == http.MethodPost && len(parts) == 3 && parts[2] == "records":
		w.WriteHeader(http.StatusAccepted)
		_, _ = w.Write([]byte(`{"data": {"id": 1000}}`))

	case r.Method == http.MethodPut && len(parts) == 4:
		_, _ = w.Write([]byte(`{"data": {}}`))

	case r.Method == http.MethodDelete && len(parts) == 4:
		w.WriteHeader(http.StatusNoContent)

	default:
		notFound()
	}
}

// testProvider returns a provider that talks to api with a fake clock.
func testProvider(t *testing.T, api *fakeAPI) *constellixProvider {
	t.Helper()
	server := httptest.NewServer(api)
	t.Cleanup(server.Close)

	c, _ := testClient(server.URL)
	p := &constellixProvider{client: c}
	p.zones = zonecache.New(p.fetchZones)
	return p
}

func TestInitializeRequiresKeys(t *testing.T) {
	tests := []struct {
		settings map[string]string
		wantErr  string
	}{
		{map[string]string{"secret_key": "s"}, "api_key"},
		{map[string]string{"api_key": "k"}, "secret_key"},
		{map[string]string{"api_key": "", "secret_key": ""}, "api_key"},
	}
	for _, tt := range tests {
		var p constellixProvider
		err := p.Initialize(tt.settings, nil, nil)
		if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
			t.Errorf("Initialize(%v) error = %v, want one naming %s", tt.settings, err, tt.wantErr)
		}
	}

	var p constellixProvider
	if err := p.Initialize(map[string]string{"api_key": "k", "secret_key": "s"}, nil, nil); err != nil {
		t.Fatalf("Initialize() error = %v", err)
	}
	if p.client == nil || p.observer == nil {
		t.Error("Initialize() left the client or the observer unset")
	}
}

func TestListZones(t *testing.T) {
	api := newFakeAPI(t)
	api.zones = nil
	for i := range 150 {
		api.zones = append(api.zones, map[string]any{"id": i, "name": fmt.Sprintf("z%d.example", i), "nameservers": testNameservers})
	}
	// Serve the zones in pages of 100, as the API does.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		page := r.URL.Query().Get("page")
		from := map[string]int{"1": 0, "2": 100}[page]
		to := min(from+100, len(api.zones))
		resp, _ := json.Marshal(map[string]any{"data": api.zones[from:to], "meta": map[string]any{"pagination": map[string]any{"total": len(api.zones)}}})
		_, _ = w.Write(resp)
	}))
	defer server.Close()
	c, _ := testClient(server.URL)
	p := &constellixProvider{client: c}
	p.zones = zonecache.New(p.fetchZones)

	zones, err := p.ListZones()
	if err != nil {
		t.Fatal(err)
	}
	seen := map[string]bool{}
	for _, z := range zones {
		seen[z] = true
	}
	if len(zones) != 150 || len(seen) != 150 {
		t.Errorf("ListZones() returned %d zones (%d distinct), want 150", len(zones), len(seen))
	}
}

func TestGetNameservers(t *testing.T) {
	p := testProvider(t, newFakeAPI(t))

	nss, err := p.GetNameservers(testZone)
	if err != nil {
		t.Fatal(err)
	}
	got := strings.Join(models.NameserversToStrings(nss), " ")
	if want := "ns11.constellix.com ns21.constellix.com ns41.constellix.net"; got != want {
		t.Errorf("GetNameservers() = %q, want %q", got, want)
	}

	// A zone that does not exist yet is not an error: preview asks before it
	// knows that the zone will be created.
	nss, err = p.GetNameservers("new.example.com")
	if err != nil || len(nss) != 0 {
		t.Errorf("GetNameservers() of a missing zone = %v, %v; want none and no error", nss, err)
	}
}

func TestEnsureZoneExists(t *testing.T) {
	api := newFakeAPI(t)
	p := testProvider(t, api)

	if err := p.EnsureZoneExists(models.MustNewDomainConfig(testZone)); err != nil {
		t.Fatal(err)
	}
	if len(api.writes) != 0 {
		t.Fatalf("an existing zone caused writes: %v", api.writes)
	}

	if err := p.EnsureZoneExists(models.MustNewDomainConfig("new.example.com")); err != nil {
		t.Fatal(err)
	}
	if want := `POST /domains {"name":"new.example.com"}`; len(api.writes) != 1 || api.writes[0] != want {
		t.Fatalf("writes = %v, want [%s]", api.writes, want)
	}

	// The new zone is usable in the same run.
	nss, err := p.GetNameservers("new.example.com")
	if err != nil || len(nss) != len(testNameservers) {
		t.Errorf("GetNameservers() of the new zone = %v, %v", nss, err)
	}
	if _, err := p.GetZoneRecords(models.MustNewDomainConfig("new.example.com")); err != nil {
		t.Errorf("GetZoneRecords() of the new zone: %v", err)
	}
	if err := p.EnsureZoneExists(models.MustNewDomainConfig("new.example.com")); err != nil || len(api.writes) != 1 {
		t.Errorf("a second EnsureZoneExists() = %v with writes %v, want no new write", err, api.writes)
	}
}

func TestEnsureZoneExistsFailures(t *testing.T) {
	tests := []struct {
		name         string
		createStatus int
		readStatus   int
		want         string
	}{
		{"created, read 404", 0, http.StatusNotFound, "was created but is not readable yet, run push again"},
		{"created, read 500", 0, http.StatusInternalServerError, "was created but is not readable yet, run push again"},
		{"creation failed", http.StatusUnprocessableEntity, 0, "creating zone new.example.com failed"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			api := newFakeAPI(t)
			api.createStatus, api.readStatus = tt.createStatus, tt.readStatus
			p := testProvider(t, api)
			dc := models.MustNewDomainConfig("new.example.com")

			check := func(what string, err error) {
				t.Helper()
				if err == nil || !strings.Contains(err.Error(), tt.want) {
					t.Errorf("%s error = %v, want %q", what, err, tt.want)
				}
			}
			check("EnsureZoneExists()", p.EnsureZoneExists(dc))
			check("second EnsureZoneExists()", p.EnsureZoneExists(dc))
			_, err := p.GetNameservers(dc.Name)
			check("GetNameservers()", err)
			_, err = p.GetZoneRecords(dc)
			check("GetZoneRecords()", err)

			if tt.createStatus != 0 && !strings.Contains(p.EnsureZoneExists(dc).Error(), "The domain is not valid") {
				t.Error("the error does not carry the API's reason")
			}
			if len(api.writes) != 1 {
				t.Errorf("writes = %v, want one creation attempt", api.writes)
			}
		})
	}
}
