package infoblox

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/DNSControl/dnscontrol/v4/models"
)

// newTestAPI creates an infobloxAPI pointing at a test server.
func newTestAPI(ts *httptest.Server, view string) *infobloxAPI {
	return &infobloxAPI{
		host:       ts.URL,
		wapiVer:    "2.12",
		view:       view,
		username:   "admin",
		password:   "password",
		httpClient: ts.Client(),
	}
}

func TestConvertA(t *testing.T) {
	raw := json.RawMessage(`{"_ref":"record:a/ZG5z:10.0.0.1","name":"host.example.com","ipv4addr":"10.0.0.1","ttl":300,"use_ttl":true}`)
	rc, err := toRecordConfig("a", raw, "example.com", 3600)
	if err != nil {
		t.Fatal(err)
	}
	if rc.Type != "A" {
		t.Errorf("expected type A, got %s", rc.Type)
	}
	if rc.TTL != 300 {
		t.Errorf("expected TTL 300, got %d", rc.TTL)
	}
	if rc.GetTargetField() != "10.0.0.1" {
		t.Errorf("expected target 10.0.0.1, got %s", rc.GetTargetField())
	}
	if rc.GetLabel() != "host" {
		t.Errorf("expected label 'host', got %q", rc.GetLabel())
	}
}

func TestConvertAAAA(t *testing.T) {
	raw := json.RawMessage(`{"_ref":"record:aaaa/ZG5z:2001:db8::1","name":"host.example.com","ipv6addr":"2001:db8::1","ttl":600,"use_ttl":true}`)
	rc, err := toRecordConfig("aaaa", raw, "example.com", 3600)
	if err != nil {
		t.Fatal(err)
	}
	if rc.Type != "AAAA" {
		t.Errorf("expected type AAAA, got %s", rc.Type)
	}
	if rc.TTL != 600 {
		t.Errorf("expected TTL 600, got %d", rc.TTL)
	}
	if rc.GetTargetField() != "2001:db8::1" {
		t.Errorf("expected target 2001:db8::1, got %s", rc.GetTargetField())
	}
}

func TestConvertCNAME(t *testing.T) {
	raw := json.RawMessage(`{"_ref":"record:cname/ZG5z:www","name":"www.example.com","canonical":"web.example.com","ttl":300,"use_ttl":true}`)
	rc, err := toRecordConfig("cname", raw, "example.com", 3600)
	if err != nil {
		t.Fatal(err)
	}
	if rc.Type != "CNAME" {
		t.Errorf("expected type CNAME, got %s", rc.Type)
	}
	if rc.GetTargetField() != "web.example.com." {
		t.Errorf("expected target web.example.com., got %s", rc.GetTargetField())
	}
}

func TestConvertMX(t *testing.T) {
	raw := json.RawMessage(`{"_ref":"record:mx/ZG5z:mx","name":"example.com","mail_exchanger":"mail.example.com","preference":10,"ttl":300,"use_ttl":true}`)
	rc, err := toRecordConfig("mx", raw, "example.com", 3600)
	if err != nil {
		t.Fatal(err)
	}
	if rc.Type != "MX" {
		t.Errorf("expected type MX, got %s", rc.Type)
	}
	if rc.MxPreference != 10 {
		t.Errorf("expected preference 10, got %d", rc.MxPreference)
	}
	if rc.GetTargetField() != "mail.example.com." {
		t.Errorf("expected target mail.example.com., got %s", rc.GetTargetField())
	}
}

func TestConvertTXT(t *testing.T) {
	raw := json.RawMessage(`{"_ref":"record:txt/ZG5z:txt","name":"example.com","text":"v=spf1 -all","ttl":300,"use_ttl":true}`)
	rc, err := toRecordConfig("txt", raw, "example.com", 3600)
	if err != nil {
		t.Fatal(err)
	}
	if rc.Type != "TXT" {
		t.Errorf("expected type TXT, got %s", rc.Type)
	}
	if rc.GetTargetTXTJoined() != "v=spf1 -all" {
		t.Errorf("expected TXT 'v=spf1 -all', got %q", rc.GetTargetTXTJoined())
	}
}

func TestConvertSRV(t *testing.T) {
	raw := json.RawMessage(`{"_ref":"record:srv/ZG5z:srv","name":"_sip._tcp.example.com","target":"sip.example.com","priority":10,"weight":20,"port":5060,"ttl":300,"use_ttl":true}`)
	rc, err := toRecordConfig("srv", raw, "example.com", 3600)
	if err != nil {
		t.Fatal(err)
	}
	if rc.Type != "SRV" {
		t.Errorf("expected type SRV, got %s", rc.Type)
	}
	if rc.SrvPriority != 10 {
		t.Errorf("expected priority 10, got %d", rc.SrvPriority)
	}
	if rc.SrvWeight != 20 {
		t.Errorf("expected weight 20, got %d", rc.SrvWeight)
	}
	if rc.SrvPort != 5060 {
		t.Errorf("expected port 5060, got %d", rc.SrvPort)
	}
}

