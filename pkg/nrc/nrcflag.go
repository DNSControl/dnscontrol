package nrc

type Flags struct {
	// EnforceOneDotPolicy instructs mustbe.Target* functions to return an error
	// if it looks like the user forgot the trailing dot. This is enforced by
	// the rule, "If a hostname contains any dots (even a single dot), there
	// must be a trailing dot."
	EnforceOneDotPolicy bool

	// SrvWeirdSplit the last field contains multiple fields.
	// Only for use with NewRecordConfigParse() && rtype=="SRV" && len(args) == 2.
	// Some APIs deliver SRV fields as two strings: "priority" and "weight port hostname."
	// This happens often enough to warrant a flag to handle this case.
	// true: last 2 args are combined and sent to NewRecordConfigParse()
	SrvWeirdSplit bool

	// TargetIsFqdnNoDot modifies how RDATA host target fields are parsed.
	// Normally a target host for MX, CNAME, and others, is assumed to be a
	// shortname (no dot) or FQDN (dot). When this flag is set, it is assumed to be a FQDN no matter what.
	// false: ends with ".": FQDN; no dot: shortname.
	// true: ends with ".": FQDN; no dot: FQDN.
	TargetIsFqdnNoDot bool

	// TxtDontParse tells NewRecordConfigParse() that the TXT data is raw bytes, not to be parsed.
	TxtDontParse bool
}
