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

type API struct {
	client   *cloudflare.Client
	options  []option.RequestOption
	APIEmail string
}

type Option func(*API) error

func NewWithAPIToken(token string, opts ...Option) (*API, error) {
	return newAPI([]option.RequestOption{option.WithAPIToken(token)}, "", opts...)
}

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
func (a *API) SetDebug(enabled bool) {
	if enabled {
		a.options = append(a.options, option.WithDebugLog(nil))
		a.reset()
	}
}
func BaseURL(s string) Option {
	return func(a *API) error { a.options = append(a.options, option.WithBaseURL(s)); a.reset(); return nil }
}
func HTTPClient(c *http.Client) Option {
	return func(a *API) error { a.options = append(a.options, option.WithHTTPClient(c)); a.reset(); return nil }
}
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

type Zone struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	NameServers []string `json:"name_servers"`
}
type Account struct {
	ID string `json:"id"`
}
type DNSRecordSettings struct {
	FlattenCNAME *bool `json:"flatten_cname,omitempty"`
}
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
type CreateDNSRecordParams DNSRecord
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
type ListDNSRecordsParams struct{}
type WorkerRoute struct {
	ID         string `json:"id,omitempty"`
	Pattern    string `json:"pattern"`
	ScriptName string `json:"script,omitempty"`
}
type ListWorkerRoutesParams struct{}
type CreateWorkerRouteParams struct {
	Pattern string `json:"pattern"`
	Script  string `json:"script,omitempty"`
}
type WorkerRoutesResponse struct{ Routes []WorkerRoute }
type CreateWorkerParams struct {
	ScriptName string
	Script     string
}
type PageRule struct {
	ID string `json:"id"`
}
type UniversalSSLSetting struct {
	Enabled bool `json:"enabled"`
}

type Ruleset struct {
	ID    string        `json:"id,omitempty"`
	Rules []RulesetRule `json:"rules"`
}
type RulesetRule struct {
	ID               string                       `json:"id,omitempty"`
	Action           string                       `json:"action"`
	Description      string                       `json:"description,omitempty"`
	Expression       string                       `json:"expression"`
	ActionParameters *RulesetRuleActionParameters `json:"action_parameters,omitempty"`
	Enabled          *bool                        `json:"enabled,omitempty"`
	Ref              string                       `json:"ref,omitempty"`
}
type RulesetRuleActionParameters struct {
	FromValue *RulesetRuleActionParametersFromValue `json:"from_value,omitempty"`
}
type RulesetRuleActionParametersFromValue struct {
	StatusCode          uint16                               `json:"status_code,omitempty"`
	TargetURL           RulesetRuleActionParametersTargetURL `json:"target_url"`
	PreserveQueryString *bool                                `json:"preserve_query_string,omitempty"`
}
type RulesetRuleActionParametersTargetURL struct {
	Expression string `json:"expression,omitempty"`
}
type UpdateEntrypointRulesetParams struct {
	Phase string        `json:"-"`
	Rules []RulesetRule `json:"rules"`
}
type DeleteRulesetRuleParams struct {
	RulesetID     string
	RulesetRuleID string
}
type NotFoundError struct{ Err error }

func (e *NotFoundError) Error() string { return e.Err.Error() }
func (e *NotFoundError) Unwrap() error { return e.Err }

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
func ZoneIdentifier(id string) string    { return id }
func AccountIdentifier(id string) string { return id }
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
func (a *API) CreateDNSRecord(ctx context.Context, zone string, params CreateDNSRecordParams) (DNSRecord, error) {
	var rec DNSRecord
	err := a.request(ctx, http.MethodPost, "/zones/"+url.PathEscape(zone)+"/dns_records", params, &rec)
	return rec, err
}
func (a *API) UpdateDNSRecord(ctx context.Context, zone string, params UpdateDNSRecordParams) (DNSRecord, error) {
	var rec DNSRecord
	err := a.request(ctx, http.MethodPut, "/zones/"+url.PathEscape(zone)+"/dns_records/"+url.PathEscape(params.ID), params, &rec)
	return rec, err
}
func (a *API) DeleteDNSRecord(ctx context.Context, zone, id string) error {
	return a.request(ctx, http.MethodDelete, "/zones/"+url.PathEscape(zone)+"/dns_records/"+url.PathEscape(id), nil, nil)
}
func (a *API) EditUniversalSSLSetting(ctx context.Context, zone string, setting UniversalSSLSetting) (UniversalSSLSetting, error) {
	var result UniversalSSLSetting
	err := a.request(ctx, http.MethodPatch, "/zones/"+url.PathEscape(zone)+"/ssl/universal/settings", setting, &result)
	return result, err
}
func (a *API) UniversalSSLSettingDetails(ctx context.Context, zone string) (UniversalSSLSetting, error) {
	var result UniversalSSLSetting
	err := a.list(ctx, "/zones/"+url.PathEscape(zone)+"/ssl/universal/settings", &result)
	return result, err
}
func (a *API) GetEntrypointRuleset(ctx context.Context, zone, phase string) (Ruleset, error) {
	var result Ruleset
	err := a.list(ctx, fmt.Sprintf("/zones/%s/rulesets/phases/%s/entrypoint", url.PathEscape(zone), url.PathEscape(phase)), &result)
	if err != nil && statusCode(err) == http.StatusNotFound {
		return result, &NotFoundError{err}
	}
	return result, err
}
func (a *API) UpdateEntrypointRuleset(ctx context.Context, zone string, params UpdateEntrypointRulesetParams) (Ruleset, error) {
	var result Ruleset
	err := a.request(ctx, http.MethodPut, fmt.Sprintf("/zones/%s/rulesets/phases/%s/entrypoint", url.PathEscape(zone), url.PathEscape(params.Phase)), params, &result)
	return result, err
}
func (a *API) DeleteRulesetRule(ctx context.Context, zone string, params DeleteRulesetRuleParams) error {
	return a.request(ctx, http.MethodDelete, fmt.Sprintf("/zones/%s/rulesets/%s/rules/%s", url.PathEscape(zone), url.PathEscape(params.RulesetID), url.PathEscape(params.RulesetRuleID)), nil, nil)
}

type RawResult struct {
	Success bool            `json:"success"`
	Errors  json.RawMessage `json:"errors"`
}

func (a *API) Raw(ctx context.Context, method, path string, body any, _ any) (RawResult, error) {
	var result RawResult
	err := a.client.Execute(ctx, method, path, body, &result)
	return result, err
}
func (a *API) ListWorkerRoutes(ctx context.Context, zone string, _ ListWorkerRoutesParams) (WorkerRoutesResponse, error) {
	var result WorkerRoutesResponse
	err := a.list(ctx, "/zones/"+url.PathEscape(zone)+"/workers/routes", &result.Routes)
	return result, err
}
func (a *API) DeleteWorkerRoute(ctx context.Context, zone, id string) (WorkerRoute, error) {
	var result WorkerRoute
	err := a.request(ctx, http.MethodDelete, "/zones/"+url.PathEscape(zone)+"/workers/routes/"+url.PathEscape(id), nil, &result)
	return result, err
}
func (a *API) CreateWorkerRoute(ctx context.Context, zone string, params CreateWorkerRouteParams) (WorkerRoute, error) {
	var result WorkerRoute
	err := a.request(ctx, http.MethodPost, "/zones/"+url.PathEscape(zone)+"/workers/routes", params, &result)
	return result, err
}
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
