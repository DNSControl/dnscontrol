package openprovider

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestAPITokenReuse(t *testing.T) {
	authCalls := 0
	zoneCalls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/auth/login":
			authCalls++
			var request map[string]string
			if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
				t.Errorf("decode login request: %v", err)
			}
			if request["username"] != "test-user" || request["password"] != "test-password" {
				t.Error("login request did not contain the expected credentials")
			}
			if _, ok := request["ip"]; ok {
				t.Error("login request unexpectedly contained an IP selector")
			}
			writeJSON(t, w, http.StatusOK, `{"code":0,"data":{"token":"token-one"},"desc":""}`)
		case "/dns/zones/example.com":
			zoneCalls++
			if r.URL.Query().Get("provider") != "openprovider" {
				t.Error("zone lookup did not select the standard Openprovider DNS service")
			}
			if r.Header.Get("Authorization") != "Bearer token-one" {
				t.Error("zone request did not reuse the authenticated token")
			}
			writeJSON(t, w, http.StatusOK, `{"code":0,"data":{"id":42,"name":"example.com"},"desc":""}`)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	client := testAPIClient(t, server.URL)
	for range 2 {
		if _, err := client.getZone("example.com"); err != nil {
			t.Fatalf("getZone: %v", err)
		}
	}
	if authCalls != 1 || zoneCalls != 2 {
		t.Fatalf("calls auth=%d zone=%d, want auth=1 zone=2", authCalls, zoneCalls)
	}
}

func TestAPIReauthenticatesAfterUnauthorized(t *testing.T) {
	authCalls := 0
	zoneCalls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/auth/login":
			authCalls++
			writeJSON(t, w, http.StatusOK, fmt.Sprintf(`{"code":0,"data":{"token":"token-%d"},"desc":""}`, authCalls))
		case "/dns/zones/example.com":
			zoneCalls++
			if r.Header.Get("Authorization") == "Bearer token-1" {
				writeJSON(t, w, http.StatusUnauthorized, `{"code":401,"data":"","desc":"expired"}`)
				return
			}
			if r.Header.Get("Authorization") != "Bearer token-2" {
				t.Error("retried request did not use the refreshed token")
			}
			writeJSON(t, w, http.StatusOK, `{"code":0,"data":{"id":42,"name":"example.com"},"desc":""}`)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	client := testAPIClient(t, server.URL)
	if _, err := client.getZone("example.com"); err != nil {
		t.Fatalf("getZone: %v", err)
	}
	if authCalls != 2 || zoneCalls != 2 {
		t.Fatalf("calls auth=%d zone=%d, want 2 each", authCalls, zoneCalls)
	}
}

func TestAPIRetriesTransientOpenproviderError(t *testing.T) {
	updateCalls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/auth/login":
			writeJSON(t, w, http.StatusOK, `{"code":0,"data":{"token":"token"},"desc":""}`)
		case "/dns/zones/example.com":
			updateCalls++
			if updateCalls < 3 {
				writeJSON(t, w, http.StatusBadRequest, `{"code":18002,"data":"","desc":"Data you sent is invalid or service is not available, try again later"}`)
				return
			}
			writeJSON(t, w, http.StatusOK, `{"code":0,"data":{"success":true},"desc":""}`)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	client := testAPIClient(t, server.URL)
	client.sleep = func(time.Duration) {}
	if err := client.updateZone(apiZone{ID: 1, Name: "example.com"}, recordUpdates{Add: []apiRecord{{Type: "A", Value: "192.0.2.1", TTL: 900}}}); err != nil {
		t.Fatalf("updateZone: %v", err)
	}
	if updateCalls != 3 {
		t.Fatalf("update calls = %d, want 3", updateCalls)
	}
}

func TestAPIDoesNotRetryOtherBadRequest(t *testing.T) {
	updateCalls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/auth/login" {
			writeJSON(t, w, http.StatusOK, `{"code":0,"data":{"token":"token"},"desc":""}`)
			return
		}
		updateCalls++
		writeJSON(t, w, http.StatusBadRequest, `{"code":12345,"data":"","desc":"invalid record"}`)
	}))
	defer server.Close()

	client := testAPIClient(t, server.URL)
	client.sleep = func(time.Duration) { t.Fatal("unexpected retry delay") }
	err := client.updateZone(apiZone{ID: 1, Name: "example.com"}, recordUpdates{Add: []apiRecord{{Type: "A", Value: "192.0.2.1", TTL: 900}}})
	if err == nil || updateCalls != 1 {
		t.Fatalf("updateZone error=%v, calls=%d; want one request and an error", err, updateCalls)
	}
}

func TestFormatRetryDiagnosticsIncludesExchangeAndRedactsToken(t *testing.T) {
	request := httptest.NewRequest(http.MethodPut, "https://api.example.test/v1/dns/zones/example.com", strings.NewReader(`{"records":[]}`))
	request.Header.Set("Authorization", "Bearer secret-token")
	request.Header.Set("Content-Type", "application/json")
	response := &http.Response{
		StatusCode: http.StatusBadRequest,
		Status:     "400 Bad Request",
		Header:     http.Header{"X-Request-ID": []string{"request-123"}},
		Request:    request,
		Body:       io.NopCloser(strings.NewReader(`{"code":18002,"desc":"try again later"}`)),
	}

	diagnostics := formatRetryDiagnostics(response, []byte(`{"records":[]}`), []byte(`{"code":18002,"desc":"try again later"}`))
	for _, expected := range []string{
		"PUT /v1/dns/zones/example.com HTTP/1.1",
		"Authorization: <redacted>",
		"X-Request-ID: request-123",
		"400 Bad Request",
		`{"code":18002,"desc":"try again later"}`,
	} {
		if !strings.Contains(diagnostics, expected) {
			t.Errorf("diagnostics do not contain %q:\n%s", expected, diagnostics)
		}
	}
	if strings.Contains(diagnostics, "secret-token") {
		t.Error("diagnostics contain the bearer token")
	}
}

