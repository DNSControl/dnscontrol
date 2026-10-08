---
name: FETCH
parameters:
  - url
  - args
ts_ignore: true
# Make sure to update fetch.d.ts if changing the docs below!
---

`FETCH` is a wrapper for the [Fetch API](https://developer.mozilla.org/en-US/docs/Web/API/Fetch_API). This allows dynamically setting DNS records based on an external data source, e.g. the API of your cloud provider.

Compared to `fetch` from Fetch API, `FETCH` will call [PANIC](PANIC.md) to terminate the execution of the script, and therefore DNSControl, if a network error occurs.

Otherwise the syntax of `FETCH` is the same as `fetch`.

`FETCH` is not enabled by default. Please read the warnings below.

> WARNING:
>
> 1. Relying on external sources adds a point of failure. If the external source doesn't work, your script won't either. Please make sure you are aware of the consequences.
> 2. Make sure DNSControl only uses verified configuration if you want to use `FETCH`. For example, an attacker can send Pull Requests to your config repo, and have your CI test malicious configurations and make arbitrary HTTP requests. Therefore, `FETCH` must be explicitly enabled with flag `--allow-fetch` on DNSControl invocation.

{% code title="dnsconfig.js" %}
```javascript
D("example.com", REG_MY_PROVIDER, DnsProvider(DSP_MY_PROVIDER),
  A("@", "1.2.3.4"),
);

FETCH("https://example.com", {
  // All three options below are optional
  headers: {"X-Authentication": "barfoo"},
  method: "POST",
  body: "Hello World",
}).then(function(r) {
  return r.text();
}).then(function(t) {
  // Example of generating record based on response
  D_EXTEND("example.com", [
    TXT("@", t.slice(0, 100)),
  ]);
});
```
{% endcode %}
