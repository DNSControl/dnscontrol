---
name: LOC_BUILDER_DD
parameters:
  - label
  - x
  - y
  - alt
  - ttl
parameters_object: true
parameter_types:
  label: string?
  x: number
  y: number
  alt: number?
  ttl: Duration?
---

`LOC_BUILDER_DD({})` actually takes an object with the following properties:

  - label (optional, defaults to `@`)
  - x (float32)
  - y (float32)
  - alt (float32, optional)
  - ttl (optional)

A helper to build [`LOC`](LOC.md) records. Supply four parameters instead of 12.

Internally assumes some defaults for [`LOC`](LOC.md) records.


The cartesian coordinates are decimal degrees, like you typically find in e.g. Google Maps.

Examples.

Big Ben:
`51.50084265331501, -0.12462541415599787`

The White House:
`38.89775977858357, -77.03655125982903`


{% code title="dnsconfig.js" %}
```javascript
D("example.com", REGISTRAR("my_registrar"), SERVICE("my_dns_provider"),
  LOC_BUILDER_DD({
    label: "big-ben",
    x: 51.50084265331501,
    y: -0.12462541415599787,
    alt: 6,
  }),
  LOC_BUILDER_DD({
    label: "white-house",
    x: 38.89775977858357,
    y: -77.03655125982903,
    alt: 19,
  }),
  LOC_BUILDER_DD({
    label: "white-house-ttl",
    x: 38.89775977858357,
    y: -77.03655125982903,
    alt: 19,
    ttl: "5m",
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
