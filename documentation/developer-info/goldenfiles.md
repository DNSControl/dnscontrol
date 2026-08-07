# Golden Files

Full testing of DNSControl requires talking to the provider's API.  The tests
take a long time to run. In some cases, the project doesn't have access to the
provider and each test must be done by reaching out to a volunteer who does
have access.

TODO(tlim): The AI Slop version of this is provider-conversion-tests.md. Take
what's useful and delete the rest.

A golden file records that API calls made during Integration Tests so that we
may replay them any time, without access to the actual API.

## How to record API calls for later replay (golden files)

1. Set credentials

export FOO_BAR=baz

2. Record the API calls performed during Integration Tests

```shell
go test -failfast -run TestDNSProviders -v ./integrationTest  -args -verbose -profile CLOUDNS -record
```

This updates `providers/cloudns/testdata/cloudns.json`

`-failfast` is used to make any failures more apparent.

Only check in data from successful tests.

{% hint style="danger" %}
**Review any files for PII!**  Do not check in any files that include API
credentials, secrets, or anything private. In theory no such information is
intentionally collected but human verification is important!
{% endhint %}

3. Update the test fixtures

```shell
go test ./providers/cloudns  -update
```

This updates `providers/cloudns/testdata/cloudns_torc.golden` (each provider is different,
some will create multiple files.

Done

## How to enable "golden files" for a provider

We'll use Digital Ocean as an example.

Create a file called `providers/digitalocean/convert_golden_test.go`

Tell the tests this provider's name.

This .Domain function should probably be renamed EnvPrefix.

```go
var testDomain = providergolden.Domain("DIGITALOCEAN")
```

Each provider has 2 functions that can be tested: the ToRC and ToNative

Some providers name them differently, and only have one or the other.

Add a `Test${FUNCTION}Golden` test for each function you wish to verify.

Here's an example of testing the toRC function:

```go
func TestToRcGolden(t *testing.T) {
        providergolden.CheckToRC(t, "digitalocean_torc", testDomain,
                func(dc *models.DomainConfig, native godo.DomainRecord) ([]*models.RecordConfig, error) {
                // ^^^ match the signature of the function ^^^
                        rc, err := toRc(dc, &native)
//                                 ^^^  -- the function name the provider uses
                        return []*models.RecordConfig{rc}, err
                })
}
```

Here's an example of testing the toNative function:

```go
func TestToReqGolden(t *testing.T) {
        providergolden.CheckToNative(t, "digitalocean_toreq", testDomain,
                func(rc *models.RecordConfig) (*godo.DomainRecordEditRequest, error) {
                // ^^^ match the signature of the function ^^^
                        return toReq(rc), nil
                               ^^^^  -- the function name the provider uses
                })
}
```

If both ToRC and ToNative are tested, you add round-trip testing:

```go
func TestConversionRoundTrip(t *testing.T) {
       providergolden.CheckRoundTrip(t, "example.com",
               func(rc *models.RecordConfig) (*domainRecord, error) {
                       return toReq("zone-1", rc)
               },
               func(dc *models.DomainConfig, native *domainRecord) ([]*models.RecordConfig, error) {
                       rc, err := toRc(dc, native)
                       return []*models.RecordConfig{rc}, err
               })
}
```
