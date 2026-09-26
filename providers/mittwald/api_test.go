package mittwald

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	mittwaldv2 "github.com/mittwald/api-client-go/mittwaldv2"
)

// fakeLookupAPI answers the requests of findProject: the domain list knows
// example.com, and only the zones of project p2 know zoneonly.example.
func fakeLookupAPI(t *testing.T) (*api, *[]string) {
	var mu sync.Mutex
	var paths []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		paths = append(paths, r.URL.Path)
		mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		var body any
		switch r.URL.Path {
		case "/v2/domains":
			body = []map[string]any{}
			if r.URL.Query().Get("domainSearchName") == "example.com" {
				body = []map[string]any{{"domain": "example.com", "domainId": "d1", "projectId": "p1"}}
			}
		case "/v2/projects":
			body = []map[string]any{{"id": "p1"}, {"id": "p2"}}
		case "/v2/projects/p1/dns-zones":
			body = []map[string]any{}
		case "/v2/projects/p2/dns-zones":
			body = []map[string]any{{"id": "z1", "domain": "zoneonly.example", "recordSet": map[string]any{}}}
		default:
			http.NotFound(w, r)
			return
		}
		_ = json.NewEncoder(w).Encode(body)
	}))
	t.Cleanup(srv.Close)
	client, err := mittwaldv2.New(context.Background(),
		mittwaldv2.WithHTTPClient(newAPIRunner(srv.Client())),
		mittwaldv2.WithAccessToken("t"),
		mittwaldv2.WithBaseURL(srv.URL+"/v2"),
	)
	if err != nil {
		t.Fatal(err)
	}
	return &api{client: client}, &paths
}

func TestFindProjectUsesTheDomainListFirst(t *testing.T) {
	a, paths := fakeLookupAPI(t)
	id, err := a.findProject("example.com")
	if err != nil || id != "p1" {
		t.Fatalf("got %q, %v", id, err)
	}
	if len(*paths) != 1 {
		t.Errorf("want one request, got %v", *paths)
	}
}

func TestFindProjectFallsBackToTheProjectsZones(t *testing.T) {
	a, _ := fakeLookupAPI(t)
	id, err := a.findProject("zoneonly.example")
	if err != nil || id != "p2" {
		t.Fatalf("got %q, %v", id, err)
	}
	if _, err := a.findProject("unknown.example"); err == nil {
		t.Error("an unknown domain is an error")
	}
}
