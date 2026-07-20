---
name: CF_TEMP_REDIRECT
parameters:
  - source
  - destination
  - modifiers...
provider: CLOUDFLAREAPI
parameter_types:
  source: string
  destination: string
  "modifiers...": RecordModifier[]
---

{% hint style="warning" %}
**WARNING:** Cloudflare is removing this feature and replacing it with a new
feature called "Dynamic Single Redirect". DNSControl will automatically
generate "Dynamic Single Redirects" for a limited number of use cases. See
[`CLOUDFLAREAPI`](../../provider/cloudflareapi.md) for details.
{% endhint %}

`CF_TEMP_REDIRECT` uses [Cloudflare](../../provider/cloudflareapi.md)-specific features ("Forwarding URL" Page Rules) to generate a HTTP 302 temporary redirect.

If _any_ [`CF_REDIRECT`](CF_REDIRECT.md) or `CF_TEMP_REDIRECT functions are used then `dnscontrol` will manage _all_ "Forwarding URL" type Page Rules for the domain. Page Rule types other than "Forwarding URL" will be left alone.

{% hint style="warning" %}
**WARNING**: Cloudflare does not currently fully document the Page Rules API and
this interface is not extensively tested. Take precautions such as making
backups and manually verifying `dnscontrol preview` output before running
`dnscontrol push`. This is especially true when mixing Page Rules that are
managed by DNSControl and those that aren't.
{% endhint %}

This example redirects the bare (aka apex, or naked) domain to www:

{% code title="dnsconfig.js" %}
```javascript
D("example.com", REG_MY_PROVIDER, DnsProvider(DSP_MY_PROVIDER),
  CF_TEMP_REDIRECT("example.com/*", "https://www.example.com/$1"),

);
```
{% endcode %}
