# Refactoring

Useful refactoring projects. Please feel free to pick up any of these.


## Documentation updates

* Improved "how to write a provider" docs. The process for creating a DNS or Registrar provider could be improved. We should provide templates to copy instead of asking people to find a similar provider.  The doc should start by creating a generic provider, then add features and options over time.  The first thing should test credentials and nothing else.  Then add a registrar (if needed) and then the "preview" functionality, then "push".

## Rewrites needed

* Rework how metadata is passed to providers and how provider state is owned.
  Consider an explicit split between state specific to a `credEntry` (such as
  account-wide caches and rate limiters) and state specific to a
  `(domain, credEntry)` (such as domain-specific configuration). Revisit client
  initialization and reuse as part of that design. Also consider a tiered
  metadata system with explicit inheritance and precedence rules, or providers
  implemented as external binaries that maintain their own state. Keep this as
  a separate future project with appropriate provider testing, rather than
  expanding the current configuration-syntax rollout.

* Future provider concurrency change: default to allowing concurrency and replace
  authored `CanConcur` declarations with `ConcurBroken: true` only for providers
  that cannot run concurrently. Derive the compatibility `CanConcur` capability
  as `!ConcurBroken`, and audit existing `Cannot`/`Unimplemented` notes to
  distinguish known limitations from untested providers. Implement in a separate PR.

* PopulateNamesFromRaw() and MakeDomainNameVarieties() seem like a silly design.  Maybe move them to pkg/nameutil, or maybe models (since that's where makeLabelNameUnicode() is)

* All the LOC() builders should be in Go, not helpers.js.

* There are multiple ways that code strips the trailing dot from a domain name.  Benchmark a few and standardize on the fastest. In pkg/nameutil. Possibly have one that is extra fast but panics if there is no "." at the end.

* models/record.go FixPostion has an easy TODO.

* models/t_txt.go has an some easy FIXMEs.

## Bad decisions to reverse

* The preview/push code is incredibly complex and difficult to understand, therefore difficult
to debug. It could probably be reworked to be more simple and easier to debug.

* External providers: It should be possible to add a provider without updating the source code to "dnscontrol". Teraform does it, why can't we?

* mustbe.TargetHost() accepts "" or "." as a special flag. This can probably be replaced by nrc.Flags{}

* models.SetLabel() should be removed (it's only used in a few places) or enhanced to also create the .NameUnicode/.NameFQDNUnicode fields. Then de-duplicate the code that does this for NewRecordConfig().

* deSEC: BUG: It mutates RecordConfigs, which will create problems if the same zone is sent to multiple providers.

* REV("1.2.3.4") returns "4.3.2.1.in-addr.arpa" with no trailing dot, leaving us guessing later on. It should add
the trailing dot when called in D(REV(foo)) but not in PTR(REV(foo)) (zone context vs. record context).  Then
downstream code could be simplified.

* NAMESERVER(), NAMESERVER_TTL() are confusing and should be reworked.
I wonder if it would be less confusing to have a feature called PARENT_NS() that takes
a single or list of nameservers and a TTL.
  * PARENT_NS() doesn't exist in a zone?  No changes shoud be done at the parent.
  * PARENT_NS() exists? Check the parent zone and make correction.
    * If the zone is a domain:
      * If there is a REGISTRAR() use it to fix things.  Else print a warning.
    * If the zone's parent is also in this config, inject NS records into it?  Warn if dups?
Or maybe a better design is:
  * REGISTRAR() should also take delegation info.
        REGISTRAR("foo", DELEGATE( "ns2", "ns2": { a: "1.2.3.4", aaaa: "1234::1", "ttl": "5m"} ))

## Actual new features:

* Handle "unknown types". dnsv2 has a way to managing unknown types. Investigate it and replace the half-written version (see rc.UnknownTypeName)

* pkg/diff2/analyze.go: Smarter diffs. Diffs could just show which fields changed. For example, if just the preference of an MX record changes, show "old -> new" just for the priority.


Projects for Dec 2026:
* Unknown types
* Fix all the issues listed in Github!
* Smarter diffs
* Glue record check
