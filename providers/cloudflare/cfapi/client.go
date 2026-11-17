// Package cfapi contains the small set of Cloudflare wire formats used by the
// DNSControl provider. Requests are executed by cloudflare-go/v7.
package cfapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"

	cloudflare "github.com/cloudflare/cloudflare-go/v7"
	"github.com/cloudflare/cloudflare-go/v7/option"
)

// API is a Cloudflare API client.
type API struct {
	client   *cloudflare.Client
	options  []option.RequestOption
	APIEmail string
}

// Option configures an API.
type Option func(*API) error

// NewWithAPIToken creates an API that authenticates with a token.
func NewWithAPIToken(token string, opts ...Option) (*API, error) {
	return newAPI([]option.RequestOption{option.WithAPIToken(token)}, "", opts...)
}

// New creates an API that authenticates with a key and email.
func New(key, email string, opts ...Option) (*API, error) {
	return newAPI([]option.RequestOption{option.WithAPIKey(key), option.WithAPIEmail(email)}, email, opts...)
}

func newAPI(auth []option.RequestOption, email string, opts ...Option) (*API, error) {
	a := &API{APIEmail: email, options: append(auth, option.WithMaxRetries(20))}
	for _, opt := range opts {
		if err := opt(a); err != nil {
			return nil, err
		}
	}
	a.client = cloudflare.NewClient(a.options...)
	return a, nil
}

func (a *API) reset() { a.client = cloudflare.NewClient(a.options...) }

// SetDebug enables debug logging.
func (a *API) SetDebug(enabled bool) {
	if enabled {
		a.options = append(a.options, option.WithDebugLog(nil))
		a.reset()
	}
}

// BaseURL sets the API base URL.
func BaseURL(s string) Option {
	return func(a *API) error { a.options = append(a.options, option.WithBaseURL(s)); a.reset(); return nil }
}

// HTTPClient sets the HTTP client.
func HTTPClient(c *http.Client) Option {
	return func(a *API) error { a.options = append(a.options, option.WithHTTPClient(c)); a.reset(); return nil }
}

// UsingRetryPolicy sets the maximum number of retries.
func UsingRetryPolicy(m int, _, _ int) Option {
	return func(a *API) error { a.options = append(a.options, option.WithMaxRetries(m)); a.reset(); return nil }
}

type envelope[T any] struct {
	Success    bool            `json:"success"`
	Result     T               `json:"result"`
	Errors     json.RawMessage `json:"errors"`
	ResultInfo struct {
		Page       int `json:"page"`
		TotalPages int `json:"total_pages"`
	} `json:"result_info"`
}

func (a *API) request(ctx context.Context, method, path string, body any, result any) error {
	if method == http.MethodDelete && result == nil {
		return a.client.Delete(ctx, path, body, nil)
	}
	var response envelope[json.RawMessage]
	if err := a.client.Execute(ctx, method, path, body, &response); err != nil {
		return err
	}
	if !response.Success {
		return fmt.Errorf("cloudflare API rejected %s %s: %s", method, path, response.Errors)
	}
	if result != nil && len(response.Result) > 0 {
		return json.Unmarshal(response.Result, result)
	}
	return nil
}

func (a *API) list(ctx context.Context, path string, result any) error {
	return a.request(ctx, http.MethodGet, path, nil, result)
}

// Zone is a Cloudflare zone.
type Zone struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	NameServers []string `json:"name_servers"`
}

// Account is a Cloudflare account.
type Account struct {
	ID string `json:"id"`
}

// DNSRecordSettings holds per-record settings.
type DNSRecordSettings struct {
	FlattenCNAME *bool `json:"flatten_cname,omitempty"`
}

