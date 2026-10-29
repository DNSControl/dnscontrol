---
name: ALL_NS
ts_return: "-1"
---

`ALL_NS` selects all nameservers from a DNS service. This is also
the default when maxNS is omitted from [SERVICE](SERVICE.md).

Use it to supply configuration metadata without limiting nameservers:

```javascript
D("example.com", REGISTRAR(),
    SERVICE("bind", ALL_NS, {default_ns: ["ns1.example.net.", "ns2.example.net."]}),
    A("@", "192.0.2.1"),
);
```

In addition to `ALL_NS`, `0` means use no nameservers; a positive integer limits the number used.

FunFact: `ALL_NS` is an alias for `-1` but please don't depend on this. Specifying `-1` to mean "all" may change some day.
