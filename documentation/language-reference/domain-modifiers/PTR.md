---
name: PTR
parameters:
  - name
  - target
  - modifiers...
parameter_types:
  name: string
  target: string
  "modifiers...": RecordModifier[]
---

`PTR` adds a [PTR Resource record](https://www.rfc-editor.org/rfc/rfc1035) to the domain.

The name is normally a relative label for the domain, or a FQDN that ends with `.`.  If magic mode is enabled (see below) it can also be an IP address, which will be replaced by the proper string automatically, thus saving the user from having to reverse the IP address manually.

Target should be a string representing the FQDN of a host.  Like all FQDNs in DNSControl, it must end with a `.`.

# Magic Mode

PTR records are complex and typos are common. Therefore DNSControl enables features to save labor and prevent typos.  This magic is only enabled when the domain ends with `in-addr.arpa.` or `ipv6.arpa.`.

*Automatic IP-to-reverse:* If the name is a valid IP address, DNSControl will replace it with a string that is appropriate for the domain. That is, if the domain ends with `in-addr.arpa` (no `.`) and name is a valid IPv4 address, the name will be replaced with the correct string to make a reverse lookup for that address. IPv6 is properly handled too.

*Extra Validation:* DNSControl considers it an error to include a name that is inappropriate for the domain.  For example `PTR("1.2.3.4", "f.co.")` is valid for the domain `D("3.2.1.in-addr.arpa",` but DNSControl will generate an error if the domain is `D("9.9.9.in-addr.arpa",`. This is because `1.2.3.4` is contained in `1.2.3.0/24` but not `9.9.9.0/24`. This validation works for IPv6, IPv4, and RFC2317 "Classless in-addr.arpa delegation" domains.

*Automatic truncation:* DNSControl will automatically truncate FQDNs as needed. If the name is a FQDN ending with `.`, DNSControl will verify that the name is contained within the CIDR block implied by domain.  For example if name is `4.3.2.1.in-addr.arpa.` (note the trailing `.`) and the domain is `2.1.in-addr.arpa` (no trailing `.`) then the name will be replaced with `4.3`.  Note that the output of `REV("1.2.3.4")` is `4.3.2.1.in-addr.arpa.`, which means the following are all equivalent:

* `PTR(REV("1.2.3.4", ...`
* `PTR("4.3.2.1.in-addr.arpa.", ...`
* `PTR("4.3", ...`    // Assuming the domain is `2.1.in-addr.arpa`

All magic is RFC2317-aware. We use the first format listed in the RFC for both [`REV()`](../top-level-functions/REV.md) and `PTR()`. The format is `FIRST/MASK.C.B.A.in-addr.arpa` where `FIRST` is the first IP address of the zone, `MASK` is the netmask of the zone (25-31 inclusive), and A, B, C are the first 3 octets of the IP address. For example `172.20.18.130/27` is located in a zone named `128/27.18.20.172.in-addr.arpa`

{% code title="dnsconfig.js" %}
```javascript
D(REV("1.2.3.0/24"), REGISTRAR, DnsProvider(BIND),
  PTR("1", "foo.example.com."),
  PTR("2", "bar.example.com."),
  PTR("3", "baz.example.com."),
  // If the first parameter is a valid IP address, DNSControl will generate the correct name:
  PTR("1.2.3.10", "ten.example.com."),    // "10"
);
```
{% endcode %}

{% code title="dnsconfig.js" %}
```javascript
D(REV("9.9.9.128/25"), REGISTRAR, DnsProvider(BIND),
  PTR("9.9.9.129", "first.example.com."),
);
```
{% endcode %}

{% code title="dnsconfig.js" %}
```javascript
D(REV("2001:db8:302::/48"), REGISTRAR, DnsProvider(BIND),
  PTR("1.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0", "foo.example.com."),  // 2001:db8:302::1
  // If the first parameter is a valid IP address, DNSControl will generate the correct name:
  PTR("2001:db8:302::2", "two.example.com."),                          // "2.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0"
  PTR("2001:db8:302::3", "three.example.com."),                        // "3.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0"
);
```
{% endcode %}

# Automatic forward and reverse lookups

DNSControl does not automatically generate forward and reverse lookups. However it is possible to write a macro that does this by using the [`D_EXTEND()`](../top-level-functions/D_EXTEND.md) function to insert `A` and `PTR` records into previously-defined domains.

{% code title="dnsconfig.js" %}
```javascript
function FORWARD_AND_REVERSE(ipaddr, fqdn) {
    D_EXTEND(dom,
        A(fqdn, ipaddr)
    );
    D_EXTEND(REV(ipaddr),
        PTR(ipaddr, fqdn)
    );
}

D("example.com", REGISTRAR, DnsProvider(DSP_NONE),
    ...,
);
D(REV("10.20.30.0/24"), REGISTRAR, DnsProvider(DSP_NONE),
    ...,
);

FORWARD_AND_REVERSE("10.20.30.77", "foo.example.com.");
FORWARD_AND_REVERSE("10.20.30.99", "bar.example.com.");
```
{% endcode %}
