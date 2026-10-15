---
name: DEFAULTS
parameters:
  - modifiers...
parameter_types:
  "modifiers...": DomainModifier[]
---

`DEFAULTS` allows you to declare a set of default arguments to apply to all subsequent domains. Subsequent calls to [`D`](D.md) will have these
arguments passed as if they were the first modifiers in the argument list.

A [REGISTRAR](../domain-modifiers/REGISTRAR.md) default supplies the registrar
when a domain has no explicit selection. An explicit registrar overrides it.
[SERVICE](../domain-modifiers/SERVICE.md) defaults supply DNS services and may
include configuration metadata. Repeating metadata for the same domain and
entry in `D()` or `D_EXTEND()` is an error, even when the values match.

## Example

We want to create backup zone files for all domains, but not actually register them. Also create a [`DefaultTTL`](../domain-modifiers/DefaultTTL.md).
The domain `example.com` will have the defaults set.

{% code title="dnsconfig.js" %}
```javascript
DEFAULTS(
  SERVICE("foo", 0),
  DefaultTTL("1d"),
);

D("example.com", REGISTRAR("my_registrar"), SERVICE("my_dns_provider"),
  A("@","1.2.3.4"),
);
```
{% endcode %}

If you want to clear the defaults, you can do the following.
The domain `example2.com` will **not** have the defaults set.

{% code title="dnsconfig.js" %}
```javascript
DEFAULTS();

D("example2.com", REGISTRAR("my_registrar"), SERVICE("my_dns_provider"),
  A("@","1.2.3.4"),
);
```
{% endcode %}