func TestConvertCAA(t *testing.T) {
	raw := json.RawMessage(`{"_ref":"record:caa/ZG5z:caa","name":"example.com","ca_flag":0,"ca_tag":"issue","ca_value":"letsencrypt.org","ttl":300,"use_ttl":true}`)
	rc, err := toRecordConfig("caa", raw, "example.com", 3600)
	if err != nil {
		t.Fatal(err)
	}
	if rc.Type != "CAA" {
		t.Errorf("expected type CAA, got %s", rc.Type)
	}
	if rc.CaaFlag != 0 {
		t.Errorf("expected flag 0, got %d", rc.CaaFlag)
	}
	if rc.CaaTag != "issue" {
		t.Errorf("expected tag 'issue', got %q", rc.CaaTag)
	}
}

func TestConvertPTR(t *testing.T) {
	raw := json.RawMessage(`{"_ref":"record:ptr/ZG5z:ptr","name":"1.0.0.10.in-addr.arpa","ptrdname":"host.example.com","ttl":300,"use_ttl":true}`)
	rc, err := toRecordConfig("ptr", raw, "0.0.10.in-addr.arpa", 3600)
	if err != nil {
		t.Fatal(err)
	}
	if rc.Type != "PTR" {
		t.Errorf("expected type PTR, got %s", rc.Type)
	}
	if rc.GetTargetField() != "host.example.com." {
		t.Errorf("expected target host.example.com., got %s", rc.GetTargetField())
	}
}

func TestConvertNSApexSkipped(t *testing.T) {
	raw := json.RawMessage(`{"_ref":"record:ns/ZG5z:ns","name":"example.com","nameserver":"ns1.example.com","ttl":3600}`)
	rc, err := toRecordConfig("ns", raw, "example.com", 3600)
	if err != nil {
		t.Fatal(err)
	}
	if rc != nil {
		t.Error("expected nil for apex NS record, got a record")
	}
}

func TestConvertNSSubdomain(t *testing.T) {
	raw := json.RawMessage(`{"_ref":"record:ns/ZG5z:ns2","name":"sub.example.com","nameserver":"ns1.sub.example.com","ttl":3600}`)
	rc, err := toRecordConfig("ns", raw, "example.com", 3600)
	if err != nil {
		t.Fatal(err)
	}
	if rc == nil {
		t.Fatal("expected non-nil for subdomain NS record")
	}
	if rc.GetLabel() != "sub" {
		t.Errorf("expected label 'sub', got %q", rc.GetLabel())
	}
}

func TestTTLInheritance(t *testing.T) {
	raw := json.RawMessage(`{"_ref":"record:a/ZG5z:10.0.0.2","name":"host.example.com","ipv4addr":"10.0.0.2","ttl":0,"use_ttl":false}`)
	rc, err := toRecordConfig("a", raw, "example.com", 3600)
	if err != nil {
		t.Fatal(err)
	}
	if rc.TTL != 3600 {
		t.Errorf("expected inherited TTL 3600, got %d", rc.TTL)
	}
}

func TestBuildRecordBodyA(t *testing.T) {
	rc := &models.RecordConfig{Type: "A", TTL: 300}
	rc.SetLabel("host", "example.com")
	if err := rc.PopulateFromString("A", "10.0.0.1", "example.com"); err != nil {
		t.Fatal(err)
	}

	recType, body, err := buildRecordBody(rc, "example.com", "default", true)
	if err != nil {
		t.Fatal(err)
	}
	if recType != "a" {
		t.Errorf("expected recType 'a', got %q", recType)
	}
	if body["ipv4addr"] != "10.0.0.1" {
		t.Errorf("expected ipv4addr 10.0.0.1, got %v", body["ipv4addr"])
	}
	if body["view"] != "default" {
		t.Errorf("expected view 'default', got %v", body["view"])
	}
	if body["use_ttl"] != true {
		t.Errorf("expected use_ttl true, got %v", body["use_ttl"])
	}
}

