package namecheap

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"testing"

	nc "github.com/billputer/go-namecheap"
)

func TestListZonesPaginates(t *testing.T) {
	requests := []string{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			t.Fatalf("ParseForm() error = %v", err)
		}
		requests = append(requests, r.Form.Get("Page"))

		want := url.Values{
			"ApiUser":  []string{"api-user"},
			"ApiKey":   []string{"api-key"},
			"UserName": []string{"api-user"},
			"ClientIp": []string{"127.0.0.1"},
			"Command":  []string{"namecheap.domains.getList"},
			"PageSize": []string{"100"},
		}

		for key, values := range want {
			if got := r.Form[key]; !reflect.DeepEqual(got, values) {
				t.Fatalf("form[%q] = %v, want %v", key, got, values)
			}
		}

		page := r.Form.Get("Page")
		switch page {
		case "1":
			fmt.Fprint(w, `<?xml version="1.0" encoding="utf-8"?>
<ApiResponse Status="OK" xmlns="http://api.namecheap.com/xml.response">
  <Errors />
  <CommandResponse Type="namecheap.domains.getList">
    <DomainGetListResult>
      <Domain ID="1" Name="alpha.example" User="api-user" Created="11/04/2014" Expires="11/04/2015" IsExpired="false" IsLocked="false" AutoRenew="false" WhoisGuard="ENABLED" />
      <Domain ID="2" Name="beta.example" User="api-user" Created="11/04/2014" Expires="11/04/2015" IsExpired="false" IsLocked="false" AutoRenew="false" WhoisGuard="ENABLED" />
    </DomainGetListResult>
    <Paging>
      <TotalItems>3</TotalItems>
      <CurrentPage>1</CurrentPage>
      <PageSize>2</PageSize>
    </Paging>
  </CommandResponse>
</ApiResponse>`)
		case "2":
			fmt.Fprint(w, `<?xml version="1.0" encoding="utf-8"?>
<ApiResponse Status="OK" xmlns="http://api.namecheap.com/xml.response">
  <Errors />
  <CommandResponse Type="namecheap.domains.getList">
    <DomainGetListResult>
      <Domain ID="3" Name="gamma.example" User="api-user" Created="11/04/2014" Expires="11/04/2015" IsExpired="false" IsLocked="false" AutoRenew="false" WhoisGuard="ENABLED" />
    </DomainGetListResult>
    <Paging>
      <TotalItems>3</TotalItems>
      <CurrentPage>2</CurrentPage>
      <PageSize>2</PageSize>
    </Paging>
  </CommandResponse>
</ApiResponse>`)
		default:
			t.Fatalf("unexpected page %q", page)
		}
	}))
	defer server.Close()

	provider := &namecheapProvider{
		client: nc.NewClient("api-user", "api-key", "api-user"),
	}
	provider.client.BaseURL = server.URL

	zones, err := provider.ListZones()
	if err != nil {
		t.Fatalf("ListZones() error = %v", err)
	}

	wantZones := []string{"alpha.example", "beta.example", "gamma.example"}
	if !reflect.DeepEqual(zones, wantZones) {
		t.Fatalf("ListZones() = %v, want %v", zones, wantZones)
	}

	wantRequests := []string{"1", "2"}
	if !reflect.DeepEqual(requests, wantRequests) {
		t.Fatalf("pages requested = %v, want %v", requests, wantRequests)
	}
}
