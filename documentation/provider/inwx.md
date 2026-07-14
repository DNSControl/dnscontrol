INWX.de is a Berlin-based domain registrar.

## Configuration

To use this provider, add an entry to `creds.json` with `TYPE` set to `INWX`
along with your INWX username and password.

**Example:**

{% code title="creds.json" %}
```json
{
  "inwx": {
    "TYPE": "INWX",
    "password": "yourPassword",
    "username": "yourUsername"
  }
}
```
{% endcode %}

### Two-factor authentication

INWX supports two-factor authentication via TOTP and does not allow TOTP codes to be reused. This means that you will only be able to log into your INWX account once every 30 seconds.

You will hit this limitation in the following two scenarios:

* You run DNSControl twice very quickly (to e.g. first use preview and then push). Waiting for 30 seconds to pass between these two invocations will work fine though.
* You use INWX as both the registrar and the DNS provider. In this case, DNSControl will try to login twice too quickly and the second login will fail because a TOTP code will be reused. The only way to support this configuration is to use a INWX account without two-factor authentication.

If you cannot work around these two limitation it is possible to create and manage sub-account - with specific permission sets - dedicated for API access with two-factor
authentication disabled. This is possible at [inwx.de/en/customer/login](https://www.inwx.de/en/customer/login).

If two-factor authentication has been enabled you will also need to provide a valid TOTP number.
This can also be done via an environment variable:

{% code title="creds.json" %}
```json
{
  "inwx": {
    "TYPE": "INWX",
    "username": "yourUsername",
    "password": "yourPassword",
    "totp": "$INWX_TOTP"
  }
}
```
{% endcode %}

and then you can run

```shell
INWX_TOTP=12345 dnscontrol preview
```

It is also possible to directly provide the shared TOTP secret using the key "totp-key" in `creds.json`.
This secret is only shown once when two-factor authentication is enabled and you'll have to make sure to write it down then.

**Important Notes**:
* Anyone with access to this `creds.json` file will have *full* access to your INWX account and will be able to transfer and/or delete your domains
* Storing the shared secret together with the password weakens two-factor authentication because both factors are stored in a single place.

{% code title="creds.json" %}
```json
{
  "inwx": {
    "TYPE": "INWX",
    "username": "yourUsername",
    "password": "yourPassword",
    "totp-key": "yourTOTPSharedSecret"
  }
}
```
{% endcode %}

### Sandbox
You can optionally also specify sandbox with a value of 1 to redirect all requests to the sandbox API instead:

{% code title="creds.json" %}
```json
{
  "inwx": {
    "TYPE": "INWX",
    "username": "yourUsername",
    "password": "yourPassword",
    "sandbox": "1"
  }
}
```
{% endcode %}

If sandbox is omitted or set to any other value the production API will be used.

## Metadata
This provider does not recognize any special metadata fields unique to INWX.

## Usage
An example `dnsconfig.js` configuration file
for `example.com` registered with INWX
and delegated to Cloudflare:

{% code title="dnsconfig.js" %}
```javascript
var REG_INWX = NewRegistrar("inwx");
var DSP_CF = NewDnsProvider("cloudflare");

D("example.com", REG_INWX, DnsProvider(DSP_CF),
    A("test", "1.2.3.4"),
);
```
{% endcode %}

## Notes

INWX enforces the [RFC 7505](https://www.rfc-editor.org/rfc/rfc7505.html#section-3) MUST NOT guidance regarding publishing both null MX and regular MX records. If a push would result in mixed null MX and regular MX records in the zone, the API responds with `FAILURE! (2308) Data management policy violation` and the record will not be persisted.

## Feature Summary

<!-- provider-features-start -->
- Provider Type
  - [Official Support](../provider/index.md#providers-with-official-support): ❌
  - DNS Provider: ✅
  - Registrar: ✅
- Provider API
  - [Concurrency Verified](../advanced-features/concurrency-verified.md): ✅
  - [dual host](../advanced-features/dual-host.md): ✅
  - create-domains: ✅
  - [get-zones](../commands/get-zones.md): ✅
- DNS extensions
  - [`ALIAS`](../language-reference/domain-modifiers/ALIAS.md): ✅
  - [`DNAME`](../language-reference/domain-modifiers/DNAME.md): ❔
  - [`LOC`](../language-reference/domain-modifiers/LOC.md): ❔
  - [`PTR`](../language-reference/domain-modifiers/PTR.md): ✅
  - [`SOA`](../language-reference/domain-modifiers/SOA.md): ❔
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
  - [`TLSA`](../language-reference/domain-modifiers/TLSA.md): ✅
- DNSSEC
  - [`AUTODNSSEC`](../language-reference/domain-modifiers/AUTODNSSEC_ON.md): ✅
  - [`DNSKEY`](../language-reference/domain-modifiers/DNSKEY.md): ❔
  - [`DS`](../language-reference/domain-modifiers/DS.md): ❔
<!-- provider-features-end -->
