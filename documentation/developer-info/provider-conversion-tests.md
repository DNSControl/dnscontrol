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
and a directory to the command you normally use:

```shell
go test -run TestDNSProviders -timeout 1h -failfast -v ./integrationTest \
  -args -verbose -profile CLOUDFLAREAPI -record providers/cloudflare/testdata
```

That writes two files named after the profile:

- `cloudflareapi.records` — every record the tests asked the provider to store,
  which is what a `CheckToNative` function is given.
- `cloudflareapi.json` — the native record each returned record came from, read
  from `RecordConfig.Original`. That is what a `CheckToRC` function is given. It
  is written only for providers that fill `Original` in.

Rename them to match the test you are about to add, and read them before
committing: they contain whatever your zone contained during the run.

{% hint style="warning" %}
`Original` is recorded as it stands once the provider has built the zone, which
is not always what the API sent. `providers/netlify` canonicalizes a CNAME, MX
or NS value in place before storing the record in `Original`, so a recorded
native carries a trailing dot the API did not send, and a golden replayed from
it never exercises the canonicalization. Check the recorded natives against the
API's own responses when a provider's converter writes to the record it was
given.
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
func TestToRecordConfigGolden(t *testing.T) {
	providergolden.CheckToRC(t, "websupport_torecordconfig", "example.com",
		func(dc *models.DomainConfig, native nativeRecord) ([]*models.RecordConfig, error) {
			rc, err := toRecordConfig(dc, native)
			return []*models.RecordConfig{rc}, err
		})
}
```

Use `CheckToNative` for the other direction: `toNative`, `toReq`,
`recordToCreateRequest`, or whatever your provider calls it. Its input is a list
of DNS records rather than native records, so it reads a `.records` file:

```go
func TestToReqGolden(t *testing.T) {
	providergolden.CheckToNative(t, "porkbun_toreq", "example.com", toReq)
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