func TestBuildRecordBodyNoView(t *testing.T) {
	rc := &models.RecordConfig{Type: "A", TTL: 300}
	rc.SetLabel("host", "example.com")
	if err := rc.PopulateFromString("A", "10.0.0.1", "example.com"); err != nil {
		t.Fatal(err)
	}

	_, body, err := buildRecordBody(rc, "example.com", "default", false)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := body["view"]; ok {
		t.Error("expected no view in update body")
	}
}

func TestGetZoneAuth(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.Path, "/zone_auth") {
			http.Error(w, "not found", 404)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"result":[{"_ref":"zone_auth/ZG5z:example.com/default"}]}`)
	}))
	defer ts.Close()

	api := newTestAPI(ts, "default")
	ref, err := api.getZoneAuth("example.com")
	if err != nil {
		t.Fatal(err)
	}
	if ref != "zone_auth/ZG5z:example.com/default" {
		t.Errorf("unexpected ref: %s", ref)
	}
}

func TestGetZoneAuthNotFound(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"result":[]}`)
	}))
	defer ts.Close()

	api := newTestAPI(ts, "default")
	_, err := api.getZoneAuth("nonexistent.com")
	if err == nil {
		t.Error("expected error for non-existent zone")
	}
}

func TestCreateRecord(t *testing.T) {
	var receivedBody string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			http.Error(w, "method not allowed", 405)
			return
		}
		body, _ := readBody(r)
		receivedBody = body
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"result":"record:a/ZG5z:new"}`)
	}))
	defer ts.Close()

	api := newTestAPI(ts, "default")
	ref, err := api.createRecord("a", map[string]any{
		"name":     "host.example.com",
		"ipv4addr": "10.0.0.1",
		"view":     "default",
		"use_ttl":  true,
		"ttl":      300,
	})
	if err != nil {
		t.Fatal(err)
	}
	if ref != "record:a/ZG5z:new" {
		t.Errorf("unexpected ref: %s", ref)
	}
	if !strings.Contains(receivedBody, "10.0.0.1") {
		t.Errorf("expected body to contain IP, got: %s", receivedBody)
	}
}

func TestDeleteRecord(t *testing.T) {
	var deletedRef string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "DELETE" {
			http.Error(w, "method not allowed", 405)
			return
		}
		deletedRef = strings.TrimPrefix(r.URL.Path, "/wapi/v2.12/")
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"result":"record:a/ZG5z:deleted"}`)
	}))
	defer ts.Close()

	api := newTestAPI(ts, "default")
	err := api.deleteRecord("record:a/ZG5z:old")
	if err != nil {
		t.Fatal(err)
	}
	if deletedRef != "record:a/ZG5z:old" {
		t.Errorf("unexpected deleted ref: %s", deletedRef)
	}
}

