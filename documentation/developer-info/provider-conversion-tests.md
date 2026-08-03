# Provider conversion tests

Most providers convert between their API's native record format and
`models.RecordConfig`. Those conversion functions are only exercised by the
integration tests, which need credentials and a test zone. When a provider's
author becomes unavailable, nobody can run them.

`pkg/providergolden` replays recorded data through those functions and compares
the result with a golden file. The recorded data is checked in, so the tests run
in `go test ./...` with no credentials and no network.

Providers are enrolled one at a time. A provider with no recorded data is
skipped, never failed.

## Enrolling a provider

### 1. Record the data

The integration tests already drive every conversion a provider has, so the
easiest way to collect the data is to record an integration run. Add `-record`
to the command you normally use:

```shell
go test -run TestDNSProviders -timeout 1h -failfast -v ./integrationTest \
  -args -verbose -profile CLOUDFLAREAPI -record
```

The recording goes to `providers/<package>/testdata`, the testdata directory of
the package the provider under test is implemented in, so `CLOUDFLAREAPI` writes
to `providers/cloudflare/testdata`. Use `-recorddir` to write somewhere else.
`go test` runs a test binary in its own package directory, so a relative
`-recorddir` is relative to `integrationTest/` and needs a `../` prefix:

```shell
go test -run TestDNSProviders -timeout 1h -failfast -v ./integrationTest \
  -args -verbose -profile CLOUDFLAREAPI -recorddir ../providers/cloudflare/testdata
```

Either way two files are written, named after the profile:

- `cloudflareapi.records` — every record the tests asked the provider to store,
  which is what a `CheckToNative` function is given.
- `cloudflareapi.json` — the native record each returned record came from, read
  from `RecordConfig.Original`. That is what a `CheckToRC` function is given. It
  is written only for providers that fill `Original` in.

Rename them to match the test you are about to add, and read them before
committing: they contain whatever your zone contained during the run.

{% hint style="warning" %}
The domain the test passes to `CheckToRC` and `CheckToNative` has to be the zone
the data was recorded against, and nothing enforces it. `providergolden.Domain`
takes it from the same `<PROVIDER>_DOMAIN` variable the integration tests use, so
a recording and the test that replays it agree as long as that variable holds the
zone the data came from. Where they disagree, native records carry labels as the
API returned them, so a fixture recorded against `realzone.net` replayed by a
test that says `example.com` produces a golden whose labels are still fully
qualified:

```
www.realzone.net 300 IN A 192.0.2.1
realzone.net 3600 IN MX 10 mail.example.org.
```

`LabelFromFQDNNoDot` and its siblings print `ERROR: ... called WRONG` when this
happens, but they return the name lowercased rather than shortened and the test
passes, so `-update` writes that golden and it becomes the baseline. Set the
domain before recording the golden, not after.

A golden recorded against a zone other than `example.com` only matches while
`<PROVIDER>_DOMAIN` still names that zone, so it does not match in a checkout
that does not set it. Record from `example.com` when the golden is to be
committed.
{% endhint %}

{% hint style="warning" %}
`Original` is recorded as it stands once the provider has built the zone, which
is not always what the API sent. A converter that canonicalizes a value in place
before storing the record in `Original` records a native that already carries
the canonicalization, and a golden replayed from it never exercises the line
that applies it. Check the recorded natives against the API's own responses when
a provider's converter writes to the record it was given.
{% endhint %}

Without `-record` nothing is collected, no file is written and the provider is
not wrapped, so a normal run is unchanged. Under `-record` the provider is
wrapped in a `models.DNSProvider`, which is all the integration tests ask of it
today. A test that type-asserts a provider to an optional interface such as
`ZoneCreator` would need the wrapper to forward that interface too.

Data can also be collected by hand. The `.json` file is a JSON array of the
native records your provider's API returns:

```json
[
  {
    "content": "192.0.2.1",
    "id": 42,
    "name": "www.example.com",
    "ttl": 3600,
    "type": "A"
  }
]
```

{% hint style="warning" %}
Record from a throwaway zone. Native records carry labels, addresses, and record
and zone IDs verbatim, which for a LAN appliance means your internal DNS ends up
in a public repository.
{% endhint %}

### 2. Add the test

The test adapts your conversion function to a uniform signature. That adapter is
the only code you write:

```go
var testDomain = providergolden.Domain("WEBSUPPORT")

func TestToRecordConfigGolden(t *testing.T) {
	providergolden.CheckToRC(t, "websupport_torecordconfig", testDomain,
		func(dc *models.DomainConfig, native nativeRecord) ([]*models.RecordConfig, error) {
			rc, err := toRecordConfig(dc, native)
			return []*models.RecordConfig{rc}, err
		})
}
```

`providergolden.Domain` returns `$WEBSUPPORT_DOMAIN`, or `example.com` when that
is unset, so the same test replays a recording of your own zone and the
committed fixtures.

Use `CheckToNative` for the other direction: `toNative`, `toReq`,
`recordToCreateRequest`, or whatever your provider calls it. Its input is a list
of DNS records rather than native records, so it reads a `.records` file:

```go
func TestToReqGolden(t *testing.T) {
	providergolden.CheckToNative(t, "porkbun_toreq", testDomain, toReq)
}
```

The `.golden` file produced by `CheckToRC` is a valid `.records` file, so the
usual way to write one is to copy it and add whatever else you want covered.

### 3. Generate the golden files

```shell
go test ./providers/websupport/ -update
```

Read the generated files before committing them. `-update` records whatever the
code does today, including bugs.

## The golden format

One line per record: the label, TTL, class, type and RDATA, followed by the
record's metadata when it has any:

```
www 300 IN A 192.0.2.1
@ 3600 IN MX 10 mail.example.com.
fwd 0 IN URL https://example.net/landing ; includePath="no" type="temporary" wildcard="no"
```

`CheckToNative` writes its golden as JSON, since a native record has no
zonefile representation.

## Updating

Run `go test ./providers/<provider>/ -update` after an intentional change and
review the diff. `-update` rewrites only the golden files: the recorded inputs
are never touched, so updating a golden never needs an API key.

## What these tests do and do not catch

They pin a provider's conversion functions against their own past behaviour, so
they catch a refactor that changes what a provider sends or understands. They do
not catch a conversion that has been wrong since the day it was written, and
they say nothing about the correction and apply path. That is still worth having
for a provider whose integration tests nobody can run.
