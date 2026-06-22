# RECORDONFIG Version 3

DELETE THIS BEFORE THE RELEASE

## Release Notes

* Replacing github.com/miekg/dns (v1) with codeberg.org/miekg/dns (v2).
* RecordConfigV3: We're adopting codeberg.org/miekg/dns in a big way!  It's now the prefered way to store a DNS record.  RecordConfig.RDATA stores a dns.RDATA interface, which can contain any DNS record type, even unofficial ones. This should save memory in the long run, but at this time it means storing a lot of data twice until legacy code is upgraded.

## Extra testing needed

* BIND: SOA handling has been rewritten to be easier to debug and more reliable. Shouldn't have any user-visible changes but please be on the lookout.
* CLOUDFLAREAPI: CF_WORKER_ROUTES() need extra testing. Internally they were represented sometimes as `WORKER_ROUTE` and sometimes as `CF_WORKER_ROUTE`. It's amazing such complex code ever worked.  Now we use `CF_WORKER_ROUTE` exclusively. The changes were core to the worker feature. Pleaes give extra attending and testing.

## Developer notes

## Potential breaking changes

## TODOs

* Fix HTTPS on SAKURACLOUD
* Port Cloudflare Single Redirects
* Port RP
* Handle `ech=IGNORE` when it is ech=0000
* PTR magic should be implemented in pkg/js??

## miekg requests

* `pkg/txtutil/{miekg.go miekg_test.go ddd/ddd.go}` come from dnsv2.  Upstream changes?
* dnsutil.Trim() -- Returning "" for z longer than s is unintuitive. I would have expected it to return s. I was going to check the output for "" to detect this, but "" also means "s == z".  I wrote txtutil.StripZone() which is more accepting of its inputs.

## dnscontrol v5

DNSControl v5 will adopt the miekg/dns version 2 ("dnsv2") module as the native way it stores DNS records. That is, the models.RecordConfig{} struct will include a pointer to
a dnsv2's RDATA struct (technically its "a reference to an interface").  This will lead
to the eventual removal of many fields in RecordConfig, such as DsKeyTag,           DsAlgorithm,        DsDigestType,       and DsDigest           , which hog memory while doing nothing if the record is not a DS record.

DNSControl v5 will also adopt unified way to create and access RecordConfigs. Currently there is no standard, therefore some places use models.&RecordConfg{} (they create the struct), others use rtypecontrol.NewRecordConfigFromStruct (the first attempt to redo RecordConfig), or various other ways.

## The one true way to create a models.RecordConfig{}

A provider typically uses an API to download all records for a particular zone or domain.
The provider converts these to RecordConfigs, for use by the rest of the system. This
function is usually called nativeToRC() or just toRC().

```go
func NewRecordConfig(origin string, name string, ttl uint32, typeAny any, args ...any) (*RecordConfig, error) {
  * origin: 
  * name: 
  * ttl:
  * typeAny: the numeric or string. dnsv2.TypeX, privatetypes.TypeY. Please use numeric when possible.
  * args: 
```

```go
func NewRecordConfigParse(origin string, name string, ttl uint32, typeAny any, data string) (*RecordConfig, error) {
```

You can save a little typing by

```go
dc.AddRecord(name, ttl, typeAny, args...)
```

func NewRecordConfigFromDnsconfigjs(origin string, name string, ttl uint32, typeAny any, data string) (*RecordConfig, error) {
// Does extra checking.

## One true way to access the fields of models.RecordConfig()

```go
    rc.DATA().(dnsv2.MX).Preference
```

## Generate

API:
label, err = dc.LabelFromShort() or dc.LabelFromFQDNNoDot()                 record_label.go
rc, err := dc.NewRecordConfig(label, ttl, type, args)  or ...Parse()        record_new.go
dc.AddRecordConfig(rc)                                                      domain.go

DNSCONFIG:
label, err = dc.LabelFromDnsconfigjs()                                      record_label.go
rc, err := dc.NewRecordConfig(label, ttl, type, args)                      record_new.go
Add metadata
dc.AddRecordConfig(rc)                                                      domain.go

TESTS: testutils_test.go
rc := models.MakeTestRC(label, ttl, type, args)                             record_helpers_test.go
rc := models.MakeTestRCParse(label, ttl, type, args)                        record_helpers_test.go
If you need many...
dc, err := models.NewDomainConfig(zone)                                     domain.go
dc.AddRecordConfig(models.MakeTestRC(label, ttl, type, args))               domain.go
dc.AddRecordConfig(models.MakeTestRCParse(label, ttl, type, args))          domain.go

### How to add a standard type:

* models/makers.go: Add a Make$TYPENAME
* models/makers.go: Add to the func init().
* models/fixhack.go: Add to the switch.
* integrationTest/helpers_integration_test.go: Add a typename() function
* integrationTest/integration_test.go: Add tests that create the type, changes each field indiviually.
* pkg/js/helpers.js: Add to list at the end.

### How to add a non-standard type.

* Pick a unique id: 
  - grep codepoint pkg/privatetypes/types_generate.yaml|sort | tail -3
* Add the custom type to types_generator.yaml and run:
  - go run types_generator.go && goimports -w *.go */*.go
* Add it to models/backfill.go

### How to add a Builder

* abc

Thoughts:

ALSO:
Let's get rid of the old fields in RecordConfig faster!
NameRaw
NameFQDNRaw
Comparable
F
ZonefilePartial
       Maybe these too:
        R53Alias           map[string]string `json:"r53_alias,omitempty"`
        AzureAlias         map[string]string `json:"azure_alias,omitempty"`
        AnswerType         string            `json:"answer_type,omitempty"`
