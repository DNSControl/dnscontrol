---
name: DMARC_BUILDER
parameters:
  - label
  - version
  - policy
  - subdomainPolicy
  - nonexistentSubdomainPolicy
  - alignmentSPF
  - alignmentDKIM
  - percent
  - rua
  - ruf
  - publicSuffixDomain
  - testMode
  - failureOptions
  - failureFormat
  - reportInterval
  - ttl
parameters_object: true
parameter_types:
  label: string?
  version: string?
  policy: "'none' | 'quarantine' | 'reject'"
  subdomainPolicy: "'none' | 'quarantine' | 'reject'?"
  nonexistentSubdomainPolicy: "'none' | 'quarantine' | 'reject'?"
  alignmentSPF: "'strict' | 's' | 'relaxed' | 'r'?"
  alignmentDKIM: "'strict' | 's' | 'relaxed' | 'r'?"
  percent: number?
  rua: string[]?
  ruf: string[]?
  publicSuffixDomain: "'y' | 'n' | 'u'"
  testMode: "'y' | 'n'"
  failureOptions: "{ SPF: boolean, DKIM: boolean } | string?"
  failureFormat: string?
  reportInterval: "Duration | number?"
  ttl: Duration?
---

DNSControl contains a `DMARC_BUILDER` which can be used to simply create
DMARC policies for your domains.


## Example

### Simple example

{% code title="dnsconfig.js" %}
```javascript
D("example.com", REG_MY_PROVIDER, DnsProvider(DSP_MY_PROVIDER),
  DMARC_BUILDER({
    policy: "reject",
    ruf: [
      "mailto:mailauth-reports@example.com",
    ],
  }),
);
```
{% endcode %}

This yield the following record:

```text
@   IN  TXT "v=DMARC1; p=reject; ruf=mailto:mailauth-reports@example.com"
```

### Advanced example

{% code title="dnsconfig.js" %}
```javascript
D("example.com", REG_MY_PROVIDER, DnsProvider(DSP_MY_PROVIDER),
  DMARC_BUILDER({
    policy: "reject",
    subdomainPolicy: "quarantine",
    alignmentSPF: "r",
    alignmentDKIM: "strict",
    rua: [
      "mailto:mailauth-reports@example.com",
      "https://dmarc.example.com/submit",
    ],
    ruf: [
      "mailto:mailauth-reports@example.com",
    ],
    failureOptions: "1",
  }),
);
```
{% endcode %}

{% code title="dnsconfig.js" %}
```javascript
D("example.com", REG_MY_PROVIDER, DnsProvider(DSP_MY_PROVIDER),
  DMARC_BUILDER({
    label: "insecure",
    policy: "none",
    ruf: [
      "mailto:mailauth-reports@example.com",
    ],
    failureOptions: {
        SPF: false,
        DKIM: true,
    },
  }),
);
```
{% endcode %}

This yields the following records:

```text
@           IN  TXT "v=DMARC1; p=reject; sp=quarantine; adkim=s; aspf=r; rua=mailto:mailauth-reports@example.com,https://dmarc.example.com/submit; ruf=mailto:mailauth-reports@example.com; fo=1"
insecure    IN  TXT "v=DMARC1; p=none; ruf=mailto:mailauth-reports@example.com; fo=d"
```

### Parameters

* `label:` The DNS label for the DMARC record (`_dmarc` prefix is added, default: `"@"`)
* `version:` The DMARC version to be used (default: `DMARC1`)
* `policy:` The DMARC policy (`p=`), must be one of `"none"`, `"quarantine"`, `"reject"`
* `subdomainPolicy:` The DMARC policy for subdomains (`sp=`), must be one of `"none"`, `"quarantine"`, `"reject"` (optional)
* `nonexistentSubdomainPolicy:` The DMARC policy for nonexistent subdomains (`np=`), must be one of `"none"`, `"quarantine"`, `"reject"` (optional)
* `alignmentSPF:` `"strict"`/`"s"` or `"relaxed"`/`"r"` alignment for SPF (`aspf=`, default: `"r"`)
* `alignmentDKIM:` `"strict"`/`"s"` or `"relaxed"`/`"r"` alignment for DKIM (`adkim=`, default: `"r"`)
* `percent:` Integer percentage (0-100) of messages to which the DMARC policy is applied (`pct=`, optional)
* `rua:` Array of aggregate report targets (optional)
* `ruf:` Array of failure report targets (optional)
* `publicSuffixDomain:` `"y"`, or `"n"` or `"u"` indicates whether the domain is a Public Suffix Domain (`psd=`, default: `"u"`) 
* `testMode:` `"y"` or `"n"` DMARC policy test mode dictates whether p, sp, or np policy tags are applied (`t=`, default: `"n"`) 
* `failureOptions:` Object or string; Object containing booleans `SPF` and `DKIM`, string is passed raw (`fo=`, default: `"0"`)
* `failureFormat:` Format of the message's failure reports (`rf=`); only emitted when `ruf` is also set (optional)
* `reportInterval:` Requested interval between aggregate reports, as a `Duration` or number of seconds (`ri=`, optional)
* `ttl:` Input for `TTL` method (optional)

### Caveats

* TXT records are automatically split using `AUTOSPLIT`.
* URIs in the `rua` and `ruf` arrays are passed raw. You must percent-encode all commas and exclamation points in the URI itself.
