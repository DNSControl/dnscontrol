package mikrotik

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/DNSControl/dnscontrol/v4/models"
)

// --- belongsToDomain ---

func TestBelongsToDomain(t *testing.T) {
	tests := []struct {
		fqdn, domain string
		want         bool
	}{
		{"example.com", "example.com", true},
		{"host.example.com", "example.com", true},
		{"deep.host.example.com", "example.com", true},
		{"example.com", "other.com", false},
		{"notexample.com", "example.com", false}, // suffix but not subdomain
		{"host.other.com", "example.com", false},
		{"a.local", "a.local", true},
		{"sub.a.local", "a.local", true},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("%s/%s", tt.fqdn, tt.domain), func(t *testing.T) {
			if got := belongsToDomain(tt.fqdn, tt.domain); got != tt.want {
				t.Errorf("belongsToDomain(%q, %q) = %v, want %v", tt.fqdn, tt.domain, got, tt.want)
			}
		})
	}
}

// --- detectZone ---

func TestDetectZone_NoHints_PublicSuffix(t *testing.T) {
	p := &mikrotikProvider{}
	tests := []struct {
		name, want string
	}{
		{"host.example.com", "example.com"},
		{"deep.sub.example.co.uk", "example.co.uk"},
		{"example.com", "example.com"},
	}
	for _, tt := range tests {
		got := p.detectZone(tt.name)
		if got != tt.want {
			t.Errorf("detectZone(%q) = %q, want %q", tt.name, got, tt.want)
		}
	}
}

func TestDetectZone_NoHints_Fallback(t *testing.T) {
	p := &mikrotikProvider{}
	tests := []struct {
		name, want string
	}{
		{"host.corp.local", "corp.local"},
		{"deep.sub.corp.local", "corp.local"},
		{"corp.local", "corp.local"},
		{"singleword", "singleword"},
	}
	for _, tt := range tests {
		got := p.detectZone(tt.name)
		if got != tt.want {
			t.Errorf("detectZone(%q) = %q, want %q", tt.name, got, tt.want)
		}
	}
}

func TestDetectZone_WithHints(t *testing.T) {
	p := &mikrotikProvider{
		zoneHints: []string{"internal.corp.local", "corp.local", "home.arpa"}, // longest-first
	}
	tests := []struct {
		name, want string
	}{
		// Matches longest hint first.
		{"host.internal.corp.local", "internal.corp.local"},
		{"internal.corp.local", "internal.corp.local"},
		// Falls through to shorter hint.
		{"other.corp.local", "corp.local"},
		{"corp.local", "corp.local"},
		// Matches a different hint.
		{"myhost.home.arpa", "home.arpa"},
		// No hint matches → public suffix fallback.
		{"host.example.com", "example.com"},
		// No hint matches → private TLD fallback (last 2 labels).
		{"host.other.local", "other.local"},
	}
	for _, tt := range tests {
		got := p.detectZone(tt.name)
		if got != tt.want {
			t.Errorf("detectZone(%q) = %q, want %q", tt.name, got, tt.want)
		}
	}
}

func TestDetectZone_HintIsGlobalTLD(t *testing.T) {
	// A hint can be a single-label "global TLD" like "local".
	p := &mikrotikProvider{
		zoneHints: []string{"internal.corp.local", "local"},
	}
	tests := []struct {
		name, want string
	}{
		{"host.internal.corp.local", "internal.corp.local"},
		{"other.corp.local", "local"},
		{"something.local", "local"},
		{"local", "local"},
	}
	for _, tt := range tests {
		got := p.detectZone(tt.name)
		if got != tt.want {
			t.Errorf("detectZone(%q) = %q, want %q", tt.name, got, tt.want)
		}
	}
}

