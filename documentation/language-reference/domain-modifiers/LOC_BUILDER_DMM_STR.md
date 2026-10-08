---
name: LOC_BUILDER_DMM_STR
parameters:
  - label
  - str
  - alt
  - ttl
parameters_object: true
parameter_types:
  label: string?
  str: string
  alt: number?
  ttl: Duration?
---

`LOC_BUILDER_DMM({})` actually takes an object with the following properties:

  - label (string, optional, defaults to `@`)
  - str (string)
  - alt (float32, optional)
  - ttl (optional)

A helper to build [`LOC`](LOC.md) records. Supply three parameters instead of 12.

Internally assumes some defaults for [`LOC`](LOC.md) records.


Accepts a string with decimal minutes (DMM) coordinates in the form: 25.24°S 153.15°E

Note that the following are acceptable forms (symbols differ):
* `25.24°S 153.15°E`
* `25.24 S 153.15 E`
* `25.24° S 153.15° E`
* `25.24S 153.15E`

{% code title="dnsconfig.js" %}
```javascript
D("example.com", REGISTRAR("my_registrar"), SERVICE("my_dns_provider"),
  LOC_BUILDER_STR({
    label: "tasmania",
    str: "42°S 147°E",
    alt: 3,
  }),
);
```
{% endcode %}

Part of the series:
 * [`LOC()`](LOC.md) - build a `LOC` by supplying all 12 parameters
 * [`LOC_BUILDER_DD({})`](LOC_BUILDER_DD.md) - accepts cartesian x, y
 * [`LOC_BUILDER_DMS_STR({})`](LOC_BUILDER_DMS_STR.md) - accepts DMS 33°51′31″S 151°12′51″E
 * [`LOC_BUILDER_DMM_STR({})`](LOC_BUILDER_DMM_STR.md) - accepts DMM 25.24°S 153.15°E
 * [`LOC_BUILDER_STR({})`](LOC_BUILDER_STR.md) - tries the coordinate string in all `LOC_BUILDER_DM*_STR()` functions until one works
