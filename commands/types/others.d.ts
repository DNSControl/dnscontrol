declare function require(name: `${string}.json`): any;
declare function require(name: `${string}.json5`): any;
declare function require(name: string): true;

/**
 * Issuer critical flag. CA that does not understand this tag will refuse to issue certificate for this domain.
 *
 * CAA record is supported only by BIND, Google Cloud DNS, Amazon Route 53 and OVH. Some certificate authorities may not support this record until the mandatory date of September 2017.
 */
declare const CAA_CRITICAL: RecordModifier;

/**
 * @deprecated
 * This disables a safety check intended to prevent:
 * 1. Two owners toggling a record between two settings.
 * 2. The other owner wiping all records at this label, which won't
 * be noticed until the next time dnscontrol is run.
 * See https://github.com/DNSControl/dnscontrol/issues/1106
 */
declare const IGNORE_NAME_DISABLE_SAFETY_CHECK: RecordModifier;

// Cloudflare aliases:

/** Proxy disabled. */
declare const CF_PROXY_OFF: RecordModifier;
/** Proxy enabled. */
declare const CF_PROXY_ON: RecordModifier;
/** Proxy+Railgun enabled. */
declare const CF_PROXY_FULL: RecordModifier;
/** Per-record CNAME flattening disabled (default) */
declare const CF_CNAME_FLATTEN_OFF: RecordModifier;
/** Per-record CNAME flattening enabled (requires Cloudflare paid plan) */
declare const CF_CNAME_FLATTEN_ON: RecordModifier;

/** Proxy default off for entire domain (the default) */
declare const CF_PROXY_DEFAULT_OFF: DomainModifier;
/** Proxy default on for entire domain */
declare const CF_PROXY_DEFAULT_ON: DomainModifier;
/** UniversalSSL off for entire domain */
declare const CF_UNIVERSALSSL_OFF: DomainModifier;
/** UniversalSSL on for entire domain */
declare const CF_UNIVERSALSSL_ON: DomainModifier;
/** Set a comment on a DNS record (works on all Cloudflare plans) */
declare function CF_COMMENT(comment: string): RecordModifier;
/** Set tags on a DNS record (requires Cloudflare paid plan) */
declare function CF_TAGS(...tags: string[]): RecordModifier;
/** Enable comment management for this domain (opt-in to sync comments) */
declare const CF_MANAGE_COMMENTS: DomainModifier;
/** Enable tag management for this domain (opt-in to sync tags, requires paid plan) */
declare const CF_MANAGE_TAGS: DomainModifier;

/**
 * Set default values for CLI variables. See: https://dnscontrol.org/cli-variables
 */
declare function CLI_DEFAULTS(vars: Record<string, unknown>): void;

/**
 * `END` permits the last item to include a comma.
 *
 * ```js
 * D("foo.com", ...
 *    A(...),
 *    A(...),
 *    A(...),
 * END)
 * ```
 */
declare const END: DomainModifier & RecordModifier;

/**
 * Permit labels like `"foo.bar.com.bar.com"` (normally an error)
 *
 * ```js
 * D("bar.com", ...
 *     A("foo.bar.com", "10.1.1.1", DISABLE_REPEATED_DOMAIN_CHECK),
 * )
 * ```
 */
declare const DISABLE_REPEATED_DOMAIN_CHECK: RecordModifier;
