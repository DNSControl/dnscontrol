# Creating new DNS Resource Types (rtypes)

- [Creating new DNS Resource Types (rtypes)](#creating-new-dns-resource-types-rtypes)
  - [How to add a CUSTOM record type (rtype)](#how-to-add-a-custom-record-type-rtype)
    - [Step 1: Pick a unique id](#step-1-pick-a-unique-id)
    - [Step 2: Describe the custom type in YAML](#step-2-describe-the-custom-type-in-yaml)
    - [Step 3: Generate the code](#step-3-generate-the-code)
    - [Step 4: Test](#step-4-test)
    - [Step 5: Write the remaining functions](#step-5-write-the-remaining-functions)
  - [How to add a new record type (rtype)](#how-to-add-a-new-record-type-rtype)
    - [Step 1: Update `pkg/js/helpers.js`](#step-1-update-pkgjshelpersjs)
    - [Step 2: Update `models/makers.go` (NOT NEEDED FOR CUSTOM TYPES)](#step-2-update-modelsmakersgo-not-needed-for-custom-types)
    - [Step 3: Document it](#step-3-document-it)
    - [Step 4: Update the matrix](#step-4-update-the-matrix)
    - [Step 5: Add a `parse_tests` test case](#step-5-add-a-parse_tests-test-case)
    - [Step 6: Add Integration tests](#step-6-add-integration-tests)
    - [Step 7: Support more providers](#step-7-support-more-providers)
    - [Step 8: Write documentation](#step-8-write-documentation)
    - [Step 9: "go generate"](#step-9-go-generate)
  - [How to enable an rtype in a provider](#how-to-enable-an-rtype-in-a-provider)
  - [How to add a "builder"](#how-to-add-a-builder)
    - [Legacy builders](#legacy-builders)

Everyone is familiar with A, AAAA, CNAME, NS and other Rtypes. However DNSControl also supports:

- **Standard record types:** Anything defined by an RFC. They must be supported by [codeberg.org/miekg/dns](https://codeberg.org/miekg/dns).
- **Custom record types:** Pseudo-types, typically specific to a particular DNS Service Provider.
- **Builders:** Pseuddo-types that generate 1 more DNS records. For example `SPF_BUILDER()` generates one or more `TXT` records.

## How to add a CUSTOM record type (rtype)

Many providers support custom DNS record types.  For example, Cloudflare has
type called `CLOUDFLAREAPI_SINGLE_REDIRECT`.

Note: This is different than a "builder". A builder is a function that can be used in `dnsconfig.js` which outputs one or more DNS records. For example, the `SPF_BUILDER()` function generates `TXT` records.  See below.

To add a custom type, follow these steps:

### Step 1: Pick a unique id

Each custom type is assigned a codepoint.

Here's the last id used. Add one to this value. (There is plenty of error-checking in the system if you guess wrong).

```shell
grep codepoint pkg/privatetypes/types_generate.yaml | sort | tail -1
```

### Step 2: Describe the custom type in YAML

Custom types are described in `pkg/privatetypes/types_generate.yaml`.  A generator will create these files in `pkg/privatetypes`:

- `t_typename.go` -- structures, parsers, etc. (header + RDATA)
- `t_typename_test.go` -- Unit test of the parser and .String() functions.
- `rdata/rdata_typename.go` -- structures, parsers, etc. (just RDATA)

Add the custom type to `pkg/privatetypes/types_generate.yaml`. `Cloudflareapi_Single_Redirect` is a good example to copy.

Here's what the fields mean:

- `name:` Must be "snake case" with first letter initial caps. This name is used to generate all the variations: `CLOUDFLAREAPI_SINGLE_REDIRECT`, `CloudflareapiSingleDirect`, and otehrs.
- `codepoint:` The unique ID you picked earlier.
- `fields:` the fields in the record type.
  - `type` should match the name of the mustbe.* function used for this field. Typically you'll use:
    - IPv4: An IPv4 address.
    - IPv6: An IPv6 address.
    - Uint8, Uint16, Uint32, Uint64, Int8, Int16, Int32, Int64, Float32, Float64: Various numeric formats.
    - Txts: A list of TXT strings.
    - Bool, BoolString: Bool and a stringified Bool.
    - TargetHost: A hostname that is a target, either a FQDN ending in `.` or `@` if it is the apex.
    - TargetHostSRV: Like TargetHost but permits SRV-compatible targets.
    - RawString: A string that is not validated, normalized, or altered in any way.
    - ToUpperRawString: Like RawString, but passed through strings.ToUpper() so that comparisons are case insensitive.
    - Bespoke types like `OpenPGPKey` and `SoaMailbox` which are used by `OPENPGPKEY` and `SOA` respectively.
  - `test_data:` is test data for the unit test. One or two simple tests is fine. The system will generate a unit test that round-trips these examples through the parser and String() functions.
  - `optionalFields:` (optional) fields that are optional. The Make*() function won't expect them, but they will always be output in the `.String()` function.
  - `runtimeFields:` (optional, rarely used) are fields that store data needed during `preview/push`. For example, in `Cloudflareapi_Single_Redirect` the API sends a `SRRRulesetID` which needs to be stored later for use with any updates.

### Step 3: Generate the code

Now that you've created the `types_generate.yaml` file, generate all the code.

```shell
cd pkg/privatetypes && go generate
```

### Step 4: Test

```shell
go test -failfast -count=1 ./...
```

You may neeed to modify the code generator `pkg/privatetypes/types_generate.go` if your custom type requires
special code, new field types, etc.

Now this type is as functional just like the standard types. The only differnece is that it is in the `privatetypes` and `privatetypesrdata` names spaces instead of the `dnsv2` and `dnsrdatav2` namespaces.

| Purpose | Standard | Custom | Notes |
| --- | --- | --- | --- |
| codepoint | `dnsv2.TypeSRV` | `privatetypes.TypeAKAMAICDN` | int representing the record type |
| struct | `dnsv2.SRV{}` | `privatetypes.AKAMAICDN{}` | the entire struct (header + RDATA) |
| RDATA struct | `dnsrdatav2.SRV{}` | `privatetypesrdata.AKAMAICDN{}` | The RDATA (fields) of the struct |

### Step 5: Write the remaining functions

Your type is now registered with the system and can be treated
the same as a standard type.

Follow the instructions in the next section to complete the process.

## How to add a new record type (rtype)

Congrats!  A new RFC has been published that defines a new DNS record type!
(Or you've added a custom type).  How do we add support to DNSControl?

Note: Since DNSControl depends on `https://codeberg.org/miekg/dns` for basic DNS record types, we must first wait for miekg to add support. He's usually quite good at adding new types but [file an issue](https://codeberg.org/miekg/dns/issues/new) if you want to make sure it is on his radar.

Addding a new type has two major parts.  First DNSControl must be updated to support it. Once that is complete, each provider must be updated to handle it.

Enable the type in DNSControl itself:

### Step 1: Update `pkg/js/helpers.js`

This enables the record to be used in `dnsconfig.js`.

- Add to list at the end. Just follow the pattern. Please keep the list sorted alphabetically.

Example:

```javascript
var CSYNC = rawrecordBuilder("CSYNC");
```

### Step 2: Update `models/makers.go` (NOT NEEDED FOR CUSTOM TYPES)

This creates a way for providers to create RDATA of this type.

- Edit `models/makers.go`
- Add a Make$TYPENAME
  - This takes arguments of any type (like NewRecordConfig()). Every argument must pass through a `mustbe.` function. See `pkg/mustbe/README.md` for details.
- Add this new Make$TYPENAME to the func init().

### Step 3: Document it

Add documentation:

- `documentation/language-reference/domain-modifiers/TYPENAME.md` (see SRV.md as an example)
- `documentation/SUMMARY.md` Add your doc to the TOC.

### Step 4: Update the matrix

In this step we make the new type appear in the feature matrics.

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

### Step 5: Add a `parse_tests` test case

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

### Step 6: Add Integration tests

Add an integration test for the new type.  The test group should create the record then attempt to change each field one at at a time.

- Edit `integrationTest/integration_test.go`
- Look for `func makeTests` and add a `testgroup()` to the end of the list.

Each `testgroup()` is a named list of tests.

{% code title="integration_test.go" lineNumbers="true" %}

```go
testgroup("MX",
  tc("MX record", tr("@", "MX", "5 foo.com.")),
  tc("Change MX pref", tr("@", "MX", "10 foo.com.")),
  tc("Change MX target", tr("@", "MX", "10 mx.foo.com.")),
  tc("Make MX and A",
      tr("@", "MX", "10 mx.foo.com."),
      tr("@", "A", "1.2.3.4"),
  ),
)
```

{% endcode %}

Line 1: `testgroup()` gives a name to a group of tests. It also tells the system to delete all records for this domain so that the tests begin with a blank slate.

Line 2-4: Each `tc()` encodes all the records of a zone.

The test framework will create the equivalent of a `dnsconfig.js` for each `tc()`. In this example, the first `dnsconfig.js` is a zone with a single MX record (`tr()` stands for "test record"). The next `dnsconfig.js` is similar but with a different MX preference. The next `dnsconfig.js` changes the target. The last `tc()` creates a zone with two records, an MX and A.

 The test framework will run the equivalent of `dnscontrol push` twice for each `tc()`. The first time it expects changes, because we are creating or modifying a record. The second run is expected to have no changes since the first run should have been successful. Otherwise, the test is marked as a failure.

Notice that some tests include `requires()`, `not()` and `only()` statements. This is how we restrict tests to certain providers. These options must be listed first in a `testgroup`. More details are in the source code. They are generally not needed because the framework now only runs a `testgroup()` if the DNS record types are compatible with the provider. Therefore, if a provider doesn't support (for example) `SRV`, the SRV tests will automatically be skipped.

The BIND provider supports all types. Therefore, it's a good way to test if your integration test worked.  To run the integration test with the BIND provider:

```shell
cd integrationTest
go test -v -args -verbose -profile BIND
```

### Step 7: Support more providers

Now we activate the type for a provider.

1. Add the record type to the list of supported types in the provider's `providers.Definition{}`. Look for the `SupportedTypes` field.
2. Run the integration tests. It should fail because we haven't added code to support the type yet.

For example, this will run the tests on Amazon AWS Route53:

```shell
export R53_DOMAIN=dnscontroltest-r53.com  # Use a test domain.
export R53_KEY_ID=CHANGE_TO_THE_ID
export R53_KEY='CHANGE_TO_THE_KEY'
pushd integrationTest
go test -v -args -verbose -profile ROUTE53
popd
```

You can save time by using the `-start` and `-end` flags. For example, to only run test 100, add `-start 100 -end 100` to the command.

Now that you've isolated the failure, update the provider's code to support this type.

(In some rare occasions the provider will simply work on the first try. Congrats! You can skip the rest of this step.)

Keep iterating between fixing the code and running the tests. When the tests all work, you are done. (Well, you might want to clean up some code a bit, but at least you know that everything is working.)

If you find bugs that aren't covered by the tests, please please please add a test that demonstrates the bug (then fix the bug, of course). This will help all future contributors. If you need help with adding tests, please ask!

### Step 8: Write documentation

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
    * [DnsProvider](language-reference/domain-modifiers/SERVICE.md)
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

### Step 9: "go generate"

Re-generate the documentation:

```shell
go generate ./...
```

This will regenerate things like the table of which providers have which features and the `dnscontrol.d.ts` file.

## How to enable an rtype in a provider

Now that DNSControl understands the new record type, each provider must be updated
to recognize the new type.

This is different for every provider. Usually the steps are:

1. Add the record type to the provider's `SupportedTypes` declaration.
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

Step 1. Add it to the list at the end of `pkg/js/helpers.js`. The 2nd parameter should be `true` if the builder doesn't take a label as the first parameter.
Step 2. Create the builder in `models/b_robert.go`
Step 3. Register the builder: (see `models/b_loc.go` as an example)

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

### Legacy builders

- Modern builders are implemented in Go, in `models/b_NAME.go`
- Legacy builders are implemented purely in JavaScript in `pkg/js/helpers.js`

Over time we hope to port all the legacy builders to Go.
