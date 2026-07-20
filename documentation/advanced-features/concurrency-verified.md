---
name: CONCURRENCY_VERIFIED
---

✅  - A checkmark means "this has been tested, and will run concurrently".  
❔  - The questionmark means "it hasn't been tested, safety unknown"  
❌  - The red "X" means "this has been tested, and it will _not_ work concurrently".

Concurrency in this context is about gathering zone data, as seen during the preview stage when fetching current data for all zones.

### Here's how to perform a test:

1. Build dnscontrol with the -race flag: go build -race (this makes a special binary)
2. Run this special binary: `dnscontrol preview` with a configuration that lists at least 4 domains using the provider in question. More domains is better.
3. The special binary runs much slower (1/10th the speed) because it is doing a lot of checking.

If it reports problems, the error message will indicate where the problem is. It might not be 100% accurate, but it will be in the right area.

### Development guidance

Loosely, each Provider has a top-level struct named `{providername}Provider`; if this state is just some simple data-types initialized once before processing starts, and then only read, then the provider is likely to be concurrency-safe.

If this contains state which is updated (eg, caches) then the provider is probably not concurrency-safe unless every routine accessing it is protected by appropriate primitives.

Eg, URLs and auth credentials are fine. An account ID determined on the first query? That probably needs to be protected, even if every fetch returns the same data.

The `-race` build is a helpful hint but is not a guarantee.

#### Multiple Providers

Most simple use-cases likely use just one copy of a given provider, managing zones in an account. But there can be multiple _distinct_ copies, each for different accounts. Someone might use this while migrating accounts, for instance. You might have two fields in `creds.json` both with a `TYPE` of your provider.

The uses of the provider objects should never create copies; each is created by a constructor, but thereafter is a singleton per constructed provider. Thus it is safe to have synchronization objects inside the provider struct.

See, for example, the `dnsimple` provider, where there is a `sync.Once` _per object_, not at a global level, so that the `.accountID` can be fetched just once per configured provider. Because `sync.Once` contains a reference to `sync.noCopy`, the `go vet` command will catch attempts to copy that object, and so will catch attempts to copy the containing `dnsimpleProvider` object.

