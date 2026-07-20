# Coding Style

# Use the Google Go Style Guide

<https://google.github.io/styleguide/go/>

# Always favor simplicity

This is a community project. The code you write today will be maintained by someone in the future that may not be a professional developer or one that is as experienced as you.

Remember what Brian Kernighan wrote:

> "Debugging is twice as hard as writing the
code in the first place. Therefore, if you write the code as cleverly as possible, you are, by definition, not smart enough to debug it." — _Brian Kernighan_

Remember the [John Woods](http://wiki.c2.com/?CodeForTheMaintainer) quote:

> "Always code as if the guy who ends up maintaining your code will be a violent
psychopath who knows where you live." — _[John Woods](http://wiki.c2.com/?CodeForTheMaintainer)_

Don't code for today-you.  Write code for six-months-from-now-you.  Have you met six-months-from-now-you? Oh, you should. Fine individual. They are quite smart. They have 6 months more experience than today-you, but sadly has had 6 months to forget what today-you knows.  The job of today-you is to write code that six-months-from-now-you can understand.

* Avoid building a complex framework to be perfectly DRY when a little bit of repetition will result in easier to understand code.
* Break things into well-defined functions that can be individually read, understood, and tested.

# Filenames

These are the filenames to use:

{% hint style="info" %}
**NOTE**: This is a new standard. Old providers are not yet compliant.
{% endhint %}

* `providers/foo/fooProvider.go` -- The main file.
* `providers/foo/records.go` -- Get/Correct the records of a DNS zone: `GetZoneRecords()` and `GetZoneRecordsCorrections()`, plus any helper functions.
* `providers/foo/convert.go` -- Convert between RecordConfig and the native API's format: `toRc()` and `toNative()`
* `providers/foo/auditrecords.go` -- The AuditRecords function and helpers
* `providers/foo/api.go` -- Code that talks to the API, preferably through a public library.
* `providers/foo/listzones.go` -- Code for listing and creating DNS zones and domains
* `providers/foo/dnssec.go` -- Code for DNSSEC support

# JavaScript annotations

Functions in `pkg/js/helpers.js` should be annotated using the [JSDoc](https://jsdoc.app/tags-param.html) `@` notation. These are used to generate autocomplete instructions for your IDE.

* Add `@param {type} foo` for a parameter named `foo` that is of type `type`.
  * Example: `@param {string} foo Description of the parameter`
* Add `@deprecated` if the function is deprecated.

# Don't conditionally add/remove trailing dots

{% hint style="info" %}
The "trailing dot" is the "." at the end of "example.com." which indicates the string is a FQDN.
{% endhint %}

DO NOT conditionally add or remove the trailing dot from a string to future-proof code. Either add it or remove it. (This applies to data received from an API call.)

DO call Go's `panic()` function if a protocol changes unexpectedly.

## Why?

It seems like future-proofing to only add a "." if the dot doesn't already exist.  It is the opposite.

Some APIs send a hostnames with a trailing "." to indicate that this is a FQDN.  Some APIs never include the trailing ".".

Zero APIs sometimes include the "." and sometimes don't include the ".". Zero APIs have a random number generator deciding if they should or shouldn't include the trailing dot.

Writing code for a situation that doesn't exist means you're writing code that never gets tested. If the world changes and suddenly the code does get executed, you're now running untested code in production. That's bad.

Therefore, if your code looks like, "add dot, but not if one exists" or "remove dot if it exists", your code is broken.  Yes, that's fine while exploring the API but once your code works, remove such conditionals.  Try the integration tests both ways: leaving the field untouched and always adding (or removing) the ".".  Only keep the way that works.

### But isn't future-proofing good? What if the API changes?

The protocol won't change.  That would break all their other users that didn't future-proof their code. Why would they make a random change like that?  A breaking change like that would (by semver rules) require a new protocol version, which would trigger code changes in DNSControl.

### But what if it changes anyway?

If the protocol does change, how do you know your future-proofed code is doing the right thing?

Let's suppose the API started sending a "." when previously they didn't.  They might do that so they can send shortnames when possible and the "." indicates that this is a FQDN. Now our future-proofed code is doing the wrong thing. It is turning "foo" into "foo." when it should be "foo.domain.com."

Let's suppose the API no longer adds a "." when it previously did. Was the change to save a byte of bandwidth or does the lack of a "." mean this is a shortname and we need to add a "." and add the domain too?  We have no way of knowing and there's a good chance we've done the wrong thing.

### What should we do instead?

Option 1: Write code that assumes it won't change.  If you need to add a dot, it is safe to just `s = s + "."`   The code will be readable by any Go developer; and less cognitive load than using a function.

Option 2: Panic if you see something unexpected.  If you are stripping a dot, panic if the dot doesn't exist.