func TestNewMikrotikProvider_ZoneHints(t *testing.T) {
	cfg := map[string]string{
		"host": "http://192.168.1.1:8080", "username": "admin", "password": "secret",
		"zonehints": "internal.corp.local, corp.local ,home.arpa",
	}
	p, err := newMikrotikProvider(cfg, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	mp := p.(*mikrotikProvider)
	if len(mp.zoneHints) != 3 {
		t.Fatalf("zoneHints len = %d, want 3", len(mp.zoneHints))
	}
	// Should be sorted longest-first.
	if mp.zoneHints[0] != "internal.corp.local" {
		t.Errorf("zoneHints[0] = %q, want internal.corp.local", mp.zoneHints[0])
	}
}

func TestNewMikrotikProvider_NoZoneHints(t *testing.T) {
	cfg := map[string]string{
		"host": "http://192.168.1.1:8080", "username": "admin", "password": "secret",
	}
	p, err := newMikrotikProvider(cfg, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	mp := p.(*mikrotikProvider)
	if len(mp.zoneHints) != 0 {
		t.Errorf("zoneHints len = %d, want 0", len(mp.zoneHints))
	}
}

// --- metaCompFunc ---

func TestMetaCompFunc(t *testing.T) {
	tests := []struct {
		name     string
		metadata map[string]string
		want     string
	}{
		{"nil_metadata", nil, ""},
		{"empty_metadata", map[string]string{}, ""},
		{"all_empty_values", map[string]string{
			"address_list": "", "comment": "", "match_subdomain": "", "regexp": "",
		}, ""},
		{"with_values", map[string]string{
			"address_list": "my-list", "comment": "test", "match_subdomain": "true", "regexp": ".*",
		}, "address_list=my-list comment=test match_subdomain=true regexp=.*"},
		{"partial_values", map[string]string{
			"comment": "just a comment",
		}, "address_list= comment=just a comment match_subdomain= regexp="},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rc := &models.RecordConfig{Metadata: tt.metadata}
			got := metaCompFunc(rc)
			if got != tt.want {
				t.Errorf("metaCompFunc() = %q, want %q", got, tt.want)
			}
		})
	}
}

// --- forwarderCompFunc ---

func TestForwarderCompFunc(t *testing.T) {
	tests := []struct {
		name     string
		metadata map[string]string
		want     string
	}{
		{"nil_metadata", nil, ""},
		{"empty_metadata", map[string]string{}, "doh_servers= verify_doh_cert="},
		{"with_values", map[string]string{
			"doh_servers": "https://dns.google/dns-query", "verify_doh_cert": "true",
		}, "doh_servers=https://dns.google/dns-query verify_doh_cert=true"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rc := &models.RecordConfig{Metadata: tt.metadata}
			got := forwarderCompFunc(rc)
			if got != tt.want {
				t.Errorf("forwarderCompFunc() = %q, want %q", got, tt.want)
			}
		})
	}
}

// --- newMikrotikProvider ---

func TestNewMikrotikProvider_Valid(t *testing.T) {
	cfg := map[string]string{
		"host": "http://192.168.1.1:8080", "username": "admin", "password": "secret",
	}
	p, err := newMikrotikProvider(cfg, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	mp := p.(*mikrotikProvider)
	if mp.host != "http://192.168.1.1:8080" {
		t.Errorf("host = %q, want http://192.168.1.1:8080", mp.host)
	}
}

func TestNewMikrotikProvider_TrailingSlash(t *testing.T) {
	cfg := map[string]string{
		"host": "http://192.168.1.1:8080/", "username": "admin", "password": "secret",
	}
	p, err := newMikrotikProvider(cfg, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	mp := p.(*mikrotikProvider)
	if mp.host != "http://192.168.1.1:8080" {
		t.Errorf("host = %q, want trailing slash stripped", mp.host)
	}
}

func TestNewMikrotikProvider_MissingHost(t *testing.T) {
	cfg := map[string]string{"username": "admin", "password": "secret"}
	_, err := newMikrotikProvider(cfg, nil)
	if err == nil {
		t.Error("expected error for missing host")
	}
}

func TestNewMikrotikProvider_MissingUsername(t *testing.T) {
	cfg := map[string]string{"host": "http://192.168.1.1:8080", "password": "secret"}
	_, err := newMikrotikProvider(cfg, nil)
	if err == nil {
		t.Error("expected error for missing username")
	}
}

func TestNewMikrotikProvider_MissingPassword(t *testing.T) {
	cfg := map[string]string{"host": "http://192.168.1.1:8080", "username": "admin"}
	_, err := newMikrotikProvider(cfg, nil)
	if err == nil {
		t.Error("expected error for missing password")
	}
}

// --- API tests using httptest ---

func newTestProvider(t *testing.T, handler http.HandlerFunc) (*mikrotikProvider, *httptest.Server) {
	t.Helper()
	srv := httptest.NewServer(handler)
	p := &mikrotikProvider{
		host:     srv.URL,
		username: "admin",
		password: "secret",
	}
	return p, srv
}

func TestDoRequest_BasicAuth(t *testing.T) {
	var gotUser, gotPass string
	var gotContentType string
	p, srv := newTestProvider(t, func(w http.ResponseWriter, r *http.Request) {
		gotUser, gotPass, _ = r.BasicAuth()
		gotContentType = r.Header.Get("Content-Type")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("[]"))
	})
	defer srv.Close()

	_, err := p.doRequest(http.MethodGet, "/test", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotUser != "admin" || gotPass != "secret" {
		t.Errorf("auth = %q:%q, want admin:secret", gotUser, gotPass)
	}
	if gotContentType != "" {
		t.Errorf("Content-Type should be empty for GET without body, got %q", gotContentType)
	}
}

func TestDoRequest_ContentType(t *testing.T) {
	var gotContentType string
	p, srv := newTestProvider(t, func(w http.ResponseWriter, r *http.Request) {
		gotContentType = r.Header.Get("Content-Type")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("{}"))
	})
	defer srv.Close()

	payload := &dnsStaticRecord{Name: "test", Type: "A"}
	_, err := p.doRequest(http.MethodPut, "/test", payload)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotContentType != "application/json" {
		t.Errorf("Content-Type = %q, want application/json", gotContentType)
	}
}

func TestDoRequest_401(t *testing.T) {
	p, srv := newTestProvider(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	})
	defer srv.Close()

	_, err := p.doRequest(http.MethodGet, "/test", nil)
	if err == nil {
		t.Fatal("expected error for 401")
	}
	if got := err.Error(); got != "mikrotik: authentication failed (401)" {
		t.Errorf("error = %q, want auth failed message", got)
	}
}

