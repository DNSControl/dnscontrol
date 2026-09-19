# Provider conversion golden files

Provider conversion golden tests replay the exact calls made at the boundary
between a provider's native record type and `models.RecordConfig`, and check the
result against recorded fixtures. They prove the current conversion code still
produces the expected output. Replay needs neither credentials nor a `*_DOMAIN`
variable.

Fixtures live in `providers/<pkg>/test_data`. `pkg/providergolden` records and
replays them. **`providers/cloudns` is a complete, minimal example**
(`api.go`, `cloudnsProvider.go`, `convert_golden_test.go`, `test_data/`) — copy it.

- [Provider conversion golden files](#provider-conversion-golden-files)
  - [Run the golden tests](#run-the-golden-tests)
  - [Update the expected output](#update-the-expected-output)
  - [Record new fixtures](#record-new-fixtures)
  - [Files](#files)
  - [Add golden tests to a provider](#add-golden-tests-to-a-provider)
    - [1. Instrument the provider](#1-instrument-the-provider)
    - [2. Add the replay tests](#2-add-the-replay-tests)
    - [3. Hydrate the fixtures](#3-hydrate-the-fixtures)

## Run the golden tests

Replay the fixtures to verify the current code still yields the recorded output
(and that conversions don't mutate their inputs):

```shell
go test ./providers/cloudns/
```

The `*Golden` tests read `test_data/` and compare. A mismatch is a test
failure: either the code regressed (fix it), or the output changed on purpose
(update the fixtures, next).

## Update the expected output

When a conversion's output *should* change, rewrite the expected-output files
from the current code. `-update` keeps the recorded **inputs** and rewrites only
the **expected outputs**:

```shell
go test ./providers/cloudns/ -update
```

Review the diff before committing. `-update` blesses whatever the code currently
emits — including a bug — so use it only when the change is intentional. (Flag
defined in `pkg/providergolden`.)

## Record new fixtures

To create fixtures for the first time, or to refresh both sides, replay a
known-good integration test with `-record`. Recording writes the input **and**
its resulting output as a matched pair, so no separate `-update` is needed:

```shell
go test -failfast -run TestDNSProviders -v ./integrationTest \
  -args -verbose -profile CLOUDNS -record
```

The recorder is injected while the provider is constructed; each instrumented
conversion reports its input and result. Only record from a **successful**
integration run. `-recorddir <dir>` overrides the provider's `test_data`
directory (a relative path resolves from the repo root). (Flags defined in
`integrationTest/helpers_test.go`.)

{% hint style="danger" %}
Recordings can contain credentials, private names, addresses, and zone IDs.
Review every file before committing it.
{% endhint %}

Mutation is checked here too: a `ToRC` conversion must not mutate its native
input, and a `ToNative` conversion must not mutate its `RecordConfig` input.
Recording reports mutations as errors; replay reports them as test failures.

## Files

`test_data/meta.json` holds the fixture version and, per domain, the recorded
`to_rc` and `to_native` function names (one provider may cover several domains).

Per direction and function:

- `ToRC` (native → `RecordConfig`):
  - `recorded_torc_input_<func>_<domain>.json`
  - `expected_torc_output_<func>_<domain>.records`
- `ToNative` (`RecordConfig` → native):
  - `recorded_tonative_input_<func>_<domain>.records`
  - `expected_tonative_output_<func>_<domain>.json`

The provider name is omitted (the package identifies it); the domain is
included. Each JSON value and each `.records` line carries an `index`. A
repeated index is one conversion that consumed or produced multiple records, so
one-to-many and many-to-one conversions stay synchronized even when the record
counts differ.

## Add golden tests to a provider

Three steps: instrument the conversion boundaries, add replay tests, then
hydrate the fixtures. `providers/cloudns` shows the whole pattern.

### 1. Instrument the provider

Add an observer field and setter. `CreateDNSProvider` calls
`SetConversionObserver` when the provider implements it (see
`pkg/providers/conversion_observer.go`):

```go
type cloudnsProvider struct {
	observer providers.ConversionObserver
	// ...
}

func (c *cloudnsProvider) SetConversionObserver(o providers.ConversionObserver) {
	c.observer = o
}
```

At every conversion boundary that integration tests exercise, wrap the call:
`Begin*` with the input before, `End*` with the same input plus the result and
error after. Use the conversion function's exact name as the observer name.

Native → `RecordConfig`:

```go
before := providers.BeginToRC(c.observer, "toRc", &records[i])
rc, err := toRc(dc, &records[i])
providers.EndToRC(c.observer, "toRc", before, &records[i], models.Records{rc}, err)
```

`RecordConfig` → native:

```go
input := models.Records{desired}
before := providers.BeginToNative(c.observer, "toReq", input)
req, err := toReq(desired)
providers.EndToNative(c.observer, "toReq", before, input, req, err)
```

The `providers.Begin*`/`End*` helpers are nil-safe, so instrumentation is inert
in production.

### 2. Add the replay tests

Add `convert_golden_test.go` (see `providers/cloudns/convert_golden_test.go`).
Each adapter runs one recorded call; the domain comes from `meta.json`, so no
env var is needed:

```go
func TestToRcGolden(t *testing.T) {
	providergolden.CheckToRC(t, "toRc",
		func(dc *models.DomainConfig, native domainRecord) (models.Records, error) {
			rc, err := toRc(dc, &native)
			return models.Records{rc}, err
		})
}

func TestToReqGolden(t *testing.T) {
	providergolden.CheckToNative(t, "toReq",
		func(_ *models.DomainConfig, records models.Records) (requestParams, error) {
			return toReq(records[0])
		})
}
```

`CheckToNative` passes all records sharing an index in one call, which supports
record-set conversions. `CheckRoundTrip` additionally verifies providers whose
two conversions are inverses. A function with no recording is skipped.

### 3. Hydrate the fixtures

Record once from a passing integration run, inspect (and redact) the new files,
commit them, then run the package normally to confirm replay:

```shell
go test -failfast -run TestDNSProviders -v ./integrationTest \
  -args -verbose -profile CLOUDNS -record   # writes providers/cloudns/test_data
go test ./providers/cloudns/                # replay must pass
```