// DNSRecord is a Cloudflare DNS record.
type DNSRecord struct {
	ID        string            `json:"id,omitempty"`
	Type      string            `json:"type,omitempty"`
	Name      string            `json:"name,omitempty"`
	Content   string            `json:"content,omitempty"`
	Meta      any               `json:"meta,omitempty"`
	Data      any               `json:"data,omitempty"`
	Priority  *uint16           `json:"priority,omitempty"`
	TTL       int               `json:"ttl,omitempty"`
	Proxied   *bool             `json:"proxied,omitempty"`
	Proxiable bool              `json:"proxiable,omitempty"`
	Comment   string            `json:"comment,omitempty"`
	Tags      []string          `json:"tags,omitempty"`
	Settings  DNSRecordSettings `json:"settings"`
}

// CreateDNSRecordParams is the request body for creating a DNS record.
type CreateDNSRecordParams DNSRecord

// UpdateDNSRecordParams is the request body for updating a DNS record.
type UpdateDNSRecordParams struct {
	ID       string            `json:"-"`
	Type     string            `json:"type,omitempty"`
	Name     string            `json:"name,omitempty"`
	Content  string            `json:"content,omitempty"`
	Data     any               `json:"data,omitempty"`
	Priority *uint16           `json:"priority,omitempty"`
	TTL      int               `json:"ttl,omitempty"`
	Proxied  *bool             `json:"proxied,omitempty"`
	Comment  *string           `json:"comment,omitempty"`
	Tags     []string          `json:"tags"`
	Settings DNSRecordSettings `json:"settings"`
}

// ListDNSRecordsParams is the parameters for listing DNS records.
type ListDNSRecordsParams struct{}

// WorkerRoute is a Cloudflare Worker route.
type WorkerRoute struct {
	ID         string `json:"id,omitempty"`
	Pattern    string `json:"pattern"`
	ScriptName string `json:"script,omitempty"`
}

// ListWorkerRoutesParams is the parameters for listing Worker routes.
type ListWorkerRoutesParams struct{}

// CreateWorkerRouteParams is the request body for creating a Worker route.
type CreateWorkerRouteParams struct {
	Pattern string `json:"pattern"`
	Script  string `json:"script,omitempty"`
}

// WorkerRoutesResponse is a list of Worker routes.
type WorkerRoutesResponse struct{ Routes []WorkerRoute }

// CreateWorkerParams is the parameters for uploading a Worker script.
type CreateWorkerParams struct {
	ScriptName string
	Script     string
}

// PageRule is a Cloudflare page rule.
type PageRule struct {
	ID string `json:"id"`
}

// UniversalSSLSetting is a zone's Universal SSL setting.
type UniversalSSLSetting struct {
	Enabled bool `json:"enabled"`
}

// Ruleset is a Cloudflare ruleset.
type Ruleset struct {
	ID    string        `json:"id,omitempty"`
	Rules []RulesetRule `json:"rules"`
}

// RulesetRule is a rule within a Ruleset.
type RulesetRule struct {
	ID               string                       `json:"id,omitempty"`
	Action           string                       `json:"action"`
	Description      string                       `json:"description,omitempty"`
	Expression       string                       `json:"expression"`
	ActionParameters *RulesetRuleActionParameters `json:"action_parameters,omitempty"`
	Enabled          *bool                        `json:"enabled,omitempty"`
	Ref              string                       `json:"ref,omitempty"`
}

// RulesetRuleActionParameters holds a rule's action parameters.
type RulesetRuleActionParameters struct {
	FromValue *RulesetRuleActionParametersFromValue `json:"from_value,omitempty"`
}

// RulesetRuleActionParametersFromValue holds redirect action parameters.
type RulesetRuleActionParametersFromValue struct {
	StatusCode          uint16                               `json:"status_code,omitempty"`
	TargetURL           RulesetRuleActionParametersTargetURL `json:"target_url"`
	PreserveQueryString *bool                                `json:"preserve_query_string,omitempty"`
}

// RulesetRuleActionParametersTargetURL is a redirect's target URL.
type RulesetRuleActionParametersTargetURL struct {
	Expression string `json:"expression,omitempty"`
}

// UpdateEntrypointRulesetParams is the request body for updating an entrypoint ruleset.
type UpdateEntrypointRulesetParams struct {
	Phase string        `json:"-"`
	Rules []RulesetRule `json:"rules"`
}