func TestDoRequest_4xxWithJSON(t *testing.T) {
	p, srv := newTestProvider(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]any{
			"detail": "input does not match any value of name",
			"error":  400,
		})
	})
	defer srv.Close()

	_, err := p.doRequest(http.MethodGet, "/test", nil)
	if err == nil {
		t.Fatal("expected error for 400")
	}
	expected := "mikrotik: API error (400): input does not match any value of name"
	if err.Error() != expected {
		t.Errorf("error = %q, want %q", err.Error(), expected)
	}
}

func TestDoRequest_4xxPlainText(t *testing.T) {
	p, srv := newTestProvider(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte("not found"))
	})
	defer srv.Close()

	_, err := p.doRequest(http.MethodGet, "/test", nil)
	if err == nil {
		t.Fatal("expected error for 404")
	}
	expected := "mikrotik: API error (404): not found"
	if err.Error() != expected {
		t.Errorf("error = %q, want %q", err.Error(), expected)
	}
}

func TestGetAllRecords(t *testing.T) {
	records := []dnsStaticRecord{
		{ID: "*1", Name: "host.example.com", Type: "A", Address: "10.0.0.1", TTL: "1d"},
		{ID: "*2", Name: "example.com", Type: "TXT", Text: "hello", TTL: "1h"},
	}
	p, srv := newTestProvider(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != apiPath {
			t.Errorf("path = %q, want %q", r.URL.Path, apiPath)
		}
		if r.Method != http.MethodGet {
			t.Errorf("method = %q, want GET", r.Method)
		}
		json.NewEncoder(w).Encode(records)
	})
	defer srv.Close()

	got, err := p.getAllRecords()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("len = %d, want 2", len(got))
	}
	if got[0].Name != "host.example.com" {
		t.Errorf("Name = %q, want host.example.com", got[0].Name)
	}
}

func TestCreateRecord(t *testing.T) {
	var gotMethod string
	var gotPath string
	var gotBody dnsStaticRecord
	p, srv := newTestProvider(t, func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		json.NewDecoder(r.Body).Decode(&gotBody)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("{}"))
	})
	defer srv.Close()

	rec := &dnsStaticRecord{Name: "new.example.com", Type: "A", Address: "10.0.0.1", TTL: "1d"}
	err := p.createRecord(rec)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotMethod != http.MethodPut {
		t.Errorf("method = %q, want PUT", gotMethod)
	}
	if gotPath != apiPath {
		t.Errorf("path = %q, want %q", gotPath, apiPath)
	}
	if gotBody.Name != "new.example.com" {
		t.Errorf("body.Name = %q, want new.example.com", gotBody.Name)
	}
}

func TestUpdateRecord(t *testing.T) {
	var gotMethod, gotPath string
	p, srv := newTestProvider(t, func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("{}"))
	})
	defer srv.Close()

	err := p.updateRecord("*5", &dnsStaticRecord{Name: "host.example.com", Type: "A", Address: "10.0.0.2"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotMethod != http.MethodPatch {
		t.Errorf("method = %q, want PATCH", gotMethod)
	}
	if gotPath != apiPath+"/*5" {
		t.Errorf("path = %q, want %s/*5", gotPath, apiPath)
	}
}

func TestDeleteRecord(t *testing.T) {
	var gotMethod, gotPath string
	p, srv := newTestProvider(t, func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(""))
	})
	defer srv.Close()

	err := p.deleteRecord("*3")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotMethod != http.MethodDelete {
		t.Errorf("method = %q, want DELETE", gotMethod)
	}
	if gotPath != apiPath+"/*3" {
		t.Errorf("path = %q, want %s/*3", gotPath, apiPath)
	}
}

