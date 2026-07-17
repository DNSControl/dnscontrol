# How to "Modernize" a provider


"Modernize" means adopting the new RecordConfig v3 structs, factories, etc.

## What work do you need to do?

This script will find any old code that needs to be updated:

```shell
$ cd providers/azuredns
$ ../../bin/is_modern.sh   
$ ../../bin/is_modern.sh
========== DomainConfig{
========== RecordConfig{
ovhProvider.go:	rec := &models.RecordConfig{
========== PopulateFromString{
ovhProvider.go:	if err := rec.PopulateFromString(rtype, r.Target, origin); err != nil {
========== SetTarget
./protocol.go:			Target:    rc.GetTargetCombined(),
./protocol.go:			Target:    rc.GetTargetCombined(),
========== GetTarget
```

## Dev tips

ProTip: Run the intergration tests after each small change. That
way if something breaks, you know it was the most recent change.

ProTip: Sometimes its useful to make the change in two steps. First, write
code that calculates something the old and new ways, compare the results
to make sure they're the same.

Original code:

```
rec.Metadata[metaOriginalIP] = rec.GetTargetField()
```

Verify our new code is correct:

```
tryOld := rec.GetTargetField()
tryNew := rec.GetTargetIP().String()
if tryOld != tryNew {
    panic("houston, we have a problem")
}
rec.Metadata[metaOriginalIP] = tryOld
```

Adopt the new code:

```
rec.Metadata[metaOriginalIP] = rec.GetTargetIP().String()
```

## Step 1: Adopt `models.NewDomainConfig()`

Change any `DomainConfig{}` to the new factory:

OLD:

```
dc := &DomainConfig{}
```

NEW:

```
dc := models.NewDomainConfig(zoneName)
```

## Step 2. Adopt `models.NewRecordConfig()`

Change any `RecordConfig{}` to the new factory:

OLD:

```
rc := &RecordConfig{}
```

NEW:

```
rc := dc.NewRecordConfig(...)            // Typical
or
rc := dc.NewRecordConfigParse(...)       // Replaces PopulateFromString()
```

If `dc` doesn't exist in this function, add it to the signature. Most functions
pass `origin` or `domain` as a string. Change that to `dc *modelsDomainConfig`.

OLD:

```go
func (a *azurednsProvider) getExistingRecords(domain string) (models.Records, []*adns.RecordSet, string, error) {
...
```

NEW:

```go
func (a *azurednsProvider) getExistingRecords(dc *models.DomainConfig) (models.Records, []*adns.RecordSet, string, error) {
    domain := dc.Name

...

    rc := dc.NewRecordConfig(...
```

You may have to follow the function calling chain a few levels.  If you use VS
Code or other editor that warns you of errors, change the signature of the
function, wait for VS Code to report errors in the callers. Fix those.  If
those don't have `dc`, change their signatures. Keep working you way up the
chain.

## Step 3. Remove obsolete setters

`PopulateFromString()` can be replaced by `dc.NewRecordConfigParse(...)`.

Since `NewRecordConfigParse()` defaults to a `txtutil.ParseQuoted`-compatible TXT parser,
`PopulateFromStringFunc(... , contents, txtutil.ParseQuoted)` can be replaced by:

```
    rc, err := dc.NewRecordConfigParse(LABEL, TTL, rType, contents)
    if err != nil { whatever }
```

If you use something other that `txtutil.ParseQuoted`, then `PopulateFromStringFunc(... , contents, MyFunc)` can be replaced by:

```
switch rType {
case "TXT":
    t := MyFunc(contents)
    rc, err := dc.NewRecordConfig(LABEL, TTL, dnsv2.TypeTXT, t)
default:
    rc, err := dc.NewRecordConfigParse(LABEL, TTL, rType, contents)
}
if err != nil { whatever }


## Step 4. Remove obsolete setters

`rc.GetTargetCombined()` is now `rc.GetRDATA().String()`

`rc.GetTargetRFC1035Quoted()` is now `rc.GetRDATA().String()`

`rc.GetTargetCombinedFunc(..., txtutil.EncodeQuoted)` is now `rc.GetRDATA().String()`

`rc.GetTargetCombinedFunc(..., MyFunc)` is now:

NEW:

```
t := MyFunc(rc.GetTargetTXTJoined())
or maybe
t := MyFunc(rc.GetTargetTXTSegmented())
```


## Step 5. Remove obsolete getters

OLD:

`SetTargetCAA()`
`SetTargetCAAStrings()`
`SetTargetCAAString()`
`SetTargetDNSKEYString()`
`SetTargetDSString()`
`SetTargetLOCString()`
`SetTargetMX()`
`SetTargetMXString()`
`SetTargetNAPTR()`
`SetTargetNAPTRString()`
`SetTargetSMIMEA()`
`SetTargetSOA()`
`SetTargetSRV()`
`SetTargetSRVPriorityString()`
`SetTargetSRVString()`
`SetTargetSSHFP()`
`SetTargetSSHFPStrings()`
`SetTargetSSHFPString()`
`SetTargetSVCBString()`
`SetTargetTLSA()`
`SetTargetTLSAString()`

In general these are no longer needed if you use `NewRecordConfig*()` functions.

However, if you want to alter an existing RC...

```go
rd := rc.GetRDATA()     // Get the generic RDATA
rdmx := rd.(dnsv2.MX)   // Cast to the MX struct 
rdmx.Preference = 999   // Alter a field.
rc.SetRDATA(rdmx)       // Save it back.
```

See the cookbook for more details.
