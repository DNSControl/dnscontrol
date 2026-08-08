# Adding provider conversion golden tests

`pkg/providergolden` records and replays the exact conversion calls exercised by
integration tests. See [goldenfiles.md](goldenfiles.md) for the recording
workflow and file formats.

## Instrument the provider

Add a `providers.ConversionObserver` field to the provider and implement
`SetConversionObserver`. Around each conversion boundary, pass the same input
to `Begin` before conversion and `End` after conversion, together with the
result and error:

```go
before := providers.BeginToRC(p.observer, "toRc", native)
records, err := toRc(dc, native)
providers.EndToRC(p.observer, "toRc", before, native, records, err)
```

For the other direction:

```go
input := models.Records{record}
before := providers.BeginToNative(p.observer, "toReq", input)
request, err := toReq(record)
providers.EndToNative(p.observer, "toReq", before, input, request, err)
```

Observe the conversion at every call site used by integration tests. Use the
conversion function's exact name as the observer name.

## Add replay tests

Adapters receive the domain from `test_data/meta.json`; no environment variable
is needed.

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
record-set conversions. `CheckRoundTrip` can additionally verify providers
whose two conversions are compatible.

An unrecorded function is skipped. Once a passing integration run is available,
record it with `-record`, inspect the new files, and run the provider package
normally to verify replay.
