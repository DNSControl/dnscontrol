## Configuration

To use this provider, add an entry to `creds.json` with `TYPE` set to `AUTODNS` along with
[username, password and a context](https://help.internetx.com/display/APIXMLEN/Authentication#Authentication-AuthenticationviaCredentials(username/password/context)).

Example:

{% code title="creds.json" %}
```json
{
  "autodns": {
    "TYPE": "AUTODNS",
    "username": "autodns.service-account@example.com",
    "password": "[***]",
    "context": "33004"
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
  "autodns": {
    "TYPE": "AUTODNS",
    "username": "autodns.service-account@example.com",
    "password": "[***]",
    "context": "33004",
    "totp": "$AUTODNS_TOTP"
  }
}
```
{% endcode %}

and then you can run

```shell
AUTODNS_TOTP=123456 dnscontrol preview
```

It is also possible to directly provide the shared TOTP secret using the key "totp-key" in `creds.json`. This secret is
only available when first enabling two-factor authentication.

**Security Warning**:
* Anyone with access to this `creds.json` file will have *full* access to your AutoDNS account
  and will be able to modify and delete your DNS entries.
* Storing the shared secret together with the password weakens two factor authentication
  because both factors are stored in a single place.

{% code title="creds.json" %}
```json
{
  "autodns": {
    "TYPE": "AUTODNS",
    "username": "autodns.service-account@example.com",
    "password": "[***]",
    "context": "33004",
    "totp-key": "yourTOTPSharedSecret"
  }
}
```
{% endcode %}

`totp` and `totp-key` must not be set at the same time. Omit both if the account does not use
two factor authentication.

### Including sub-user zones

By default DNSControl only sees zones owned directly by the configured user. If
you authenticate as a master/admin user and want `get-zones` (and the `all`
keyword) to also include zones owned by sub-users — the same optional "include
subusers" toggle the AutoDNS web UI offers — set `children` to `"true"`:

{% code title="creds.json" %}
```json
{
  "autodns": {
    "TYPE": "AUTODNS",
    "username": "autodns.service-account@example.com",
    "password": "[***]",
    "context": "33004",
    "children": "true"
  }
}
```
{% endcode %}

## Usage

An example configuration:

{% code title="dnsconfig.js" %}
```javascript
var REG_NONE = NewRegistrar("none");
var DSP_AUTODNS = NewDnsProvider("autodns");

D("example.com", REG_NONE, DnsProvider(DSP_AUTODNS),
    A("test", "1.2.3.4"),
);
```
{% endcode %}

## Feature Summary

<!-- provider-features-start -->
- Provider Type
  - [Official Support](../provider/index.md#providers-with-official-support): ❌
  - DNS Provider: ✅
  - Registrar: ✅
- Provider API
  - [Concurrency Verified](../advanced-features/concurrency-verified.md): ✅
  - [dual host](../advanced-features/dual-host.md): ❌
  - create-domains: ❌
  - [get-zones](../commands/get-zones.md): ✅
- DNS extensions
  - [`ALIAS`](../language-reference/domain-modifiers/ALIAS.md): ✅
  - [`DNAME`](../language-reference/domain-modifiers/DNAME.md): ❔
  - [`LOC`](../language-reference/domain-modifiers/LOC.md): ❔
  - [`PTR`](../language-reference/domain-modifiers/PTR.md): ✅
  - [`SOA`](../language-reference/domain-modifiers/SOA.md): ❔
- Service discovery
  - [`DHCID`](../language-reference/domain-modifiers/DHCID.md): ❔
  - [`NAPTR`](../language-reference/domain-modifiers/NAPTR.md): ❔
  - [`SRV`](../language-reference/domain-modifiers/SRV.md): ✅
  - [`SVCB`](../language-reference/domain-modifiers/SVCB.md): ❔
- Security
  - [`CAA`](../language-reference/domain-modifiers/CAA.md): ✅
  - [`HTTPS`](../language-reference/domain-modifiers/HTTPS.md): ❔
  - [`SMIMEA`](../language-reference/domain-modifiers/SMIMEA.md): ❔
  - [`SSHFP`](../language-reference/domain-modifiers/SSHFP.md): ❌
  - [`TLSA`](../language-reference/domain-modifiers/TLSA.md): ❌
- DNSSEC
  - [`AUTODNSSEC`](../language-reference/domain-modifiers/AUTODNSSEC_ON.md): ❔
  - [`DNSKEY`](../language-reference/domain-modifiers/DNSKEY.md): ❔
  - [`DS`](../language-reference/domain-modifiers/DS.md): ❌
<!-- provider-features-end -->