// DeleteRulesetRuleParams identifies a ruleset rule to delete.
type DeleteRulesetRuleParams struct {
	RulesetID     string
	RulesetRuleID string
}

// NotFoundError indicates the requested resource does not exist.
type NotFoundError struct{ Err error }

func (e *NotFoundError) Error() string { return e.Err.Error() }
func (e *NotFoundError) Unwrap() error { return e.Err }

// ListZones returns all zones.
func (a *API) ListZones(ctx context.Context) ([]Zone, error) {
	var all []Zone
	for page := 1; ; page++ {
		var response envelope[[]Zone]
		path := "/zones?page=" + strconv.Itoa(page) + "&per_page=50"
		if err := a.client.Get(ctx, path, nil, &response); err != nil {
			return nil, err
		}
		if !response.Success {
			return nil, fmt.Errorf("cloudflare API rejected GET %s: %s", path, response.Errors)
		}
		all = append(all, response.Result...)
		if response.ResultInfo.TotalPages <= page {
			return all, nil
		}
	}
}

// CreateZone creates a zone.
func (a *API) CreateZone(ctx context.Context, name string, jumpStart bool, account Account, zoneType string) (Zone, error) {
	var z Zone
	err := a.request(ctx, http.MethodPost, "/zones", struct {
		Name      string  `json:"name"`
		JumpStart bool    `json:"jump_start"`
		Account   Account `json:"account"`
		Type      string  `json:"type"`
	}{name, jumpStart, account, zoneType}, &z)
	return z, err
}

// ZoneIdentifier returns the zone ID unchanged.
func ZoneIdentifier(id string) string { return id }

// AccountIdentifier returns the account ID unchanged.
func AccountIdentifier(id string) string { return id }

// ListDNSRecords returns all DNS records in a zone.
func (a *API) ListDNSRecords(ctx context.Context, zone string, _ ListDNSRecordsParams) ([]DNSRecord, any, error) {
	var all []DNSRecord
	for page := 1; ; page++ {
		var response envelope[[]DNSRecord]
		path := fmt.Sprintf("/zones/%s/dns_records?page=%d&per_page=100", url.PathEscape(zone), page)
		if err := a.client.Get(ctx, path, nil, &response); err != nil {
			return nil, nil, err
		}
		if !response.Success {
			return nil, nil, fmt.Errorf("cloudflare API rejected GET %s: %s", path, response.Errors)
		}
		all = append(all, response.Result...)
		if response.ResultInfo.TotalPages <= page {
			return all, nil, nil
		}
	}
}

// CreateDNSRecord creates a DNS record.
func (a *API) CreateDNSRecord(ctx context.Context, zone string, params CreateDNSRecordParams) (DNSRecord, error) {
	var rec DNSRecord
	err := a.request(ctx, http.MethodPost, "/zones/"+url.PathEscape(zone)+"/dns_records", params, &rec)
	return rec, err
}

// UpdateDNSRecord updates a DNS record.
func (a *API) UpdateDNSRecord(ctx context.Context, zone string, params UpdateDNSRecordParams) (DNSRecord, error) {
	var rec DNSRecord
	err := a.request(ctx, http.MethodPut, "/zones/"+url.PathEscape(zone)+"/dns_records/"+url.PathEscape(params.ID), params, &rec)
	return rec, err
}

// DeleteDNSRecord deletes a DNS record.
func (a *API) DeleteDNSRecord(ctx context.Context, zone, id string) error {
	return a.request(ctx, http.MethodDelete, "/zones/"+url.PathEscape(zone)+"/dns_records/"+url.PathEscape(id), nil, nil)
}

// EditUniversalSSLSetting updates a zone's Universal SSL setting.
func (a *API) EditUniversalSSLSetting(ctx context.Context, zone string, setting UniversalSSLSetting) (UniversalSSLSetting, error) {
	var result UniversalSSLSetting
	err := a.request(ctx, http.MethodPatch, "/zones/"+url.PathEscape(zone)+"/ssl/universal/settings", setting, &result)
	return result, err
}

