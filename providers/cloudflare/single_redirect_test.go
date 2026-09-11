package cloudflare

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/DNSControl/dnscontrol/v5/models"
	"github.com/DNSControl/dnscontrol/v5/pkg/js"
	"github.com/DNSControl/dnscontrol/v5/pkg/normalize"
	"github.com/DNSControl/dnscontrol/v5/pkg/zonecache"
	_ "github.com/DNSControl/dnscontrol/v5/providers/cloudflare/rtypes/cfsingleredirect"
	"github.com/cloudflare/cloudflare-go"
)

const redirectEntrypoint = "/zones/zone-id/rulesets/phases/http_request_dynamic_redirect/entrypoint"
const redirectRulePrefix = "/zones/zone-id/rulesets/ruleset-id/rules/"

// redirectAPI models only the Rulesets API operations used by these tests.
// The real Cloudflare SDK serializes every request; no credentials or network
// access are needed. A replacement without an ID is a new rule, appended by PUT.
type redirectAPI struct {
	rules           []cloudflare.RulesetRule
	writes          []string
	bodies          []map[string]json.RawMessage
	failMethod      string
	failStatus      int
	deleteNoContent bool
}

func (a *redirectAPI) RoundTrip(r *http.Request) (*http.Response, error) {
	respond := func(status int, body any) (*http.Response, error) {
		var data []byte
		if body != nil {
			var err error
			data, err = json.MarshalIndent(body, "", "  ")
			if err != nil {
				return nil, err
			}
		}
		return &http.Response{StatusCode: status, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(string(data))), Request: r}, nil
	}
	result := func() (*http.Response, error) {
		return respond(http.StatusOK, struct {
			Success bool               `json:"success"`
			Result  cloudflare.Ruleset `json:"result"`
		}{true, cloudflare.Ruleset{ID: "ruleset-id", Rules: a.rules}})
	}
	if r.Method != http.MethodGet {
		a.writes = append(a.writes, r.Method+" "+r.URL.Path)
		body := map[string]json.RawMessage{}
		if r.Body != nil {
			defer r.Body.Close()
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				return nil, err
			}
		}
		a.bodies = append(a.bodies, body)
	}
	if r.Method == a.failMethod {
		return respond(a.failStatus, map[string]any{"success": false, "errors": []any{map[string]any{"code": 1000, "message": "synthetic failure"}}})
	}
	switch {
	case r.Method == http.MethodGet && r.URL.Path == redirectEntrypoint:
		return result()
	case r.Method == http.MethodPut && r.URL.Path == redirectEntrypoint:
		var rules []cloudflare.RulesetRule
		if err := json.Unmarshal(a.bodies[len(a.bodies)-1]["rules"], &rules); err != nil {
			return nil, err
		}
		for i := range rules {
			if rules[i].ID == "" {
				rules[i].ID = fmt.Sprintf("created-%d-%d", len(a.writes), i)
			}
		}
		a.rules = rules
		return result()
	case strings.HasPrefix(r.URL.Path, redirectRulePrefix):
		id := strings.TrimPrefix(r.URL.Path, redirectRulePrefix)
		i := slices.IndexFunc(a.rules, func(rule cloudflare.RulesetRule) bool { return rule.ID == id })
		if i < 0 {
			return nil, fmt.Errorf("unknown rule ID %q", id)
		}
		switch r.Method {
		case http.MethodDelete:
			a.rules = slices.Delete(a.rules, i, i+1)
			if a.deleteNoContent {
				return respond(http.StatusNoContent, nil)
			}
			return result()
		case http.MethodPatch:
			data, err := json.Marshal(a.bodies[len(a.bodies)-1])
			if err != nil {
				return nil, err
			}
			var rule cloudflare.RulesetRule
			if err := json.Unmarshal(data, &rule); err != nil {
				return nil, err
			}
			rule.ID = id
			a.rules[i] = rule
			return result()
		}
	}
	return nil, fmt.Errorf("unexpected request: %s %s", r.Method, r.URL)
}

