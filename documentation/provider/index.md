# Service Providers

## Provider Features

The table below shows various features supported, or not supported by DNSControl providers. This table is automatically generated from metadata supplied by the provider when they register themselves inside dnscontrol.

An empty space may indicate the feature is not supported by a provider, or it may simply mean the feature has not been investigated and implemented yet. If a feature you need is missing from a provider that supports it, we'd love your contribution to ensure it works correctly and add it to this matrix.

If a feature is definitively not supported for whatever reason, we would also like a PR to clarify why it is not supported, and fill in this entire matrix.

<!-- provider-matrix-start -->
Jump to a table:

- [Provider Type](#provider-type)
- [Provider API](#provider-api)
- [DNS extensions](#dns-extensions)
- [Service discovery](#service-discovery)
- [Security](#security)
- [DNSSEC](#dnssec)

### Provider Type <!--(table 1/6)-->

| Provider name | Official Support | DNS Provider | Registrar |
| ------------- | ---------------- | ------------ | --------- |
| [`ADGUARDHOME`](adguardhome.md) | ❌ | ✅ | ❌ |
| [`AKAMAIEDGEDNS`](akamaiedgedns.md) | ❌ | ✅ | ❌ |
| [`ALIDNS`](alidns.md) | ❌ | ✅ | ❌ |
| [`AUTODNS`](autodns.md) | ❌ | ✅ | ✅ |
| [`AXFRDDNS`](axfrddns.md) | ❌ | ✅ | ❌ |
| [`AZURE_DNS`](azuredns.md) | ✅ | ✅ | ❌ |
| [`AZURE_PRIVATE_DNS`](azureprivatedns.md) | ✅ | ✅ | ❌ |
| [`BIND`](bind.md) | ✅ | ✅ | ❌ |
| [`BUNNY_DNS`](bunnydns.md) | ❌ | ✅ | ❌ |
| [`CLOUDFLAREAPI`](cloudflareapi.md) | ✅ | ✅ | ❌ |
| [`CLOUDNS`](cloudns.md) | ❌ | ✅ | ✅ |
| [`CNR`](cnr.md) | ❌ | ✅ | ✅ |
| [`CSCGLOBAL`](cscglobal.md) | ✅ | ✅ | ✅ |
| [`DESEC`](desec.md) | ❌ | ✅ | ❌ |
| [`DIGITALOCEAN`](digitalocean.md) | ❌ | ✅ | ❌ |
| [`DNSCALE`](dnscale.md) | ❌ | ✅ | ❌ |
| [`DNSIMPLE`](dnsimple.md) | ❌ | ✅ | ✅ |
| [`DNSMADEEASY`](dnsmadeeasy.md) | ❌ | ✅ | ❌ |
| [`DNSOVERHTTPS`](dnsoverhttps.md) | ❌ | ❌ | ✅ |
| [`DOMAINNAMESHOP`](domainnameshop.md) | ❌ | ✅ | ❌ |
| [`DYNADOT`](dynadot.md) | ❌ | ❌ | ✅ |
| [`EASYNAME`](easyname.md) | ❌ | ❌ | ✅ |
| [`EXOSCALE`](exoscale.md) | ❌ | ✅ | ❌ |
| [`FORTIGATE`](fortigate.md) | ❌ | ✅ | ❌ |
| [`GANDI_V5`](gandiv5.md) | ❌ | ✅ | ✅ |
| [`GCLOUD`](gcloud.md) | ✅ | ✅ | ❌ |
| [`GCORE`](gcore.md) | ❌ | ✅ | ❌ |
| [`GIDINET`](gidinet.md) | ❌ | ✅ | ✅ |
| [`HEDNS`](hedns.md) | ❌ | ✅ | ❌ |
| [`HETZNER`](hetzner.md) | ❌ | ✅ | ❌ |
| [`HETZNER_V2`](hetznerv2.md) | ❌ | ✅ | ❌ |
| [`HOSTINGDE`](hostingde.md) | ❌ | ✅ | ✅ |
| [`HUAWEICLOUD`](huaweicloud.md) | ❌ | ✅ | ❌ |
| [`INFOMANIAK`](infomaniak.md) | ❌ | ✅ | ❌ |
| [`INTERNETBS`](internetbs.md) | ❌ | ❌ | ✅ |
| [`INWX`](inwx.md) | ❌ | ✅ | ✅ |
| [`JOKER`](joker.md) | ❌ | ✅ | ❌ |
| [`LINODE`](linode.md) | ❌ | ✅ | ❌ |
| [`LOOPIA`](loopia.md) | ❌ | ✅ | ✅ |
| [`LUADNS`](luadns.md) | ❌ | ✅ | ❌ |
| [`MIKROTIK`](mikrotik.md) | ❌ | ✅ | ❌ |
| [`MYTHICBEASTS`](mythicbeasts.md) | ❌ | ✅ | ❌ |
| [`NAMECHEAP`](namecheap.md) | ❌ | ✅ | ✅ |
| [`NAMEDOTCOM`](namedotcom.md) | ❌ | ✅ | ✅ |
| [`NETBIRD`](netbird.md) | ❌ | ✅ | ❌ |
| [`NETCUP`](netcup.md) | ❌ | ✅ | ❌ |
| [`NETLIFY`](netlify.md) | ❌ | ✅ | ❌ |
| [`NETNOD`](netnod.md) | ❌ | ✅ | ❌ |
| [`NS1`](ns1.md) | ❌ | ✅ | ❌ |
| [`OPENSRS`](opensrs.md) | ❌ | ❌ | ✅ |
| [`ORACLE`](oracle.md) | ❌ | ✅ | ❌ |
| [`OVH`](ovh.md) | ❌ | ✅ | ✅ |
| [`PACKETFRAME`](packetframe.md) | ❌ | ✅ | ❌ |
| [`PORKBUN`](porkbun.md) | ❌ | ✅ | ✅ |
| [`POWERDNS`](powerdns.md) | ❌ | ✅ | ❌ |
| [`REALTIMEREGISTER`](realtimeregister.md) | ❌ | ✅ | ✅ |
| [`ROUTE53`](route53.md) | ✅ | ✅ | ✅ |
| [`RWTH`](rwth.md) | ❌ | ✅ | ❌ |
| [`SAKURACLOUD`](sakuracloud.md) | ❌ | ✅ | ❌ |
| [`SOFTLAYER`](softlayer.md) | ❌ | ✅ | ❌ |
| [`TENCENTDNS`](tencentdns.md) | ❌ | ✅ | ✅ |
| [`TRANSIP`](transip.md) | ❌ | ✅ | ❌ |
| [`UNIFI`](unifi.md) | ❌ | ✅ | ❌ |
| [`VERCEL`](vercel.md) | ❌ | ✅ | ❌ |
| [`VULTR`](vultr.md) | ❌ | ✅ | ❌ |


### Provider API <!--(table 2/6)-->

| Provider name | [Concurrency Verified](../advanced-features/concurrency-verified.md) | [dual host](../advanced-features/dual-host.md) | create-domains | get-zones |
| ------------- | -------------------------------------------------------------------- | ---------------------------------------------- | -------------- | --------- |
| [`ADGUARDHOME`](adguardhome.md) | ❔ | ❔ | ❌ | ❌ |
| [`AKAMAIEDGEDNS`](akamaiedgedns.md) | ❔ | ✅ | ✅ | ✅ |
| [`ALIDNS`](alidns.md) | ✅ | ✅ | ❌ | ✅ |
| [`AUTODNS`](autodns.md) | ✅ | ❌ | ❌ | ✅ |
| [`AXFRDDNS`](axfrddns.md) | ✅ | ❌ | ❌ | ❌ |
| [`AZURE_DNS`](azuredns.md) | ✅ | ✅ | ✅ | ✅ |
| [`AZURE_PRIVATE_DNS`](azureprivatedns.md) | ❔ | ❌ | ✅ | ✅ |
| [`BIND`](bind.md) | ✅ | ✅ | ✅ | ✅ |
| [`BUNNY_DNS`](bunnydns.md) | ❔ | ❌ | ✅ | ✅ |
| [`CLOUDFLAREAPI`](cloudflareapi.md) | ✅ | ❌ | ✅ | ✅ |
| [`CLOUDNS`](cloudns.md) | ✅ | ✅ | ✅ | ✅ |
| [`CNR`](cnr.md) | ✅ | ✅ | ✅ | ✅ |
| [`CSCGLOBAL`](cscglobal.md) | ✅ | ❔ | ❌ | ✅ |
| [`DESEC`](desec.md) | ✅ | ❔ | ✅ | ✅ |
| [`DIGITALOCEAN`](digitalocean.md) | ✅ | ✅ | ✅ | ✅ |
| [`DNSCALE`](dnscale.md) | ❌ | ❔ | ✅ | ✅ |
| [`DNSIMPLE`](dnsimple.md) | ✅ | ❌ | ❌ | ✅ |
| [`DNSMADEEASY`](dnsmadeeasy.md) | ❔ | ✅ | ✅ | ✅ |
| [`DNSOVERHTTPS`](dnsoverhttps.md) | ❔ | ❔ | ❌ | ❔ |
| [`DYNADOT`](dynadot.md) | ❔ | ❔ | ❌ | ❔ |
| [`EASYNAME`](easyname.md) | ❔ | ❔ | ❌ | ❔ |
| [`EXOSCALE`](exoscale.md) | ❌ | ❌ | ❌ | ❔ |
| [`FORTIGATE`](fortigate.md) | ❔ | ❔ | ✅ | ✅ |
| [`GANDI_V5`](gandiv5.md) | ✅ | ❔ | ❌ | ✅ |
| [`GCLOUD`](gcloud.md) | ✅ | ✅ | ✅ | ✅ |
| [`GCORE`](gcore.md) | ✅ | ✅ | ✅ | ✅ |
| [`GIDINET`](gidinet.md) | ✅ | ✅ | ❌ | ✅ |
| [`HEDNS`](hedns.md) | ❔ | ✅ | ✅ | ✅ |
| [`HETZNER`](hetzner.md) | ✅ | ✅ | ✅ | ✅ |
| [`HETZNER_V2`](hetznerv2.md) | ✅ | ✅ | ✅ | ✅ |
| [`HOSTINGDE`](hostingde.md) | ❔ | ✅ | ✅ | ✅ |
| [`HUAWEICLOUD`](huaweicloud.md) | ❔ | ✅ | ✅ | ✅ |
| [`INFOMANIAK`](infomaniak.md) | ❔ | ❔ | ❌ | ❔ |
| [`INTERNETBS`](internetbs.md) | ❔ | ❔ | ❌ | ❔ |
| [`INWX`](inwx.md) | ✅ | ✅ | ✅ | ✅ |
| [`JOKER`](joker.md) | ❌ | ❌ | ✅ | ✅ |
| [`LINODE`](linode.md) | ❔ | ❌ | ❌ | ✅ |
| [`LOOPIA`](loopia.md) | ❔ | ✅ | ❌ | ✅ |
| [`LUADNS`](luadns.md) | ✅ | ✅ | ✅ | ✅ |
| [`MIKROTIK`](mikrotik.md) | ❌ | ❌ | ❌ | ✅ |
| [`MYTHICBEASTS`](mythicbeasts.md) | ✅ | ✅ | ❌ | ✅ |
| [`NAMECHEAP`](namecheap.md) | ✅ | ❌ | ❌ | ✅ |
| [`NAMEDOTCOM`](namedotcom.md) | ❔ | ✅ | ❌ | ✅ |
| [`NETBIRD`](netbird.md) | ✅ | ❌ | ✅ | ✅ |
| [`NETCUP`](netcup.md) | ❔ | ❌ | ❌ | ❌ |
| [`NETLIFY`](netlify.md) | ✅ | ❌ | ❌ | ✅ |
| [`NETNOD`](netnod.md) | ❔ | ✅ | ✅ | ✅ |
| [`NS1`](ns1.md) | ✅ | ✅ | ✅ | ✅ |
| [`OPENSRS`](opensrs.md) | ❔ | ❔ | ❌ | ❔ |
| [`ORACLE`](oracle.md) | ❔ | ✅ | ✅ | ✅ |
| [`OVH`](ovh.md) | ❔ | ✅ | ❌ | ✅ |
| [`PACKETFRAME`](packetframe.md) | ❔ | ❌ | ❌ | ❔ |
| [`PORKBUN`](porkbun.md) | ✅ | ❌ | ❌ | ✅ |
| [`POWERDNS`](powerdns.md) | ❔ | ✅ | ✅ | ✅ |
| [`REALTIMEREGISTER`](realtimeregister.md) | ❔ | ❌ | ✅ | ✅ |
| [`ROUTE53`](route53.md) | ✅ | ✅ | ✅ | ✅ |
| [`RWTH`](rwth.md) | ❔ | ❌ | ❌ | ✅ |
| [`SAKURACLOUD`](sakuracloud.md) | ❔ | ❌ | ✅ | ✅ |
| [`SOFTLAYER`](softlayer.md) | ❔ | ❔ | ❌ | ❔ |
| [`TENCENTDNS`](tencentdns.md) | ❔ | ✅ | ✅ | ✅ |
| [`TRANSIP`](transip.md) | ✅ | ❌ | ❌ | ✅ |
| [`UNIFI`](unifi.md) | ❌ | ❔ | ❌ | ❌ |
| [`VERCEL`](vercel.md) | ❔ | ❌ | ❌ | ❌ |
| [`VULTR`](vultr.md) | ❔ | ❔ | ✅ | ✅ |


### DNS extensions <!--(table 3/6)-->

| Provider name | [`ALIAS`](../language-reference/domain-modifiers/ALIAS.md) | [`DNAME`](../language-reference/domain-modifiers/DNAME.md) | [`LOC`](../language-reference/domain-modifiers/LOC.md) | [`PTR`](../language-reference/domain-modifiers/PTR.md) | [`SOA`](../language-reference/domain-modifiers/SOA.md) |
| ------------- | ---------------------------------------------------------- | ---------------------------------------------------------- | ------------------------------------------------------ | ------------------------------------------------------ | ------------------------------------------------------ |
| [`ADGUARDHOME`](adguardhome.md) | ✅ | ❔ | ❔ | ❔ | ❔ |
| [`AKAMAIEDGEDNS`](akamaiedgedns.md) | ✅ | ❔ | ✅ | ✅ | ❌ |
| [`ALIDNS`](alidns.md) | ❌ | ❔ | ❔ | ❌ | ❔ |
| [`AUTODNS`](autodns.md) | ✅ | ❔ | ❔ | ✅ | ❔ |
| [`AXFRDDNS`](axfrddns.md) | ❌ | ✅ | ✅ | ✅ | ❌ |
| [`AZURE_DNS`](azuredns.md) | ❌ | ❔ | ❌ | ✅ | ❔ |
| [`AZURE_PRIVATE_DNS`](azureprivatedns.md) | ❌ | ❔ | ❌ | ✅ | ❔ |
| [`BIND`](bind.md) | ❔ | ✅ | ✅ | ✅ | ✅ |
| [`BUNNY_DNS`](bunnydns.md) | ✅ | ❔ | ❌ | ✅ | ❌ |
| [`CLOUDFLAREAPI`](cloudflareapi.md) | ✅ | ❔ | ✅ | ✅ | ❔ |
| [`CLOUDNS`](cloudns.md) | ✅ | ✅ | ✅ | ✅ | ❔ |
| [`CNR`](cnr.md) | ✅ | ✅ | ✅ | ✅ | ❌ |
| [`DESEC`](desec.md) | ❔ | ❔ | ✅ | ✅ | ❔ |
| [`DIGITALOCEAN`](digitalocean.md) | ❌ | ❌ | ❌ | ❌ | ❌ |
| [`DNSCALE`](dnscale.md) | ✅ | ❔ | ❌ | ✅ | ❔ |
| [`DNSIMPLE`](dnsimple.md) | ✅ | ❔ | ❌ | ✅ | ❔ |
| [`DNSMADEEASY`](dnsmadeeasy.md) | ✅ | ❔ | ❌ | ✅ | ❔ |
| [`DOMAINNAMESHOP`](domainnameshop.md) | ❔ | ❔ | ❌ | ❌ | ❌ |
| [`EXOSCALE`](exoscale.md) | ✅ | ❔ | ❌ | ✅ | ❔ |
| [`FORTIGATE`](fortigate.md) | ❔ | ❔ | ❌ | ❌ | ❔ |
| [`GANDI_V5`](gandiv5.md) | ✅ | ❔ | ❌ | ✅ | ❔ |
| [`GCLOUD`](gcloud.md) | ✅ | ❔ | ❌ | ✅ | ❔ |
| [`GCORE`](gcore.md) | ✅ | ❔ | ❌ | ✅ | ❔ |
| [`GIDINET`](gidinet.md) | ❌ | ❌ | ❌ | ❌ | ❌ |
| [`HEDNS`](hedns.md) | ✅ | ❌ | ✅ | ✅ | ❌ |
| [`HETZNER`](hetzner.md) | ❌ | ❔ | ❌ | ❌ | ❌ |
| [`HETZNER_V2`](hetznerv2.md) | ❌ | ❔ | ❌ | ✅ | ❌ |
| [`HOSTINGDE`](hostingde.md) | ✅ | ❔ | ❌ | ✅ | ✅ |
| [`HUAWEICLOUD`](huaweicloud.md) | ❌ | ❔ | ❌ | ❌ | ❌ |
| [`INFOMANIAK`](infomaniak.md) | ❔ | ✅ | ❔ | ❔ | ❔ |
| [`INWX`](inwx.md) | ✅ | ❔ | ❔ | ✅ | ❔ |
| [`JOKER`](joker.md) | ❌ | ❔ | ❌ | ❌ | ❌ |
| [`LINODE`](linode.md) | ❔ | ❔ | ❌ | ❔ | ❔ |
| [`LOOPIA`](loopia.md) | ❌ | ❔ | ✅ | ❌ | ❌ |
| [`LUADNS`](luadns.md) | ✅ | ❔ | ❌ | ✅ | ❔ |
| [`MIKROTIK`](mikrotik.md) | ❌ | ❔ | ❌ | ❌ | ❌ |
| [`MYTHICBEASTS`](mythicbeasts.md) | ❌ | ❔ | ❌ | ✅ | ❔ |
| [`NAMECHEAP`](namecheap.md) | ✅ | ❔ | ❌ | ❌ | ❔ |
| [`NAMEDOTCOM`](namedotcom.md) | ✅ | ❔ | ❌ | ❌ | ❔ |
| [`NETBIRD`](netbird.md) | ❌ | ❌ | ❌ | ❌ | ❌ |
| [`NETCUP`](netcup.md) | ❔ | ❔ | ❌ | ❌ | ❔ |
| [`NETLIFY`](netlify.md) | ✅ | ❔ | ❌ | ❌ | ❔ |
| [`NETNOD`](netnod.md) | ✅ | ❌ | ❌ | ✅ | ❌ |
| [`NS1`](ns1.md) | ✅ | ✅ | ❌ | ✅ | ❔ |
| [`ORACLE`](oracle.md) | ✅ | ❔ | ❔ | ✅ | ❔ |
| [`OVH`](ovh.md) | ❌ | ❔ | ❔ | ❌ | ❔ |
| [`PACKETFRAME`](packetframe.md) | ❔ | ❔ | ❔ | ✅ | ❔ |
| [`PORKBUN`](porkbun.md) | ✅ | ❔ | ❌ | ❌ | ❌ |
| [`POWERDNS`](powerdns.md) | ✅ | ✅ | ❔ | ✅ | ✅ |
| [`REALTIMEREGISTER`](realtimeregister.md) | ✅ | ❔ | ✅ | ❌ | ❌ |
| [`ROUTE53`](route53.md) | ❌ | ❔ | ❌ | ✅ | ❔ |
| [`RWTH`](rwth.md) | ❌ | ❔ | ❌ | ✅ | ❔ |
| [`SAKURACLOUD`](sakuracloud.md) | ✅ | ❌ | ❌ | ✅ | ❌ |
| [`SOFTLAYER`](softlayer.md) | ❔ | ❔ | ❌ | ❔ | ❔ |
| [`TENCENTDNS`](tencentdns.md) | ✅ | ❔ | ❔ | ✅ | ❔ |
| [`TRANSIP`](transip.md) | ✅ | ❌ | ❌ | ❌ | ❌ |
| [`UNIFI`](unifi.md) | ❌ | ❔ | ❌ | ❌ | ❔ |
| [`VERCEL`](vercel.md) | ✅ | ❌ | ❌ | ❌ | ❌ |
| [`VULTR`](vultr.md) | ❌ | ❔ | ❌ | ❌ | ❔ |


### Service discovery <!--(table 4/6)-->

| Provider name | [`DHCID`](../language-reference/domain-modifiers/DHCID.md) | [`NAPTR`](../language-reference/domain-modifiers/NAPTR.md) | [`SRV`](../language-reference/domain-modifiers/SRV.md) | [`SVCB`](../language-reference/domain-modifiers/SVCB.md) |
| ------------- | ---------------------------------------------------------- | ---------------------------------------------------------- | ------------------------------------------------------ | -------------------------------------------------------- |
| [`AKAMAIEDGEDNS`](akamaiedgedns.md) | ❔ | ✅ | ✅ | ❔ |
| [`ALIDNS`](alidns.md) | ❔ | ❌ | ✅ | ❔ |
| [`AUTODNS`](autodns.md) | ❔ | ❔ | ✅ | ❔ |
| [`AXFRDDNS`](axfrddns.md) | ✅ | ✅ | ✅ | ✅ |
| [`AZURE_DNS`](azuredns.md) | ❔ | ❌ | ✅ | ❔ |
| [`AZURE_PRIVATE_DNS`](azureprivatedns.md) | ❔ | ❌ | ✅ | ❔ |
| [`BIND`](bind.md) | ✅ | ✅ | ✅ | ✅ |
| [`BUNNY_DNS`](bunnydns.md) | ❌ | ❌ | ✅ | ✅ |
| [`CLOUDFLAREAPI`](cloudflareapi.md) | ❔ | ✅ | ✅ | ✅ |
| [`CLOUDNS`](cloudns.md) | ❌ | ✅ | ✅ | ❌ |
| [`CNR`](cnr.md) | ✅ | ✅ | ✅ | ✅ |
| [`CSCGLOBAL`](cscglobal.md) | ❔ | ❔ | ✅ | ❔ |
| [`DESEC`](desec.md) | ❔ | ✅ | ✅ | ✅ |
| [`DIGITALOCEAN`](digitalocean.md) | ❌ | ❌ | ✅ | ❌ |
| [`DNSCALE`](dnscale.md) | ❔ | ❔ | ✅ | ✅ |
| [`DNSIMPLE`](dnsimple.md) | ❔ | ✅ | ✅ | ✅ |
| [`DNSMADEEASY`](dnsmadeeasy.md) | ❔ | ❔ | ✅ | ❔ |
| [`DOMAINNAMESHOP`](domainnameshop.md) | ❔ | ❌ | ✅ | ❔ |
| [`EXOSCALE`](exoscale.md) | ❔ | ❔ | ✅ | ❔ |
| [`GANDI_V5`](gandiv5.md) | ❔ | ❔ | ✅ | ❔ |
| [`GCLOUD`](gcloud.md) | ❔ | ❔ | ✅ | ✅ |
| [`GCORE`](gcore.md) | ❔ | ❌ | ✅ | ✅ |
| [`GIDINET`](gidinet.md) | ❌ | ❌ | ✅ | ❌ |
| [`HEDNS`](hedns.md) | ❌ | ✅ | ✅ | ✅ |
| [`HETZNER`](hetzner.md) | ❔ | ❌ | ✅ | ❔ |
| [`HETZNER_V2`](hetznerv2.md) | ❔ | ❌ | ✅ | ✅ |
| [`HOSTINGDE`](hostingde.md) | ❔ | ❌ | ✅ | ❔ |
| [`HUAWEICLOUD`](huaweicloud.md) | ❔ | ❌ | ✅ | ❌ |
| [`INFOMANIAK`](infomaniak.md) | ❔ | ❔ | ✅ | ❔ |
| [`INWX`](inwx.md) | ❔ | ✅ | ✅ | ✅ |
| [`JOKER`](joker.md) | ❔ | ✅ | ✅ | ❌ |
| [`LINODE`](linode.md) | ❔ | ❔ | ✅ | ❔ |
| [`LOOPIA`](loopia.md) | ❌ | ✅ | ✅ | ❌ |
| [`LUADNS`](luadns.md) | ❔ | ❔ | ✅ | ❔ |
| [`MIKROTIK`](mikrotik.md) | ❔ | ❌ | ✅ | ❌ |
| [`MYTHICBEASTS`](mythicbeasts.md) | ❔ | ❔ | ✅ | ❔ |
| [`NAMECHEAP`](namecheap.md) | ❔ | ❔ | ❌ | ❔ |
| [`NAMEDOTCOM`](namedotcom.md) | ❔ | ❔ | ✅ | ❔ |
| [`NETBIRD`](netbird.md) | ❌ | ❌ | ❌ | ❌ |
| [`NETCUP`](netcup.md) | ❔ | ❔ | ✅ | ❔ |
| [`NETLIFY`](netlify.md) | ❔ | ❌ | ✅ | ❔ |
| [`NETNOD`](netnod.md) | ❌ | ✅ | ✅ | ✅ |
| [`NS1`](ns1.md) | ✅ | ✅ | ✅ | ✅ |
| [`ORACLE`](oracle.md) | ❔ | ✅ | ✅ | ❔ |
| [`OVH`](ovh.md) | ❔ | ❔ | ✅ | ❔ |
| [`PACKETFRAME`](packetframe.md) | ❔ | ❔ | ✅ | ❔ |
| [`PORKBUN`](porkbun.md) | ❔ | ❌ | ✅ | ✅ |
| [`POWERDNS`](powerdns.md) | ✅ | ✅ | ✅ | ✅ |
| [`REALTIMEREGISTER`](realtimeregister.md) | ❌ | ✅ | ✅ | ❔ |
| [`ROUTE53`](route53.md) | ❔ | ❔ | ✅ | ✅ |
| [`RWTH`](rwth.md) | ❔ | ❌ | ✅ | ❔ |
| [`SAKURACLOUD`](sakuracloud.md) | ❌ | ❌ | ✅ | ✅ |
| [`SOFTLAYER`](softlayer.md) | ❔ | ❔ | ✅ | ❔ |
| [`TENCENTDNS`](tencentdns.md) | ❔ | ❔ | ✅ | ❔ |
| [`TRANSIP`](transip.md) | ❌ | ✅ | ✅ | ❌ |
| [`UNIFI`](unifi.md) | ❔ | ❔ | ✅ | ❔ |
| [`VERCEL`](vercel.md) | ❌ | ❌ | ✅ | ❌ |
| [`VULTR`](vultr.md) | ❔ | ❔ | ✅ | ❔ |


### Security <!--(table 5/6)-->

| Provider name | [`CAA`](../language-reference/domain-modifiers/CAA.md) | [`HTTPS`](../language-reference/domain-modifiers/HTTPS.md) | [`SMIMEA`](../language-reference/domain-modifiers/SMIMEA.md) | [`SSHFP`](../language-reference/domain-modifiers/SSHFP.md) | [`TLSA`](../language-reference/domain-modifiers/TLSA.md) |
| ------------- | ------------------------------------------------------ | ---------------------------------------------------------- | ------------------------------------------------------------ | ---------------------------------------------------------- | -------------------------------------------------------- |
| [`AKAMAIEDGEDNS`](akamaiedgedns.md) | ✅ | ❔ | ❔ | ✅ | ✅ |
| [`ALIDNS`](alidns.md) | ✅ | ❔ | ❔ | ❌ | ❌ |
| [`AUTODNS`](autodns.md) | ✅ | ❔ | ❔ | ❌ | ❌ |
| [`AXFRDDNS`](axfrddns.md) | ✅ | ✅ | ✅ | ✅ | ✅ |
| [`AZURE_DNS`](azuredns.md) | ✅ | ❔ | ❔ | ❌ | ❌ |
| [`AZURE_PRIVATE_DNS`](azureprivatedns.md) | ❌ | ❔ | ❔ | ❌ | ❌ |
| [`BIND`](bind.md) | ✅ | ✅ | ✅ | ✅ | ✅ |
| [`BUNNY_DNS`](bunnydns.md) | ✅ | ✅ | ❔ | ❌ | ✅ |
| [`CLOUDFLAREAPI`](cloudflareapi.md) | ✅ | ✅ | ❔ | ✅ | ✅ |
| [`CLOUDNS`](cloudns.md) | ✅ | ❌ | ❔ | ✅ | ✅ |
| [`CNR`](cnr.md) | ✅ | ❌ | ❔ | ✅ | ✅ |
| [`CSCGLOBAL`](cscglobal.md) | ✅ | ❔ | ❔ | ❔ | ❔ |
| [`DESEC`](desec.md) | ✅ | ✅ | ✅ | ✅ | ✅ |
| [`DIGITALOCEAN`](digitalocean.md) | ✅ | ❌ | ❌ | ❌ | ❌ |
| [`DNSCALE`](dnscale.md) | ✅ | ✅ | ❔ | ✅ | ✅ |
| [`DNSIMPLE`](dnsimple.md) | ✅ | ✅ | ❔ | ✅ | ✅ |
| [`DNSMADEEASY`](dnsmadeeasy.md) | ✅ | ❔ | ❔ | ❌ | ❌ |
| [`DOMAINNAMESHOP`](domainnameshop.md) | ✅ | ❔ | ❔ | ❌ | ❔ |
| [`EXOSCALE`](exoscale.md) | ✅ | ❔ | ❔ | ❔ | ❌ |
| [`GANDI_V5`](gandiv5.md) | ✅ | ❔ | ❔ | ✅ | ✅ |
| [`GCLOUD`](gcloud.md) | ✅ | ✅ | ❔ | ✅ | ✅ |
| [`GCORE`](gcore.md) | ✅ | ✅ | ❔ | ❌ | ❌ |
| [`GIDINET`](gidinet.md) | ❌ | ❌ | ❔ | ❌ | ❌ |
| [`HEDNS`](hedns.md) | ✅ | ✅ | ❌ | ✅ | ❌ |
| [`HETZNER`](hetzner.md) | ✅ | ❔ | ❔ | ❌ | ✅ |
| [`HETZNER_V2`](hetznerv2.md) | ✅ | ✅ | ❔ | ❌ | ✅ |
| [`HOSTINGDE`](hostingde.md) | ✅ | ❔ | ❔ | ✅ | ✅ |
| [`HUAWEICLOUD`](huaweicloud.md) | ✅ | ❌ | ❔ | ❌ | ❌ |
| [`INFOMANIAK`](infomaniak.md) | ✅ | ❔ | ❔ | ✅ | ✅ |
| [`INWX`](inwx.md) | ✅ | ✅ | ❔ | ✅ | ✅ |
| [`JOKER`](joker.md) | ✅ | ❌ | ❔ | ❌ | ❌ |
| [`LINODE`](linode.md) | ✅ | ❔ | ❔ | ❔ | ❔ |
| [`LOOPIA`](loopia.md) | ✅ | ❌ | ❔ | ✅ | ✅ |
| [`LUADNS`](luadns.md) | ✅ | ✅ | ❔ | ✅ | ✅ |
| [`MIKROTIK`](mikrotik.md) | ❌ | ❌ | ❔ | ❌ | ❌ |
| [`MYTHICBEASTS`](mythicbeasts.md) | ✅ | ❔ | ❔ | ✅ | ✅ |
| [`NAMECHEAP`](namecheap.md) | ✅ | ❔ | ❔ | ❔ | ❌ |
| [`NETBIRD`](netbird.md) | ❌ | ❌ | ❌ | ❌ | ❌ |
| [`NETCUP`](netcup.md) | ✅ | ❔ | ❔ | ❔ | ✅ |
| [`NETLIFY`](netlify.md) | ✅ | ❔ | ❔ | ❌ | ❌ |
| [`NETNOD`](netnod.md) | ✅ | ✅ | ❔ | ✅ | ✅ |
| [`NS1`](ns1.md) | ✅ | ✅ | ❔ | ❔ | ✅ |
| [`ORACLE`](oracle.md) | ✅ | ❔ | ❔ | ✅ | ✅ |
| [`OVH`](ovh.md) | ✅ | ❔ | ❔ | ✅ | ✅ |
| [`PORKBUN`](porkbun.md) | ✅ | ✅ | ❔ | ✅ | ✅ |
| [`POWERDNS`](powerdns.md) | ✅ | ✅ | ❔ | ✅ | ✅ |
| [`REALTIMEREGISTER`](realtimeregister.md) | ✅ | ❔ | ❔ | ✅ | ✅ |
| [`ROUTE53`](route53.md) | ✅ | ✅ | ❔ | ✅ | ✅ |
| [`RWTH`](rwth.md) | ✅ | ❔ | ❔ | ✅ | ❌ |
| [`SAKURACLOUD`](sakuracloud.md) | ✅ | ✅ | ❔ | ❌ | ❌ |
| [`TENCENTDNS`](tencentdns.md) | ✅ | ❔ | ❔ | ❔ | ❔ |
| [`TRANSIP`](transip.md) | ✅ | ❌ | ❔ | ✅ | ✅ |
| [`UNIFI`](unifi.md) | ❌ | ❔ | ❔ | ❌ | ❌ |
| [`VERCEL`](vercel.md) | ✅ | ✅ | ❔ | ❌ | ❌ |
| [`VULTR`](vultr.md) | ✅ | ❔ | ❔ | ✅ | ❌ |


### DNSSEC <!--(table 6/6)-->

| Provider name | [`AUTODNSSEC`](../language-reference/domain-modifiers/AUTODNSSEC_ON.md) | [`DNSKEY`](../language-reference/domain-modifiers/DNSKEY.md) | [`DS`](../language-reference/domain-modifiers/DS.md) |
| ------------- | ----------------------------------------------------------------------- | ------------------------------------------------------------ | ---------------------------------------------------- |
| [`AKAMAIEDGEDNS`](akamaiedgedns.md) | ✅ | ❔ | ❌ |
| [`ALIDNS`](alidns.md) | ❌ | ❔ | ❔ |
| [`AUTODNS`](autodns.md) | ❔ | ❔ | ❌ |
| [`AXFRDDNS`](axfrddns.md) | ✅ | ❌ | ✅ |
| [`BIND`](bind.md) | ✅ | ✅ | ✅ |
| [`BUNNY_DNS`](bunnydns.md) | ✅ | ❔ | ❌ |
| [`CLOUDFLAREAPI`](cloudflareapi.md) | ❔ | ❌ | ✅ |
| [`CLOUDNS`](cloudns.md) | ✅ | ❌ | ❌ |
| [`DESEC`](desec.md) | ✅ | ✅ | ✅ |
| [`DIGITALOCEAN`](digitalocean.md) | ❌ | ❌ | ❌ |
| [`DNSIMPLE`](dnsimple.md) | ✅ | ❔ | ❌ |
| [`DNSMADEEASY`](dnsmadeeasy.md) | ❔ | ❔ | ❌ |
| [`DOMAINNAMESHOP`](domainnameshop.md) | ❌ | ❔ | ❔ |
| [`GANDI_V5`](gandiv5.md) | ❔ | ❔ | ❌ |
| [`GCLOUD`](gcloud.md) | ✅ | ❔ | ❔ |
| [`GCORE`](gcore.md) | ✅ | ❔ | ❌ |
| [`GIDINET`](gidinet.md) | ❌ | ❌ | ❌ |
| [`HEDNS`](hedns.md) | ❌ | ❌ | ❌ |
| [`HETZNER`](hetzner.md) | ❌ | ❔ | ✅ |
| [`HETZNER_V2`](hetznerv2.md) | ❌ | ❔ | ✅ |
| [`HOSTINGDE`](hostingde.md) | ✅ | ❔ | ✅ |
| [`HUAWEICLOUD`](huaweicloud.md) | ✅ | ❔ | ❌ |
| [`INFOMANIAK`](infomaniak.md) | ❔ | ❔ | ✅ |
| [`INWX`](inwx.md) | ✅ | ❔ | ❔ |
| [`JOKER`](joker.md) | ❔ | ❌ | ❌ |
| [`LOOPIA`](loopia.md) | ❌ | ❌ | ❌ |
| [`MIKROTIK`](mikrotik.md) | ❌ | ❔ | ❌ |
| [`NETBIRD`](netbird.md) | ❌ | ❌ | ❌ |
| [`NETLIFY`](netlify.md) | ❌ | ❔ | ❌ |
| [`NETNOD`](netnod.md) | ❌ | ❌ | ❌ |
| [`NS1`](ns1.md) | ✅ | ❔ | ✅ |
| [`ORACLE`](oracle.md) | ❔ | ❔ | ❌ |
| [`PORKBUN`](porkbun.md) | ❌ | ❔ | ❌ |
| [`POWERDNS`](powerdns.md) | ✅ | ✅ | ✅ |
| [`REALTIMEREGISTER`](realtimeregister.md) | ✅ | ❔ | ❌ |
| [`SAKURACLOUD`](sakuracloud.md) | ❌ | ❌ | ❌ |
| [`TRANSIP`](transip.md) | ❌ | ❌ | ❌ |
| [`VERCEL`](vercel.md) | ❌ | ❌ | ❌ |

<!-- provider-matrix-end -->

### Providers with "official support"

Official support means:

* New releases will block if any of these providers do not pass integration tests.
* The DNSControl maintainers prioritize fixing bugs in these providers (though we gladly accept PRs).
* New features will work on these providers (unless the provider does not support it).
* StackOverflow maintains test accounts with those providers for running integration tests.

Providers in this category and their maintainers are:

|Name|Maintainer|
|---|---|
|[`AZURE_DNS`](azuredns.md)|@vatsalyagoel|
|[`BIND`](bind.md)|@TomOnTime|
|[`CLOUDFLAREAPI`](cloudflareapi.md)|@tresni|
|[`CSCGLOBAL`](cscglobal.md)|@mikenz|
|[`GCLOUD`](gcloud.md)|@riyadhalnur|
|[`ROUTE53`](route53.md)|@tresni|

### Providers with "contributor support"

The other providers are supported by community members, usually the original contributor.

Due to the large number of DNS providers in the world, the DNSControl team can not support and test all providers.  Test frameworks are provided to help community members support their code independently.

Expectations of maintainers:

* Maintainers are expected to support their provider and/or help find a new maintainer.
* Maintainers should set up test accounts and periodically verify that all tests pass (`pkg/js/parse_tests` and `integrationTest`).
* Contributors are encouraged to add new tests and refine old ones. (Test-driven development is encouraged.)
* Bugs will be referred to the maintainer or their designate.
* Maintainers must be responsible to bug reports and PRs.  If a maintainer is unresponsive for more than 2 months, we will consider disabling the provider.  First we will put out a call for new maintainer. If nobody volunteers, the provider may be disabled.
* Tom needs to know your real email address.  Please email tal at what exit dot org so he has it.

Providers in this category and their maintainers are:

|Name|Maintainer|
|---|---|
|[`ADGUARDHOME`](adguardhome.md)|@ishanjain28|
|[`ALIDNS`](alidns.md)|@bytemain|
|[`AZURE_PRIVATE_DNS`](azureprivatedns.md)|@matthewmgamble|
|[`AKAMAIEDGEDNS`](akamaiedgedns.md)|@edglynes|
|[`AXFRDDNS`](axfrddns.md)|@hnrgrgr|
|[`BUNNY_DNS`](bunnydns.md)|@ppmathis|
|[`CLOUDFLAREAPI`](cloudflareapi.md)|@tresni|
|[`CLOUDNS`](cloudns.md)|@pragmaton|
|[`CNR`](cnr.md)|@KaiSchwarz-cnic|
|[`CSCGLOBAL`](cscglobal.md)|@Air-New-Zealand|
|[`DESEC`](desec.md)|@D3luxee|
|[`DIGITALOCEAN`](digitalocean.md)|@chicks-net|
|[`DNSCALE`](dnscale.md)|@dnscale-ops|
|[`DNSIMPLE`](dnsimple.md)|@onlyhavecans|
|[`DNSMADEEASY`](dnsmadeeasy.md)|@vojtad|
|[`DNSOVERHTTPS`](dnsoverhttps.md)|@mikenz|
|[`DOMAINNAMESHOP`](domainnameshop.md)|@SimenBai|
|[`EASYNAME`](easyname.md)|@tresni|
|[`EXOSCALE`](exoscale.md)|@pierre-emmanuelJ|
|[`GANDI_V5`](gandiv5.md)|@TomOnTime|
|[`GCORE`](gcore.md)|@xddxdd|
|[`GIDINET`](gidinet.md)|@zupolgec|
|[`HEDNS`](hedns.md)|@rblenkinsopp|
|[`HETZNER`](hetzner.md)|@das7pad|
|[`HOSTINGDE`](hostingde.md)|@membero|
|[`HUAWEICLOUD`](huaweicloud.md)|@huihuimoe|
|[`INFOMANIAK`](infomaniak.md)|@jbelien|
|[`INTERNETBS`](internetbs.md)|@pragmaton|
|[`INWX`](inwx.md)|@patschi|
|[`LINODE`](linode.md)|@koesie10|
|[`LOOPIA`](loopia.md)|@systemcrash|
|[`LUADNS`](luadns.md)|@riku22|
|[`MIKROTIK`](mikrotik.md)|@hedger|
|[`NAMECHEAP`](namecheap.md)|@willpower232|
|[`NETCUP`](netcup.md)|@kordianbruck|
|[`NETLIFY`](netlify.md)|@SphericalKat|
|[`NETNOD`](netnod.md)|@Netnod|
|[`NS1`](ns1.md)|@costasd|
|[`OPENSRS`](opensrs.md)|@philhug|
|[`ORACLE`](oracle.md)|@kallsyms|
|[`OVH`](ovh.md)|@masterzen|
|[`PACKETFRAME`](packetframe.md)|@hamptonmoore|
|[`POWERDNS`](powerdns.md)|@jpbede|
|[`REALTIMEREGISTER`](realtimeregister.md)|@PJEilers|
|[`ROUTE53`](route53.md)|@tresni|
|[`RWTH`](rwth.md)|@MisterErwin|
| [`SAKURACLOUD`](sakuracloud.md) | @ttkzw |
| [`SOFTLAYER`](softlayer.md) | @jamielennox |
| [`TENCENTDNS`](tencentdns.md) | @cylonchau |
| [`TRANSIP`](transip.md) | @blackshadev |
|[`VERCEL`](vercel.md)|@SukkaW|
|[`VULTR`](vultr.md)|@pgaskin|

### Requested providers

We have received requests for the following providers. If you would like to contribute code to support this provider, we'd be glad to help in any way.

*(The list below is sorted alphabetically.)*

* [1984 Hosting](https://github.com/DNSControl/dnscontrol/issues/1251) (#1251)
* [BookMyName](https://github.com/DNSControl/dnscontrol/issues/3451) (#3451)
* [Constellix (DNSMadeEasy)](https://github.com/DNSControl/dnscontrol/issues/842) (#842)
* [CoreDNS](https://github.com/DNSControl/dnscontrol/issues/1284) (#1284)
* [EU.ORG](https://github.com/DNSControl/dnscontrol/issues/1176) (#1176)
* [EnCirca](https://github.com/DNSControl/dnscontrol/issues/1048) (#1048)
* [GoDaddy](https://github.com/DNSControl/dnscontrol/issues/2596) (#2596)
* [IPv64](https://github.com/DNSControl/dnscontrol/issues/3471) (#3471)
* [Imperva](https://github.com/DNSControl/dnscontrol/issues/1484) (#1484)
* [Infoblox DNS](https://github.com/DNSControl/dnscontrol/issues/1077) (#1077)
* [Joker.com](https://github.com/DNSControl/dnscontrol/issues/854) (#854)
* [Netim](https://github.com/DNSControl/dnscontrol/issues/3511) (#3511)
* [Plesk](https://github.com/DNSControl/dnscontrol/issues/2261) (#2261)
* [Rackspace Cloud DNS](https://github.com/DNSControl/dnscontrol/issues/2980) (#2980)
* [RcodeZero](https://github.com/DNSControl/dnscontrol/issues/884) (#884)
* [Sav.com](https://github.com/DNSControl/dnscontrol/issues/3633) (#3633)
* [Scaleway](https://github.com/DNSControl/dnscontrol/issues/3606) (#3606)
* [Spaceship](https://github.com/DNSControl/dnscontrol/issues/3452) (#3452)
* [SynergyWholesale](https://github.com/DNSControl/dnscontrol/issues/1605) (#1605)
* [UltraDNS by Neustar / CSCGlobal](https://github.com/DNSControl/dnscontrol/issues/1533) (#1533)
* [Yandex Cloud DNS](https://github.com/DNSControl/dnscontrol/issues/3737) (#3737)

#### Q: Why are the above GitHub issues marked "closed"?

A: Following [provider requests](../developer-info/provider-request.md), the request is closed once it is added to this list. If someone chooses to implement the provider, they re-open the issue.

#### Q: Would someone write a provider for me?

A: The maintainer of DNSControl does not write new providers.  New providers are contributed by the community.

DNSControl tries to make writing a provider as easy as possible.  DNSControl does most of the work for you, you only have to write code to authenticate, download DNS records, and perform create/modify/delete operations on those records. Please read the directions for [Writing new DNS providers](../advanced-features/writing-providers.md).  The DNS maintainers will gladly coach you through the process.
