---
name: REVCOMPAT
parameters:
  - rfc
parameter_types:
  rfc: string
ts_return: string
---

`REVCOMPAT()` controls which RFC the [`REV()`](REV.md) function adheres to.

Include one of these two commands near the top `dnsconfig.js` (at the global level):

{% code title="dnsconfig.js" %}
```javascript
REVCOMPAT("rfc2317");  // RFC 2117: Compatible with old files.
REVCOMPAT("rfc4183");  // RFC 4183: Adopt the newer standard.
```
{% endcode %}

`REVCOMPAT()` is global for all of `dnsconfig.js`. It must appear before any use of `REV()`; If not, behavior is undefined.

# RFC 4183 vs RFC 2317

RFC 2317 and RFC 4183 are two different ways to implement reverse lookups for CIDR blocks that are not on 8-bit boundaries (/24, /16, /8).

Originally DNSControl implemented the older standard, which only specifies what to do for /8, /16, /24 - /32.  Using `REV()` for /9-17 and /17-23 CIDRs was an error.

v4 defaults to RFC 2317.  In v5.0 the default will change to RFC 4183. `REVCOMPAT()` is provided for those that wish to retain the old behavior.

For more information, see [Opinion #9](../../advanced-features/opinions.md#opinion-9-rfc-4183-is-better-than-rfc-2317).

# Transition plan

What's the default behavior if `REVCOMPAT()` is not used?

| Version | /9 to /15 and /17 to /23 | /25 to 32 | Warnings                   |
|---------|--------------------------|-----------|----------------------------|
| v4      | RFC 4183                 | RFC 2317  | Only if /25 - /32 are used |
| v5      | RFC 4183                 | RFC 4183  | none                       |

No warnings are generated if the `REVCOMPAT()` function is used.
