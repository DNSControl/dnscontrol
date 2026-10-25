package cfapi

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestDNSRecordWireFormat(t *testing.T) {
	client, err := NewWithAPIToken("test", BaseURL("https://cloudflare.invalid"), HTTPClient(&http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		if r.Method != http.MethodPost || r.URL.Path != "/zones/zone-id/dns_records" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL)
		}
		body, _ := io.ReadAll(r.Body)
		if !strings.Contains(string(body), `"content":"192.0.2.1"`) || !strings.Contains(string(body), `"type":"A"`) {
			t.Errorf("unexpected DNS body: %s", body)
		}
		return &http.Response{StatusCode: 200, Header: http.Header{"Content-Type": {"application/json"}}, Body: io.NopCloser(strings.NewReader(`{"success":true,"result":{"id":"record-id","type":"A","content":"192.0.2.1"}}`)), Request: r}, nil
	})}), UsingRetryPolicy(0, 0, 0))
	if err != nil {
		t.Fatal(err)
	}
	record, err := client.CreateDNSRecord(context.Background(), "zone-id", CreateDNSRecordParams{Type: "A", Name: "www", Content: "192.0.2.1", TTL: 300})
	if err != nil || record.ID != "record-id" {
		t.Fatalf("record = %+v, error = %v", record, err)
	}
}

func TestUploadWorkerUsesJavaScriptBody(t *testing.T) {
	const script = `addEventListener("fetch", () => {})`
	client, err := NewWithAPIToken("test", BaseURL("https://cloudflare.invalid"), HTTPClient(&http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		if r.Method != http.MethodPut || r.URL.Path != "/accounts/account-id/workers/scripts/test-script" || r.Header.Get("Content-Type") != "application/javascript" {
			t.Errorf("unexpected request: %s %s %s", r.Method, r.URL, r.Header.Get("Content-Type"))
		}
		body, _ := io.ReadAll(r.Body)
		if string(body) != script {
			t.Errorf("script body = %q", body)
		}
		return &http.Response{StatusCode: 200, Header: http.Header{"Content-Type": {"application/json"}}, Body: io.NopCloser(strings.NewReader(`{"success":true,"result":{}}`)), Request: r}, nil
	})}), UsingRetryPolicy(0, 0, 0))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := client.UploadWorker(context.Background(), "account-id", CreateWorkerParams{ScriptName: "test-script", Script: script}); err != nil {
		t.Fatal(err)
	}
}

func TestMissingEntrypointRuleset(t *testing.T) {
	client, err := NewWithAPIToken("test", BaseURL("https://cloudflare.invalid"), HTTPClient(&http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: 404, Header: http.Header{"Content-Type": {"application/json"}}, Body: io.NopCloser(strings.NewReader(`{"success":false,"errors":[{"code":1000,"message":"not found"}]}`)), Request: r}, nil
	})}), UsingRetryPolicy(0, 0, 0))
	if err != nil {
		t.Fatal(err)
	}
	_, err = client.GetEntrypointRuleset(context.Background(), "zone-id", "http_request_dynamic_redirect")
	if _, ok := errors.AsType[*NotFoundError](err); !ok {
		t.Fatalf("expected NotFoundError, got %v", err)
	}
}
