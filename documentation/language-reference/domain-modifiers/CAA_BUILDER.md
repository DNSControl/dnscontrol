---
name: CAA_BUILDER
parameters:
  - label
  - iodef
  - iodef_critical
  - issue
  - issue_critical
  - issuewild
  - issuewild_critical
  - issuevmc
  - issuevmc_critical
  - issuemail
  - issuemail_critical
  - ttl
parameters_object: true
parameter_types:
  label: string?
  iodef: string?
  iodef_critical: boolean?
  issue: "string[]|'none'?"
  issue_critical: boolean?
  issuewild: "string[]|'none'?"
  issuewild_critical: boolean?
  issuevmc: "string[]|'none'?"
  issuevmc_critical: boolean?
  issuemail: "string[]|'none'?"
  issuemail_critical: boolean?
  ttl: Duration?
---

`CAA_BUILDER` adds a [Certification Authority Authorization record](https://www.rfc-editor.org/rfc/rfc8659) to a domain.

`CAA_BUILDER` eases the creation of [`CAA`](CAA.md) records. Instead of creating each [`CAA`](CAA.md) record
individually, you can simply configure your report mail address, the
authorized certificate authorities and the builder cares about the rest.

## Example

### Simple example

{% code title="dnsconfig.js" %}
```javascript
D("example.com", REG_MY_PROVIDER, DnsProvider(DSP_MY_PROVIDER),
  CAA_BUILDER({
    label: "@",
    iodef: "mailto:test@example.com",
    iodef_critical: true,
    issue: [
      "letsencrypt.org",
      "comodoca.com",
    ],
    issuewild: "none",
  }),
);
```
{% endcode %}

`CAA_BUILDER` builds multiple records:

{% code title="dnsconfig.js" %}
```javascript
D("example.com", REG_MY_PROVIDER, DnsProvider(DSP_MY_PROVIDER),
  CAA("@", "iodef", "mailto:test@example.com", CAA_CRITICAL),
  CAA("@", "issue", "letsencrypt.org"),
  CAA("@", "issue", "comodoca.com"),
  CAA("@", "issuewild", ";"),
);
```
{% endcode %}

which in turns yield the following records:

```text
@ 300 IN CAA 128 iodef "mailto:test@example.com"
@ 300 IN CAA 0 issue "letsencrypt.org"
@ 300 IN CAA 0 issue "comodoca.com"
@ 300 IN CAA 0 issuewild ";"
```

### Example with CAA_CRITICAL flag on all records

The same example can be enriched with CAA_CRITICAL on all records:

{% code title="dnsconfig.js" %}
```javascript
D("example.com", REG_MY_PROVIDER, DnsProvider(DSP_MY_PROVIDER),
  CAA_BUILDER({
    label: "@",
    iodef: "mailto:test@example.com",
    iodef_critical: true,
    issue: [
      "letsencrypt.org",
      "comodoca.com",
    ],
    issue_critical: true,
    issuewild: "none",
    issuewild_critical: true,
  }),
);
```
{% endcode %}

`CAA_BUILDER` then builds (the same) multiple records - all with CAA_CRITICAL flag set:

{% code title="dnsconfig.js" %}
```javascript
D("example.com", REG_MY_PROVIDER, DnsProvider(DSP_MY_PROVIDER),
  CAA("@", "iodef", "mailto:test@example.com", CAA_CRITICAL),
  CAA("@", "issue", "letsencrypt.org", CAA_CRITICAL),
  CAA("@", "issue", "comodoca.com", CAA_CRITICAL),
  CAA("@", "issuewild", ";", CAA_CRITICAL),
);
```
{% endcode %}

which in turns yield the following records:

```text
@ 300 IN CAA 128 iodef "mailto:test@example.com"
@ 300 IN CAA 128 issue "letsencrypt.org"
@ 300 IN CAA 128 issue "comodoca.com"
@ 300 IN CAA 128 issuewild ";"
```

### Parameters

* `label:` The label of the CAA record. (Optional. Default: `"@"`)
* `iodef:` Report all violation to configured mail address. (Optional. Default: `""`)
* `iodef_critical:` This can be `true` or `false`. If enabled and CA does not support this record, then certificate issue will be refused. (Optional. Default: `false`)
* `issue:` An array of CAs which are allowed to issue certificates. (Optional. Default: `[]`. Use `"none"` to refuse all CAs)
* `issue_critical:` This can be `true` or `false`. If enabled and CA does not support this record, then certificate issue will be refused. (Optional. Default: `false`)
* `issuewild:` An array of CAs which are allowed to issue wildcard certificates. (Optional. Default: `[]`. Can be simply `"none"` to refuse issuing wildcard certificates for all CAs)
* `issuewild_critical:` This can be `true` or `false`. If enabled and CA does not support this record, then certificate issue will be refused. (Optional. Default: `false`)
* `issuevmc:` An array of CAs which are allowed to issue VMC certificates. (Optional. Default: `[]`. Use `"none"` to refuse all CAs)
* `issuevmc_critical:` This can be `true` or `false`. If enabled and CA does not support this record, then certificate issue will be refused. (Optional. Default: `false`)
* `issuemail:` An array of CAs which are allowed to issue email certificates. (Optional. Default: `[]`. Use `"none"` to refuse all CAs)
* `issuemail_critical:` This can be `true` or `false`. If enabled and CA does not support this record, then certificate issue will be refused. (Optional. Default: `false`)
* `ttl:` Input for `TTL` method (optional)
