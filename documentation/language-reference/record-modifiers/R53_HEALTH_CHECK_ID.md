---
name: R53_HEALTH_CHECK_ID
parameters:
  - health_check_id
parameter_types:
  health_check_id: string
ts_return: RecordModifier
provider: ROUTE53
---

`R53_HEALTH_CHECK_ID` associates a [Route 53 health check](https://docs.aws.amazon.com/Route53/latest/DeveloperGuide/health-checks-creating.html) with a record. This is typically used with [`R53_WEIGHT()`](R53_WEIGHT.md) so that Route 53 stops routing traffic to unhealthy endpoints.

The `health_check_id` is the ID of a Route 53 health check that you create separately (e.g. via the AWS Console, CLI, or Terraform). DNSControl does not manage the health checks themselves, only their association with DNS records.

{% code title="dnsconfig.js" %}
```javascript
D("example.com", REG_MY_PROVIDER, DnsProvider("ROUTE53"),
  A("www", "1.2.3.4", R53_WEIGHT(70, "primary"), R53_HEALTH_CHECK_ID("12345678-1234-1234-1234-123456789012")),
  A("www", "5.6.7.8", R53_WEIGHT(30, "secondary"), R53_HEALTH_CHECK_ID("87654321-4321-4321-4321-210987654321")),
);
```
{% endcode %}