func redirectRule(id, name, host, destination string) cloudflare.RulesetRule {
	enabled, preserve := true, true
	return cloudflare.RulesetRule{
		ID: id, Ref: "ref-" + id, Description: name, Action: "redirect", Enabled: &enabled,
		Expression: fmt.Sprintf(`http.host eq %q`, host),
		ActionParameters: &cloudflare.RulesetRuleActionParameters{FromValue: &cloudflare.RulesetRuleActionParametersFromValue{
			StatusCode: 301, PreserveQueryString: &preserve,
			TargetURL: cloudflare.RulesetRuleActionParametersTargetURL{Expression: fmt.Sprintf(`concat(%q, http.request.uri.path)`, destination)},
		}},
	}
}

func redirectFixture(t *testing.T) (*cloudflareProvider, *redirectAPI, *models.DomainConfig) {
	t.Helper()
	a := &redirectAPI{rules: []cloudflare.RulesetRule{
		redirectRule("meta-id", "Meta", "meta.example.com", "https://meta.example.net"),
		redirectRule("chat-id", "Chat", "chat.example.com", "https://chat.example.net"),
		redirectRule("fallback-id", "Fallback", "example.com", "https://main.example.net"),
	}}
	a.rules[2].Expression = `http.host eq "example.com" or ends_with(http.host, ".example.com")`
	client, err := cloudflare.NewWithAPIToken("synthetic-token", cloudflare.BaseURL("https://cloudflare.invalid"), cloudflare.HTTPClient(&http.Client{Transport: a}), cloudflare.UsingRateLimit(100000), cloudflare.UsingRetryPolicy(0, 0, 0))
	if err != nil {
		t.Fatal(err)
	}
	c := &cloudflareProvider{cfClient: client, manageSingleRedirects: true}
	c.zoneCache = zonecache.New(func() (map[string]cloudflare.Zone, error) {
		return map[string]cloudflare.Zone{"example.com": {ID: "zone-id", Name: "example.com"}}, nil
	})
	dc := models.MustNewDomainConfig("example.com")
	for _, rule := range a.rules {
		dc.Records = append(dc.Records, dc.MustNewRecordConfig("@", 1, "CLOUDFLAREAPI_SINGLE_REDIRECT", rule.Description, rule.ActionParameters.FromValue.StatusCode, rule.Expression, rule.ActionParameters.FromValue.TargetURL.Expression))
	}
	return c, a, dc
}

func redirectPlan(t *testing.T, c *cloudflareProvider, dc *models.DomainConfig) []*models.Correction {
	t.Helper()
	existing, err := c.getSingleRedirects(dc, "zone-id")
	if err != nil {
		t.Fatal(err)
	}
	// Each preview/push starts from a fresh desired config; preprocessing mutates it.
	desired, err := dc.Copy()
	if err != nil {
		t.Fatal(err)
	}
	corrs, _, err := c.GetZoneRecordsCorrections(desired, existing)
	if err != nil {
		t.Fatal(err)
	}
	return corrs
}

func applyRedirectPlan(t *testing.T, corrs []*models.Correction) {
	t.Helper()
	for _, corr := range corrs {
		t.Log(corr.Msg)
		if err := corr.F(); err != nil {
			t.Fatal(err)
		}
	}
}

func assertRedirectNoop(t *testing.T, c *cloudflareProvider, dc *models.DomainConfig) {
	t.Helper()
	if corrs := redirectPlan(t, c, dc); len(corrs) != 0 {
		t.Fatalf("second reconciliation planned %d corrections: %v", len(corrs), corrs)
	}
}

func TestSingleRedirectUpdatePreservesOrder(t *testing.T) {
	c, a, dc := redirectFixture(t)
	before, _ := json.Marshal(a.rules[1:])
	rd := dc.Records[0].AsCLOUDFLAREAPISINGLEREDIRECT()
	rd.SRThen = `concat("https://new-meta.example.net", http.request.uri.path)`
	dc.Records[0].SetRDATA(rd)
	corrs := redirectPlan(t, c, dc)
	if len(corrs) != 1 {
		t.Fatalf("want one correction, got %d", len(corrs))
	}
	if len(a.writes) != 0 {
		t.Fatal("preview wrote to the API")
	}
	applyRedirectPlan(t, corrs)
	var names []string
	for _, rule := range a.rules {
		names = append(names, rule.Description)
	}
	if !reflect.DeepEqual(names, []string{"Meta", "Chat", "Fallback"}) {
		t.Errorf("rule order = %v; requests to meta.example.com now hit Fallback first", names)
	}
	if a.rules[0].ID != "meta-id" {
		t.Errorf("Meta identity changed: %s", a.rules[0].ID)
	}
	if !reflect.DeepEqual(a.writes, []string{"PATCH " + redirectRulePrefix + "meta-id"}) {
		t.Errorf("mutations = %v; want one in-place PATCH", a.writes)
	}
	if a.rules[0].ActionParameters.FromValue.TargetURL.Expression != rd.SRThen {
		t.Error("destination not updated in original slot")
	}
	after, _ := json.Marshal(a.rules[1:])
	if string(before) != string(after) {
		t.Error("unrelated rules changed")
	}
	assertRedirectNoop(t, c, dc)
}

