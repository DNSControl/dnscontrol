---
name: LOC
parameters:
  - deg1
  - min1
  - sec1
  - deg2
  - min2
  - sec2
  - altitude
  - size
  - horizontal_precision
  - vertical_precision
parameter_types:
  name: string
  target: string
  deg1: number
  min1: number
  sec1: number
  deg2: number
  min2: number
  sec2: number
  altitude: number
  size: number
  horizontal_precision: number
  vertical_precision: number
---

`LOC` add a [Location record](https://www.rfc-editor.org/rfc/rfc1876) to the domain.

The parameter number types ingested are as follows:

```
name: string
target: string
deg1: uint32
min1: uint32
sec1: float32
deg2: uint32
min2: uint32
sec2: float32
altitude: float32
size: float32
horizontal_precision: float32
vertical_precision: float32
```

## Description ##

Strictly follows [RFC 1876](https://datatracker.ietf.org/doc/html/rfc1876).

A LOC record holds a geographical position. In the zone file, it may look like:

```text
;
pipex.net.                    LOC   52 14 05 N 00 08 50 E 10m
```

On the wire, it is in a binary format.

A use case for LOC is suggested in the RFC:

> Some uses for the LOC RR have already been suggested, including the
   USENET backbone flow maps, a "visual traceroute" application showing the geographical path of an IP packet, and network management applications that could use LOC RRs to generate a map of hosts and routers being managed.

There is the UK based [https://find.me.uk](https://find.me.uk/) whereby you can do:

```sh
dig loc <uk-postcode>.find.me.uk
```

There are some behaviours that you should be aware of, however:

> If omitted, minutes and seconds default to zero, size defaults to 1m,
   horizontal precision defaults to 10000m, and vertical precision defaults to 10m.  These defaults are chosen to represent typical ZIP/postal code area sizes, since it is often easy to find approximate geographical location by ZIP/postal code.

Alas, the world does not revolve around US ZIP codes, but here we are. Internally, the LOC record type will supply defaults where values were absent on DNS import. One must supply the `LOC()` js helper all parameters. If that seems like too much work, see also helper functions:

 * [`LOC_BUILDER_DD({})`](LOC_BUILDER_DD.md) - build a `LOC` by supplying only **d**ecimal **d**egrees.
 * [`LOC_BUILDER_DMS_STR({})`](LOC_BUILDER_DMS_STR.md) - accepts DMS 33°51′31″S 151°12′51″E
 * [`LOC_BUILDER_DMM_STR({})`](LOC_BUILDER_DMM_STR.md) - accepts DMM 25.24°S 153.15°E
 * [`LOC_BUILDER_STR({})`](LOC_BUILDER_STR.md) - tries the coordinate string in all `LOC_BUILDER_DM*_STR()` functions until one works

## Format ##

The coordinate format for `LOC()` is:

`degrees,minutes,seconds,[NnSs],deg,min,sec,[EeWw],altitude,size,horizontal_precision,vertical_precision`

where: altitude: [-100000.00 .. 42849672.95] BY .01 (altitude in meters) size, horizontal_precision, vertical_precision: [0 .. 90000000.00] (size/precision in meters)

values outside of the above ranges are gated to within the ranges.

## Examples ##

{% code title="dnsconfig.js" %}
```javascript
D("example.com", REG_MY_PROVIDER, DnsProvider(DSP_MY_PROVIDER),
  // LOC "subdomain", d1, m1, s1, "[NnSs]", d2, m2, s2, "[EeWw]", alt, siz, hp, vp)
  //42 21 54     N  71 06  18     W -24m 30m
  LOC("@", 42, 21, 54,     "N", 71,  6, 18,     "W", -24.01,   30,    0,  0),
  //42 21 43.952 N  71 5   6.344  W -24m 1m 200m 10m
  LOC("a", 42, 21, 43.952, "N", 71,  5,  6.344, "W", -24.33,    1,  200, 10),
  //52 14 05     N  00 08  50     E 10m
  LOC("b", 52, 14,  5,     "N",  0,  8, 50,     "E",  10,    0,    0,  0),
  //32  7 19     S 116  2  25     E 10m
  LOC("c", 32,  7, 19,     "S",116,  2, 25,     "E",  10,    0,    0,  0),
  //42 21 28.764 N  71 00  51.617 W -44m 2000m
  LOC("d", 42, 21, 28.764, "N", 71,  0, 51.617, "W", -44, 2000,    0,  0),
);
```
{% endcode %}
