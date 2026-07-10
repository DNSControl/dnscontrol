---
name: R53_ALIAS
parameters:
  - name
  - type
  - target
  - modifiers...
parameter_types:
  name: string
  type: '"SOA" | "A" | "TXT" | "CNAME" | "MX" | "NAPTR" | "PTR" | "SRV" | "SPF" | "AAAA" | "CAA" | "DS" | "TLSA" | "SSHFP" | "SVCB" | "HTTPS"'
  target: string
  "modifiers...": RecordModifier[]
provider: ROUTE53
---

`R53_ALIAS` is a Route53 specific virtual record type that points a record at either another record or an AWS entity (like a Cloudfront distribution, an ELB, etc...). It is analogous to a `CNAME`, but is usually resolved at request-time and served as an `A` record. Unlike `CNAME` records, `ALIAS` records can be used at the zone apex (`@`)

Unlike the regular [`ALIAS`](ALIAS.md) directive, `R53_ALIAS` is only supported on Route53. Attempting to use `R53_ALIAS` on another provider than Route53 will result in an error.

The name should be the relative label for the domain.

Target should be a string representing the target. If it is a single label we will assume it is a relative name on the current domain. If it contains *any* dots, it should be a fully qualified domain name, ending with a `.`.

The Target can be any of:

* _CloudFront distribution_: in this case specify the domain name that CloudFront assigned when you created your distribution (note that your CloudFront distribution must include an alternate domain name that matches the record you're adding)
* _Elastic Beanstalk environment_: specify the `CNAME` attribute for the environment. The environment must have a regionalized domain name. To get the `CNAME`, you can use either the [AWS Console](https://docs.aws.amazon.com/elasticbeanstalk/latest/dg/customdomains.html), [AWS Elastic Beanstalk API](https://docs.aws.amazon.com/elasticbeanstalk/latest/api/API_DescribeEnvironments.html), or the [AWS CLI](https://docs.aws.amazon.com/cli/latest/reference/elasticbeanstalk/describe-environments.html).
* _ELB load balancer_: specify the DNS name that is associated with the load balancer. To get the DNS name you can use either the AWS Console (on the EC2 page, choose Load Balancers, select the right one, choose the description tab), [ELB API](https://docs.aws.amazon.com/elasticloadbalancing/latest/APIReference/API_DescribeLoadBalancers.html), the [AWS ELB CLI](https://docs.aws.amazon.com/cli/latest/reference/elb/describe-load-balancers.html), or the [AWS ELBv2 CLI](https://docs.aws.amazon.com/cli/latest/reference/elbv2/describe-load-balancers.html).
* _S3 bucket_ (configured as website): specify the domain name of the Amazon S3 website endpoint in which you configured the bucket (for instance s3-website-us-east-2.amazonaws.com). For the available values refer to the [Amazon S3 Website Endpoints](https://docs.aws.amazon.com/general/latest/gr/rande.html#s3_region).
* _Another Route53 record_: specify the value of the name of another record in the same hosted zone.

For all the target type, excluding 'another record', you have to specify the `Zone ID` of the target. This is done by using the [`R53_ZONE`](../record-modifiers/R53_ZONE.md) record modifier.

The zone id can be found depending on the target type:

* _CloudFront distribution_: specify `Z2FDTNDATAQYW2`
* _Elastic Beanstalk environment_: specify the hosted zone ID for the region in which the environment has been created. Refer to the [List of regions and hosted Zone IDs](https://docs.aws.amazon.com/general/latest/gr/rande.html#elasticbeanstalk_region).
* _ELB load balancer_: specify the value of the hosted zone ID for the load balancer. You can find it in [the List of regions and hosted Zone IDs](https://docs.aws.amazon.com/general/latest/gr/rande.html#elb_region)
* _S3 bucket_ (configured as website): specify the hosted zone ID for the region that you created the bucket in. You can find it in [the List of regions and hosted Zone IDs](https://docs.aws.amazon.com/general/latest/gr/rande.html#s3_region)
* _Another Route 53 record_: you can either specify the correct zone id or do not specify anything and DNSControl will figure out the right zone id. (Note: Route53 alias can't reference a record in a different zone).

Target health evaluation can be enabled with the [`R53_EVALUATE_TARGET_HEALTH`](../record-modifiers/R53_EVALUATE_TARGET_HEALTH.md) record modifier.

{% code title="dnsconfig.js" %}
```javascript
D("example.com", REG_MY_PROVIDER, DnsProvider("ROUTE53"),
  R53_ALIAS("foo", "A", "bar"),                              // record in same zone
  R53_ALIAS("foo", "A", "bar", R53_ZONE("Z35SXDOTRQ7X7K")),  // record in same zone, zone specified
  R53_ALIAS("foo", "A", "blahblah.elasticloadbalancing.us-west-1.amazonaws.com.", R53_ZONE("Z368ELLRRE2KJ0"), R53_EVALUATE_TARGET_HEALTH(true)),     // a classic ELB in us-west-1 with target health evaluation enabled
  R53_ALIAS("foo", "A", "blahblah.elasticbeanstalk.us-west-2.amazonaws.com.", R53_ZONE("Z38NKT9BP95V3O")),     // an Elastic Beanstalk environment in us-west-2
  R53_ALIAS("foo", "A", "blahblah-bucket.s3-website-us-west-1.amazonaws.com.", R53_ZONE("Z2F56UZL2M1ACD")),     // a website S3 Bucket in us-west-1
);
```
{% endcode %}