func TestSingleRedirectTTLNoop(t *testing.T) {
	for _, declaration := range []string{
		`CF_SINGLE_REDIRECT("Meta", 301, 'http.host eq "meta.example.com"', 'concat("https://meta.example.net", http.request.uri.path)')`,
		`CF_SINGLE_REDIRECT("Meta", 301, 'http.host eq "meta.example.com"', 'concat("https://meta.example.net", http.request.uri.path)', TTL(999))`,
		`CF_REDIRECT("meta.example.com/*", "https://meta.example.net/$1")`,
		`CF_TEMP_REDIRECT("meta.example.com/*", "https://meta.example.net/$1")`,
	} {
		t.Run(declaration, func(t *testing.T) {
			c, a, _ := redirectFixture(t)
			config, err := js.ExecuteJavascriptString([]byte(`D("example.com", "none", `+declaration+`);`), false, nil)
			if err != nil {
				t.Fatal(err)
			}
			if errs := normalize.ValidateAndNormalizeConfig(config); len(errs) != 0 {
				t.Fatal(errs)
			}
			dc := config.Domains[0]
			rd := dc.Records[0].AsCLOUDFLAREAPISINGLEREDIRECT()
			a.rules = a.rules[:1]
			a.rules[0].Description = rd.SRName
			a.rules[0].Expression = rd.SRWhen
			a.rules[0].ActionParameters.FromValue.StatusCode = rd.Code
			a.rules[0].ActionParameters.FromValue.TargetURL.Expression = rd.SRThen
			corrs := redirectPlan(t, c, dc)
			for _, corr := range corrs {
				t.Log(corr.Msg)
			}
			if len(corrs) != 0 {
				t.Errorf("unchanged HTTP redirect planned %d corrections (desired TTL %d, native TTL 1)", len(corrs), dc.Records[0].TTL)
			}
			assertRedirectNoop(t, c, dc)
			if len(a.writes) != 0 {
				t.Fatal("no-op reconciliation wrote to API")
			}
		})
	}
}

func TestSingleRedirectMixedChanges(t *testing.T) {
	c, a, dc := redirectFixture(t)
	// Remove Chat, edit Meta, and add a name that sorts ahead of Meta. The
	// generic diff must not pair the new rule with Meta's existing identity.
	dc.Records = slices.Delete(dc.Records, 1, 2)
	rd := dc.Records[0].AsCLOUDFLAREAPISINGLEREDIRECT()
	rd.Code = 302
	dc.Records[0].SetRDATA(rd)
	dc.Records = append(dc.Records, dc.MustNewRecordConfig("@", 1, "CLOUDFLAREAPI_SINGLE_REDIRECT", "A-new", 301, `http.host eq "new.example.com"`, `concat("https://new.example.net", http.request.uri.path)`))
	applyRedirectPlan(t, redirectPlan(t, c, dc))
	if len(a.rules) != 3 {
		t.Fatalf("got %d rules", len(a.rules))
	}
	if a.rules[0].ID != "meta-id" || a.rules[0].Description != "Meta" || a.rules[0].ActionParameters.FromValue.StatusCode != 302 {
		t.Errorf("surviving Meta rule lost its identity/position: %+v", a.rules[0])
	}
	if a.rules[1].ID != "fallback-id" || a.rules[2].Description != "A-new" {
		t.Errorf("unexpected rule order: %+v", a.rules)
	}
	assertRedirectNoop(t, c, dc)
}

