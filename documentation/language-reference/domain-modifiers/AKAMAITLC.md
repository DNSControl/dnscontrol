---
name: AKAMAITLC
parameters:
  - name
  - answer_type
  - target
  - modifiers...
provider: AKAMAIEDGEDNS
parameter_types:
  name: string
  answer_type: '"DUAL" | "A" | "AAAA"'
  target: string
  "modifiers...": RecordModifier[]
---

`AKAMAITLC` is a proprietary Top-Level CNAME (TLC) record type specific to Akamai Edge DNS.
It allows CNAME-like functionality at the zone apex (`@`) of a domain where regular CNAME records
are not permitted.

The difference between `AKAMAITLC` and `CNAME` is that `AKAMAITLC` records are resolved by Akamai Edge DNS
servers instead of the client's resolver. This is similar to how `AKAMAICDN` records work, except that `AKAMAITLC`
records can be pointed to any domain, not just Akamai properties. If you are pointing to an Akamai property,
you should use `AKAMAICDN` instead.

Important restrictions:
- Can only be used at the zone apex (`@`)
- Limited to one `AKAMAITLC` record per zone
- Cannot coexist with an `AKAMAICDN` record at the apex

The `answer_type` parameter controls which record types are returned when clients resolve the target:
- `DUAL`: Returns both IPv4 (`A`) and IPv6 (`AAAA`) records
- `A`: Returns only IPv4 records
- `AAAA`: Returns only IPv6 records

## Example
{% code title="dnsconfig.js" %}
```javascript
D("example.com", REGISTRAR("my_registrar"), SERVICE("my_dns_provider"),
    // Redirect example.com to google.com, returning both A and AAAA records
    AKAMAITLC("@", "DUAL", "google.com."),
);
```
{% endcode %}
