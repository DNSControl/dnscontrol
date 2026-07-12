# Cookbook

This document provides "cookbook" recipes for doing common tasks.

## Create a `models.DomainConfig`

- What: Create a `models.DomainConfig`
- Why: Providers are handed a `models.DomainConfig` that is already created. However often when we write tests we need to create a list of `models.RecordConfig`s, which are stored in a `models.DomainConfig`.

Recommended:

```go
dc, err := models.NewDomainConfig(zone)
dc.AddRecordConfig(models.MakeTestRC(label, ttl, type, args))
dc.AddRecordConfig(models.MakeTestRCParse(label, ttl, type, args))
```

Deprecated:

```go
dc := &models.DomainConfig{Name: origin}
```

## Create a `models.RecordConfig`

- What: Create a `models.RecordConfig`
- Why: One of the most important functions of a provider is to translate the native DNS records (as received from the API) to standard `models.RecordConfig` structs. Previously there were many ways to do this. Starting in DNSControl v5, we've standaredized on the following factories.

Recommended:

There are two primary ways to create an RC:

```go
rc, err := dc.NewRecordConfig(LABEL, TTL, TYPE_STR_OR_NUM, ARGS)
rc, err := dc.NewRecordConfigParse(LABEL, TTL, TYPE_STR_OR_NUM, RFC1038_STRING)
```

- These are a method of `models.DomainConfig` (typically the variable name is `dc`). This is done so that you don't have to pass additional parameters such as the zone name (required for normalizing labels).
- `NewRecordConfig()` takes a list of arguments. It doesn't matter if the arguments are strings, ints, netip.Addrs... the function will convert them to the correct type and return and error if they can't be converted.
- `NewRecordConfigParse()` takes the arguments as one long string, which is parsed. If your provider returns (for example) the MX record data as `10 mx.example.com.` and the SRV record data as `4 100 123 three.example.com.`, you can just send the whole string to this function. This replaces `models.PopulateFromString()`

- `LABEL`: Must be the output of one of these functions:
  - `models.LabelFromShort()`: Use this if your provider always gives you the shortname (`foo` of `foo.example.com`)
  - `models.LabelFromFQDNNoDot()`: Use this if your provider always gives you the FQDN (`foo.example.com`)
  - `models.LabelFromFQDNWithDot()`: Use this if your provider always give syou the FQDN+"." (`foo.example.com.`)
- Why doesn't NewRecordConfig just test the string and do the right thing?
  - There are too many ambiguous cases to get this correct every time.
  - It is faster and more accurate to simply have multiple functions, one for each situation.
  - The truth is that your provider's API is going to only deliver the label one way. They're not going to change, as that would break too much code.

- `TTL` must be the desired TTL or `0` if it is unknown. Unknown TTLs are converted into the default TTL.

- `TYPE_STR_OR_NUM` can be either the record type's constant (`dnsv2.TypeA - (`dnsv2.TypeA`, `dnsv2.TypeMX`, `dnsv2.TypeCNAME`, `privatetypes.CLOUDFLAREAPISINGLEREDIRECT`) or the string (`"A"`, `"MX"`, `"CNAME"`, `"CLOUDFLAREAPI_SINGLE_REDIRECT"`). Please use the constant when possible.

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

Typically either `dc.NewRecordConfig()` or `dc.NewRecordConfigParse()` will
satisfy all of your needs.  However occasionally there are situations where a
particular record type needs special handling.  We recommend using a switch
statement to handle the special case:

