---
name: DnsProvider
parameters:
  - name
  - nsCount
parameter_types:
  name: string
  nsCount: number?
---

DnsProvider indicates that the specified provider should be used to manage records for this domain. The name must match the name used with [NewDnsProvider](../top-level-functions/NewDnsProvider.md).

The nsCount parameter determines how the nameservers will be managed from this provider.

Leaving the parameter out means "fetch and use all nameservers from this provider as authoritative". ie: `DnsProvider("name")`

Using `0` for nsCount means "do not fetch nameservers from this domain, or give them to the registrar".

Using a different number, ie: `DnsProvider("name",2)`, means "fetch all nameservers from this provider, but limit it to this many.

See [this page](../../advanced-features/nameservers.md) for a detailed explanation of how DNSControl handles nameservers and NS records.

If a domain (`D()`) does not include any `DnsProvider()` functions, the DNS records will not be modified. In fact, if you want to control the Registrar for a domain but not the DNS records themselves, simply do not include a `DnsProvider()` function for that `D()`.
