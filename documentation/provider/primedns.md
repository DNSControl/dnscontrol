## Configuration

To use this provider, add an entry to `creds.json` with `TYPE` set to `PRIMEDNS`
along with a PrimeDNS API key.

Example:

{% code title="creds.json" %}
```json
{
  "primedns": {
    "TYPE": "PRIMEDNS",
    "api_key": "pdns_your-api-key-here"
  }
}
```
{% endcode %}

The key needs the scopes `dns:read` and `dns:write`. `dnscontrol create-zones` also needs `zones:create`, and
`dnscontrol get-zones ... all` needs `domains:read` (it lists the zones of the account).
A key can be limited to some zones (`allowed_zones`) when you create it; that is enough for normal use.

Optional: `api_url` overrides the API base URL (default `https://api.primedns.net/v1`).

## Metadata

This provider does not recognize any special metadata fields unique to PrimeDNS.

## Usage

An example configuration:

{% code title="dnsconfig.js" %}
```javascript
var REG_NONE = NewRegistrar("none");
var DSP_PRIMEDNS = NewDnsProvider("primedns");

D("example.com", REG_NONE, DnsProvider(DSP_PRIMEDNS),
    A("@", "192.0.2.1"),
    A("www", "192.0.2.1"),
    AAAA("@", "2001:db8::1"),
    CNAME("blog", "example.github.io."),
    MX("@", 10, "mail.example.com."),
    TXT("@", "v=spf1 include:_spf.google.com ~all"),
    CAA("@", "issue", "letsencrypt.org"),
END);
```
{% endcode %}

## Activation

Create an API key in the PrimeDNS panel (Account, API Keys) at <https://primedns.net>.

## Supported record types

PrimeDNS supports A, AAAA, ALIAS, CAA, CNAME, DNAME, DS, HTTPS, LOC, MX, NAPTR, NS, OPENPGPKEY, PTR, SRV, SSHFP, SVCB, TLSA and TXT.
The records PrimeDNS manages itself (the SOA record and the apex NS records that point to its own name servers) are not shown or changed by DNSControl.
CDS, CDNSKEY and URI records of a zone exist at PrimeDNS but DNSControl has no such types, so they are left alone.

## Nameservers

PrimeDNS assigns its name servers when a zone is added, so `GetNameservers` returns an empty list. To let DNSControl update the
delegation at a registrar, declare them in the domain:

{% code title="dnsconfig.js" %}
```javascript
D("example.com", REG_NAMECHEAP, DnsProvider(DSP_PRIMEDNS),
    NAMESERVER("ns1.primedns.net"),
    NAMESERVER("ns2.primedns.net"),
    // ...
END);
```
{% endcode %}

## Extra NS records at the apex

PrimeDNS keeps all records of one name and type on one TTL (RFC 2181 section 5.2). Its own NS records at the apex have a TTL of
8000 seconds, so an NS record you add at the apex (for example a second provider) must use the same TTL, or the API refuses it
with `TTL_MISMATCH`. DNSControl reports this before it changes anything:

{% code title="dnsconfig.js" %}
```javascript
D("example.com", REG_NONE, DnsProvider(DSP_PRIMEDNS),
    NS("@", "ns1.other-provider.example.", TTL(8000)),
END);
```
{% endcode %}

## Rate limit

The API limits requests per minute (300 to 1000, depending on the plan of the key's owner). When it answers 429 or a temporary
5xx error, the provider waits (honoring `Retry-After`) and retries.

## TXT records

PrimeDNS stores TXT content as typed. A value that starts with a double quote is read as zone file notation, so the quotes around the
text are not part of it. DNSControl handles this: a TXT value with literal double quotes is sent in escaped form and comes back unchanged.