```go
    switch rtype {
      case dnsv2.TypeMX:
        preference := extractPreference(nativeRec)
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

```go
rc := &models.RecordConfig{Name: label, Type: "MX"}
rc.MxPreference = pref
...
```

## Create a native record from `models.RecordConfig`

When creating and updating DNS records, most APIs require you to create
a record in their native format.

The fields of a DNS record are called the `RDATA` (resource data). There is a getter that
returns the generic interface (`dnsv2.RDATA`):

The `.String()` function generates a zonefile-like string representing every field in the struct.

```go
rd := rc.GetRDATA()     // The generic RDATA
fmt.Printf("Like in a zonefile: %s\n", rd.String())
```

If you know the RDATA's type, you can cast it to the specific type and access the individual fields:

```go
rdmx := rd.(dnsv2.MX)   // Cast to the MX record
fmt.Printf("my MX is preference=%d target=%q\n", rdmx.Preference, rdmx.Mx)
```

Here's how you alter the fields:

```go
rdmx.Preference = 999   // Change a fields.
rc.SetRDATA(rdmx)       // Update the record.
```

## Create test `models.RecordConfig` data

This is for creating test data only. They panic on error.

```go
dc, err := models.NewDomainConfig(zone)
dc.AddTestRC("www", 0, dnsv2.TypeA, "1.2.3.4")
dc.AddTestRC("mail", 0, dnsv2.TypeMX, 10, "mx1.example.com.")
dc.AddTestRCParse("mail", 0, dnsv2.TypeMX, "20 mx2.example.com.")
```

If you want to create an `models.RecordConfig` without adding it to a `dc`, there are `Must` versions
of `dc.NewRecordConfig()1 and `dc.NewRecordConfigParse()`:

```go
dc, err := models.NewDomainConfig(zone)
rc0 := dc.MustNewRecordConfig("www", 0, dnsv2.TypeA, "1.2.3.4")
rc1 := dc.MustNewRecordConfig("mail", 0, dnsv2.TypeMX, 10, "mx1.example.com.")
rc2 := dc.MustNewRecordConfigParse("mail", 0, dnsv2.TypeMX, "20 mx2.example.com.")
```

## How to add a new RFC STANDARD record type (rtype)

Congrats!  A new RFC has been published that defines a new DNS record type!
How do we add support to DNSControl?

