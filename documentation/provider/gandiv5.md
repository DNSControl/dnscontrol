`GANDI_V5` uses the v5 API and can act as a registrar provider or a DNS provider. It is only able to work with domains migrated to the new LiveDNS API, which should be all domains.

* API Documentation: https://api.gandi.net/docs
* API Endpoint: https://api.gandi.net/
* Sandbox API Documentation: https://api.sandbox.gandi.net/docs/
* Sandbox API Endpoint: https://api.sandbox.gandi.net/

## Configuration

To use this provider, add an entry to `creds.json` with `TYPE` set to `GANDI_V5` along with other settings:

* (mandatory, string) your Gandi.net access credentials (see below) - one of:
  * `token`: Personal Access Token (PAT)
  * `apikey` API Key (deprecated)
* `apiurl`: (optional, string) the endpoint of the API. When empty or absent the production endpoint is used (default) ; you can use it to select the Sandbox API Endpoint instead.
* `sharing_id`: (optional, string) let you scope to a specific organization. When empty or absent calls are not scoped to a specific organization.

When both `token` and `apikey` are defined, the priority is given to `token` which will be used for API communication (as if `apikey` was not set). See [the Authentication section](#authentication) for details on obtaining these credentials.

The [sharing_id](https://api.gandi.net/docs/reference/#Sharing-ID) selects between different organizations which your account is a member of; to manage domains in multiple organizations, you can use multiple `creds.json` entries.

How to find the `sharing_id`: The sharing ID is the second hex string found in the URL on the portal. Log into the Gandi website, click on "organizations" in the leftnav, and click on the organization name.  The URL will be something like:

```text
https://admin.gandi.net/organizations/[not this hex string]/PLTS/[sharing id]/profile
```

Example:

{% code title="creds.json" %}
```json
{
  "gandi": {
    "TYPE": "GANDI_V5",
    "token": "your-gandi-personal-access-token",
    "sharing_id": "your-sharing_id"
  }
}
```
{% endcode %}

## Authentication

(Cf [official documentation of the API](https://api.gandi.net/docs/authentication/) The **Personal Access Token** (PAT) is configured in the [Account Settings of the Gandi Admin application](https://admin.gandi.net/organizations/account/pat), then click on "Create a token" button. Choose an organisation (if your account happens to have multiple ones). Then, choose a name (limited to 42 chars), an expiration date. You can choose to limit the scope to a select number of products (domain names). Finally, choose the permissions : the needed one is "Manage domain name technical configurations" (in French: "Gérer la configuration technique des domaines"), which automatically implies "See and renew domain names" (in French: "Voir et renouveler les domaines"). You then have only one (1) chance to copy and save the token somewhere.

The **API Key** is the previous (deprecated) mechanism used to do api calls. To generate or delete your API key, go to User Settings, "Manage the user account and security settings", the "Authentication options" tab, then regenerate the "Production API key" under "Developer access"

## Metadata
This provider does not recognize any special metadata fields unique to Gandi.

## Limitations
This provider does not support using `ALIAS` in combination with DNSSEC, whether `AUTODNSSEC` or otherwise.

This provider only supports `ALIAS` on the `"@"` zone apex, not on any other names.

Support for NAPTR records is limited. We have been unable to create test data that Gandi doesn't reject. If you require NAPTR support, please open an issue and include NAPTR records that you were able to create using the Gandi web portal. Your help will be greatly appreciated!

## Usage
An example configuration:

{% code title="dnsconfig.js" %}
```javascript
var REG_GANDI = NewRegistrar("gandi");
var DSP_GANDI = NewDnsProvider("gandi");

D("example.com", REG_GANDI, DnsProvider(DSP_GANDI),
    A("test", "1.2.3.4"),
);
```
{% endcode %}

If you are converting from the old "GANDI" provider, simply change "GANDI" to "GANDI_V5" in `dnsconfig.js`. Be sure to test with `dnscontrol preview` before running `dnscontrol push`.

## New domains
If a domain does not exist in your Gandi account, DNSControl will *not* automatically add it with the `create-domains` command. You'll need to do that via the web UI manually.

## Common errors

### Error getting corrections

```text
Error getting corrections: 401: The server could not verify that you authorized to access the document you requested. Either you supplied the wrong credentials (e.g., bad api key), or your access token has expired
```

This is the error you'll see if your `token` (or (deprecated) `apikey`) in `creds.json` is wrong or invalid.

### Domain does not exist in profile

```text
WARNING: Domain 'example.com' does not exist in the 'secname' profile and will be added automatically.
```

This error is caused by the internal `ListZones()` functions returning no domain names.  This is usually because your `creds.json` information is pointing at an empty organization or no organization.  The solution is to set `sharing_id` in `creds.json`.

### get-zones "nameonly" returns nothing

If a `dnscontrol get-zones --format=nameonly CredId - all` returns nothing, this is usually because your `creds.json`  information is pointing at an empty organization or no organization.  The solution is to set `sharing_id` in `creds.json`.

## Development

### Debugging
Set `GANDI_V5_DEBUG` environment variable to a [boolean-compatible](https://pkg.go.dev/strconv#ParseBool) value to dump all API calls made by this provider.

### Testing
Set `apiurl` key to the endpoint url for the sandbox (https://api.sandbox.gandi.net/), along with corresponding `token` (or (deprecated) `apikey`) created in this sandbox environment (Cf https://api.sandbox.gandi.net/docs/sandbox/) to make all API calls against Gandi sandbox environment.

## Feature Summary

<!-- provider-features-start -->
- Provider Type
  - [Official Support](../provider/index.md#providers-with-official-support): ❌
  - DNS Provider: ✅
  - Registrar: ✅
- Provider API
  - [Concurrency Verified](../advanced-features/concurrency-verified.md): ✅
  - [dual host](../advanced-features/dual-host.md): ❔
  - create-domains: ❌
  - [get-zones](../commands/get-zones.md): ✅
- DNS extensions
  - [`ALIAS`](../language-reference/domain-modifiers/ALIAS.md): ✅
  - [`DNAME`](../language-reference/domain-modifiers/DNAME.md): ❔
  - [`LOC`](../language-reference/domain-modifiers/LOC.md): ✅
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
  - [`AUTODNSSEC`](../language-reference/domain-modifiers/AUTODNSSEC_ON.md): ❔
  - [`DNSKEY`](../language-reference/domain-modifiers/DNSKEY.md): ❔
  - [`DS`](../language-reference/domain-modifiers/DS.md): ❌
<!-- provider-features-end -->
