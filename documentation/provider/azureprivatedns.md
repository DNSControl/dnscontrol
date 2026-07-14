## Configuration

This provider is for the [Azure Private DNS Service](https://learn.microsoft.com/en-us/azure/dns/private-dns-overview).  This provider can only manage Azure Private DNS zones and will not manage public Azure DNS zones. To use this provider, add an entry to `creds.json` with `TYPE` set to `AZURE_PRIVATE_DNS`
along with the API credentials.

Example:

{% code title="creds.json" %}
```json
{
  "azure_private_dns_main": {
    "TYPE": "AZURE_PRIVATE_DNS",
    "SubscriptionID": "AZURE_PRIVATE_SUBSCRIPTION_ID",
    "ResourceGroup": "AZURE_PRIVATE_RESOURCE_GROUP",
    "TenantID": "AZURE_PRIVATE_TENANT_ID",
    "ClientID": "AZURE_PRIVATE_CLIENT_ID",
    "ClientSecret": "AZURE_PRIVATE_CLIENT_SECRET"
  }
}
```
{% endcode %}

You can also use environment variables:

```shell
export AZURE_SUBSCRIPTION_ID=XXXXXXXXX
export AZURE_RESOURCE_GROUP=YYYYYYYYY
export AZURE_TENANT_ID=ZZZZZZZZ
export AZURE_CLIENT_ID=AAAAAAAAA
export AZURE_CLIENT_SECRET=BBBBBBBBB
```

{% code title="creds.json" %}
```json
{
  "azure_private_dns_main": {
    "TYPE": "AZURE_PRIVATE_DNS",
    "SubscriptionID": "$AZURE_PRIVATE_SUBSCRIPTION_ID",
    "ResourceGroup": "$AZURE_PRIVATE_RESOURCE_GROUP",
    "ClientID": "$AZURE_PRIVATE_CLIENT_ID",
    "TenantID": "$AZURE_PRIVATE_TENANT_ID",
    "ClientSecret": "$AZURE_PRIVATE_CLIENT_SECRET"
  }
}
```
{% endcode %}

## Metadata
This provider does not recognize any special metadata fields unique to Azure Private DNS.

## Usage
An example configuration:

{% code title="dnsconfig.js" %}
```javascript
var REG_NONE = NewRegistrar("none");
var DSP_AZURE_PRIVATE_MAIN = NewDnsProvider("azure_private_dns_main");

D("example.com", REG_NONE, DnsProvider(DSP_AZURE_PRIVATE_MAIN),
    A("test", "1.2.3.4"),
);
```
{% endcode %}

## Test credentials

If you want to create credentials without learning all about Entra ID (formerly AAD).  Here's what I did.  You will create an named API key (in this case, called `dns-api-test`) and give it access to the specific zones it should access. This is probably best for testing. For production use, you should understand Entra ID and set up proper access.

1. Get a shell

* Start the Azure Portal: https://portal.azure.com
* Click the `>_` Cloud Shell button at the top.
* Choose Bash.

2. Create an API key called `dns-api-test`

NOTE: You can use the same credential data for `AZURE_DNS` and `AZURE_PRIVATE_DNS` but `creds.json` must have a separate entry for each. All the fields except `TYPE` will be the same.

```
az ad sp create-for-rbac --name dns-api-test

{
  "appId": "74d472fe-d9e7-4b5f-a7df-76fefb146394",
  "displayName": "dns-api-test",
  "password": "REDACTED_SECRET",
  "tenant": "3b08b773-7594-4677-a7f0-44d0afac51b4"
}
```

3. Show the zone's acess path:

Use your own Resource Group in `--resource-group` and change the `--name` to the DNS zone name you created through the portal.

```
az network private-dns zone show \
  --resource-group DNSControlTest  \
  --name dnscontroltest-azurep.com \
  --query id \
  -o tsv
/subscriptions/02efc9e4-732d-4a8d-8b82-5b37e10eb89d/resourceGroups/dnscontroltest/providers/Microsoft.Network/privateDnsZones/dnscontroltest-azurep.com
```

The `/subscriptions/02efc9e4....` output is the path to this zone.

4. Give your API key access to that zone

```
az role assignment create \
  --assignee 74d472fe-d9e7-4b5f-a7df-76fefb146394 \
  --role "DNS Zone Contributor" \
  --scope /subscriptions/02efc9e4-732d-4a8d-8b82-5b37e10eb89d/resourceGroups/dnscontroltest/providers/Microsoft.Network/privateDnsZones/dnscontroltest-azurep.com
```

For AZURE_DNS the commands are slightly different.

## Activation
DNSControl depends on a standard [Client credentials Authentication](https://docs.microsoft.com/en-us/cli/azure/create-an-azure-service-principal-azure-cli?view=azure-cli-latest) with permission to list, create and update private zones.

## New domains

If a domain does not exist in your Azure account, DNSControl will *not* automatically add it with the `push` command. You can do that manually via the control panel.

## Caveats

The ResourceGroup is case sensitive.

## Feature Flags

<!-- provider-features-start -->
- Provider Type
  - [Official Support](../provider/index.md#providers-with-official-support): ✅
  - DNS Provider: ✅
  - Registrar: ❌
- Provider API
  - [Concurrency Verified](../advanced-features/concurrency-verified.md): ❔
  - [dual host](../advanced-features/dual-host.md): ❌
  - create-domains: ✅
  - [get-zones](../commands/get-zones.md): ✅
- DNS extensions
  - [`ALIAS`](../language-reference/domain-modifiers/ALIAS.md): ❌
  - [`DNAME`](../language-reference/domain-modifiers/DNAME.md): ❔
  - [`LOC`](../language-reference/domain-modifiers/LOC.md): ❌
  - [`PTR`](../language-reference/domain-modifiers/PTR.md): ✅
  - [`SOA`](../language-reference/domain-modifiers/SOA.md): ❔
- Service discovery
  - [`DHCID`](../language-reference/domain-modifiers/DHCID.md): ❔
  - [`NAPTR`](../language-reference/domain-modifiers/NAPTR.md): ❌
  - [`SRV`](../language-reference/domain-modifiers/SRV.md): ✅
  - [`SVCB`](../language-reference/domain-modifiers/SVCB.md): ❔
- Security
  - [`CAA`](../language-reference/domain-modifiers/CAA.md): ❌
  - [`HTTPS`](../language-reference/domain-modifiers/HTTPS.md): ❔
  - [`SMIMEA`](../language-reference/domain-modifiers/SMIMEA.md): ❔
  - [`SSHFP`](../language-reference/domain-modifiers/SSHFP.md): ❌
  - [`TLSA`](../language-reference/domain-modifiers/TLSA.md): ❌
- DNSSEC
  - [`AUTODNSSEC`](../language-reference/domain-modifiers/AUTODNSSEC_ON.md): ❔
  - [`DNSKEY`](../language-reference/domain-modifiers/DNSKEY.md): ❔
  - [`DS`](../language-reference/domain-modifiers/DS.md): ❔
<!-- provider-features-end -->
