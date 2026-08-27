# Refactoring

Useful refactoring projects. Please feel free to pick up any of these.


## Code that can probably be deleted

* RegisterCustomRecordType()/GetCustomRecordType() is no longer needed. Remove.

* I don't think the metadata "orig_custom_type" is used any more. We store to it but don't use it.

## Rewrites needed

* PTR() "magic" should be reworked as a builder called PTR(). It will be much more
cleaner and more testable. Plus it will consolidate the code into one place instead
of being some in LabelFromDnsconfigjs() and other places.

* PopulateNamesFromRaw() and MakeDomainNameVarieties() seem like a silly design.  Maybe move them to pkg/nameutil, or maybe models (since that's where makeLabelNameUnicode() is)

* All the LOC() builders should be in Go, not helpers.js.

* There are multiple ways that code strips the trailing dot from a domain name.  Benchmark a few and standardize on the fastest. In pkg/nameutil. Possibly have one that is extra fast but panics if there is no "." at the end.

* pkg/normalize/validate.go has half of the Transform code, the other half is in pkg/transform.  An easy thing would be to move the part of pkg/normalize/validate.go and validate_test.go into pkg/normalize/transform{,_test}.go.  The more difficult thing would be to move all or most of the code to pkg/transform.

* models/record.go FixPostion has an easy TODO.

* models/t_txt.go has an some easy FIXMEs.

## Bad decisions to reverse

* Providers, not Registrars + DNS Service Providers.  It should be possible to make a PROVIDER() function that returns
something that is both a Reg and a DSP.

* mustbe.TargetHost() accepts "." as a special flag. This can probably be replaced by nrc.Flags{}

* models.SetLabel() should be removed (it's only used in a few places) or enhanced to also create the .NameUnicode/.NameFQDNUnicode fields. Then de-duplicate the code that does this for NewRecordConfig().

* deSEC: BUG: It mutates RecordConfigs, which will create problems if the same zone is sent to multiple providers.


## Actual new features:

* Handle "unknown types". dnsv2 has a way to managing unknown types. Investigate it and replace the half-written version (see rc.UnknownTypeName)

* pkg/diff2/analyze.go: Smarter diffs. Diffs could just show which fields changed. For example, if just the preference of an MX record changes, show "old -> new" just for the priority.
