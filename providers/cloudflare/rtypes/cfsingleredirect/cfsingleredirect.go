package cfsingleredirect

import (
	"fmt"

	"github.com/DNSControl/dnscontrol/v4/models"
	"github.com/DNSControl/dnscontrol/v4/pkg/domaintags"
	"github.com/DNSControl/dnscontrol/v4/pkg/rtypecontrol"
)

func init() {
	rtypecontrol.Register(&SingleRedirectConfig{})
}

// SingleRedirectConfig contains info about a Cloudflare Single Redirect.
type SingleRedirectConfig struct {
	//
	Code uint16 `json:"code,omitempty"` // 301 or 302
	//
	// SR == SingleRedirect
	SRName           string `json:"sr_name,omitempty"`          // How is this displayed to the user
	SRWhen           string `json:"sr_when,omitempty"`          // Condition for redirect
	SRThen           string `json:"sr_then,omitempty"`          // Formula for redirect
	SRRRulesetID     string `json:"sr_rulesetid,omitempty"`     // ID of the ruleset containing this rule (populated by API)
	SRRRulesetRuleID string `json:"sr_rulesetruleid,omitempty"` // ID of this rule within the ruleset (populated by API)
	SRDisplay        string `json:"sr_display,omitempty"`       // How is this displayed to the user (SetTarget) for CF_SINGLE_REDIRECT
}

// Name returns the text (all caps) name of the rtype.
func (handle *SingleRedirectConfig) Name() string {
	return "CLOUDFLAREAPI_SINGLE_REDIRECT"
}

// FromArgs populates a RecordConfig from the raw ([]any) args.
func (handle *SingleRedirectConfig) FromArgs(dcn *domaintags.DomainNameVarieties, rec *models.RecordConfig, args []any) error {
	// Pave the args to be the expected types.
	if err := rtypecontrol.PaveArgs(args, "swss"); err != nil {
		return err
	}

	// Unpack the args:
	var name = args[0].(string)
	var code = args[1].(uint16)
	var when = args[2].(string)
	var then = args[3].(string)

	// Validate
	if code != 301 && code != 302 && code != 303 && code != 307 && code != 308 {
		return fmt.Errorf("%s: code (%03d) is not 301,302,303,307,308", rec.FilePos, code)
	}

	// Calclate the Comparable and ZonefilePartial values:
	display := targetFromRaw(name, code, when, then)
	rec.Comparable = display
	rec.ZonefilePartial = display

	// Set the fields
	rec.F = &SingleRedirectConfig{
		Code:      code,
		SRName:    name,
		SRWhen:    when,
		SRThen:    then,
		SRDisplay: display,
	}

	// Usually these fields do not need to be changed.  The caller sets appropriate values.
	// But Cloudflare Single Redirects always use "@" as the name and TTL=1.  We override here.
	rec.Name = "@"
	rec.NameRaw = "@"
	rec.NameUnicode = "@"
	rec.NameFQDN = dcn.NameASCII
	rec.NameFQDNRaw = dcn.NameRaw
	rec.NameFQDNUnicode = dcn.NameUnicode
	rec.TTL = 1

	// Fill in the legacy fields:
	handle.CopyToLegacyFields(rec)
	return nil
}

// FromStruct populates a RecordConfig from a struct, which will be stored in rec.F.
func (handle *SingleRedirectConfig) FromStruct(dcn *domaintags.DomainNameVarieties, rec *models.RecordConfig, name string, fields any) error {
	panic("CLOUDFLAREAPI_SINGLE_REDIRECT: FromStruct not implemented")
}

// targetFromRaw create the display text used for a normal Redirect.
func targetFromRaw(name string, code uint16, when, then string) string {
	return fmt.Sprintf("name=(%s) code=(%03d) when=(%s) then=(%s)",
		name,
		code,
		when,
		then,
	)
}

// CopyToLegacyFields copies data from rec.F to the legacy fields in rec.
func (handle *SingleRedirectConfig) CopyToLegacyFields(rec *models.RecordConfig) {
	_ = rec.SetTarget(rec.F.(*SingleRedirectConfig).SRDisplay)
}

// CopyFromLegacyFields populates rec.F from the legacy RecordType fields.
func (handle *SingleRedirectConfig) CopyFromLegacyFields(rec *models.RecordConfig) {
	// Nothing needs to be copied.  The CLOUDFLAREAPI_SINGLE_REDIRECT is built in FromArgs.

	// However, we add some assertions here to catch mistakes.
	if rec.F == nil {
		panic("assertion failed: SingleRedirectConfig CopyFromLegacyFields called with rec.F == nil")
	}
	if rec.ZonefilePartial == "" {
		panic("assertion failed: SingleRedirectConfig CopyFromLegacyFields called with rec.ZonefilePartial == \"\"")
	}
	if rec.Comparable == "" {
		panic("assertion failed: SingleRedirectConfig CopyFromLegacyFields called with rec.Comparable == \"\"")
	}
}
