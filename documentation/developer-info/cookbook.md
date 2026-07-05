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

There are 

```go
rc, err := dc.NewRecordConfig(LABEL, TTL, TYPE_STR_OR_NUM, ARGS)
rc, err := dc.NewRecordConfigParse(LABEL, TTL, TYPE_STR_OR_NUM, RFC1038_STRING)
```

- These are a method of `models.DomainConfig` (typically the variable name is `dc`). This is done so that you don't have to pass additional parameters such as the zone name (required for normalizing labels).
- `NewRecordConfig()` takes a list of arguments which are converted automatically.
- `NewRecordConfigParse()` takes the arguments as a string, which is parsed. This replaces `models.PopulateFromString()`

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

```go
rd := rc.GetRDATA()     // The generic RDATA
fmt.Printf("Like in a zonefile: %s\n", rd.String())
```

If you know the RDATA's type, you can cast it to the specific type and manipulate it:

```go
rdmx := rd.(dnsv2.MX)   // Cast to the MX record
fmt.Printf("my MX is preference=%d target=%q\n", rdmx.Preference, rdmx.Mx)
```

You can even change it and set it back 

```go
rdmx.Preference = 999   // Change a fields.
rc.SetRDATA(rdmx)       // Update the record.
rc.RecomputeV3Fields()  // Compute any derived fields.
```

## Create test `models.RecordConfig` data

This is for creating test data only. They panic on error.

```go
dc, err := models.NewDomainConfig(zone)
dc.AddTestRC("www", 0, dnsv2.TypeA, "1.2.3.4")
dc.AddTestRC("mail", 0, dnsv2.TypeMX, 10, "mx.example.com.")
```

TODO: Create a MustMakeRC().

## How to add a RFC STANDARD record type (rtype) 

Congrats!  A new RFC has been published that defines a new DNS record type!
How do we add support to DNSControl?

Since DNSControl depends on `https://codeberg.org/miekg/dns` for basic DNS
record types, we must first wait for miekg to add support. He's usually quite
good at adding new types but file an issue if you want to make sure it is on
his radar.

Now there are two major steps.  First DNSControl must be updated to support it. Once that
is complete, each provider must be updated to handle it.

Updating DNSControl itself:

* models/makers.go: Add a Make$TYPENAME
* models/makers.go: Add to the func init().
* models/fixhack.go: Add to the switch statement.
* integrationTest/helpers_integration_test.go: Add a typename() function
* integrationTest/integration_test.go: Add tests that create the type, changes each field indiviually.
* pkg/js/helpers.js: Add to list at the end.
* TODO: Add a CanUseTYPENAME
* TODO: Add documentation

Updating providers:

This is different for every provider. Usually the steps are:

* Add CanUseTYPENAME to the init() function
* update the toNative() function to support the type.
* update GetZoneRecordsCorrections()'s create/update/delete functions to support the type.

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
  - TODO: Describe the YAML format
* Generate the code:
  - `cd pkg/privatetypes && go generate`
* Add it to the switch statement in `models/backfill.go`
  - This should be a no-op for new types.
* Test.
  - You may need to update the code generator `pkg/privatetypes/types_generate.go` 
* TODO: Add a CanUseTYPENAME
* TODO: Add documentation


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
