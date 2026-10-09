---
name: AZURE_ALIAS
parameters:
  - name
  - type
  - target
  - modifiers...
parameter_types:
  name: string
  type: '"A" | "AAAA" | "CNAME"'
  target: string
  "modifiers...": RecordModifier[]
provider: AZURE_DNS
---

AZURE_ALIAS is a Azure specific virtual record type that points a record at either another record or an Azure entity.
It is analogous to a CNAME, but is usually resolved at request-time and served as an A record.
Unlike CNAMEs, ALIAS records can be used at the zone apex (`@`)

Unlike the regular ALIAS directive, AZURE_ALIAS is only supported on AZURE.
Attempting to use AZURE_ALIAS on another provider than Azure will result in an error.

The name should be the relative label for the domain.

The type can be any of the following:
* A
* AAAA
* CNAME

Target should be the Azure Id representing the target. It starts `/subscription/`. The resource id can be found in https://resources.azure.com/.

The Target can :

* Point to a public IP resource from a DNS `A/AAAA` record set.
You can create an A/AAAA record set and make it an alias record set to point to a public IP resource (standard or basic).
The DNS record set changes automatically if the public IP address changes or is deleted.
Dangling DNS records that point to incorrect IP addresses are avoided.
There is a current limit of 20 alias records sets per resource.
* Point to a Traffic Manager profile from a DNS `A/AAAA/CNAME` record set.
You can create an A/AAAA or CNAME record set and use alias records to point it to a Traffic Manager profile.
It's especially useful when you need to route traffic at a zone apex, as traditional CNAME records aren't supported for a zone apex.
For example, say your Traffic Manager profile is myprofile.trafficmanager.net and your business DNS zone is contoso.com.
You can create an alias record set of type A/AAAA for contoso.com (the zone apex) and point to myprofile.trafficmanager.net.
* Point to an Azure Content Delivery Network (CDN) endpoint.
This is useful when you create static websites using Azure storage and Azure CDN.
* Point to another DNS record set within the same zone.
Alias records can reference other record sets of the same type.
For example, a DNS CNAME record set can be an alias to another CNAME record set.
This arrangement is useful if you want some record sets to be aliases and some non-aliases.

{% code title="dnsconfig.js" %}
```javascript
D("example.com", REGISTRAR("my_registrar"), SERVICE("AZURE_DNS"),
  AZURE_ALIAS("foo", "A", "/subscriptions/726f8cd6-6459-4db4-8e6d-2cd2716904e2/resourceGroups/test/providers/Microsoft.Network/trafficManagerProfiles/testpp2"), // record for traffic manager
  AZURE_ALIAS("foo", "CNAME", "/subscriptions/726f8cd6-6459-4db4-8e6d-2cd2716904e2/resourceGroups/test/providers/Microsoft.Network/dnszones/example.com/A/quux."), // record in the same zone
);
```
{% endcode %}
