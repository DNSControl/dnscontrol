---
name: NO_PURGE
---

`NO_PURGE` indicates that existing records should not be deleted from a domain. Records will be added and updated, but not removed.

Suppose a domain is managed by both DNSControl and a third-party system. This creates a problem because DNSControl will try to delete records inserted by the other system.

By setting `NO_PURGE` on a domain, this tells DNSControl not to delete the records found in the domain.

It is similar to [`IGNORE`](IGNORE.md) but more general.

The original reason for `NO_PURGE` was that a legacy system was adopting DNSControl. Previously the domain was managed via Microsoft DNS Server's GUI. ActiveDirectory was in use, so various records were being inserted behind the scenes. It was decided to use DNSControl to simply insert a few records. The `NO_PURGE` setting instructed DNSControl not to delete the existing records.

In this example DNSControl will insert "foo.example.com" into the zone, but otherwise leave the zone alone. Changes to "foo"'s IP address will update the record. Removing the A("foo", ...) record from DNSControl will leave the record in place.

{% code title="dnsconfig.js" %}
```javascript
D("example.com", REG_MY_PROVIDER, DnsProvider(DSP_MY_PROVIDER), NO_PURGE,
  A("foo","1.2.3.4"),
);
```
{% endcode %}

The main caveat of `NO_PURGE` is that intentionally deleting records becomes more difficult. Suppose a `NO_PURGE` zone has an record such as A("ken", "1.2.3.4"). Removing the record from `dnsconfig.js` will not delete "ken" from the domain. DNSControl has no way of knowing the record was deleted from the file The DNS record must be removed manually. Users of `NO_PURGE` are prone to finding themselves with an accumulation of orphaned DNS records. That's easy to fix for a small zone but can be a big mess for large zones.

## Support

Prior to DNSControl v4.0.0, not all providers supported `NO_PURGE`.

With introduction of `diff2` algorithm (enabled by default in v4.0.0), `NO_PURGE` works with all providers.

## See also

* [`PURGE`](PURGE.md) is the default, thus this command is a no-op
* [`IGNORE`](IGNORE.md) is similar to `NO_PURGE` but is more selective
