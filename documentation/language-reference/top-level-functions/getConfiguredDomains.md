---
name: getConfiguredDomains
ts_is_function: true
ts_return: string[]
---

`getConfiguredDomains` getConfiguredDomains is a helper function that returns the domain names
configured at the time the function is called. Calling this function early or later in
`dnsconfig.js` may return different results. Typical usage is to iterate over all
domains at the end of your configuration file.

Example for adding records to all configured domains:
{% code title="dnsconfig.js" %}
```javascript
var domains = getConfiguredDomains();
for(i = 0; i < domains.length; i++) {
  D_EXTEND(domains[i],
    TXT("_important", "BLA") // I know, not really creative.
  )
}
```
{% endcode %}

This will end up in following modifications: (All output assumes the `--full` flag)

```text
******************** Domain: domain1.tld
----- Getting nameservers from: registrar
----- DNS Provider: registrar...2 corrections
#1: CREATE TXT _important.domain1.tld "BLA" ttl=43200
#2: REFRESH zone domain1.tld

******************** Domain: domain2.tld
----- Getting nameservers from: registrar
----- DNS Provider: registrar...2 corrections
#1: CREATE TXT _important.domain2.tld "BLA" ttl=43200
#2: REFRESH zone domain2.tld
```

Example for adding DMARC report records:

This example might be more useful, specially for configuring the DMARC report records. According to DMARC RFC you need to specify `domain2.tld._report.dmarc.domain1.tld` to allow `domain2.tld` to send aggregate/forensic email reports to `domain1.tld`. This can be used to do this in an easy way, without using the wildcard from the RFC.

{% code title="dnsconfig.js" %}
```javascript
var domains = getConfiguredDomains();
for(i = 0; i < domains.length; i++) {
    D_EXTEND("domain1.tld",
        TXT(domains[i] + "._report._dmarc", "v=DMARC1")
    );
}
```
{% endcode %}

This will end up in following modifications:

```text
******************** Domain: domain2.tld
----- Getting nameservers from: registrar
----- DNS Provider: registrar...4 corrections
#1: CREATE TXT domain1.tld._report._dmarc.domain2.tld "v=DMARC1" ttl=43200
#2: CREATE TXT domain3.tld._report._dmarc.domain2.tld "v=DMARC1" ttl=43200
#3: CREATE TXT domain4.tld._report._dmarc.domain2.tld "v=DMARC1" ttl=43200
#4: REFRESH zone domain2.tld
```
