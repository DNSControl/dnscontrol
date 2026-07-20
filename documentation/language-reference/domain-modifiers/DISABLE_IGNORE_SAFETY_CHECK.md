---
name: DISABLE_IGNORE_SAFETY_CHECK
---

`DISABLE_IGNORE_SAFETY_CHECK` disables the safety check. Normally it is an error to insert records that match an `IGNORE` pattern. This disables that safety check for the entire domain.

It replaces the per-record `IGNORE_NAME_DISABLE_SAFETY_CHECK` which is deprecated as of DNSControl v4.0.0.0.

See [`IGNORE`](../domain-modifiers/IGNORE.md) for more information.

## Syntax

```javascript
D("example.com", REG_MY_PROVIDER, DnsProvider(DSP_MY_PROVIDER),
    DISABLE_IGNORE_SAFETY_CHECK,
    ...
    TXT("myhost", "mytext"),
    IGNORE("myhost", "*", "*"),
    ...
```
