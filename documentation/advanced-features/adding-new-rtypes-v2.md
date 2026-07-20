# Creating new DNS Resource Types (rtypes) (v4.28 and later)

Everyone is familiar with A, AAAA, CNAME, NS and other Rtypes. However there are new record types being added all the time. Each new record type requires special handling by DNSControl.

Version v4.28.0 greatly simplified how to add new record types. As a demonstration of this new method it added the "RP" type and ported the existing "CLOUDFLAREAPI_SINGLE_REDIRECT" type.  All other records still use the old method. The old and new methods co-exist, though eventually we hope to migrate everything to the new method.

# What's new?

* OLD: the RecordConfig struct keeps getting larger as new record types require more fields.
* NEW: the RecordConfig struct has a pointer to a struct describing the fields of the record.
* Benefit: Saves memory.

* OLD: helpers.js performs validation, executes builders, etc.  Since we don't have a test framework for Javascript, this is brittle and difficult to debug.
* NEW: helpers.js packs up the fields, whatever they are, and handes them off to Go code for processing. The Go test framework is available.
* Benefit: More testable, easier to  develop as you don't need to know 2 languages.

* OLD: Critical things like IDN processing, normalization (downcasing), and validation happen late in the pipeline by `pkg/normalize`.
* NEW: The factory that creates a RecordConfig performs all that.
* Benefit: No code has to be concerned with "has this RecordConfig been normalized/IDN-ized yet?". This makes it easier to write and debug code.

* OLD: Code that affects a Record Type is splattered all over the code base.
* NEW: Code related to a Record Type is all in one file.
* Benefit: Centralize all concerns about a record type in one file (with some exceptions that we hope to fix eventually).

# Overview

There are two parts to adding a new Record Type (rtype). First we activate it in the parser for dnsconfig.js. Then we update any provider to be aware of the rtype.

Activate it in dnsconfig.js:

1. Update `pkg/js/helpers.js` (add just 1 line!)
2. Create a file in `pkg/rtype` (for example, `pkg/rtype/rp.go`) with a parser.

Update providers to be aware of the rtype:

1. Update the toRC() function (whatever it may be called).
1. Update the toNative() and any create/delete/change functions.

# Updating the parser

In these examples, the new type will be called `THING` (or `thing`).

Step 1: Update helpers.js

At the end of the file, add a single line named after the record type.

```
var RP = rawrecordBuilder('RP');
```

