# Creating new DNS Resource Types (rtypes)

- [Creating new DNS Resource Types (rtypes)](#creating-new-dns-resource-types-rtypes)
  - [How to add a CUSTOM record type (rtype)](#how-to-add-a-custom-record-type-rtype)
    - [Step 1. Pick a unique id](#step-1-pick-a-unique-id)
    - [Step 2. Describe the custom type in YAML](#step-2-describe-the-custom-type-in-yaml)
    - [Step 3. Generate the code](#step-3-generate-the-code)
    - [Step 4. Test](#step-4-test)
    - [Step 5. Write the remaining functions](#step-5-write-the-remaining-functions)
  - [How to add a new record type (rtype)](#how-to-add-a-new-record-type-rtype)
    - [Step 1. Update `pkg/js/helpers.js`](#step-1-update-pkgjshelpersjs)
    - [Step 2. Update `models/makers.go` (NOT NEEDED FOR CUSTOM TYPES)](#step-2-update-modelsmakersgo-not-needed-for-custom-types)
    - [Step 3. Update `models/populatelegacy.go`](#step-3-update-modelspopulatelegacygo)
    - [Step 4. Update `models/populaterd.go`](#step-4-update-modelspopulaterdgo)
    - [Step 5. Add a `CanUseTYPENAME`](#step-5-add-a-canusetypename)
    - [Step 6. Document it](#step-6-document-it)
    - [Step 7. Update the matrix](#step-7-update-the-matrix)
    - [Step 8: Add a `parse_tests` test case](#step-8-add-a-parse_tests-test-case)
    - [Step 9. Test it out with BIND](#step-9-test-it-out-with-bind)
    - [Step 10. Add an integration test helper](#step-10-add-an-integration-test-helper)
    - [Step 11. Add Integration tests](#step-11-add-integration-tests)
    - [Step 12: Support more providers](#step-12-support-more-providers)
    - [Step 13: Write documentation](#step-13-write-documentation)
    - [Step 14: "go generate"](#step-14-go-generate)
  - [How to enable an rtype in a provider](#how-to-enable-an-rtype-in-a-provider)
  - [How to add a "builder"](#how-to-add-a-builder)

Everyone is familiar with A, AAAA, CNAME, NS and other Rtypes. However DNSControl supports

- All standard (RFC) record types. Anything that [codeberg.org/miekg/dns supports](https://codeberg.org/miekg/dns)
- Custom record types. DNS Service Providers often add their own unique record types.
- Builders. Functions that can be used in `dnsconfig.js` that generate one or more records. For example, `SPF_BUILDER()` generates one or more `TXT` records.
  - Modern builders are implemented in Go, in `models/b_NAME.go`
  - Legacy builders are implemented purely in JavaScript in `pkg/js/helpers.js`
  - In the future, we plan on migrating all builders to Go.

The fields of standard and custom types are stored in a struct called an `RDATA`, which is stored in `models.RecordConfig.rdata`. Use the .GetRDATA() and .SetRDATA() getter/setter to access them.

Legacy types store their fields directly in `models.RecordConfig`. For example, the MX preference
is in `models.RecordConfig{MxPreference: 10}`. This means space is taken to store the MX Preference even when the record type is something else. While this seemed ok back in 2016, the structure is now very big. The goal of RecordConfig V3 is to save memory by eliminating those fields.

While we transition to RecordConfig V3, the data is stored in both places (RDATA and the legacy fields). Data is converted bidirectionally. If RDATA is nil, DNSControl will create it automatically by plucking the data from the legacy fields.  The reverse also happens: when `SetRDATA()` is called, it populates any legacy fields.

Eventually we'll get rid of the legacy code.

The new system consolidates everything related to a record type to the record type's "object" (as much as Go has objects).  However the legacy code spreads logic for a given record type in many places in the code.  Those places are marked with the comment  `// rtype_variations`

## How to add a CUSTOM record type (rtype)

Many providers support custom DNS record types.  For example, Cloudflare has
type called `CLOUDFLAREAPI_SINGLE_REDIRECT`.

Note: This is different than a "builder". A builder is a function that can be used in `dnsconfig.js` which outputs one or more DNS records. For example, the `SPF_BUILDER()` function generates `TXT` records.  See below.

To add a custom type, follow these steps:

### Step 1. Pick a unique id

Each custom type is assigned a codepoint.

Here's the last id used. Add one to this value. (There is plenty of error-checking in the system if you guess wrong).

```shell
grep codepoint pkg/privatetypes/types_generate.yaml | sort | tail -1
```

### Step 2. Describe the custom type in YAML

Custom types are described in `pkg/privatetypes/types_generate.yaml`.  The generator will create 3 Go files in `pkg/privatetypes`:

- `t_typename.go` -- structures, parsers, etc. (header + RDATA)
- `t_typename_test.go` -- Unit test of the parser and .String() functions.
- `rdata/rdata_typename.go` -- structures, parsers, etc. (just RDATA)

Add the custom type to `pkg/privatetypes/types_generate.yaml`. `Cloudflareapi_Single_Redirect` is a good example to copy.

Here's what the fields mean:

- `name:` Must be "snake case" with first letter initial caps. This name is used to generate all the variations: `CLOUDFLAREAPI_SINGLE_REDIRECT`, `CloudflareapiSingleDirect`, and otehrs.
- `codepoint:` The unique ID you picked earlier.
- `fields:` the fields in the record type.
  - `type` should match the name of the mustbe.* function used for this field. Typically you'll use:
    - TargetHost: A hostname that is a target, either a FQDN ending in `.` or `@` if it is the apex.
    - IPv4: An IPv4 address.
    - IPv6: An IPv6 address.
    - Uint8, Uint16, Uint32, Uint64, Int8, Int16, Int32, Int64, Float32, Float64: Various numeric formats.
    - Bespoke types like `OpenPGPKey` and `SoaMailbox` which are used by `OPENPGPKEY` and `SOA` respectively.
    - RawString: A string that is not validated, normalized, or altered in any way.
    - ToUpperRawString: Like RawString, but passed through strings.ToUpper() so that comparisons are case insensitive.
  - `test_data:` is test data for the unit test. One or two simple tests is fine. The system will generate a unit test that round-trips these examples through the parser and String() functions.
  - `optionalFields:` (optional) fields that are optional. The Make*() function won't expect them, but they will always be output in the `.String()` function.
  - `runtimeFields:` (optional, rarely used) are fields that store data needed during `preview/push`. For example, in `Cloudflareapi_Single_Redirect` the API sends a `SRRRulesetID` which needs to be stored later for use with any updates.

### Step 3. Generate the code

Now that you've created the `types_generate.yaml` file, generate all the code.

```shell
cd pkg/privatetypes && go generate
```

### Step 4. Test

```shell
go test -failfast -count=1 ./...
```

Feelf free to aadd to the code generator `pkg/privatetypes/types_generate.go` if your custom type requires
special code, new field types, etc.

Now this type is as functional as a standard type, except it is in the `privatetypes` and `privatetypesrdata` names spaces instead of the `dnsv2` and `dnsrdatav2` namespaces.

Standard types:

- `dnsv2.TypeSRV` -- the codepoint
- `dnsv2.SRV{}` -- the entire struct (header + RDATA) (rarely used)
- `dnsrdatav2.SRV{}` -- the RDATA struct

Custom types:

- `privatetypes.TypeAKAMAICDN` -- the codepoint
- `privatetypes.AKAMAICDN{}` -- the entire struct (header + RDATA) (rarely used)
- `privatetypesrdata.AKAMAICDN{}` -- the RDATA struct

### Step 5. Write the remaining functions

Your type is now registered with the system and can be treated
the same as a standard type.

Follow the instructions in the next section to complete the process.

## How to add a new record type (rtype)

Congrats!  A new RFC has been published that defines a new DNS record type!
(Or you've added a custom type).  How do we add support to DNSControl?

Note: Since DNSControl depends on `https://codeberg.org/miekg/dns` for basic DNS record types, we must first wait for miekg to add support. He's usually quite good at adding new types but [file an issue](https://codeberg.org/miekg/dns/issues/new) if you want to make sure it is on his radar.

Addding a new type has 2 major parts.  First DNSControl must be updated to support it. Once that is complete, each provider must be updated to handle it.

Enable the type in DNSControl itself:

### Step 1. Update `pkg/js/helpers.js`

- Add to list at the end. Just follow the pattern.
- This enables the record to be used in `dnsconfig.js`.

### Step 2. Update `models/makers.go` (NOT NEEDED FOR CUSTOM TYPES)

- Add a Make$TYPENAME
  - This takes arguments of any type (like NewRecordConfig()). Every argument must pass through a `mustbe.` function. See `pkg/mustbe/README.md` for details.
- Add this new Make$TYPENAME to the func init().

### Step 3. Update `models/populatelegacy.go`

- Add your new type to the switch statement.
- This protects backwards compatibility by populating the legacy fields with data from RDATA. For new rtypes, there shouldn't be any legacy fields.

### Step 4. Update `models/populaterd.go`

- Add your new type to the switch statement.
  - This protects forward compatibility by creating RDATA from the legacy fields. For new rtypes, there shouldn't be any legacy fields.

### Step 5. Add a `CanUseTYPENAME`

Since not all providers support this new record type, add a "capability" so that providers can mark themselves as willing.

You'll need to have Stringer installed:

```shell
go tool stringer
```

- Update `pkg/providers/capabilities.go` (search for CanUseSRV and add something similar. Please add it in alphabetical order!)
- Update `build/generate/featureMatrix.go` (search for SRV and do something similar for your type)
- Run: `cd pkg/providers && go generate`

### Step 6. Document it

Add documentation:

- `documentation/language-reference/domain-modifiers/TYPENAME.md` (see SRV.md as an example)
- `documentation/SUMMARY.md` Add your doc to the TOC.

### Step 7. Update the matrix

Add this feature to the feature matrix in `dnscontrol/build/generate/featureMatrix.go`. Add it to the variable `matrix` maintaining alphabetical ordering, which should look like this:

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

Then add it later in the file with a `setCapability()` statement, which should look like this:

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

Add the capability to the list of features that zones are validated against (i.e. if you want DNSControl to report an error if this feature is used with a DNS provider that doesn't support it). That's in the `checkProviderCapabilities` function in `pkg/normalize/validate.go`. It should look like this:

{% code title="pkg/normalize/validate.go" %}

```diff
var providerCapabilityChecks = []pairTypeCapability{
...
+   capabilityCheck("FOO", providers.CanUseFOO),
...
```

{% endcode %}

DNSControl will warn/error if this new record is used with a provider that does not support the capability.

- Add the capability to the validations in `pkg/normalize/validate.go` by adding it to `providerCapabilityChecks`
- Some capabilities can't be tested for. If such testing can't be done, add it to the whitelist in function `TestCapabilitiesAreFiltered` in `pkg/normalize/capabilities_test.go`

If the capabilities testing is not configured correctly, `go test ./...` will report something like the `MISSING` message below. In this example we removed `providers.CanUseCAA` from the `providerCapabilityChecks` list.

```text
--- FAIL: TestCapabilitiesAreFiltered (0.00s)
    capabilities_test.go:66: ok: providers.CanUseAlias (0) is checked for with "ALIAS"
    capabilities_test.go:68: MISSING: providers.CanUseCAA (1) is not checked by checkProviderCapabilities
    capabilities_test.go:66: ok: providers.CanUseNAPTR (3) is checked for with "NAPTR"
```

### Step 8: Add a `parse_tests` test case

Add at least one test case to the `pkg/js/parse_tests` directory. Test `013-mx.js` is a very simple one and is good for cloning. See also `017-txt.js`.

Run these tests via:

```shell
cd pkg/js
go test ./...

# To run only your new test, use the -run flag. In this example, we only run test 058:
go test -v -run 'TestParsedFiles/058'
```

If these tests pass you know the `dnsconfig.js` and `helpers.js` code is working correctly.

The tests also verify that for every "capability" there is a validation. This is explained in Step 2 (search for `TestCapabilitiesAreFiltered` or `MISSING`)

### Step 9. Test it out with BIND

The `BIND` provider supports all record types. Update `providers/bind` to support this
record type.  The next section describes how to enable a new record type on a provider.

Here's how to run the integration tests on the `BIND` provider:

```shell
cd integrationTest
go test -failfast -v -args -verbose -profile BIND
```

### Step 10. Add an integration test helper

- Edit `integrationTest/helpers_integration_test.go`
- Add a typename() function (alphabetically). For example, there are functions like `mx()` and `a()` which make it easy to write test cases.

### Step 11. Add Integration tests

Add at least one test case to the `integrationTest/integration_test.go` file.  Add tests that create the type then changes each field individually.  For example, the MX records are tested by creating an MX record, changing the target, changing the preference, then deleting the record.

Look for `func makeTests` and add the test to the end of this list.

Each `testgroup()` is a named list of tests.

{% code title="integration_test.go" lineNumbers="true" %}

```go
testgroup("MX",
  tc("MX record", mx("@", 5, "foo.com.")),
  tc("Change MX pref", mx("@", 10, "foo.com.")),
  tc("Change MX target", mx("@", 10, "mx.foo.com.")),
  tc("Make MX and A",
      mx("@", 10, "mx.foo.com."),
      a("@", "1.2.3.4"),
  ),
)
```

{% endcode %}

Line 1: `testgroup()` gives a name to a group of tests. It also tells the system to delete all records for this domain so that the tests begin with a blank slate.

Line 2-4: Each `tc()` encodes all the records of a zone.

The test framework will create the equivalent of a `dnsconfig.js` for each `tc()`. In this example, the first `dnsconfig.js` is a zone with a single MX record. The next `dnsconfig.js` is similar but with a different MX preference. The next `dnsconfig.js` changes the target. The last `tc()` creates a zone with two records, an MX and A.

 The test framework will run the equivalent of `dnscontrol push` twice for each `tc()`. The first time it expects changes. The second run it expects no changes are required because the first was successful. Otherwise, the test will fail.

If you look at the tests for `CAA`, it inserts a few records then attempts to modify each field of a record one at a time. This test was useful because it turns out we hadn't written the code to properly see a change in priority. We fixed this bug before the code made it into production.

Notice that some tests include `requires()`, `not()` and `only()` statements. This is how we restrict tests to certain providers. These options must be listed first in a `testgroup`. More details are in the source code.

To run the integration test with the BIND provider:

```shell
cd integrationTest
go test -v -args -verbose -profile BIND
```

### Step 12: Support more providers

Now add support in other providers. Add the `providers.CanUse...` flag to the provider and re-run the integration tests:

For example, this will run the tests on Amazon AWS Route53:

```shell
export R53_DOMAIN=dnscontroltest-r53.com  # Use a test domain.
export R53_KEY_ID=CHANGE_TO_THE_ID
export R53_KEY='CHANGE_TO_THE_KEY'
pushd integrationTest
go test -v -args -verbose -profile ROUTE53
popd
```

The test should reveal any bugs. Keep iterating between fixing the code and running the tests. When the tests all work, you are done. (Well, you might want to clean up some code a bit, but at least you know that everything is working.)

If you find bugs that aren't covered by the tests, please please please add a test that demonstrates the bug (then fix the bug, of course). This will help all future contributors. If you need help with adding tests, please ask!

### Step 13: Write documentation

Add a new Markdown file to `documentation/language-reference/domain-modifiers`. Copy an existing file (`CNAME.md` is a good example). The section between the lines of `---` is called the front matter and it has the following keys:

- `name`: The name of the record. This should match the file name and the name of the record in `helpers.js`.
- `parameters`: A list of parameter names, in order. Feel free to use spaces in the name if necessary. Your last parameter should be `modifiers...` to allow arbitrary modifiers like `TTL` to be applied to your record.
- `parameter_types`: an object with parameter names as keys and TypeScript type names as values. Check out existing record documentation if you’re not sure to put for a parameter. Note that this isn’t displayed on the website, it’s only used to generate the `.d.ts` file.

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

### Step 14: "go generate"

Re-generate the documentation:

```shell
go generate ./...
```

This will regenerate things like the table of which providers have which features and the `dnscontrol.d.ts` file.

## How to enable an rtype in a provider

Now that DNSControl understands the new record type, each provider must be updated
to recognize the new type.

This is different for every provider. Usually the steps are:

1. Add `CanUseTYPENAME` to the init() function
2. Update the toNative() function to support the type when `GetZoneRecords()` runs.
3. Update `GetZoneRecordsCorrections()`'s create/update/delete functions to support the type.
4. Run [the integration tests](https://docs.dnscontrol.org/developer-info/integration-tests) until they all pass.

ProTip: The integration tests work by running teach test case twice. The first time they expect changes, the second time they expect no changes:

- If there is an error in the first run, that probably indicates the toNative() or GetZoneRecordsCorrections() is broken. That is, the record is not being created properly.
- If there is an error in the second run, that probably indicates toRC() is broken (the native record is not being parsed properly) or the modify/delete functionality has a bug (the new record is not being generated properly).

It can be useful to check the DNS server provider's web console to see if the records are being created properly.

## How to add a "builder"

A builder is a function that can be used in `dnsconfig.js` which outputs one or
more DNS records. For example, the `SPF_BUILDER()` function generates `TXT`
records.

Assume the builder is named "robert" and appears in `dnsconfig.js` as `ROBERT(label, data)`

Step 1. Create the builder in `models/b_robert.go`
Step 2. Register the builder: (see `models/b_loc.go` as an example)

```go
func init() {
        RegisterBuilder("LOC", BuilderLOC)
}
```

Step 3. Add the `BuilderNAME()` function

Add a function called BuilderROBERT() with this signature:

```go
func BuilderROBERT(dc *DomainConfig, ttl uint32, args []any, subdomain string) (Records, error) {
```
