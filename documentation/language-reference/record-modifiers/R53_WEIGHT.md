---
name: R53_WEIGHT
parameters:
  - weight
  - set_identifier
parameter_types:
  weight: number
  set_identifier: string
ts_return: RecordModifier
provider: ROUTE53
---

`R53_WEIGHT` configures [Route 53 weighted routing](https://docs.aws.amazon.com/Route53/latest/DeveloperGuide/routing-policy-weighted.html) for a record. It distributes traffic across multiple resources based on the weights you assign.

`weight` is an integer between 0 and 255. Route 53 distributes traffic proportionally based on the weights assigned to each record with the same name and type. A weight of 0 means no traffic is routed to that resource unless all other records also have weight 0.

`set_identifier` is a unique string that differentiates this record from other weighted records with the same name and type.

You can optionally associate a health check using [`R53_HEALTH_CHECK_ID()`](R53_HEALTH_CHECK_ID.md) to remove unhealthy endpoints from the rotation.

{% code title="dnsconfig.js" %}
```javascript
D("example.com", REG_MY_PROVIDER, DnsProvider("ROUTE53"),
  // 70% of traffic to east, 30% to west
  A("www", "1.2.3.4", R53_WEIGHT(70, "web-east")),
  A("www", "5.6.7.8", R53_WEIGHT(30, "web-west")),

  // Weighted CNAME records
  CNAME("cdn", "east.cdn.example.com.", R53_WEIGHT(70, "cdn-east")),
  CNAME("cdn", "west.cdn.example.com.", R53_WEIGHT(30, "cdn-west")),

  // Weighted R53_ALIAS records
  R53_ALIAS("api", "A", "alb-east.us-east-1.elb.amazonaws.com.", R53_ZONE("Z35SXDOTRQ7X7K"), R53_WEIGHT(60, "api-east")),
  R53_ALIAS("api", "A", "alb-west.us-west-2.elb.amazonaws.com.", R53_ZONE("Z1H1FL5HABSF5"), R53_WEIGHT(40, "api-west")),

  // With health checks
  A("api", "10.0.1.1", R53_WEIGHT(50, "api-primary"), R53_HEALTH_CHECK_ID("12345678-1234-1234-1234-123456789012")),
  A("api", "10.0.2.1", R53_WEIGHT(50, "api-secondary"), R53_HEALTH_CHECK_ID("87654321-4321-4321-4321-210987654321")),
);
```
{% endcode %}
