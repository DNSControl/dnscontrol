## Important Note
Hurricane Electric does not currently expose an official JSON or XML API, and as such, this provider interacts directly
with the web interface. Because there is no officially supported API, this provider may cease to function if Hurricane
Electric changes their interface, and you should be willing to accept this possibility before relying on this provider.

## Configuration

To use this provider, add an entry to `creds.json` with `TYPE` set to `HEDNS`
along with
your `dns.he.net` account username and password. These are the same username
and password used to log in to the [web interface](https://dns.he.net).

{% code title="creds.json" %}
```json
{
  "hedns": {
    "TYPE": "HEDNS",
    "username": "yourUsername",
    "password": "yourPassword"
  }
}
```
{% endcode %}

### Two factor authentication

If two-factor authentication has been enabled on your account you will also need to provide a valid TOTP code.
This can also be done via an environment variable:

{% code title="creds.json" %}
```json
{
  "hedns": {
    "TYPE": "HEDNS",
    "username": "yourUsername",
    "password": "yourPassword",
    "totp": "$HEDNS_TOTP"
  }
}
```
{% endcode %}

and then you can run

```shell
HEDNS_TOTP=12345 dnscontrol preview
```

It is also possible to directly provide the shared TOTP secret using the key "totp-key" in `creds.json`. This secret is
only available when first enabling two-factor authentication.

**Security Warning**:
* Anyone with access to this `creds.json` file will have *full* access to your Hurricane Electric account and will be
  able to modify and delete your DNS entries
* Storing the shared secret together with the password weakens two factor authentication because both factors are stored
  in a single place.

{% code title="creds.json" %}
```json
{
  "hedns": {
    "TYPE": "HEDNS",
    "username": "yourUsername",
    "password": "yourPassword",
    "totp-key": "yourTOTPSharedSecret"
  }
}
```
{% endcode %}

### Persistent Sessions

Normally this provider will refresh authentication with each run of dnscontrol. This can lead to issues when using
two-factor authentication if two runs occur within the time period of a single TOTP token (30 seconds), as reusing the
same token is explicitly disallowed by RFC 6238 (TOTP).

To work around this limitation, if multiple requests need to be made, the option `"session-file-path"` can be set in
`creds.json`, which is the directory where a `.hedns-session` file will be created. This can be used to allow reuse of an
existing session between runs, without the need to re-authenticate.

This option is disabled by default when this key is not present,

**Security Warning**:
* Anyone with access to this `.hedns-session` file will be able to use the existing session (until it expires) and have
  *full* access to your Hurricane Electric account and will be able to modify and delete your DNS entries.
* It should be stored in a location only trusted users can access.

{% code title="creds.json" %}
```json
{
  "hedns": {
    "TYPE": "HEDNS",
    "username": "yourUsername",
    "password": "yourPassword",
    "totp-key": "yourTOTPSharedSecret",
    "session-file-path": "."
  }
}
```
{% endcode %}

## Metadata

This provider supports the following record-level metadata:

| Modifier | Description |
|---|---|
| `HEDNS_DYNAMIC_ON` | Enable [Dynamic DNS](https://dns.he.net/) on the record. The record will be assigned a DDNS key that can be used to update its value via the HE DDNS API (`https://dyn.dns.he.net/nic/update`). |
| `HEDNS_DYNAMIC_OFF` | Explicitly disable Dynamic DNS on the record. **Warning:** this will clear any associated DDNS key. |
| `HEDNS_DDNS_KEY("key")` | Enable Dynamic DNS and set a specific DDNS key (token) on the record. Implies `HEDNS_DYNAMIC_ON`. |

### Dynamic DNS behavior

* When a record has Dynamic DNS enabled and is subsequently modified by dnscontrol (e.g. TTL change), the dynamic flag is **preserved** automatically. You do not need to re-specify `HEDNS_DYNAMIC_ON` on every run unless you want to be explicit.
* If you do not specify any `HEDNS_DYNAMIC_*` modifier on a record that is already dynamic on the provider, the dynamic state is **inherited** — the record stays dynamic.
* DDNS keys are **write-only**: dnscontrol will set the key you specify but cannot read back the current key from HE DNS. This means:
  * A key-only change (same record data, new key) requires changing another field (e.g. TTL) to trigger an update.
  * The `get-zones` export will include `HEDNS_DYNAMIC_ON` for dynamic records but will not include the DDNS key.

## Usage

An example configuration:

{% code title="dnsconfig.js" %}
```javascript
var REG_NONE = NewRegistrar("none");
var DSP_HEDNS = NewDnsProvider("hedns");

D("example.com", REG_NONE, DnsProvider(DSP_HEDNS),
    // Standard static record
    A("test", "1.2.3.4"),

    // Dynamic DNS record (HE DNS assigns/preserves the DDNS key)
    A("dyn", "0.0.0.0", HEDNS_DYNAMIC_ON),

    // Dynamic DNS record with a specific DDNS key
    A("dyn2", "0.0.0.0", HEDNS_DDNS_KEY("my-secret-token")),

    // Dynamic AAAA record
    AAAA("dyn6", "::1", HEDNS_DYNAMIC_ON),

    // Explicitly non-dynamic record (clears any prior DDNS key)
    A("static", "5.6.7.8", HEDNS_DYNAMIC_OFF),
);
```
{% endcode %}

## Feature Flags

<!-- provider-features-start -->
- Provider Type
  - Official Support: ❌
  - DNS Provider: ✅
  - Registrar: ❌
- Provider API
  - [Concurrency Verified](../advanced-features/concurrency-verified.md): ❔
  - [dual host](../advanced-features/dual-host.md): ✅
  - create-domains: ✅
  - get-zones: ✅
- DNS extensions
  - [`ALIAS`](../language-reference/domain-modifiers/ALIAS.md): ✅
  - [`DNAME`](../language-reference/domain-modifiers/DNAME.md): ❔
  - [`LOC`](../language-reference/domain-modifiers/LOC.md): ✅
  - [`PTR`](../language-reference/domain-modifiers/PTR.md): ✅
  - [`SOA`](../language-reference/domain-modifiers/SOA.md): ❌
- Service discovery
  - [`DHCID`](../language-reference/domain-modifiers/DHCID.md): ❔
  - [`NAPTR`](../language-reference/domain-modifiers/NAPTR.md): ✅
  - [`SRV`](../language-reference/domain-modifiers/SRV.md): ✅
  - [`SVCB`](../language-reference/domain-modifiers/SVCB.md): ✅
- Security
  - [`CAA`](../language-reference/domain-modifiers/CAA.md): ✅
  - [`HTTPS`](../language-reference/domain-modifiers/HTTPS.md): ✅
  - [`SMIMEA`](../language-reference/domain-modifiers/SMIMEA.md): ❔
  - [`SSHFP`](../language-reference/domain-modifiers/SSHFP.md): ✅
  - [`TLSA`](../language-reference/domain-modifiers/TLSA.md): ❌
- DNSSEC
  - [`AUTODNSSEC`](../language-reference/domain-modifiers/AUTODNSSEC_ON.md): ❌
  - [`DNSKEY`](../language-reference/domain-modifiers/DNSKEY.md): ❔
  - [`DS`](../language-reference/domain-modifiers/DS.md): ❌
<!-- provider-features-end -->
