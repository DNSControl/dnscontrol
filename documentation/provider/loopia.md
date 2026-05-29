Loopia is a 💩 provider of DNS. Using DNSControl hides some of the 💩. If you are stuck with Loopia, hopefully this will reduce the pain.

They provide DNS services, both as a registrar, and a provider. They provide support in English and other regional variants (Norwegian, Serbian, Swedish).

This plugin is based on API documents found at [https://www.loopia.com/api/](https://www.loopia.com/api/) and by observing API responses. Hat tip to GitHub @hazzeh whose code for the LEGO Loopia implementation was helpful.

Sadly the Loopia API has some problems:
* API calls are limited to 60 calls per minute.  If you go above this, you will have to wait before you can make changes.
* When rate-limited, you will not receive a single HTTP error: The errors propagate from the back-end, with no headers, or Retry-After or anything useful.
* There are no guarantees of idempotency from their API.

## Unimplemented API methods

 * `removeDomain` is not implemented for safety reasons. Should you wish to remove a domain, do so from the Loopia control panel.
 * `addDomain`
 * `transferDomain` (to Loopia)

This effectively means that this plugin does not access registrar functions.

## Errors

You may occasionally see this error
```text
HTTP Post Error: Post "https://api.loopia.se/RPCSERV": context deadline exceeded (Client.Timeout exceeded while awaiting headers)
```

The API endpoint didn't answer. Try again. 🤷

## Configuration

To use this provider, add an entry to `creds.json` with `TYPE` set to `LOOPIA` along with your Loopia API login credentials.

Example:

{% code title="creds.json" %}
```json
{
  "loopia": {
    "TYPE": "LOOPIA",
    "username": "your-loopia-api-account-id@loopiaapi",
    "password": "your-loopia-api-account-password",
    "debug": "true" // Set to true for extra debug output. Remove or set to false to prevent extra debug output.
  }
}
```
{% endcode %}

### Variables

* `username` - string - your @loopiaapi created username
* `password` - string - your loopia API password
* `debug` - string - Set to true for extra debug output. Remove or set to false to prevent extra debug output.
* `rate_limit_per` - string - See [Rate Limiting](#rate-limiting) below.
* `region` - string - See [Regions](#regions) below.
* `modify_name_servers` - string - See [Modify Name Servers](#modify-name-servers) below.
* `fetch_apex_ns_entries` - string - See [Fetch NS Entries](#fetch-apex-ns-entries) below.

There is no test endpoint. Fly free, grasshopper.

Turning on debug will show the XML requests and responses, and include the username and password from your `creds.json` file. If you want to share these, like for a GitHub issue, be sure to redact those from the XML.

### Fetch Apex NS Entries

`creds.json` setting: `fetch_apex_ns_entries`

... or use locally hard-coded variables:

```go
  defaultNS1       = "ns1.loopia.se."
  defaultNS2       = "ns2.loopia.se."
```

API calls to loopia can be expensive time-wise. Set this to "false" (off) to skip the API call to fetch the apex (`@`) entries, and use Loopia's default NS servers.

This setting defaults to "true" (on).

### Modify Name Servers

`creds.json` setting: `modify_name_servers`

Setting this to "true" (on) allows you to modify NS entries.

Loopia is weird. NS entries are inaccessible in the control panel. But you can see them. Perhaps dnscontrol added an NS that you cannot delete now? Toggle this setting to "true" in order to treat all NS entries as any other - making them accessible to modification. Beware the consequences of changing from default NS entries. Likely nothing will happen since the glue records provided won't match those in the domain, and you will need to manually inform Loopia of this so they can update the glue records.

In short: enable this setting to be able to delete NS entries. No `NS()` in your `dnsconfig.js`? Existing ones will be deleted. Have some `NS()` or `NAMESERVER()` entries? They'll be added.

This setting defaults to "false" (off).

### Regions

`creds.json` setting: `region`

Loopia operate in a few regions. Norway (`no`), Serbia (`rs`), Sweden (`se`).

For the parameter `region`, specify one of `no`, `rs`, `se`, or omit, or leave empty for the default `se` Sweden.

As of writing, `no` was broken 💩 and produced:

```text
HTTP Post Error: Post "https://api.loopia.no/RPCSERV": x509: “*.loopia.rs” certificate name does not match input
```

### Rate Limiting

`creds.json` setting: `rate_limit_per`

Loopia rate limits requests to 60 per minute.

From their [web-site](https://www.loopia.com/api/rate_limiting/):
```text
You can make up to 60 calls per minute to LoopiaAPI. Of those, a maximum of 15 can be domain searches.
```

Depending on how many requests you make, you may encounter a limit. Modification of each DNS record requires at least one API call. 🤦

Example: If the rate is 60/min and you make two requests every second, the 31st request will be rejected. You will then have to wait for 29 seconds, until the first request’s age reaches one minute. At that time, it will be dropped from the calculation, and you can make another request. One second later, and generally every time an old request’s age falls out of the sliding window counting interval, you can make another request.

Your per minute quota is 60 requests and in your settings you specified `Minute`. DNSControl will perform at most one request per second. DNSControl will emit a warning in case it breaches the quota.

The setting `rate_limit_per` controls this behavior and accepts a case-insensitive value of
- `Hour`
- `Minute`
- `Second`

The default for `rate_limit_per` is `Second`.

In your `creds.json` for all `LOOPIA` provider entries:

{% code title="creds.json" %}
```json
{
  "loopia": {
    "TYPE": "LOOPIA",
    "username": "your-loopia-api-account-id@loopiaapi",
    "password": "your-loopia-api-account-password",
    "debug": "true", // Set to true for extra debug output. Remove or set to false to prevent extra debug output.
    "rate_limit_per": "Minute"
  }
}
```
{% endcode %}

## Usage

Here's an example DNS Configuration `dnsconfig.js` using the provider module. Even though it shows how you use Loopia as Domain Registrar AND DNS Provider, you're not forced to do that (thank god).

{% code title="dnsconfig.js" %}
```javascript
var REG_LOOPIA = NewRegistrar("loopia");
var DSP_LOOPIA = NewDnsProvider("loopia");

// Set Default TTL for all RR to reflect our Backend API Default
// If you use additional DNS Providers, configure a default TTL
// per domain using the domain modifier DefaultTTL instead.
DEFAULTS(
    NAMESERVER_TTL(3600),
    DefaultTTL(3600)
);

D("example.com", REG_LOOPIA, DnsProvider(DSP_LOOPIA),
    //NAMESERVER("ns1.loopia.se."), //default
    //NAMESERVER("ns2.loopia.se."), //default
    A("elk1", "192.0.2.1"),
    A("test", "192.0.2.2"),
);
```
{% endcode %}

## Special notes about newer standards

Loopia does not yet support [RFC7505](https://www.rfc-editor.org/rfc/rfc7505), so null `MX` records are currently prohibited.

Until such a time when they do begin to support this, Loopias `auditrecords.go` code prohibits this.

## Metadata

This provider does not recognize any special metadata fields unique to LOOPIA.

## get-zones

`dnscontrol get-zones` is implemented for this provider.

## New domains

If a dnszone does not exist in your LOOPIA account, DNSControl will *not* automatically add it with the `dnscontrol push` or `dnscontrol preview` command. You'll need to do that via the control panel manually or using the command `dnscontrol create-domains`. This is because it could lead to unwanted costs on customer-side that you may want to avoid.

## Debug Mode

As shown in the configuration examples above, this can be activated on demand and it can be used to check the API commands sent to Loopia.

## Feature Flags

<!-- provider-features-start -->
- Provider Type
  - [Official Support](../provider/index.md#providers-with-official-support): ❌
  - DNS Provider: ✅
  - Registrar: ✅
- Provider API
  - [Concurrency Verified](../advanced-features/concurrency-verified.md): ❔
  - [dual host](../advanced-features/dual-host.md): ✅
  - create-domains: ❌
  - [get-zones](../commands/get-zones.md): ✅
- DNS extensions
  - [`ALIAS`](../language-reference/domain-modifiers/ALIAS.md): ❌
  - [`DNAME`](../language-reference/domain-modifiers/DNAME.md): ❔
  - [`LOC`](../language-reference/domain-modifiers/LOC.md): ✅
  - [`PTR`](../language-reference/domain-modifiers/PTR.md): ❌
  - [`SOA`](../language-reference/domain-modifiers/SOA.md): ❌
- Service discovery
  - [`DHCID`](../language-reference/domain-modifiers/DHCID.md): ❌
  - [`NAPTR`](../language-reference/domain-modifiers/NAPTR.md): ✅
  - [`SRV`](../language-reference/domain-modifiers/SRV.md): ✅
  - [`SVCB`](../language-reference/domain-modifiers/SVCB.md): ❌
- Security
  - [`CAA`](../language-reference/domain-modifiers/CAA.md): ✅
  - [`HTTPS`](../language-reference/domain-modifiers/HTTPS.md): ❌
  - [`SMIMEA`](../language-reference/domain-modifiers/SMIMEA.md): ❔
  - [`SSHFP`](../language-reference/domain-modifiers/SSHFP.md): ✅
  - [`TLSA`](../language-reference/domain-modifiers/TLSA.md): ✅
- DNSSEC
  - [`AUTODNSSEC`](../language-reference/domain-modifiers/AUTODNSSEC_ON.md): ❌
  - [`DNSKEY`](../language-reference/domain-modifiers/DNSKEY.md): ❌
  - [`DS`](../language-reference/domain-modifiers/DS.md): ❌
<!-- provider-features-end -->
