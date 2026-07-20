package cfsingleredirect

import (
	"fmt"

	"github.com/DNSControl/dnscontrol/v4/models"
	"github.com/DNSControl/dnscontrol/v4/pkg/domaintags"
	"github.com/DNSControl/dnscontrol/v4/pkg/rtypecontrol"
)

func init() {
	rtypecontrol.Register(&CfRedirect{})
	rtypecontrol.Register(&CfTempRedirect{})
}

// CfRedirect represents the CF_REDIRECT rtype, which is a builder that produces CLOUDFLAREAPI_SINGLE_REDIRECT.
type CfRedirect struct{}

// Name returns the text (all caps) name of the rtype.
func (handle *CfRedirect) Name() string {
	return "CF_REDIRECT"
}

// FromArgs populates a RecordConfig from the raw ([]any) args.
func (handle *CfRedirect) FromArgs(dcn *domaintags.DomainNameVarieties, rec *models.RecordConfig, args []any) error {
	return fromArgsHelper(dcn, rec, args, 301)
}

// FromStruct populates a RecordConfig from a struct, which will be stored in rec.F.
func (handle *CfRedirect) FromStruct(dcn *domaintags.DomainNameVarieties, rec *models.RecordConfig, name string, fields any) error {
	panic("CF_REDIRECT: FromStruct not implemented")
}

// CopyToLegacyFields copies data from rec.F to the legacy fields in rec.
func (handle *CfRedirect) CopyToLegacyFields(rec *models.RecordConfig) {
	// Nothing needs to be copied.  The CLOUDFLAREAPI_SINGLE_REDIRECT FromArgs copies everything needed.
}

// CopyFromLegacyFields populates rec.F from the legacy RecordType fields.
func (handle *CfRedirect) CopyFromLegacyFields(rec *models.RecordConfig) {
	// Nothing needs to be copied.  The CLOUDFLAREAPI_SINGLE_REDIRECT is built in FromArgs.

	// However, we add some assertions here to catch mistakes.
	if rec.F == nil {
		panic("assertion failed: CfRedirect CopyFromLegacyFields called with rec.F == nil")
	}
	if rec.ZonefilePartial == "" {
		panic("assertion failed: CfRedirect CopyFromLegacyFields called with rec.ZonefilePartial == \"\"")
	}
	if rec.Comparable == "" {
		panic("assertion failed: CfRedirect CopyFromLegacyFields called with rec.Comparable == \"\"")
	}
}

// CfTempRedirect represents the CF_TEMP_REDIRECT rtype, which is a builder that produces CLOUDFLAREAPI_SINGLE_REDIRECT.
type CfTempRedirect struct{}

// Name returns the text (all caps) name of the rtype.
func (handle *CfTempRedirect) Name() string {
	return "CF_TEMP_REDIRECT"
}

// FromArgs populates a RecordConfig from the raw ([]any) args.
func (handle *CfTempRedirect) FromArgs(dcn *domaintags.DomainNameVarieties, rec *models.RecordConfig, args []any) error {
	return fromArgsHelper(dcn, rec, args, 302)
}

// FromStruct populates a RecordConfig from a struct, which will be stored in rec.F.
func (handle *CfTempRedirect) FromStruct(dcn *domaintags.DomainNameVarieties, rec *models.RecordConfig, name string, fields any) error {
	panic("CF_TEMP_REDIRECT: FromStruct not implemented")
}

// CopyToLegacyFields copies data from rec.F to the legacy fields in rec.
func (handle *CfTempRedirect) CopyToLegacyFields(rec *models.RecordConfig) {
	// Nothing needs to be copied.  The CLOUDFLAREAPI_SINGLE_REDIRECT FromArgs copies everything needed.
}

// CopyFromLegacyFields copies data from rec.F to the legacy fields in rec.
func (handle *CfTempRedirect) CopyFromLegacyFields(rec *models.RecordConfig) {
	// Nothing needs to be copied.  The CLOUDFLAREAPI_SINGLE_REDIRECT is built in FromArgs.

	// However, we add some assertions here to catch mistakes.
	if rec.F == nil {
		panic("assertion failed: CfTempRedirect CopyFromLegacyFields called with rec.F == nil")
	}
	if rec.ZonefilePartial == "" {
		panic("assertion failed: CfTempRedirect CopyFromLegacyFields called with rec.ZonefilePartial == \"\"")
	}
	if rec.Comparable == "" {
		panic("assertion failed: CfTempRedirect CopyFromLegacyFields called with rec.Comparable == \"\"")
	}
}

func fromArgsHelper(dcn *domaintags.DomainNameVarieties, rec *models.RecordConfig, args []any, code int) error {

	// Pave the args to be the expected types.
	if err := rtypecontrol.PaveArgs(args, "ss"); err != nil {
		return err
	}

	// Convert old-style patterns to new-style rules:
	prWhen := args[0].(string)
	prThen := args[1].(string)
	srWhen, srThen, err := makeRuleFromPattern(prWhen, prThen)
	if err != nil {
		return err
	}

	// Create a rule name:
	name := fmt.Sprintf("%03d,%s,%s", code, prWhen, prThen)

	sr := SingleRedirectConfig{}
	rec.Type = sr.Name() // This record is now a CLOUDFLAREAPI_SINGLE_REDIRECT
	err = sr.FromArgs(dcn, rec, []any{name, code, srWhen, srThen})
	if err != nil {
		return err
	}

	return nil
}
