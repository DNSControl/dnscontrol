---
name: CF_SINGLE_REDIRECT
parameters:
  - name
  - code
  - when
  - then
  - modifiers...
provider: CLOUDFLAREAPI
parameter_types:
  name: string
  code: number
  when: string
  then: string
  "modifiers...": RecordModifier[]
---

`CF_SINGLE_REDIRECT` is a [Cloudflare](../../provider/cloudflareapi.md)-specific feature for creating HTTP redirects.  301, 302, 303, 307, 308 are supported. Typically one uses 302 (temporary) or 301 (permanent).

This feature manages dynamic "Single Redirects". (Single Redirects can be static or dynamic but DNSControl only maintains dynamic redirects).

DNSControl will delete any "single redirects" it doesn't recognize (i.e. ones created via the web UI) so please be careful.

Cloudflare documentation: <https://developers.cloudflare.com/rules/url-forwarding/single-redirects/>

{% code title="dnsconfig.js" %}
```javascript
D("example.com", REG_MY_PROVIDER, DnsProvider(DSP_MY_PROVIDER),
  CF_SINGLE_REDIRECT('redirect www.example.com', 302, 'http.host eq "www.example.com"', 'concat("https://otherplace.com", http.request.uri.path)'),
  CF_SINGLE_REDIRECT('redirect yyy.example.com', 302, 'http.host eq "yyy.example.com"', 'concat("https://survey.stackoverflow.co", "")'),
  CF_TEMP_REDIRECT("*example.com/*", "https://contests.otherexample.com/$2"),
);
```
{% endcode %}

The fields are:

* name: The name (basically a comment)
* code: Any of 301, 302, 303, 307, 308. May be a number or string.
* when: What Cloudflare sometimes calls the "rule expression".
* then: The replacement expression.

DNSControl does not currently choose the order of the rules.  New rules are added to the end of the list. Use Cloudflare's dashboard to re-order the rule, DNSControl should not change them.  (In the future we hope to add a feature where the order the rules appear in dnsconfig.js is maintained in the dashboard.)

## `CF_REDIRECT` and `CF_TEMP_REDIRECT`

`CF_REDIRECT` and `CF_TEMP_REDIRECT` used to manage Cloudflare Page Rules. However that feature is going away.  To help with the migration, DNSControl now translates those commands into CF_SINGLE_REDIRECT equivalents.  The conversion process is a transpiler that only understands certain formats. Please submit a Github issue if you find something it can't handle.
