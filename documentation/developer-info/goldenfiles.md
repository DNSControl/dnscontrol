# Provider conversion golden files

- [Provider conversion golden files](#provider-conversion-golden-files)
  - [Recording](#recording)
  - [Files](#files)
  - [Intentional output changes](#intentional-output-changes)
  - [Mutation checks](#mutation-checks)

Provider conversion tests replay the exact calls made at the boundary between a
provider's native record type and `models.RecordConfig`. The fixtures live in
`providers/<package>/test_data` and require neither credentials nor a domain
environment variable when replayed.

## Recording

Run a known-good integration test with `-record`:

```shell
go test -failfast -run TestDNSProviders -v ./integrationTest \
  -args -verbose -profile CLOUDNS -record
```

The conversion observer is injected while the provider is constructed. Each
instrumented conversion reports its input and result. `-record` writes both as
a matched, indexed pair, so recording does not require a later `-update` step.
Only check in recordings from successful integration tests.

Use `-recorddir` to override the provider's normal `test_data` directory. A
relative path is resolved from the repository root.

{% hint style="danger" %}
Review every recorded file for credentials, private names, addresses, zone IDs,
and other sensitive data before committing it.
{% endhint %}

## Files

`test_data/meta.json` stores the fixture version and the domains and conversion
functions recorded for the provider. A provider may have more than one domain.

For native-to-RecordConfig (`ToRC`) conversions:

- `recorded_torc_input_<func>_<domain>.json`
- `expected_torc_output_<func>_<domain>.records`

For RecordConfig-to-native (`ToNative`) conversions:

- `recorded_tonative_input_<func>_<domain>.records`
- `expected_tonative_output_<func>_<domain>.json`

The provider name is omitted because the containing package already identifies
it. The domain is included because one provider can be tested against several
zones.

Every JSON value has an `index` and a `value`. Every `.records` line starts
with the same integer and a tab. Repeated indexes represent one conversion
which consumes or produces multiple records. Consequently, one-to-many and
many-to-one conversions remain synchronized even when record counts differ.

## Intentional output changes

`-record` replaces both sides of the fixture pair using a passing integration
run. `-update` is separate: it preserves recorded inputs and rewrites only the
expected outputs using the current conversion code.

```shell
go test ./providers/cloudns -update
```

Review the resulting diff. `-update` can bless current behavior, including a
bug, so use it only when an output change is intentional.

## Mutation checks

The observer snapshots both sides of each boundary. A `ToRC` conversion must
not mutate its native input, and a `ToNative` conversion must not mutate its
`RecordConfig` input. Recording reports these mutations as errors; replay tests
report them as test failures.
