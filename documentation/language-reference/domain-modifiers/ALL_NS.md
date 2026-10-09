---
name: ALL_NS
ts_return: "-1"
---

`ALL_NS` equals `-1` and selects all nameservers from a DNS service. This is also
the default when maxNS is omitted from [SERVICE](SERVICE.md).

Use it to supply configuration metadata without limiting nameservers:

```javascript
D("example.com", REGISTRAR("none"),
    SERVICE("bind", ALL_NS, {default_ns: ["ns1.example.net.", "ns2.example.net."]}),
    A("@", "192.0.2.1"),
);
```

`0` means use no nameservers; a positive integer limits the number used.