func TestGetAllForwarders(t *testing.T) {
	fwds := []dnsForwarder{
		{ID: "*1", Name: "fwd1", DNSServers: "1.1.1.1"},
	}
	p, srv := newTestProvider(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != forwardersPath {
			t.Errorf("path = %q, want %q", r.URL.Path, forwardersPath)
		}
		json.NewEncoder(w).Encode(fwds)
	})
	defer srv.Close()

	got, err := p.getAllForwarders()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("len = %d, want 1", len(got))
	}
	if got[0].Name != "fwd1" {
		t.Errorf("Name = %q, want fwd1", got[0].Name)
	}
}

func TestCreateForwarder(t *testing.T) {
	var gotMethod string
	p, srv := newTestProvider(t, func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("{}"))
	})
	defer srv.Close()

	err := p.createForwarder(&dnsForwarder{Name: "new-fwd", DNSServers: "8.8.8.8"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotMethod != http.MethodPut {
		t.Errorf("method = %q, want PUT", gotMethod)
	}
}

func TestUpdateForwarder(t *testing.T) {
	var gotMethod, gotPath string
	p, srv := newTestProvider(t, func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("{}"))
	})
	defer srv.Close()

	err := p.updateForwarder("*2", &dnsForwarder{Name: "updated", DNSServers: "8.8.4.4"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotMethod != http.MethodPatch {
		t.Errorf("method = %q, want PATCH", gotMethod)
	}
	if gotPath != forwardersPath+"/*2" {
		t.Errorf("path = %q, want %s/*2", gotPath, forwardersPath)
	}
}

func TestDeleteForwarder(t *testing.T) {
	var gotMethod, gotPath string
	p, srv := newTestProvider(t, func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(""))
	})
	defer srv.Close()

	err := p.deleteForwarder("*4")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotMethod != http.MethodDelete {
		t.Errorf("method = %q, want DELETE", gotMethod)
	}
	if gotPath != forwardersPath+"/*4" {
		t.Errorf("path = %q, want %s/*4", gotPath, forwardersPath)
	}
}

// --- GetZoneRecords ---

func TestGetZoneRecords_FiltersAndConverts(t *testing.T) {
	records := []dnsStaticRecord{
		{ID: "*1", Name: "host.example.com", Type: "A", Address: "10.0.0.1", TTL: "1d"},
		{ID: "*2", Name: "other.other.com", Type: "A", Address: "10.0.0.2", TTL: "1d"},
		{ID: "*3", Name: "example.com", Type: "TXT", Text: "hello", TTL: "1h"},
		{ID: "*4", Name: "dyn.example.com", Type: "A", Address: "10.0.0.3", TTL: "1d", Dynamic: "true"},
		{ID: "*5", Name: "off.example.com", Type: "A", Address: "10.0.0.4", TTL: "1d", Disabled: "true"},
	}

	p, srv := newTestProvider(t, func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(records)
	})
	defer srv.Close()

	dc, _ := models.NewDomainConfig("example.com")
	rcs, err := p.GetZoneRecords(dc)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Should include *1 and *3, but not *2 (wrong domain), *4 (dynamic), *5 (disabled).
	if len(rcs) != 2 {
		t.Fatalf("len = %d, want 2 (got records: %v)", len(rcs), rcs)
	}
}

func TestGetZoneRecords_ForwarderZone(t *testing.T) {
	fwds := []dnsForwarder{
		{ID: "*1", Name: "fwd1", DNSServers: "1.1.1.1"},
		{ID: "*2", Name: "fwd2", DNSServers: "8.8.8.8", Disabled: "true"},
	}

	p, srv := newTestProvider(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == forwardersPath {
			json.NewEncoder(w).Encode(fwds)
		} else {
			json.NewEncoder(w).Encode([]dnsStaticRecord{})
		}
	})
	defer srv.Close()

	dc, _ := models.NewDomainConfig(ForwarderZone)
	rcs, err := p.GetZoneRecords(dc)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Should include fwd1 but not fwd2 (disabled).
	if len(rcs) != 1 {
		t.Fatalf("len = %d, want 1", len(rcs))
	}
	if rcs[0].GetLabel() != "fwd1" {
		t.Errorf("Label = %q, want fwd1", rcs[0].GetLabel())
	}
}

// --- GetNameservers ---

func TestGetNameservers(t *testing.T) {
	p := &mikrotikProvider{}
	ns, err := p.GetNameservers("example.com")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(ns) != 0 {
		t.Errorf("len = %d, want 0", len(ns))
	}
}