func TestUpdateRecord(t *testing.T) {
	var updatedRef string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "PUT" {
			http.Error(w, "method not allowed", 405)
			return
		}
		updatedRef = strings.TrimPrefix(r.URL.Path, "/wapi/v2.12/")
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"result":"record:a/ZG5z:updated"}`)
	}))
	defer ts.Close()

	api := newTestAPI(ts, "default")
	err := api.updateRecord("record:a/ZG5z:existing", map[string]any{
		"ipv4addr": "10.0.0.2",
		"use_ttl":  true,
		"ttl":      600,
	})
	if err != nil {
		t.Fatal(err)
	}
	if updatedRef != "record:a/ZG5z:existing" {
		t.Errorf("unexpected updated ref: %s", updatedRef)
	}
}

func TestHTTPError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, `{"error":"unauthorized"}`, 401)
	}))
	defer ts.Close()

	api := newTestAPI(ts, "default")
	_, err := api.getZoneAuth("example.com")
	if err == nil {
		t.Error("expected error for 401 response")
	}
	if !strings.Contains(err.Error(), "401") {
		t.Errorf("expected 401 in error, got: %s", err.Error())
	}
}

func TestNewInfobloxMissingHost(t *testing.T) {
	_, err := newInfoblox(map[string]string{
		"username": "admin",
		"password": "pass",
	})
	if err == nil {
		t.Error("expected error for missing host")
	}
}

func TestNewInfobloxMissingUsername(t *testing.T) {
	_, err := newInfoblox(map[string]string{
		"host":     "https://grid.example.com",
		"password": "pass",
	})
	if err == nil {
		t.Error("expected error for missing username")
	}
}

func TestNewInfobloxMissingPassword(t *testing.T) {
	_, err := newInfoblox(map[string]string{
		"host":     "https://grid.example.com",
		"username": "admin",
	})
	if err == nil {
		t.Error("expected error for missing password")
	}
}

func TestNewInfobloxDefaults(t *testing.T) {
	p, err := newInfoblox(map[string]string{
		"host":     "https://grid.example.com",
		"username": "admin",
		"password": "pass",
	})
	if err != nil {
		t.Fatal(err)
	}
	if p.api.wapiVer != "2.12" {
		t.Errorf("expected default WAPI version 2.12, got %s", p.api.wapiVer)
	}
	if p.api.view != "default" {
		t.Errorf("expected default view, got %s", p.api.view)
	}
}

func TestEnsureTrailingDot(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"example.com", "example.com."},
		{"example.com.", "example.com."},
		{"", ""},
	}
	for _, tc := range tests {
		got := ensureTrailingDot(tc.input)
		if got != tc.expected {
			t.Errorf("ensureTrailingDot(%q) = %q, want %q", tc.input, got, tc.expected)
		}
	}
}

func TestEffectiveTTL(t *testing.T) {
	if got := effectiveTTL(true, 300, 3600); got != 300 {
		t.Errorf("expected 300, got %d", got)
	}
	if got := effectiveTTL(false, 0, 3600); got != 3600 {
		t.Errorf("expected 3600, got %d", got)
	}
}

func TestBuildRecordBodyTXT(t *testing.T) {
	rc := &models.RecordConfig{Type: "TXT", TTL: 300}
	rc.SetLabel("@", "example.com")
	if err := rc.SetTargetTXT("v=spf1 include:_spf.example.com -all"); err != nil {
		t.Fatal(err)
	}

	recType, body, err := buildRecordBody(rc, "example.com", "default", true)
	if err != nil {
		t.Fatal(err)
	}
	if recType != "txt" {
		t.Errorf("expected recType 'txt', got %q", recType)
	}
	if body["text"] != "v=spf1 include:_spf.example.com -all" {
		t.Errorf("expected text value, got %v", body["text"])
	}
}

func TestBuildRecordBodyMX(t *testing.T) {
	rc := &models.RecordConfig{Type: "MX", TTL: 300}
	rc.SetLabel("@", "example.com")
	if err := rc.SetTargetMX(10, "mail.example.com."); err != nil {
		t.Fatal(err)
	}

	recType, body, err := buildRecordBody(rc, "example.com", "default", true)
	if err != nil {
		t.Fatal(err)
	}
	if recType != "mx" {
		t.Errorf("expected recType 'mx', got %q", recType)
	}
	if body["mail_exchanger"] != "mail.example.com" {
		t.Errorf("expected mail_exchanger without trailing dot, got %v", body["mail_exchanger"])
	}
	if body["preference"] != uint16(10) {
		t.Errorf("expected preference 10, got %v", body["preference"])
	}
}

func TestBuildRecordBodySRV(t *testing.T) {
	rc := &models.RecordConfig{Type: "SRV", TTL: 300}
	rc.SetLabel("_sip._tcp", "example.com")
	if err := rc.SetTargetSRV(10, 20, 5060, "sip.example.com."); err != nil {
		t.Fatal(err)
	}

	recType, body, err := buildRecordBody(rc, "example.com", "default", true)
	if err != nil {
		t.Fatal(err)
	}
	if recType != "srv" {
		t.Errorf("expected recType 'srv', got %q", recType)
	}
	if body["target"] != "sip.example.com" {
		t.Errorf("expected target without trailing dot, got %v", body["target"])
	}
	if body["priority"] != uint16(10) {
		t.Errorf("expected priority 10, got %v", body["priority"])
	}
}

func TestBuildRecordBodyCAA(t *testing.T) {
	rc := &models.RecordConfig{Type: "CAA", TTL: 300}
	rc.SetLabel("@", "example.com")
	if err := rc.SetTargetCAA(0, "issue", "letsencrypt.org"); err != nil {
		t.Fatal(err)
	}

	recType, body, err := buildRecordBody(rc, "example.com", "default", true)
	if err != nil {
		t.Fatal(err)
	}
	if recType != "caa" {
		t.Errorf("expected recType 'caa', got %q", recType)
	}
	if body["ca_flag"] != uint8(0) {
		t.Errorf("expected ca_flag 0, got %v", body["ca_flag"])
	}
	if body["ca_tag"] != "issue" {
		t.Errorf("expected ca_tag 'issue', got %v", body["ca_tag"])
	}
}

// readBody is a test helper that reads the request body as a string.
func readBody(r *http.Request) (string, error) {
	defer r.Body.Close()
	body := new(strings.Builder)
	_, err := strings.NewReader("").WriteTo(body)
	if err != nil {
		return "", err
	}
	buf := make([]byte, 4096)
	n, _ := r.Body.Read(buf)
	return string(buf[:n]), nil
}
