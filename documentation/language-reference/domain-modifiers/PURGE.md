---
name: PURGE
---

`PURGE` is the default setting for all domains.  Therefore `PURGE` is a no-op. It is included for completeness only.

A domain with a mixture of `NO_PURGE` and `PURGE` parameters will abide by the last one.

These three examples all are equivalent.

`PURGE` is the default:

{% code title="dnsconfig.js" %}
```javascript
D("example.com", REGISTRAR("my_registrar"), SERVICE("my_dns_provider"),
);
```
{% endcode %}

Purge is the default, but we set it anyway:

{% code title="dnsconfig.js" %}
```javascript
D("example.com", REGISTRAR("my_registrar"), SERVICE("my_dns_provider"),
  PURGE,
);
```
{% endcode %}

Since the "last command wins", this is the same as `PURGE`:

{% code title="dnsconfig.js" %}
```javascript
D("example.com", REGISTRAR("my_registrar"), SERVICE("my_dns_provider"),
  PURGE,
  NO_PURGE,
  PURGE,
  NO_PURGE,
  PURGE,
);
```
{% endcode %}