In this example, the first `RP` is the name of the function that users will type in dnsconfig.js.  For example, `A("label", "10.2.3.4"),` (though the "A" record type hasn't been ported to the new system yet).

Step 2: Create the parser

Create a file in `pkg/rtype/thing.go` named after the record type (all lowercase).

Copy `pkg/rtype/rp.go` as it is a good prototype.

Step 2a: Update init()

Update the init function to register your new type. That is, change `RP` to `THING`.

```
func init() {
	rtypecontrol.Register(&THING{})
}
```

Step 2b: Create the struct

Create a struct that will store the fields of this rtype.

If this is a standard rtype, borrow from miekg/dns:

```
type THING struct {
	dns.THING
}
```

If this is not a standard type, list the fields:

```
type THING struct {
	ThingField1 uint32
    ThingField2 string
    ThingField3 string
	ThingField4 uint32
}
```

Step 2c: Create Name()

Create a function `Name()` that outputs the name of the type.

```
func (handle *THING) Name() string {
	return "THING"
}
```

Step 2d: Create FromArgs

The "FromArgs" function receives an array of `any` which can contain any type.  The "PaveArgs" function will convert them to the types you need. For example, it will convert numbers to strings, or strings to numbers.

* "s": Convert to string
* "i": Convert to int16

`args[0]` is the label. You can skip it as that is already processed for you.  (If you want to modify the label, see cfsingle.go as an example of how to do that.)

If THING takes 4 parameters (2 ints and 2 strings), you might pave the arguments as follows:

```
	if err := rtypecontrol.PaveArgs(args[1:], "issi"); err != nil {
```

Now you can fill the struct as needed:

```
	fields := &THING{
		dns.THING{
			ThingField1: args[1].(uint16),
			ThingField2:  args[2].(string),
			ThingField3:  args[3].(string),
			ThingField4:  args[4].(uint16),
		},
	}
```

Now call FromStruct to finsh up.

```
	return handle.FromStruct(dc, rec, args[0].(string), fields)
```

FromStruct does many things:

1. Installs the struct in .F
2. Converts any names to IDN and Unicode equivalents for future reference.
2. Performs any validation (for example, if ThingField1 has to be between 0 and 999)
2. Generate the .ZonefilePartial field: This is what the record outputs in a zonefile.
2. Generate the .Comparable field: This is an opaque string used to compare two RecordConfigs. If the strings are not an exact match, they are considered "not equal".

Create the CopyToLegacyFields function

This updates any of the legacy fields. The most important is the .target field, which we usually store a copy of the .ZonefilePartial.

When we migrate other rtypes this will populate the legacy RecordConfig fields. For example, when we migrate `SRV`, this function will populate the `Srv*` fields.  Then, eventually, we'll remove those legacy fields.

TODO: `js/parse_test`

## Add a capability for the record type

You'll need to mark which providers support this record type. The initial PR should implement this record for the `BIND` provider at a minimum.  `BIND` outputs non-standard rtypes as a comment.

-   Add the capability to the file `dnscontrol/providers/capabilities.go` (look for `CanUseAlias` and add it to the end of the list.)
-   Run stringer to auto-update the file `dnscontrol/providers/capability_string.go`

```shell
pushd providers/
go tool stringer -type=Capability
popd
```
alternatively

```shell
pushd providers/
go generate
popd
```

-   Add this feature to the feature matrix in `dnscontrol/build/generate/featureMatrix.go`. Add it to the variable `matrix` maintaining alphabetical ordering, which should look like this:

    {% code title="dnscontrol/build/generate/featureMatrix.go" %}
    ```diff
    func matrixData() *FeatureMatrix {
        const (
            ...
            DomainModifierCaa    = "[`CAA`](language-reference/domain-modifiers/CAA.md)"
    +       DomainModifierFoo    = "[`FOO`](language-reference/domain-modifiers/FOO.md)"
            DomainModifierLoc    = "[`LOC`](language-reference/domain-modifiers/LOC.md)"
            ...
        )
        matrix := &FeatureMatrix{
            Providers: map[string]FeatureMap{},
            Features: []string{
                ...
                DomainModifierCaa,
    +           DomainModifierFoo,
                DomainModifierLoc,
                ...
            },
        }
    ```
    {% endcode %}

    then add it later in the file with a `setCapability()` statement, which should look like this:

    {% code title="dnscontrol/build/generate/featureMatrix.go" %}
    ```diff
    ...
    +       setCapability(
    +           DomainModifierFoo,
    +           providers.CanUseFOO,
    +       )
    ...
    ```
    {% endcode %}

-   Add the capability to the list of features that zones are validated against (i.e. if you want DNSControl to report an error if this feature is used with a DNS provider that doesn't support it). That's in the `checkProviderCapabilities` function in `pkg/normalize/validate.go`. It should look like this:

    {% code title="pkg/normalize/validate.go" %}
    ```diff
    var providerCapabilityChecks = []pairTypeCapability{
    ...
    +   capabilityCheck("FOO", providers.CanUseFOO),
    ...
    ```
    {% endcode %}

-   Mark the `bind` provider as supporting this record type by updating `dnscontrol/providers/bind/bindProvider.go` (look for `providers.CanUse` and you'll see what to do).

DNSControl will warn/error if this new record is used with a provider that does not support the capability.

-   Add the capability to the validations in `pkg/normalize/validate.go` by adding it to `providerCapabilityChecks`
-   Some capabilities can't be tested for. If such testing can't be done, add it to the whitelist in function `TestCapabilitiesAreFiltered` in `pkg/normalize/capabilities_test.go`

If the capabilities testing is not configured correctly, `go test ./...` will report something like the `MISSING` message below. In this example we removed `providers.CanUseCAA` from the `providerCapabilityChecks` list.

```text
--- FAIL: TestCapabilitiesAreFiltered (0.00s)
    capabilities_test.go:66: ok: providers.CanUseAlias (0) is checked for with "ALIAS"
    capabilities_test.go:68: MISSING: providers.CanUseCAA (1) is not checked by checkProviderCapabilities
    capabilities_test.go:66: ok: providers.CanUseNAPTR (3) is checked for with "NAPTR"
```

## Step 6: Add an `integrationTest` test case

(see old doc)

## Step 8: Write documentation

Add a new Markdown file to `documentation/language-reference/domain-modifiers`. Copy an existing file (`CNAME.md` is a good example). The section between the lines of `---` is called the front matter and it has the following keys:

-   `name`: The name of the record. This should match the file name and the name of the record in `helpers.js`.
-   `parameters`: A list of parameter names, in order. Feel free to use spaces in the name if necessary. Your last parameter should be `modifiers...` to allow arbitrary modifiers like `TTL` to be applied to your record.
-   `parameter_types`: an object with parameter names as keys and TypeScript type names as values. Check out existing record documentation if you’re not sure to put for a parameter. Note that this isn’t displayed on the website, it’s only used to generate the `.d.ts` file.

The rest of the file is the documentation. You can use Markdown syntax to format the text.

Add the new file `FOO.md` to the documentation table of contents `documentation/SUMMARY.md` > `Domain Modifiers`, and/or to the `Service Provider specific` section if you made a record specific to a provider, and to the `Record Modifiers` section if you created any `*_BUILDER` or `*_HELPER` or similar functions for the new record type:

{% code title="documentation/SUMMARY.md" %}
```diff
...
* Domain Modifiers
...
    * [DnsProvider](language-reference/domain-modifiers/DnsProvider.md)
+   * [FOO](language-reference/domain-modifiers/FOO.md)
    * [FRAME](language-reference/domain-modifiers/FRAME.md)
...
    * Service Provider specific
...
        * ClouDNS
            * [CLOUDNS_WR](language-reference/domain-modifiers/CLOUDNS_WR.md)
+       * ASDF
+           * [ASDF_NINJA](language-reference/domain-modifiers/ASDF_NINJA.md)
...
* Record Modifiers
...
    * [DMARC_BUILDER](language-reference/domain-modifiers/DMARC_BUILDER.md)
+   * [FOO_HELPER](language-reference/record-modifiers/FOO_HELPER.md)
    * [SPF_BUILDER](language-reference/domain-modifiers/SPF_BUILDER.md)
...
```
{% endcode %}

TODO Run the integration tests for BIND

## Step 9: "Update the index page"

Edit `documentation/SUMMARY.md` and add a link to the new record type's doc.

## Step 9: "go generate"

Re-generate the documentation:

```shell
go generate ./...
```

This will regenerate things like the table of which providers have which features and the `dnscontrol.d.ts` file.

# Update providers

When a provider needs to create a THING, they have two choices

If you have the fields already in variables of the right type, use NewRecordConfigFromStruct:

```
rec, err = rtypecontrol.NewRecordConfigFromStruct(name, ttl, "THING", rtype.THING{a, b, c, d}, dc)
```

If you have the fields in variables that are strings that need to be converted, use `NewRecordConfigFromRaw()`:

```
rec, err := rtypecontrol.NewRecordConfigFromRaw("CF_REDIRECT", ttl, []any{pattern, target}, dc)
```

There's a good chance you know the zoneName but don't have a complete models.DomainConfig(). That's ok. As long as you know the zone name, we can fake it: (this is a hack we'll figure out how to eliminate eventually).

```
dc := models.MakeFakeDomainConfig(zoneName)
```

# Tips for "builders"

A "builder" is a function that create other records.  For example, SPF_BUILDER() creates a `TXT()` record.

A good example of a builder is `providers/cloudflare/rtypes/cfsingleredirect/cfredirect.go`

Simply do the processing you need, then create the resulting rtype you want:

In cfredirect.go, CF_REDIRECT is a builder that generates a CLOUDFLAREAPI_SINGLE_REDIRECT, represented by the SingleRedirectConfig struct:

```
	sr := SingleRedirectConfig{}
	rec.Type = sr.Name() // This record is now a CLOUDFLAREAPI_SINGLE_REDIRECT
	err = sr.FromArgs(dc, rec, []any{name, code, srWhen, srThen})
	if err != nil {
		return err
	}
```
