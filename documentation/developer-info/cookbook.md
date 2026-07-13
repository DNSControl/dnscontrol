# Cookbook

This document provides "cookbook" recipes for doing common tasks.

- [Cookbook](#cookbook)
  - [Create a `models.DomainConfig`](#create-a-modelsdomainconfig)
  - [Create a `models.RecordConfig`](#create-a-modelsrecordconfig)
  - [Getters/Setters for RDATA in `models.RecordConfig`](#getterssetters-for-rdata-in-modelsrecordconfig)
  - [Create `models.RecordConfig` literals for testdata](#create-modelsrecordconfig-literals-for-testdata)
  - [How to create a "builder"](#how-to-create-a-builder)
  - [How to manipulate domain/zone names](#how-to-manipulate-domainzone-names)
  - [How to label imports](#how-to-label-imports)

## Create a `models.DomainConfig`

- What: Create a `models.DomainConfig`
- Why: Providers are handed a `models.DomainConfig` that is already created. However often when we write tests we need to create a list of `models.RecordConfig`'s which are stored in a `models.DomainConfig`.

Recommended:

```go
dc, err := models.NewDomainConfig(zone)
dc.AddRecordConfig(models.MakeTestRC(label, ttl, type, args))
dc.AddRecordConfig(models.MakeTestRCParse(label, ttl, type, args))
fmt.Printf("Count: %d", len(dc.Records))
```

Deprecated:

```go
dc := &models.DomainConfig{Name: origin}
```

## Create a `models.RecordConfig`

- What: Create a `models.RecordConfig`
- Why: One of the most important functions of a provider is to translate the native DNS records (as received from the API) to standard `models.RecordConfig` structs. Previously there were many ways to do this. Starting in DNSControl v5, we've standaredized on two factories:

Recommended:

```go
rc1, err := dc.NewRecordConfig(LABEL, TTL, TYPE_STR_OR_NUM, ARGS)
rc2, err := dc.NewRecordConfigParse(LABEL, TTL, TYPE_STR_OR_NUM, RFC1038_STRING)
```

- These are a method of `models.DomainConfig` (typically the variable name is `dc`). Records only exist in the context of a domain. For example, if an MX record's target it "@", this is meaningless unless we know what the domain is. This also makes the call shorter: it eliminates parameters that contain info already stored in `dc` such as the zone name (required for normalizing labels).
- `NewRecordConfig()` takes a list of arguments. It doesn't matter if the arguments are strings, ints, `netip.Addrs`... the function will convert them to the correct type and return and error if they can't be converted.
- `NewRecordConfigParse()` takes the arguments as one long string, which is parsed. If your provider returns (for example) the MX record data as `10 mx.example.com.` and the SRV record data as `4 100 123 three.example.com.`, you can just send the whole string to this function. This replaces `models.PopulateFromString()`

- `LABEL`: Must be the output of one of these functions:
  - `models.LabelFromShort()`: Use this if your provider always gives you the shortname (`foo` of `foo.example.com`)
  - `models.LabelFromFQDNNoDot()`: Use this if your provider always gives you the FQDN (`foo.example.com`)
  - `models.LabelFromFQDNWithDot()`: Use this if your provider always give syou the FQDN+"." (`foo.example.com.`)
- Why 3 functions? Can't NewRecordConfig figure it out?
  - There are ambiguous cases that make it impossible to guess.
  - It is faster and more accurate to simply have multiple functions, one for each situation.
  - The truth is that your provider's API is going to only deliver the label one way. They're not going to change, as that would break too much code.

- `TTL` must be the desired TTL or `0` if it is unknown. Unknown TTLs are converted into the default TTL.

- `TYPE_STR_OR_NUM` can be either the record type's codepoint or a string:
  - codepoint: e.g. `dnsv2.TypeA`, `dnsv2.TypeMX`, `dnsv2.TypeCNAME`, `privatetypes.CLOUDFLAREAPISINGLEREDIRECT`
  - string (`"A"`, `"MX"`, `"CNAME"`, `"CLOUDFLAREAPI_SINGLE_REDIRECT"`)
  - Please use the constant when possible. Eventually we'll eliminate the strings. However, if all you have is the string, please let the factory do the conversion for you.

- `ARGS` is a list of fields in the order they appear in the struct.  The type doesn't matter as they will be converted automatically.  No need to convert strings to ints and so on. Even IP addresses are handled properly. Examples:
  - `dc.NewRecordConfig("mxhost", 0, dnsv2.TypeMX, "10", "mx.example.com.")`
  - `dc.NewRecordConfig("mxback", 0, dnsv2.TypeMX, 20, "mx2.example.com.")`
  - `dc.NewRecordConfig("www", 0, dnsv2.TypeA, "1.2.3.4")`
  - `addr, _ := netip.ParseAddr("192.168.1.1"); dc.NewRecordConfig("www", 0, dnsv2.TypeA, addr)`
  - `dc.NewRecordConfig("public", 0, dnsv2.TypeLOC, 42, 21, 54, "N", 71, 6, 18, "W", -24.05, 30, 0, 0)`
  - `dc.NewRecordConfig("public", 0, dnsv2.TypeLOC, "42", "21", "54", "N", "71", "6", "18", "W", "-24.05", 30, "0", 0)`

- `RFC1038_STRING` is a string that is parsed like the fields in a ZoneFile
  - `dc.NewRecordConfigParse("mxhost", 0, dnsv2.TypeMX, "10 mx.example.com.")`
  - `dc.NewRecordConfigParse("mxback", 0, dnsv2.TypeMX, "20 mx2.example.com.")`
  - `dc.NewRecordConfigParse("www", 0, dnsv2.TypeA, "1.2.3.4")`
  - `dc.NewRecordConfigParse("public", 0, dnsv2.TypeLOC, "42 21 54 N 71 6 18 W -24.05 30 0 0)`

Typically either `dc.NewRecordConfig()` or `dc.NewRecordConfigParse()` will satisfy all of your needs.  However occasionally there are situations where a particular record type needs special handling.  We recommend using a switch statement to handle the special case.

In this example, `*Parse` works just fine for all cases except `MX` records. Therefore, we process `MX` records first as a special case:

```go
    switch rtype {
      case dnsv2.TypeMX:
        preference := extractPreference(nativeRec)
        target := extractTarget(nativeRec)
        rc, err := dc.NewRecordConfig(label, ttl, dnsv2.TypeMX, preference, target)
      default:
        rc, err := dc.NewRecordConfigParse(label, 0, rtype, combined_fields)
    }
    if err != nil {
      return err
    }
    dc.AddRecord(rc)
```

Deprecated:

Please do not create your own `RecordConfig`'s:

```go
rc := &models.RecordConfig{Name: label, Type: "MX"}
rc.MxPreference = pref
...
```

## Getters/Setters for RDATA in `models.RecordConfig`

The fields of a DNS record are called the `RDATA` (resource data).

Accessing the RDATA using the `.GetRDATA()` getter, and the `.SetRDATA()` setter.

`.GetRDATA()`) returns the generic interface (`dnsv2.RDATA`).  The `.String()` function generates a zonefile-like string representing every field in the struct:

```go
rd := rc.GetRDATA()     // The generic RDATA
fmt.Printf("Like in a zonefile: %s\n", rd.String())
```

If you know the RDATA's type, you can cast it to the specific type and access the individual fields:

```go
rdmx := rd.(dnsv2.MX)   // Cast to the MX record
fmt.Printf("my MX is preference=%d target=%q\n", rdmx.Preference, rdmx.Mx)
fmt.Printf("Like in a zonefile: %s\n", rdmx.String()) // Same as rd.String()
```

To alter fields call `GetRDATA()`, change the fields, then call `SetRDATA()` to save it back.  `GetRDATA()` returns a copy of the RDATA, therefore mutating it without saving it back is fruitless.

```go
rd := rc.GetRDATA()     // The generic RDATA
rdmx := rd.(dnsv2.MX)   // Cast to the MX record
rdmx.Preference = 999   // Change a field.
rc.SetRDATA(rdmx)       // Update the record.
```

## Create `models.RecordConfig` literals for testdata

This is for creating test data only. They panic on error.

```go
dc, err := models.NewDomainConfig(zone)
dc.AddTestRC("www", 0, dnsv2.TypeA, "1.2.3.4")
dc.AddTestRC("mail", 0, dnsv2.TypeMX, 10, "mx1.example.com.")
dc.AddTestRCParse("mail", 0, dnsv2.TypeMX, "20 mx2.example.com.")
```

If you want to create an `models.RecordConfig` without adding it to a `dc`, there are `Must` versions
of `dc.NewRecordConfig()` and `dc.NewRecordConfigParse()`:

```go
dc, err := models.NewDomainConfig(zone)
rc0 := dc.MustNewRecordConfig("www", 0, dnsv2.TypeA, "1.2.3.4")
rc1 := dc.MustNewRecordConfig("mail", 0, dnsv2.TypeMX, 10, "mx1.example.com.")
rc2 := dc.MustNewRecordConfigParse("mail", 0, dnsv2.TypeMX, "20 mx2.example.com.")
```

## How to create a "builder"

A builder is a function that can be used in `dnsconfig.js` which outputs one or
more DNS records. For example, the `SPF_BUILDER()` function generates `TXT`
records.

Assume the builder is named "robert" and appears in `dnsconfig.js` as `ROBERT(label, data)`

- Create the builder in `models/b_robert.go`
. Register the builder:

```go
func init() {
        RegisterBuilder("LOC", BuilderLOC)
}
```

(see `models/b_loc.go` for an example)

- Create the `BuilderNAME()` function

Add a function called BuilderROBERT() with this signature:

```go
func BuilderROBERT(dc *DomainConfig, ttl uint32, args []any, metadata map[string]string, subdomain string) (Records, error) {
```

- `dc`: The domain the builder was called in.
- `ttl`: the desired TTL or `0` if it is unknown. Unknown TTLs are converted into the default TTL.
- `args`: the arguments passed to the function in `dnsconfig.js`. Each should be passed through a `mustbe.*` function before use.
- `metadata`: Any `{foo: "foo"}` (Javascript objects) passed to the function in `dnsconfig.js`.
- `subdomain`: If the builder was used in a `D_EXTEND()`, the subdomain will be non-nil.
  - If `D("example.com")` is followed by `D_EXTEND("foo.example.com")`, subdomain will be `foo`.
  - To calculate the label: `name, _ := dc.LabelFromDnsconfigjs(args[0].(string), subdomain)`
  - To calcuate a target name: `name, _ := mustbe.TagetHostWithSubdoman(dc.Name, subdomain, name)`

The builder function can do basically anything and generate as many records as it wants. `SPF_BUILDER()` returns many records. `LOC()` returns just one record.

All builders must have unit tests.

## How to manipulate domain/zone names

How to remove a domain from a name?

```go
txtutil.StripZone()
```

How to add a domain to a shortname?

```go
txtutil.Extend()
```

## How to label imports

To avoid confusion between old and new DNS modules, we always import them with explicit `v1` and `v2` names.

Here is the canonical list:

```go
import (
    dnsv1 "github.com/miekg/dns"
    dnsv2 "codeberg.org/miekg/dns"
    dnsutilv1 "github.com/miekg/dns/dnsutil"
    dnsutilv2 "codeberg.org/miekg/dns/dnsutil"
    dnsrdatav2 "codeberg.org/miekg/dns/rdata"
    dnstestv2 "codeberg.org/miekg/dns/dnstest"
    svcbv1 "github.com/miekg/dns/svcb"
    svcbv2 "codeberg.org/miekg/dns/svcb"
)
```
