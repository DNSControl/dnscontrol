# JSON Reports

DNSControl can generate a machine-parseable report of changes.

The report is JSON-formatted and contains the zonename, the provider or registrar name, the number of changes (corrections), and the correction details. All values are in text, values may contain `<`,`>` and `&` escape as needed.

To generate the report, add the `--report <filename>` option to a `preview` or `push` command.

The report lists the changes that would be (preview) or are (push) attempted, whether they are successful or not.

If a fatal error happens during the run, no report is generated.

## Sample output

{% code title="report.json" %}
```json
[
  {
    "domain": "private.example.com",
    "corrections": 10,
    "correction_details": [
      "± MODIFY private.example.com A (1.1.1.1 ttl=60) -> (1.1.1.6 ttl=300)",
      "+ CREATE private.example.com A 1.1.1.7 ttl=300",
      "± MODIFY-TTL private.example.com TXT \"v=spf1 include:spf.protection.outlook.com -all\" ttl=(60->300)",
      "+ CREATE private.example.com TXT \"v=DKIM1; k=rsa; p=xxxx....xxx\" ttl=300",
      "+ CREATE private.example.com MX 0 private-example-com.mail.protection.outlook.com. ttl=300",
      "+ CREATE *.private.example.com A 1.1.1.6 ttl=300",
      "+ CREATE *.private.example.com A 1.1.1.7 ttl=300",
      "+ CREATE ns101.private.example.com A 1.1.1.1 ttl=300",
      "+ CREATE ns102.private.example.com A 1.0.0.2 ttl=300",
      "- DELETE out-of-band.private.example.com TXT \"This out-of-band TXT record should be removed.\" ttl=300"
    ],
    "provider": "bind"
  },
  {
    "domain": "private.example.com",
    "corrections": 0,
    "registrar": "none"
  },
  {
    "domain": "admin.example.com",
    "corrections": 5,
    "correction_details": [
      "± MODIFY admin.example.com A (1.1.1.1 ttl=60) -> (1.1.1.6 ttl=300)",
      "+ CREATE admin.example.com A 1.1.1.7 ttl=300",
      "± MODIFY-TTL admin.example.com TXT \"v=spf1 include:spf.protection.outlook.com -all\" ttl=(60->300)",
      "+ CREATE admin.example.com TXT \"v=DKIM1; k=rsa; p=xxxx....xxx\" ttl=300",
      "- DELETE out-of-band.admin.example.com TXT \"This out-of-band TXT record should be removed.\" ttl=300"
    ],
    "provider": "bind"
  },
  {
    "domain": "admin.example.com",
    "corrections": 0,
    "registrar": "none"
  }
]
```
{% endcode %}
