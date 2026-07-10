# Creating new DNS Resource Types (rtypes)

Everyone is familiar with A, AAAA, CNAME, NS and other Rtypes. However there are new record types being added all the time. Each new record type requires special handling by DNSControl.

If a record simply has a single "target", then there is little to do because it is handled similarly to A, CNAME, and so on. However if there are multiple fields within the record you have more work to do.

Our general philosophy is:

-   Internally the individual fields of a record are kept separate. If a particular provider combines them into one big string, that kind of thing is done in the provider code at the end of the food chain. For example, an MX record has a Target (`aspmx.l.google.com.`) and a preference (`10`). Some systems combine this into one string (`10 aspmx.l.google.com.`). We keep the two values separate in `RecordConfig` and leave it up to the individual providers to merge them when required. An earlier implementation kept everything combined and we found ourselves constantly parsing and re-parsing the target. It was inefficient and lead to many bugs.
-   Anywhere we have a special case for a particular Rtype, we use a `switch` statement and have a `case` for every single record type, usually with a `default:` case that calls `panic()`. This way developers adding a new record type will quickly find where they need to add code (the panic will tell them where). Before we did this, missing implementation code would go unnoticed for months.
-   Keep things alphabetical. If you are adding your record type to a case statement, function library, or whatever, please list it alphabetically along with the others when possible.

Step 2 requires `stringer` which is available using
```shell
go tool stringer
```
## Step 1: Update `RecordConfig` in `models/record.go`

If the record has any unique fields, add them to `RecordConfig`. The field name should be the record type, then the field name as used in `github.com/miekg/dns/types.go`. For example, the `CAA` record has a field called `Flag`, therefore the field name in `RecordConfig` is CaaFlag (not `CaaFlags` or `CAAFlags`).

Here are some examples:

```go
type RecordConfig struct {
  ...
  MxPreference uint16            `json:"mxpreference,omitempty"`
  SrvPriority  uint16            `json:"srvpriority,omitempty"`
  SrvWeight    uint16            `json:"srvweight,omitempty"`
  SrvPort      uint16            `json:"srvport,omitempty"`
  CaaTag       string            `json:"caatag,omitempty"`
  CaaFlag      uint8             `json:"caaflag,omitempty"`
  ...
}
```

It is important to leave the `omitempty` flag present so that tests for other record types do not start to fail because your new record types insist on being present.

## Step 2: Add a capability for the record

You'll need to mark which providers support this record type. The initial PR should implement this record for the `bind` provider at a minimum.

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

## Step 3: Add a helper function

Add a function to `pkg/js/helpers.js` for the new record type. This is the JavaScript file that defines `dnsconfig.js`'s functions like [`A()`](../language-reference/domain-modifiers/A.md) and [`MX()`](../language-reference/domain-modifiers/MX.md). Look at the definition of `A`, `MX` and `CAA` for good examples to use as a base.

Please add the function alphabetically with the others. Also, please run [prettier](https://github.com/prettier/prettier) on the file to ensure your code conforms to our coding standard:

```shell
npm install prettier
node_modules/.bin/prettier --write pkg/js/helpers.js
```

## Step 4: Search for `#rtype_variations`

Anywhere a `rtype` requires special handling has been marked with a comment that includes the string `#rtype_variations`. Search for this string and add your new type to this code.

## Step 5: Add a `parse_tests` test case

Add at least one test case to the `pkg/js/parse_tests` directory. Test `013-mx.js` is a very simple one and is good for cloning. See also `017-txt.js`.

Run these tests via:

```shell
pushd pkg/js/
go test ./...
popd
```

If this works, then you know the `dnsconfig.js` and `helpers.js` code is working correctly.

As you debug, if there are places that haven't been marked `#rtype_variations` that should be, add such a comment. Every time you do this, an angel gets its wings.

The tests also verify that for every "capability" there is a validation. This is explained in Step 2 (search for `TestCapabilitiesAreFiltered` or `MISSING`)

## Step 6: Add an `integrationTest` test case

Add at least one test case to the `integrationTest/integration_test.go` file. Look for `func makeTests` and add the test to the end of this list.

Each `testgroup()` is a named list of tests.

{% code title="integration_test.go" lineNumbers="true" %}
```go
testgroup("MX",
  tc("MX record", mx("@", 5, "foo.com.")),
  tc("Change MX pref", mx("@", 10, "foo.com.")),
  tc("MX record",
      mx("@", 10, "foo.com."),
      mx("@", 20, "bar.com."),
  ),
)
```
{% endcode %}

Line 1: `testgroup()` gives a name to a group of tests. It also tells the system to delete all records for this domain so that the tests begin with a blank slate.

Line 2: Each `tc()` encodes all the records of a zone. The test framework will try to do the smallest changes to bring the zone up to date. In this case, we know the zone is empty, so this will add one MX record.

Line 3: In this example, we just change one field of an existing record. To get to this configuration, the provider will have to either change the priority on an existing record, or delete the old record and insert a new one. Either way, this test case assures us that the diff'ing functionality is working properly.

If you look at the tests for `CAA`, it inserts a few records then attempts to modify each field of a record one at a time. This test was useful because it turns out we hadn't written the code to properly see a change in priority. We fixed this bug before the code made it into production.

Line 4: In this example, the next zone adds a second MX record. To get to this configuration, the provider will add an additional MX record to the same label. New tests don't need to do this kind of test because we're pretty sure that that part of the diffing engine works fine. It is here as an example.

Also notice that some tests include `requires()`, `not()` and `only()` statements. This is how we restrict tests to certain providers. These options must be listed first in a `testgroup`. More details are in the source code.

To run the integration test with the BIND provider:

```shell
pushd integrationTest
go test -v -args -verbose -profile BIND
popd
```

Once the code works for BIND, consider submitting a PR at this point. (The earlier you submit a PR, the earlier we can provide feedback.)

If you find places that haven't been marked `#rtype_variations` but should be, please add that comment. Every time you fail to do this, God kills a cute little kitten. Please do it for the kittens.

## Step 7: Support more providers

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

## Step 9: "go generate"

Re-generate the documentation:

```shell
go generate ./...
```

This will regenerate things like the table of which providers have which features and the `dnscontrol.d.ts` file.
