---
name: CSYNC
parameters:
  - name
  - serial
  - flags
  - types
  - modifiers...
parameter_types:
  name: string
  serial: number
  flags: number
  types: string
  "modifiers...": RecordModifier[]
---

`CSYNC` adds a [Child-to-Parent Synchronization record](https://www.rfc-editor.org/rfc/rfc7477) to a domain. The name should be the relative label for the record. A CSYNC record is normally published at the apex (`@`) of a child zone. It tells the parent's operator which records should be copied from the child zone into the parent zone.

Serial is the SOA serial number of the child zone that the CSYNC record applies to.

Flags is a bit field:

- `1` (immediate): the parent may process the record immediately, without first checking the SOA serial.
- `2` (soaminimum): the parent should only process the record if the child's SOA serial is greater than or equal to the CSYNC serial.
- `3` is 1 + 2.

Types is a space-separated list of the record types to synchronize, such as `"A NS AAAA"`.

{% code title="dnsconfig.js" %}
```javascript
D("example.com", REGISTRAR(), SERVICE('myservice'),
  CSYNC("@", 2026101100, 2, "A NS AAAA"),
);
```
{% endcode %}
