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

* name: The name used to match this rule between deployments. Use a unique, stable name for each rule in a zone.
* code: Any of 301, 302, 303, 307, 308. May be a number or string.
* when: What Cloudflare sometimes calls the "rule expression".
* then: The replacement expression.

DNSControl does not currently choose the order of the rules. New rules are added to the end of the list. Use Cloudflare's dashboard to reorder them. Editing the status code, condition, or destination of a rule with the same unique name preserves its Cloudflare rule ID and current position. Its enabled state is also preserved. A condition or status edit preserves the existing query-string behavior; changing the destination derives that behavior from the new destination, as for a new rule.

Changing a rule's name is a replacement: the old rule is deleted and the new rule is appended. Names are case-sensitive. Rules with duplicate names cannot be reliably matched during edits; use unique names when position matters.

Preserving positions does not restore an already misordered ruleset, enforce declaration order, or detect order-only drift. When conditions overlap, put specific rules before broader fallbacks in Cloudflare's dashboard. In the future we hope to support declarative ordering as a separate feature.

Single Redirects are HTTP rules and have no DNS TTL. DNSControl normalizes their internal TTL to 1 when planning Cloudflare changes, including when a default TTL or an explicit `TTL()` modifier is present. This does not control browser redirect caching.

## `CF_REDIRECT` and `CF_TEMP_REDIRECT`

`CF_REDIRECT` and `CF_TEMP_REDIRECT` used to manage Cloudflare Page Rules. However that feature is going away.  To help with the migration, DNSControl now translates those commands into CF_SINGLE_REDIRECT equivalents.  The conversion process is a transpiler that only understands certain formats. Please submit a Github issue if you find something it can't handle.

These generated rules share the same ordered list as explicit `CF_SINGLE_REDIRECT` rules. Their generated names include the status code, source pattern, and destination. Changing any of those produces a replacement that is appended. To edit a redirect while retaining its position, use an explicit `CF_SINGLE_REDIRECT` with a stable name.