// UniversalSSLSettingDetails returns a zone's Universal SSL setting.
func (a *API) UniversalSSLSettingDetails(ctx context.Context, zone string) (UniversalSSLSetting, error) {
	var result UniversalSSLSetting
	err := a.list(ctx, "/zones/"+url.PathEscape(zone)+"/ssl/universal/settings", &result)
	return result, err
}

// GetEntrypointRuleset returns a zone's entrypoint ruleset for a phase.
func (a *API) GetEntrypointRuleset(ctx context.Context, zone, phase string) (Ruleset, error) {
	var result Ruleset
	err := a.list(ctx, fmt.Sprintf("/zones/%s/rulesets/phases/%s/entrypoint", url.PathEscape(zone), url.PathEscape(phase)), &result)
	if err != nil && statusCode(err) == http.StatusNotFound {
		return result, &NotFoundError{err}
	}
	return result, err
}

// UpdateEntrypointRuleset replaces a zone's entrypoint ruleset for a phase.
func (a *API) UpdateEntrypointRuleset(ctx context.Context, zone string, params UpdateEntrypointRulesetParams) (Ruleset, error) {
	var result Ruleset
	err := a.request(ctx, http.MethodPut, fmt.Sprintf("/zones/%s/rulesets/phases/%s/entrypoint", url.PathEscape(zone), url.PathEscape(params.Phase)), params, &result)
	return result, err
}

// DeleteRulesetRule deletes a rule from a ruleset.
func (a *API) DeleteRulesetRule(ctx context.Context, zone string, params DeleteRulesetRuleParams) error {
	return a.request(ctx, http.MethodDelete, fmt.Sprintf("/zones/%s/rulesets/%s/rules/%s", url.PathEscape(zone), url.PathEscape(params.RulesetID), url.PathEscape(params.RulesetRuleID)), nil, nil)
}

// RawResult is the status portion of an API response.
type RawResult struct {
	Success bool            `json:"success"`
	Errors  json.RawMessage `json:"errors"`
}

// Raw performs an arbitrary API request.
func (a *API) Raw(ctx context.Context, method, path string, body any, _ any) (RawResult, error) {
	var result RawResult
	err := a.client.Execute(ctx, method, path, body, &result)
	return result, err
}

// ListWorkerRoutes returns a zone's Worker routes.
func (a *API) ListWorkerRoutes(ctx context.Context, zone string, _ ListWorkerRoutesParams) (WorkerRoutesResponse, error) {
	var result WorkerRoutesResponse
	err := a.list(ctx, "/zones/"+url.PathEscape(zone)+"/workers/routes", &result.Routes)
	return result, err
}

// DeleteWorkerRoute deletes a Worker route.
func (a *API) DeleteWorkerRoute(ctx context.Context, zone, id string) (WorkerRoute, error) {
	var result WorkerRoute
	err := a.request(ctx, http.MethodDelete, "/zones/"+url.PathEscape(zone)+"/workers/routes/"+url.PathEscape(id), nil, &result)
	return result, err
}

// CreateWorkerRoute creates a Worker route.
func (a *API) CreateWorkerRoute(ctx context.Context, zone string, params CreateWorkerRouteParams) (WorkerRoute, error) {
	var result WorkerRoute
	err := a.request(ctx, http.MethodPost, "/zones/"+url.PathEscape(zone)+"/workers/routes", params, &result)
	return result, err
}

// UploadWorker uploads a Worker script.
func (a *API) UploadWorker(ctx context.Context, account string, params CreateWorkerParams) (any, error) {
	// Workers scripts use a JavaScript body rather than the JSON body used by the other endpoints.
	var result envelope[json.RawMessage]
	err := a.client.Put(ctx, "/accounts/"+url.PathEscape(account)+"/workers/scripts/"+url.PathEscape(params.ScriptName), nil, &result, option.WithRequestBody("application/javascript", []byte(params.Script)))
	return result.Result, err
}

func statusCode(err error) int {
	if apiErr, ok := errors.AsType[*cloudflare.Error](err); ok {
		return apiErr.StatusCode
	}
	return 0
}