func TestSingleRedirectDefinitionEdits(t *testing.T) {
	for _, field := range []string{"status", "expression", "destination-query"} {
		t.Run(field, func(t *testing.T) {
			c, a, dc := redirectFixture(t)
			disabled, preserve := false, false
			a.rules[0].Enabled = &disabled
			a.rules[0].ActionParameters.FromValue.PreserveQueryString = &preserve
			rd := dc.Records[0].AsCLOUDFLAREAPISINGLEREDIRECT()
			switch field {
			case "status":
				rd.Code = 308
			case "expression":
				rd.SRWhen = `http.host eq "new-meta.example.com"`
			case "destination-query":
				rd.SRThen = `concat("https://new-meta.example.net", http.request.uri.path, "?fixed=1")`
			}
			dc.Records[0].SetRDATA(rd)
			applyRedirectPlan(t, redirectPlan(t, c, dc))
			rule := a.rules[0]
			if rule.ID != "meta-id" || rule.Ref != "ref-meta-id" || rule.Enabled == nil || *rule.Enabled {
				t.Fatalf("lost identity or disabled state: %+v", rule)
			}
			if rule.Action != "redirect" || rule.Description != rd.SRName || rule.Expression != rd.SRWhen {
				t.Errorf("wrong definition: %+v", rule)
			}
			fv := rule.ActionParameters.FromValue
			if fv.StatusCode != rd.Code || fv.TargetURL.Expression != rd.SRThen || fv.PreserveQueryString == nil || *fv.PreserveQueryString {
				t.Errorf("wrong redirect parameters: %+v", fv)
			}
			body := a.bodies[0]
			for _, key := range []string{"id", "version", "last_updated", "position", "rules"} {
				if _, present := body[key]; present {
					t.Errorf("PATCH should omit %q", key)
				}
			}
			assertRedirectNoop(t, c, dc)
		})
	}
}

func TestSingleRedirectUpdateFailure(t *testing.T) {
	for _, status := range []int{http.StatusBadRequest, http.StatusOK} {
		t.Run(fmt.Sprint(status), func(t *testing.T) {
			c, a, dc := redirectFixture(t)
			a.failMethod, a.failStatus = http.MethodPatch, status
			before, _ := json.Marshal(a.rules)
			rd := dc.Records[0].AsCLOUDFLAREAPISINGLEREDIRECT()
			rd.Code = 302
			dc.Records[0].SetRDATA(rd)
			corrs := redirectPlan(t, c, dc)
			if len(corrs) != 1 {
				t.Fatalf("want one correction, got %d", len(corrs))
			}
			if err := corrs[0].F(); err == nil || !strings.Contains(err.Error(), "synthetic failure") {
				t.Errorf("expected API error, got %v", err)
			}
			after, _ := json.Marshal(a.rules)
			if string(before) != string(after) {
				t.Error("failed update changed live rules")
			}
			if len(redirectPlan(t, c, dc)) != 1 {
				t.Error("failed update disappeared from next preview")
			}
		})
	}
}

func TestSingleRedirectRenameIsReplacement(t *testing.T) {
	c, a, dc := redirectFixture(t)
	rd := dc.Records[0].AsCLOUDFLAREAPISINGLEREDIRECT()
	rd.SRName = "Renamed Meta"
	dc.Records[0].SetRDATA(rd)
	applyRedirectPlan(t, redirectPlan(t, c, dc))
	if a.rules[0].ID != "chat-id" || a.rules[1].ID != "fallback-id" || a.rules[2].Description != rd.SRName || a.rules[2].ID == "meta-id" {
		t.Errorf("rename should be a new rule appended after survivors: %+v", a.rules)
	}
	assertRedirectNoop(t, c, dc)
}

func TestSingleRedirectDeleteResponses(t *testing.T) {
	for _, noContent := range []bool{false, true} {
		t.Run(fmt.Sprint(noContent), func(t *testing.T) {
			c, a, dc := redirectFixture(t)
			a.deleteNoContent = noContent
			dc.Records = slices.Delete(dc.Records, 1, 2)
			applyRedirectPlan(t, redirectPlan(t, c, dc))
			if len(a.rules) != 2 || a.rules[0].ID != "meta-id" || a.rules[1].ID != "fallback-id" {
				t.Errorf("delete changed surviving rules: %+v", a.rules)
			}
			assertRedirectNoop(t, c, dc)
		})
	}
}
