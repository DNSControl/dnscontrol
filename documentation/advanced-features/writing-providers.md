# Writing new DNS providers

- [Writing new DNS providers](#writing-new-dns-providers)
  - [Overview](#overview)
  - [Step 1: General advice](#step-1-general-advice)
  - [Step 2: Pick a base provider](#step-2-pick-a-base-provider)
  - [Step 3: Create the driver skeleton](#step-3-create-the-driver-skeleton)
  - [Step 4: Activate the driver](#step-4-activate-the-driver)
  - [Step 5: Onboarding metadata for `dnscontrol init` (optional)](#step-5-onboarding-metadata-for-dnscontrol-init-optional)
  - [Step 6: Implement the provider](#step-6-implement-the-provider)
  - [Step 7: Create `auditrecords.go`](#step-7-create-auditrecordsgo)
  - [Step 8: Unit Test](#step-8-unit-test)
  - [Step 9: Integration Test](#step-9-integration-test)
  - [Step 10: Verify TXT records](#step-10-verify-txt-records)
  - [Step 11: Update docs, CICD and other files](#step-11-update-docs-cicd-and-other-files)
  - [Step 12: Capabilities](#step-12-capabilities)
  - [Step 13: Automated code tests](#step-13-automated-code-tests)
  - [Step 14: Dependencies](#step-14-dependencies)
  - [Step 15: Update `pr_integration_tests.yml`](#step-15-update-pr_integration_testsyml)
  - [Step 16: Check your work](#step-16-check-your-work)
  - [Step 17: Submit a PR](#step-17-submit-a-pr)
  - [Step 18: After the PR is merged](#step-18-after-the-pr-is-merged)

Writing a new DNS provider is a relatively straightforward process. You essentially need to implement the [providers.DNSServiceProvider interface.](https://pkg.go.dev/github.com/DNSControl/dnscontrol/v4/pkg/providers#DNSServiceProvider) and the system takes care of the rest.

Please do note that if you submit a new provider you will be assigned bugs related to the provider in the future (unless you designate someone else as the maintainer). [More details here](../provider/index.md).

Please follow the [DNSControl Code Style Guide](styleguide-code.md) and the [DNSControl Documentation Style Guide](styleguide-doc.md).

An important concept you'll need to understand is "existing vs. desired". "Existing" is when you've gathered data that describes how things are now: i.e. a list of DNS records
we've downloaded via the API, that represents the current state of the zone. "Desired" describes how we want the world to be: i.e. the zone as described in `dnsconfig.js`.
So, for example, you'll frequently see code with two variables: `var existing, desired []*models.RecordConfig`, or variables with names that begin with `e` or `d`, such as `var ex, dx int`. A mnemonic is "D, desired, comes from dnsconfig.js".

## Overview

I'll ignore all the small stuff and get to the point.

A typical provider implements 3 methods and DNSControl takes care of the rest:

* GetZoneRecords() -- Download the list of DNS records.
* GetZoneRecordsCorrections() -- Generate a list of corrections.
* GetNameservers() -- Query the API and return the list of parent nameservers.

These three functions are all that's needed for `dnscontrol preview` and `dnscontrol push`.

The goal of `GetZoneRecords()` is to download all the DNS records for that zone, convert them to `models.RecordConfig` format, and return them as one big list (`models.Records`).

The goal of `GetZoneRecordsCorrections()` is to return a list of corrections. Each correction is a text string describing the change ("Delete CNAME record foo") and a function that, if called, will make the change (i.e. call the API and delete record foo).  `dnscontrol preview` simply prints the text strings.  `dnscontrol push` prints the strings and calls the functions. Because of how Go's functions work, the function will have everything it needs to make the change. Pretty cool, eh?

Calculating the difference between existing and desired is difficult. Luckily the work is done for you by `pkg/diff2`.  `GetZoneRecordsCorrections()` calls a a function in the `pkg/diff2` module that generates a list of changes (usually an ADD, CHANGE, or DELETE) that can easily be turned into the API calls mentioned previously.

So, what does all this mean?

It basically means that writing a provider is as simple as writing code that (1) downloads the existing records, (2) converts each records into `models.RecordConfig`, (3) write functions that perform adds, changes, and deletions.

If you are new to Go, there are plenty of providers you can copy from. In fact, many non-Go programmers [have learned Go by contributing to DNSControl](https://everythingsysadmin.com/2017/08/go-get-up-to-speed.html).

Now that you understand the general process, here are the details.

## Step 1: General advice

A provider can be a DnsProvider, a Registrar, or both. We recommend you write the DnsProvider first, release it, and then write the Registrar if needed.

If you have any questions, please discuss them in the GitHub issue related to the request for this provider.

This document is constantly being updated.  Please let us know what was confusing so we can update this document with advice for future authors (or even better send a PR! We love PRs!).

## Step 2: Pick a base provider

It's a good idea to start by copying a similar provider.

How can you tell a provider is similar?

Each DNS provider's API falls into one of 4 category. Some update one DNS record at a time. Others, the only change they permit is to upload the entire zone even if only one record changed! Others are somewhere in between: all records at a label must be updated at once, or all records in a RecordSet (the label + rType).

In summary, provider APIs basically fall into four general categories:

* Updates are done one record at a time (ByRecord)
* Updates are done one label at a time (ByLabel)
* Updates are done one label+type at a time (ByRecordSet)
* Updates require the entire zone to be uploaded (ByZone).

DNSControl provides 4 helper functions that do all the hard work for you.  As input, they take the existing zone (what was downloaded via the API) and the desired zone (what is in `dnsconfig.js`).  They return a list of instructions. Implement handlers for the instructions and DNSControl is able to perform `dnscontrol push`.

The functions are:

* [diff2.ByRecord()](https://pkg.go.dev/github.com/DNSControl/dnscontrol/v4/pkg/diff2#ByRecord) -- Updates are done one DNS record at a time. New records are added. Changes and deletes refer to an ID assigned to the record by the provider.
* [diff2.ByLabel()](https://pkg.go.dev/github.com/DNSControl/dnscontrol/v4/pkg/diff2#ByLabel) -- Updates are done for an entire label. Adds and changes are done by sending one or more records that will appear at that label (i.e. `www.example.com`). Deletes delete all records at that label.
* [diff2.ByRecordSet()](https://pkg.go.dev/github.com/DNSControl/dnscontrol/v4/pkg/diff2#ByRecordSet) -- Similar to ByLabel() but updates are done on the label+type level. If `www.example.com` has 2 A records and 2 MX records, updates must replace all the A records, or all the MX records, or add records of a different type.
* [diff2.ByZone()](https://pkg.go.dev/github.com/DNSControl/dnscontrol/v4/pkg/diff2#ByZone) -- Updates are done by uploading the entire zone every time.

To determine your provider's category, review your API documentation.

To find a similar provider to copy, look at which `diff2.By*()` function is used: `grep diff2.By providers/*.go`

The file [`pkg/diff2/diff2.go`](https://github.com/DNSControl/dnscontrol/blob/main/pkg/diff2/diff2.go) has instructions about how to use the diff2 system.

FYI: We are in the middle of converting to a new way to create `models.RecordConfig` structs. So far, BIND and CLOUDFLAREAPI are two providers that do things the new way. Old code creates RecordConfigs directly: `rc = &models.RecordConfig`. New code uses factories: `NewRecordConfig()` or `NewRecordConfigParse()`. Any new providers should use the new methods.

## Step 3: Create the driver skeleton

Create a directory for the provider called `providers/name` where `name` is all lowercase and represents the commonly-used name for the service.

The main driver should be called `providers/name/nameProvider.go`. The API abstraction is usually in a separate file (often called `api.go`).

Directory names should be consistent.  It should be all lowercase and match the ALLCAPS provider name. Avoid `_`s.

## Step 4: Activate the driver

Edit [providers/\_all/all.go](https://github.com/DNSControl/dnscontrol/blob/main/pkg/providers/_all/all.go). Add the provider list so DNSControl knows it exists.

## Step 5: Onboarding metadata for `dnscontrol init` (optional)

This step allows users of your provider to use the interactive [`dnscontrol init`](../commands/init.md) wizard.

Register a `CredsMetadata` block in the same `init()` function where you call `RegisterDomainServiceProviderType` and `RegisterMaintainer`. A simple provider reduces to a few lines:

```go
providers.RegisterCredsMetadata("MYPROVIDER", providers.CredsMetadata{
    DisplayName: "My Provider",
    Kind:        providers.KindDNS, // or providers.KindDNS | providers.KindRegistrar
    DocsURL:     "https://docs.dnscontrol.org/provider/myprovider",
    PortalURL:   "https://portal.example.com/api-tokens",
    Fields: []providers.CredsField{
        {Key: "apitoken", Label: "API Token", Required: true, Secret: true},
    },
})
```

A field can carry any of the following flags:

| Flag | Effect |
| ---- | ---- |
| `Required` | Empty answers are rejected. |
| `Secret` | Input is masked. |
| `Multiline` | Opens `$EDITOR` so PEM blocks and other multi line values can be entered intact. |
| `Choices` | Input is restricted to a fixed list. |
| `EnvVar` | When the environment variable is set, its value becomes the default. |
| `Internal` | Marks a selector whose answer only drives `ShowIf` logic. The value is not written to `creds.json`. |
| `ShowIf` | Only ask this field when earlier field answers match the given key/value map. Used to branch between auth methods. |
| `Default` | Suggested value shown in the prompt. |
| `Validator` | Custom function that rejects invalid values with an error message. |

The optional `PostWrite` hook on `CredsMetadata` lets the provider
prepare local resources after the wizard writes `creds.json` (BIND uses
this to create the zone files directory).

The BIND and TransIP registrations in this repository are worked examples maintainers can copy from:

* [`providers/bind/bindProvider.go`][bind-source]: the simple shape, plus a `PostWrite` hook that creates the zone files directory.
* [`providers/transip/transipProvider.go`][transip-source]: an auth method selector (`Internal` plus `ShowIf`) that branches between a short lived access token and an account name paired with a PEM private key.

[bind-source]: https://github.com/StackExchange/dnscontrol/blob/main/providers/bind/bindProvider.go
[transip-source]: https://github.com/StackExchange/dnscontrol/blob/main/providers/transip/transipProvider.go

Providers without registered metadata still work; users just create the
`creds.json` entry manually, using the help of the provider's documentation page.

## Step 6: Implement the provider

**If you are implementing a DNS Registrar:**

Implement all the calls in the [providers.Registrar interface](https://pkg.go.dev/github.com/DNSControl/dnscontrol/v4/pkg/providers#Registrar).

The function `GetRegistrarCorrections()` returns a list of corrections to be made. These are in the form of functions that DNSControl can call to actually make the corrections.

**If you are implementing a DNS Service Provider:**

Implement all the calls in the [providers.DNSServiceProvider interface](https://pkg.go.dev/github.com/DNSControl/dnscontrol/v4/pkg/providers#DNSServiceProvider).

* The function that converts the API's native records to `models.RecordConfig` structs should be called toRC().
* There are helper functions (factories) for creating `models.RecordConfig`'s. See [The Cookbook](developer-info/cookbook.md) "Create a `models.RecordConfig`" for details.

The function `GetDomainCorrections()` is a bit interesting. It returns a list of corrections to be made. These are in the form of functions that DNSControl can call to actually make the corrections.

* The "create" function will probably need to convert an `models.RecordConfig` to the native API struct. Please name this function toNative()
* To access or change the RDATA in a field, use `rd := rc.GetRDATA()` and `rc.SetRDATA(rd)`. Full details are in [The Cookbook](developer-info/cookbook.md) "Getters/Setters for RDATA in `models.RecordConfig`".
* Of course, if you need to create a `models.RecordConfig` please use the factories listed above.

The remaining steps assume you're creating a DNS Service Provider.

## Step 7: Create `auditrecords.go`

The `auditrecords.go` file lists special cases that a provider doesn't support.

For example, the `AXFRDDNS` provider doesn't support empty TXT records.  `providers/axfrddns/auditrecords.go` records that fact. The integration test system will skip any tests with empty TXT records.

Use an empty list for now (copy `providers/bind/auditrecords.go`). When integration testing, you'll add to it.

## Step 8: Unit Test

ProTip: Unit tests tests functions in isolation ("just the function").

Add unit tests for any complex algorithms in the new code. Good examples include: custom parsers, complex string manipulation functions, the toRC() function (if it is complex).

Run the unit tests with this command:

```shell
    go test ./...
```

## Step 9: Integration Test

ProTip: Integration tests are tests that verify the parts of the system are working together as expected.

Integration testing is the most important kind of testing when adding a new provider. Integration tests run add/change/delete operations on a real domain. You'll need to set up an account and add a domain (technically a "zone") to the account.

{% hint style="danger" %}
All records will be deleted from the test domain!  Use a OTE domain or a real domain that isn't otherwise in use and can be destroyed.
{% endhint %}

* Edit [integrationTest/profiles.json](https://github.com/DNSControl/dnscontrol/blob/main/integrationTest/profiles.json):
  * Add the `creds.json` info required for this provider in the form of environment variables.

Now you can run the integration tests.

For example, test BIND:

```shell
cd integrationTest
export BIND_DOMAIN='example.com'
go test -v -args -verbose -profile BIND
```

(BIND is a good place to start since it doesn't require API keys.)

This will run the tests on Amazon AWS Route53:

```shell
export R53_DOMAIN='dnscontroltest-r53.com'    # Use a test domain.
export R53_KEY_ID='REDACTED_ID'
export R53_KEY='REDACTED_KEY'
cd integrationTest
go test -v -args -verbose -profile ROUTE53
```

Some useful `go test` tips:

* Flags before `-args` go to the `go test` program. Flags after `-args` go to DNSControl.
* Run only certain tests using the `-start` and `-end` flags.
  * Rather than running all the tests, run just the tests you want.
  * These flags must be *after* the `-args` flag.
  * Example: `go test -v -args -verbose -profile ROUTE53 -start 10 -end 20` run tests 10-20 inclusive.
  * Example: `go test -v -args -verbose -profile ROUTE53 -start 5 -end 5` runs only test 5.
  * Example: `go test -v -args -verbose -profile ROUTE53 -start 20` skip the first 19 tests.
  * Example: `go test -v -args -verbose -profile ROUTE53 -end 20` only run the first 20 tests.
* Slow tests? Add `-timeout n` to increase the timeout for tests
  * `go test` kills the tests after 10 minutes by default.  Some providers need more time.
  * This flag must be *before* the `-args` flag.
  * Example:  `go test -timeout 20m -v -args -verbose -profile CLOUDFLAREAPI`

You can opt out of tests if they will always fail for that provider. For example, if it doesn't support a particular feature. Look at `func makeTests()` in [integrationTest/integration_test.go](https://github.com/DNSControl/dnscontrol/blob/2f65533e1b92c2967229a92a304fff7c14f7f4b6/integrationTest/integration_test.go#L675) for more details.

You can also opt-out of special edge cases (like not supporting a "null MX") by adding to the `auditrecoreds.go` file.

The [Integration Testing](../developer-info/integration-tests) page has more tips.

## Step 10: Verify TXT records

There is a potential bug in how TXT records are handled. Sadly we haven't found an automated way to test for this bug.  The manual steps are here in [documentation/testing-txt-records.md](testing-txt-records.md)

## Step 11: Update docs, CICD and other files

* Edit `documentation/providers.md`:
  * Remove the provider from the `Requested providers` list (near the end of the doc) (if needed).
  * Add the new provider to the [Providers with "contributor support"](../provider/index.md#providers-with-contributor-support) section.
* Edit `README.md`:
  * Add the provider to the list.
* Edit `documentation/SUMMARY.md`:
  * Add the provider to the "Providers" list.
* Create `documentation/provider/PROVIDERNAME.md`:
  * Use one of the other files in that directory as a base.

{% hint style="success" %}
**Need feedback?** Submit a draft PR!  It's a great way to get early feedback, ask about fixing
a particular integration test, or request feedback.
{% endhint %}

## Step 12: Capabilities

Some DNS providers have features that others do not.  For example some support the SRV record.  A provider announces what it can do using the capabilities system.

If a provider doesn't advertise a particular capability, the integration test system skips the appropriate tests.  Therefore you might want to initially develop the provider with no particular capabilities advertised and code until all the integration tests work.  Then enable capabilities one at a time to finish off the project.

Don't feel obligated to implement everything at once. In fact, we'd prefer a few small PRs than one big one. Focus on getting the basic provider working well before adding these extras.

Operational features have names like `providers.CanUseSRV` and `providers.CanUseAlias`.  The list of optional "capabilities" are in the file `dnscontrol/pkg/providers/providers.go` (look for `CanUseAlias`).

Capabilities are processed early by DNSControl.  For example if a provider doesn't support SRV records, DNSControl will error out when parsing `dnscontrol.js` rather than waiting until the API fails at the very end.

Enable optional capabilities in the `nameProvider.go` file and run the integration tests to see what works and what doesn't.  Fix any bugs and repeat, repeat, repeat until you have all the capabilities you want to implement.

FYI: If a provider's capabilities changes, run `go generate` to update the documentation.

## Step 13: Automated code tests

We use a number of automated code-checking systems. Please run your code through all of them and fix all warnings and errors.  Some of the automated fixes may not alway sbe perfect. Therefore, it is best to commit your code before running these and verify that you agree with the changes.

Modernize your code:

```shell
go run golang.org/x/tools/go/analysis/passes/modernize/cmd/modernize@latest -fix ./...
```

Vet the code:

```shell
go vet ./...
```

Use golangci-lint: (install [golangci-lint](https://golangci-lint.run/docs/welcome/install/local/))

```shell
golangci-lint run ./...
staticcheck ./...
```

Use staticcheck:

```shell
go install honnef.co/go/tools/cmd/staticcheck@latest
staticcheck ./...
```

Commit any changes.

## Step 14: Dependencies

See [documentation/release-engineering.md](../release/release-engineering.md) for tips about managing modules and checking for outdated dependencies.

## Step 15: Update `pr_integration_tests.yml`

Edit `.github/workflows/pr_integration_tests.yml`

* Add your providers `_DOMAIN` env variable:
  * Add it to the `env` section of `integration-tests`.
  * Please keep this list sorted alphabetically.
  * To find this section, search for `PROVIDER SECRET LIST`.

For example, the entry for BIND looks like:

{% code title=".github/workflows/pr_integration_tests.yml" %}

```yaml
        BIND_DOMAIN: ${{ vars.BIND_DOMAIN }}
```

{% endcode %}

* Add your providers other ENV variables:
  * Every provider requires different variables set to perform the integration tests.  The list of such variables is in `integrationTest/profiles.json`.
  * You've already added `*_DOMAIN` to `pr_integration_tests.yml`. Now we're going to add the remaining ones.

To find this section, search for `PROVIDER SECRET LIST`.

For example, the entry for CLOUDFLAREAPI looks like this:

{% code title=".github/workflows/pr_integration_tests.yml" %}

```yaml
        CLOUDFLAREAPI_ACCOUNTID: ${{ secrets.CLOUDFLAREAPI_ACCOUNTID }}
        CLOUDFLAREAPI_TOKEN: ${{ secrets.CLOUDFLAREAPI_TOKEN }}
```

{% endcode %}

## Step 16: Check your work

You're almost done!

These are the things we'll be checking when you submit the PR.  Please try to complete all or as many of these as possible.

1. Run `go generate ./...` to make sure all generated files are fresh.
2. Make sure the following files were created and/or updated:

* `.github/CODEOWNERS`
* `README.md`
* `.github/workflows/pr_integration_tests.yml` (env variables for your provider)
* `documentation/SUMMARY.md`
* `documentation/provider/index.md` (the autogenerated table + the second one; make sure it is removed from the `requested` list)
* `documentation/provider/`PROVIDERNAME`.md`
* `integrationTest/profiles.json`
* `pkg/providers/_all/all.go`

3. Review the code for style issues, remove debug statements, make sure all exported functions have a comment, and generally tighten up the code.
4. Verify you're using the most recent version of anything you import.  (See [Step 14](#step-14-dependencies))
5. Re-run the [integration test](#step-9-integration-test) one last time.

* Post the results as a comment to your PR.

6. Re-read the [maintainer's responsibilities](../provider/index.md#providers-with-contributor-support) bullet list.  By submitting a provider you agree to maintain it, respond to bugs, periodically re-run the integration test to verify nothing has broken, and if we don't hear from you for 2 months we may disable the provider.

## Step 17: Submit a PR

At this point you can submit a PR.

The PR should include the sentence: "Please create the GitHub label 'provider-PROVIDERNAME'" (change `PROVIDERNAME` to the name of your provider.)  This is

Actually you can submit the PR earlier if you just want feedback, or have questions.  However if you haven't submitted a PR by now, this is the time to do it.

## Step 18: After the PR is merged

1. Close any related GitHub issues.
2. Would you like your provider to be tested automatically as part of every PR?  Sure you would!  Follow the instructions in [Bring-Your-Own-Secrets for automated testing](byo-secrets.md)