Since DNSControl depends on `https://codeberg.org/miekg/dns` for basic DNS
record types, we must first wait for miekg to add support. He's usually quite
good at adding new types but [file an issue](https://codeberg.org/miekg/dns/issues/new)
if you want to make sure it is on his radar.

Now there are two major steps.  First DNSControl must be updated to support it. Once that
is complete, each provider must be updated to handle it.

Enable the type in DNSControl itself:

* `pkg/js/helpers.js`:
  - Add to list at the end. Just follow the pattern.
  - This enables the record to be used in `dnsconfig.js`.
* `models/makers.go`: (NOT NEEDED FOR CUSTOM TYPES)
  - Add a Make$TYPENAME
  - This takes arguments of any type (like NewRecordConfig()). Every argument must pass through a `mustbe.` function. See `pkg/mustbe/README.md` for details.
* `models/makers.go`: (NOT NEEDED FOR CUSTOM TYPES)
  - Add this new Make$TYPENAME to the func init().
* `models/populatelegacy.go`: Add to the switch statement.
  - This protects backwards compatibility by populating the legacy fields with data from RDATA. For new rtypes, there shouldn't be any legacy fields.
* `models/populaterd.go:
  - Add to the switch statement.
  - This protects forward compatibility by creating RDATA from the legacy fields. For new rtypes, there shouldn't be any legacy fields.
* `integrationTest/helpers_integration_test.go`:
  - Add a typename() function (alphabetically). For example, there are functions like `mx()` and `a()` which make it easy to write test cases.
* `integrationTest/integration_test.go`:
  - Add tests that create the type, changes each field individually.
  - For example, the MX records are tested by creating an MX record, changing the target, changing the preference, then deleting the record.
* Add a `CanUseTYPENAME`:
  - Since not all providers support this new record type, add a "capability" so that providers can mark themselves as willing.
  - Update `pkg/providers/capabilities.go` (search for CanUseSRV and add something similar. Please add it in alphabetical order!)
  - Update `build/generate/featureMatrix.go` (search for SRV and do something similar for your type)
  - Run: `cd pkg/providers && go generate`
* Add documentation:
  - `documentation/language-reference/domain-modifiers/TYPENAME.md` (see SRV.md as an example)
  - `documentation/SUMMARY.md` Add your doc to the TOC.

Enable the type in a provider:

This is different for every provider. Usually the steps are:

* Add `CanUseTYPENAME` to the init() function
* Update the toNative() function to support the type when `GetZoneRecords()` runs.
* Update `GetZoneRecordsCorrections()`'s create/update/delete functions to support the type.

## How to add a CUSTOM record type (rtype)

Many providers support custom DNS record types.  For example, Cloudflare has
type called `CLOUDFLAREAPI_SINGLE_REDIRECT`.

Note: This is different than a "builder". A builder is a function in
`dnsconfig.js` which outputs one or more DNS records. For example, the
`SPF_BUILDER()` function generates `TXT` records.  See below.

Process overview:

* Pick a unique id: Here's the last id used. Add one to this value. (There is plenty of error-checking in the system if you guess wrong).
  - `grep codepoint pkg/privatetypes/types_generate.yaml | sort | tail -1`
* Add the custom type to `pkg/privatetypes/types_generate.yaml`
  - `Cloudflareapi_Single_Redirect` is a good example to copy.
  - `name:` Must be "snake case" with first letter initial caps.
  - `codepoint:` The unique ID you picked earlier.
  - `fields:` the fields in the record type.
    - The "type" should match mustbe.* functions. Typically you'll use:
      - TargetHost: A hostname that is a target, either a FQDN ending in `.` or `@` if it is the apex.
      - IPv4: An IPv4 address.
      - IPv6: An IPv6 address.
      - Uint8, Uint16, Uint32, Uint64
      - Int8, Int16, Int32, Int64
      - Float32, Float64
      - Bespoke types like `OpenPGPKey` and `SoaMailbox` which are used by `OPENPGPKEY` and `SOA` respectively.
      - RawString: A string that is not validated, normalized, or altered in any way.
      - ToUpperRawString: Like RawString, but passed through strings.ToUpper() so that comparisons are case insensitive.
    - `test_data:` is test data for the unit test. One or two simple tests is fine.
    - `optionalFields:` (optional) fields that are optional. The Make*() function won't expect them, but they will always be output in the `.String()` function.
    - `runtimeFields:` (optional, rarely used) are fields that store data needed during `preview/push`. For example, in `Cloudflareapi_Single_Redirect` the API sends a `SRRRulesetID` which needs to be stored later for use with any updates.

* Generate the code:
  - Now that you've created the `types_generate.yaml` file, generate all the code.
  - `cd pkg/privatetypes && go generate`

* Test.
  - `go test -failfast -count=1 ./...`
  - You may need to update the code generator `pkg/privatetypes/types_generate.go`

Now this type is as functional as a standard type. Follow the `How to add a new RFC STANDARD record type (rtype)` instructions above.

Standard types:
- `dnsv2.TypeSRV` -- the codepoint
- `dnsv2.SRV{}` -- the entire struct (header + RDATA) (rarely used)
- `dnsrdatav2.SRV{}` -- the RDATA struct

Custom types:
- `privatetypes.TypeAKAMAICDN` -- the codepoint
- `privatetypes.AKAMAICDN{}` -- the entire struct (header + RDATA) (rarely used)
- `privatetypesrdata.AKAMAICDN{}` -- the RDATA struct


## How to add a "builder"

A builder is a function that can be used in `dnsconfig.js` which outputs one or
more DNS records. For example, the `SPF_BUILDER()` function generates `TXT`
records.

Assume the builder is named "robert" and appears in `dnsconfig.js` as `ROBERT(label, data)`

* Create the builder in `models/b_robert.go`
* Register the builder: (see `models/b_loc.go` as an example)

```go
func init() {
        RegisterBuilder("LOC", BuilderLOC)
}
```

* Add the `BuilderNAME()` function

Add a function called BuilderROBERT() with this signature:

```go
func BuilderROBERT(dc *DomainConfig, ttl uint32, args []any, subdomain string) (Records, error) {
```

## How to manipulate domain/zone names

How to remove a domain from a name?

```go
txtutil.StripZone()
```

How to add a domain to a shortname?

```go
txtutil.Extend()
```

## What to import?

To avoid confusion between old and new DNS modules, we always import them with explicit `v1` and `v2` names:

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
