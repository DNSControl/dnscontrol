---
name: CF_WORKER_ROUTE
parameters:
  - pattern
  - script
parameter_types:
  pattern: string
  script: string
provider: CLOUDFLAREAPI
---

`CF_WORKER_ROUTE` uses the [Cloudflare Workers](https://developers.cloudflare.com/workers/) API to manage [worker routes](https://developers.cloudflare.com/workers/platform/routes) for a given domain.

If _any_ `CF_WORKER_ROUTE` function is used then `dnscontrol` will manage _all_ Worker Routes for the domain. To be clear: this means it will delete existing routes that were created outside of DNSControl.

{% hint style="warning" %}
**WARNING**: This interface is not extensively tested. Take precautions such as making
backups and manually verifying `dnscontrol preview` output before running
`dnscontrol push`.
{% endhint %}

This example assigns the patterns `api.example.com/*` and `example.com/api/*` to a `my-worker` script:

{% code title="dnsconfig.js" %}
```javascript
D("example.com", REG_MY_PROVIDER, DnsProvider(DSP_MY_PROVIDER),
    CF_WORKER_ROUTE("api.example.com/*", "my-worker"),
    CF_WORKER_ROUTE("example.com/api/*", "my-worker"),
);
```
{% endcode %}
