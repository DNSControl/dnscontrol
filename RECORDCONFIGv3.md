
DELETE THIS BEFORE THE RELEASE

### Release Notes:

* Replacing github.com/miekg/dns (v1) with codeberg.org/miekg/dns (v2).
* RecordConfigV3: We're adopting codeberg.org/miekg/dns in a big way!  It's now the prefered way to store a DNS record.  RecordConfig.RDATA stores a dns.RDATA interface, which can contain any DNS record type, even unofficial ones. This should save memory in the long run, but at this time it means storing a lot of data twice until legacy code is upgraded.

### Extra testing needed:

* BIND: SOA handling has been rewritten to be easier to debug and more reliable. Shouldn't have any user-visible changes but please be on the lookout.

### Developer notes

* There is now a factory for `models.DomainConfig{}` called `models.NewDomainConfig(name)`. This is now the only supported way to make a new `DomainConfig`.

### Potential breaking changes

### TODOs:
* Fix HTTPS on SAKURACLOUD
* Port Cloudflare Single Redirects
* Port RP
* Handle `ech=IGNORE` when it is ech=0000
* PTR magic should be implemented in pkg/js??

### miekg requests

* `pkg/txtutil/{miekg.go miekg_test.go ddd/ddd.go}` come from dnsv2.  Upstream changes?



# dnscontrol v5

DNSControl v5 will adopt the miekg/dns version 2 ("dnsv2") module as the native way it stores DNS records. That is, the models.RecordConfig{} struct will include a pointer to
a dnsv2's RDATA struct (technically its "a reference to an interface").  This will lead
to the eventual removal of many fields in RecordConfig, such as DsKeyTag,           DsAlgorithm,        DsDigestType,       and DsDigest           , which hog memory while doing nothing if the record is not a DS record.

DNSControl v5 will also adopt unified way to create and access RecordConfigs. Currently there is no standard, therefore some places use models.&RecordConfg{} (they create the struct), others use rtypecontrol.NewRecordConfigFromStruct (the first attempt to redo RecordConfig), or various other ways.


## The one true way to create a models.RecordConfig{}

A provider typically uses an API to download all records for a particular zone or domain.
The provider converts these to RecordConfigs, for use by the rest of the system. This
function is usually called nativeToRC() or just toRC().

func NewRecordConfig(origin string, name string, ttl uint32, typeAny any, args ...any) (*RecordConfig, error) {
  * origin: 
  * name: 
  * ttl:
  * typeAny: the numeric or string. dnsv2.TypeX, privatetypes.TypeY. Please use numeric when possible.
  * args: 

func NewRecordConfigParse(origin string, name string, ttl uint32, typeAny any, data string) (*RecordConfig, error) {

You can save a little typing by 
dc.AddRecord(name, ttl, typeAny, args...)

## One true way to access the fields of models.RecordConfig()

    rc.DATA().(dnsv2.MX).Preference

## Generate