func TestAPIErrorDoesNotExposeSecrets(t *testing.T) {
	const (
		username = "secret-user"
		password = "secret-password"
		token    = "secret-token"
	)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/auth/login" {
			writeJSON(t, w, http.StatusOK, `{"code":0,"data":{"token":"`+token+`"},"desc":""}`)
			return
		}
		writeJSON(t, w, http.StatusOK, `{"code":800,"data":"details omitted","desc":"Unknown DNS zone"}`)
	}))
	defer server.Close()

	client, err := newAPIClient(server.URL, username, password)
	if err != nil {
		t.Fatal(err)
	}
	_, err = client.getZone("missing.example")
	if err == nil {
		t.Fatal("getZone returned no error")
	}
	message := err.Error()
	for _, secret := range []string{username, password, token} {
		if strings.Contains(message, secret) {
			t.Errorf("error contains secret material")
		}
	}
	if !strings.Contains(message, "code 800") || !strings.Contains(message, "Unknown DNS zone") {
		t.Errorf("error lacks useful API details: %s", message)
	}
	if !isNotFound(err) {
		t.Errorf("code 800 was not classified as not found: %v", err)
	}
}

func TestListZonesPagination(t *testing.T) {
	offsets := []string{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/auth/login" {
			writeJSON(t, w, http.StatusOK, `{"code":0,"data":{"token":"token"},"desc":""}`)
			return
		}
		offsets = append(offsets, r.URL.Query().Get("offset"))
		if r.URL.Query().Get("provider") != "openprovider" {
			t.Error("zone listing did not select the standard Openprovider DNS service")
		}
		start := 0
		count := pageSize
		if offsets[len(offsets)-1] == "500" {
			start = 500
			count = 1
		}
		zones := make([]apiZone, 0, count)
		for i := range count {
			zones = append(zones, apiZone{ID: int64(start + i), Name: fmt.Sprintf("zone-%03d.example", start+i)})
		}
		data, _ := json.Marshal(zoneListResponse{Results: zones, Total: 501})
		writeJSON(t, w, http.StatusOK, `{"code":0,"data":`+string(data)+`,"desc":""}`)
	}))
	defer server.Close()

	zones, err := testAPIClient(t, server.URL).listZones()
	if err != nil {
		t.Fatalf("listZones: %v", err)
	}
	if len(zones) != 501 {
		t.Fatalf("listZones returned %d zones, want 501", len(zones))
	}
	if strings.Join(offsets, ",") != "0,500" {
		t.Errorf("offsets = %v, want [0 500]", offsets)
	}
}

func TestDecodeMalformedResponse(t *testing.T) {
	err := decodeResponse("test operation", http.StatusBadGateway, []byte("not-json"), nil)
	if err == nil || !strings.Contains(err.Error(), "test operation") || !strings.Contains(err.Error(), "HTTP 502") {
		t.Fatalf("decodeResponse error = %v", err)
	}
}

func TestMutationRejectsFalseSuccess(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/auth/login" {
			writeJSON(t, w, http.StatusOK, `{"code":0,"data":{"token":"token"},"desc":""}`)
			return
		}
		writeJSON(t, w, http.StatusOK, `{"code":0,"data":{"success":false},"desc":""}`)
	}))
	defer server.Close()

	client := testAPIClient(t, server.URL)
	err := client.updateZone(apiZone{ID: 1, Name: "example.com"}, recordUpdates{Add: []apiRecord{{Type: "A", Value: "192.0.2.1", TTL: 600}}})
	if err == nil || !strings.Contains(err.Error(), "success=false") {
		t.Fatalf("updateZone error = %v", err)
	}
}

func TestNewAPIClientValidatesURL(t *testing.T) {
	for _, baseURL := range []string{"", "ftp://api.example", "https://user:pass@api.example/v1", "https://api.example/v1?token=x"} {
		if _, err := newAPIClient(baseURL, "user", "password"); err == nil {
			t.Errorf("newAPIClient(%q) returned no error", baseURL)
		}
	}
	for _, baseURL := range []string{"http://api.example", "https://api.example/v1"} {
		if _, err := newAPIClient(baseURL, "user", "password"); err != nil {
			t.Errorf("newAPIClient(%q) returned an error: %v", baseURL, err)
		}
	}
}

func testAPIClient(t *testing.T, baseURL string) *apiClient {
	t.Helper()
	client, err := newAPIClient(baseURL, "test-user", "test-password")
	if err != nil {
		t.Fatal(err)
	}
	return client
}

func writeJSON(t *testing.T, w http.ResponseWriter, status int, body string) {
	t.Helper()
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if _, err := w.Write([]byte(body)); err != nil {
		t.Errorf("write response: %v", err)
	}
}
