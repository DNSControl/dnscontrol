# Service Providers

## Provider Features

The table below shows various features supported, or not supported by DNSControl providers. This table is automatically generated from metadata supplied by the provider when they register themselves inside dnscontrol.

An empty space may indicate the feature is not supported by a provider, or it may simply mean the feature has not been investigated and implemented yet. If a feature you need is missing from a provider that supports it, we'd love your contribution to ensure it works correctly and add it to this matrix.

If a feature is definitively not supported for whatever reason, we would also like a PR to clarify why it is not supported, and fill in this entire matrix.

<!-- provider-matrix-start -->
Jump to a section:

- [Provider Support Level and Type](#provider-support-level-and-type)
- [Provider API-level Features](#provider-api-level-features)
- [Supported Record Types by Provider](#supported-record-types-by-provider)

### Provider Support Level and Type <!--(section 1/3)-->

| Provider name | [Official Support](../provider/index.md#providers-with-official-support) | DNS Provider | Registrar |
| ------------- | ------------------------------------------------------------------------ | ------------ | --------- |
| [ADGUARDHOME](adguardhome.md) | ❌ | ✅ | ❌ |
| [AKAMAIEDGEDNS](akamaiedgedns.md) | ❌ | ✅ | ❌ |
| [ALIDNS](alidns.md) | ❌ | ✅ | ❌ |
| [AUTODNS](autodns.md) | ❌ | ✅ | ✅ |
| [AXFRDDNS](axfrddns.md) | ❌ | ✅ | ❌ |
| [AZURE_DNS](azuredns.md) | ✅ | ✅ | ❌ |
| [AZURE_PRIVATE_DNS](azureprivatedns.md) | ✅ | ✅ | ❌ |
| [BIND](bind.md) | ✅ | ✅ | ❌ |
| [BUNNY_DNS](bunnydns.md) | ❌ | ✅ | ❌ |
| [CLOUDFLAREAPI](cloudflareapi.md) | ✅ | ✅ | ❌ |
| [CLOUDNS](cloudns.md) | ❌ | ✅ | ✅ |
| [CLOUDPRESS](cloudpress.md) | ❌ | ✅ | ❌ |
| [CNR](cnr.md) | ❌ | ✅ | ✅ |
| [CONSTELLIX](constellix.md) | ❌ | ✅ | ❌ |
| [CSCGLOBAL](cscglobal.md) | ✅ | ✅ | ✅ |
| [DESEC](desec.md) | ❌ | ✅ | ❌ |
| [DIGITALOCEAN](digitalocean.md) | ❌ | ✅ | ❌ |
| [DNSCALE](dnscale.md) | ❌ | ✅ | ❌ |
| [DNSIMPLE](dnsimple.md) | ❌ | ✅ | ✅ |
| [DNSMADEEASY](dnsmadeeasy.md) | ❌ | ✅ | ❌ |
| [DNSOVERHTTPS](dnsoverhttps.md) | ❌ | ❌ | ✅ |
| [DOMAINNAMESHOP](domainnameshop.md) | ❌ | ✅ | ❌ |
| [DYNADOT](dynadot.md) | ❌ | ❌ | ✅ |
| [DYNU](dynu.md) | ❌ | ✅ | ❌ |
| [EASYNAME](easyname.md) | ❌ | ❌ | ✅ |
| [EXOSCALE](exoscale.md) | ❌ | ✅ | ❌ |
| [FORTIGATE](fortigate.md) | ❌ | ✅ | ❌ |
| [GANDI_V5](gandiv5.md) | ❌ | ✅ | ✅ |
| [GCLOUD](gcloud.md) | ✅ | ✅ | ❌ |
| [GCORE](gcore.md) | ❌ | ✅ | ❌ |
| [GIDINET](gidinet.md) | ❌ | ✅ | ✅ |
| [GIGAHOST](gigahost.md) | ❌ | ✅ | ❌ |
| [HEDNS](hedns.md) | ❌ | ✅ | ❌ |
| [HETZNER_V2](hetznerv2.md) | ❌ | ✅ | ❌ |
| [HOSTINGDE](hostingde.md) | ❌ | ✅ | ✅ |
| [HUAWEICLOUD](huaweicloud.md) | ❌ | ✅ | ❌ |
| [INFOBLOX](infoblox.md) | ❌ | ✅ | ❌ |
| [INFOMANIAK](infomaniak.md) | ❌ | ✅ | ❌ |
| [INTERNETBS](internetbs.md) | ❌ | ❌ | ✅ |
| [INWX](inwx.md) | ❌ | ✅ | ✅ |
| [JOKER](joker.md) | ❌ | ✅ | ❌ |
| [LINODE](linode.md) | ❌ | ✅ | ❌ |
| [LOOPIA](loopia.md) | ❌ | ✅ | ✅ |
| [LUADNS](luadns.md) | ❌ | ✅ | ❌ |
| [MIKROTIK](mikrotik.md) | ❌ | ✅ | ❌ |
| [MISAKA](misaka.md) | ❌ | ✅ | ❌ |
| [MITTWALD](mittwald.md) | ❌ | ✅ | ❌ |
| [MYTHICBEASTS](mythicbeasts.md) | ❌ | ✅ | ❌ |
| [NAMECHEAP](namecheap.md) | ❌ | ✅ | ✅ |
| [NAMEDOTCOM](namedotcom.md) | ❌ | ✅ | ✅ |
| [NETBIRD](netbird.md) | ❌ | ✅ | ❌ |
| [NETCUP](netcup.md) | ❌ | ✅ | ❌ |
| [NETLIFY](netlify.md) | ❌ | ✅ | ❌ |
| [NETNOD](netnod.md) | ❌ | ✅ | ❌ |
| [NEXDNS](nexdns.md) | ❌ | ✅ | ❌ |
| [NS1](ns1.md) | ❌ | ✅ | ❌ |
| [OPENPROVIDER](openprovider.md) | ❌ | ✅ | ❌ |
| [OPENSRS](opensrs.md) | ❌ | ❌ | ✅ |
| [OPENWRT](openwrt.md) | ❌ | ✅ | ❌ |
| [ORACLE](oracle.md) | ❌ | ✅ | ❌ |
| [OVH](ovh.md) | ❌ | ✅ | ✅ |
| [PACKETFRAME](packetframe.md) | ❌ | ✅ | ❌ |
| [PORKBUN](porkbun.md) | ❌ | ✅ | ✅ |
| [POWERDNS](powerdns.md) | ❌ | ✅ | ❌ |
| [REALTIMEREGISTER](realtimeregister.md) | ❌ | ✅ | ✅ |
| [ROUTE53](route53.md) | ✅ | ✅ | ✅ |
| [RWTH](rwth.md) | ❌ | ✅ | ❌ |
| [SAKURACLOUD](sakuracloud.md) | ❌ | ✅ | ❌ |
| [SCALEWAY](scaleway.md) | ❌ | ✅ | ❌ |
| [SOFTLAYER](softlayer.md) | ❌ | ✅ | ❌ |
| [SPACESHIP](spaceship.md) | ❌ | ✅ | ✅ |
| [TENCENTDNS](tencentdns.md) | ❌ | ✅ | ✅ |
| [TRANSIP](transip.md) | ❌ | ✅ | ❌ |
| [UNIFI](unifi.md) | ❌ | ✅ | ❌ |
| [VERCEL](vercel.md) | ❌ | ✅ | ❌ |
| [VULTR](vultr.md) | ❌ | ✅ | ❌ |
| [WEBSUPPORT](websupport.md) | ❌ | ✅ | ❌ |


### Provider API-level Features <!--(section 2/3)-->

| Provider name | [Concurrent fetching](../advanced-features/concurrency-verified.md) | [dual host](../advanced-features/dual-host.md) | create-domains | [get-zones](../commands/get-zones.md) | [AUTODNSSEC](../language-reference/domain-modifiers/AUTODNSSEC_ON.md) |
| ------------- | ------------------------------------------------------------------- | ---------------------------------------------- | -------------- | ------------------------------------- | --------------------------------------------------------------------- |
| [ADGUARDHOME](adguardhome.md) | ✅ | ✅ | ❌ | ❌ | ❌ |
| [AKAMAIEDGEDNS](akamaiedgedns.md) | ✅ | ✅ | ✅ | ✅ | ✅ |
| [ALIDNS](alidns.md) | ✅ | ✅ | ❌ | ✅ | ❌ |
| [AUTODNS](autodns.md) | ✅ | ❌ | ✅ | ✅ | ❌ |
| [AXFRDDNS](axfrddns.md) | ✅ | ❌ | ❌ | ❌ | ✅ |
| [AZURE_DNS](azuredns.md) | ✅ | ✅ | ✅ | ✅ | ❌ |
| [AZURE_PRIVATE_DNS](azureprivatedns.md) | ✅ | ❌ | ❌ | ✅ | ❌ |
| [BIND](bind.md) | ✅ | ✅ | ✅ | ✅ | ✅ |
| [BUNNY_DNS](bunnydns.md) | ✅ | ❌ | ✅ | ✅ | ✅ |
| [CLOUDFLAREAPI](cloudflareapi.md) | ✅ | ❌ | ✅ | ✅ | ❌ |
| [CLOUDNS](cloudns.md) | ✅ | ✅ | ✅ | ✅ | ✅ |
| [CLOUDPRESS](cloudpress.md) | ✅ | ❌ | ✅ | ✅ | ✅ |
| [CNR](cnr.md) | ✅ | ✅ | ✅ | ✅ | ✅ |
| [CONSTELLIX](constellix.md) | ✅ | ✅ | ✅ | ✅ | ❌ |
| [CSCGLOBAL](cscglobal.md) | ✅ | ✅ | ❌ | ✅ | ❌ |
| [DESEC](desec.md) | ✅ | ❔ | ✅ | ✅ | ✅ |
| [DIGITALOCEAN](digitalocean.md) | ✅ | ✅ | ✅ | ✅ | ❌ |
| [DNSCALE](dnscale.md) | ❌ | ✅ | ✅ | ❌ | ❌ |
| [DNSIMPLE](dnsimple.md) | ✅ | ❌ | ❌ | ✅ | ✅ |
| [DNSMADEEASY](dnsmadeeasy.md) | ✅ | ✅ | ✅ | ✅ | ❌ |
| [DNSOVERHTTPS](dnsoverhttps.md) | ✅ | ✅ | ❌ | ❌ | ❌ |
| [DOMAINNAMESHOP](domainnameshop.md) | ✅ | ❔ | ❌ | ❌ | ❌ |
| [DYNADOT](dynadot.md) | ✅ | ✅ | ❌ | ❌ | ❌ |
| [DYNU](dynu.md) | ❌ | ✅ | ❌ | ❌ | ❌ |
| [EASYNAME](easyname.md) | ✅ | ✅ | ❌ | ❌ | ❌ |
| [EXOSCALE](exoscale.md) | ❌ | ❌ | ✅ | ❌ | ❌ |
| [FORTIGATE](fortigate.md) | ✅ | ❌ | ✅ | ✅ | ❌ |
| [GANDI_V5](gandiv5.md) | ✅ | ✅ | ❌ | ✅ | ❌ |
| [GCLOUD](gcloud.md) | ✅ | ✅ | ✅ | ✅ | ✅ |
| [GCORE](gcore.md) | ✅ | ✅ | ✅ | ✅ | ✅ |
| [GIDINET](gidinet.md) | ✅ | ✅ | ❌ | ✅ | ❌ |
| [GIGAHOST](gigahost.md) | ✅ | ✅ | ❌ | ✅ | ❌ |
| [HEDNS](hedns.md) | ✅ | ✅ | ✅ | ✅ | ❌ |
| [HETZNER_V2](hetznerv2.md) | ✅ | ✅ | ✅ | ✅ | ❌ |
| [HOSTINGDE](hostingde.md) | ✅ | ✅ | ✅ | ✅ | ✅ |
| [HUAWEICLOUD](huaweicloud.md) | ✅ | ✅ | ✅ | ✅ | ✅ |
| [INFOBLOX](infoblox.md) | ✅ | ❌ | ❌ | ❌ | ❌ |
| [INFOMANIAK](infomaniak.md) | ✅ | ✅ | ❌ | ❌ | ❌ |
| [INTERNETBS](internetbs.md) | ✅ | ✅ | ❌ | ❌ | ❌ |
| [INWX](inwx.md) | ✅ | ✅ | ✅ | ✅ | ✅ |
| [JOKER](joker.md) | ❌ | ❌ | ✅ | ✅ | ❌ |
| [LINODE](linode.md) | ✅ | ❌ | ❌ | ❌ | ❌ |
| [LOOPIA](loopia.md) | ✅ | ✅ | ❌ | ✅ | ❌ |
| [LUADNS](luadns.md) | ✅ | ✅ | ✅ | ✅ | ❌ |
| [MIKROTIK](mikrotik.md) | ❌ | ❌ | ❌ | ✅ | ❌ |
| [MISAKA](misaka.md) | ✅ | ❌ | ✅ | ✅ | ❔ |
| [MITTWALD](mittwald.md) | ❌ | ❌ | ❌ | ❌ | ❌ |
| [MYTHICBEASTS](mythicbeasts.md) | ✅ | ✅ | ❌ | ❌ | ❌ |
| [NAMECHEAP](namecheap.md) | ✅ | ❌ | ❌ | ✅ | ❌ |
| [NAMEDOTCOM](namedotcom.md) | ✅ | ✅ | ❌ | ✅ | ❌ |
| [NETBIRD](netbird.md) | ✅ | ❌ | ✅ | ✅ | ❌ |
| [NETCUP](netcup.md) | ✅ | ❌ | ❌ | ❌ | ❌ |
| [NETLIFY](netlify.md) | ✅ | ❌ | ❌ | ✅ | ❌ |
| [NETNOD](netnod.md) | ✅ | ✅ | ✅ | ✅ | ❌ |
| [NEXDNS](nexdns.md) | ✅ | ❌ | ✅ | ✅ | ❔ |
| [NS1](ns1.md) | ✅ | ✅ | ✅ | ✅ | ✅ |
| [OPENPROVIDER](openprovider.md) | ✅ | ❌ | ✅ | ✅ | ❔ |
| [OPENSRS](opensrs.md) | ✅ | ✅ | ❌ | ❌ | ❌ |
| [OPENWRT](openwrt.md) | ✅ | ✅ | ❌ | ❌ | ❌ |
| [ORACLE](oracle.md) | ✅ | ✅ | ✅ | ✅ | ❌ |
| [OVH](ovh.md) | ✅ | ✅ | ❌ | ✅ | ❌ |
| [PACKETFRAME](packetframe.md) | ✅ | ❌ | ❌ | ❌ | ❌ |
| [PORKBUN](porkbun.md) | ✅ | ❌ | ❌ | ✅ | ❌ |
| [POWERDNS](powerdns.md) | ✅ | ✅ | ✅ | ✅ | ✅ |
| [REALTIMEREGISTER](realtimeregister.md) | ✅ | ❌ | ✅ | ✅ | ✅ |
| [ROUTE53](route53.md) | ✅ | ✅ | ✅ | ✅ | ❌ |
| [RWTH](rwth.md) | ✅ | ❌ | ❌ | ✅ | ❔ |
| [SAKURACLOUD](sakuracloud.md) | ✅ | ❌ | ✅ | ✅ | ❌ |
| [SCALEWAY](scaleway.md) | ✅ | ❌ | ❌ | ✅ | ❌ |
| [SOFTLAYER](softlayer.md) | ✅ | ✅ | ❌ | ❌ | ❌ |
| [SPACESHIP](spaceship.md) | ✅ | ❌ | ❌ | ✅ | ❌ |
| [TENCENTDNS](tencentdns.md) | ✅ | ✅ | ✅ | ✅ | ❌ |
| [TRANSIP](transip.md) | ✅ | ❌ | ❌ | ✅ | ❌ |
| [UNIFI](unifi.md) | ❌ | ✅ | ❌ | ❌ | ❌ |
| [VERCEL](vercel.md) | ✅ | ❌ | ❌ | ❌ | ❌ |
| [VULTR](vultr.md) | ✅ | ✅ | ✅ | ✅ | ✅ |
| [WEBSUPPORT](websupport.md) | ✅ | ❌ | ❌ | ❌ | ❌ |


### Supported Record Types by Provider <!--(section 3/3)-->

NOTE: "Unimplemented" means that the provider's API supports this record type but DNSControl's implementation does not (yet) implement it. "Not supported" means that the record type is not supported and the reason has been documented. A record type that is not listed is simply not supported.

- [ADGUARDHOME](adguardhome.md): [A](../language-reference/domain-modifiers/A.md) [AAAA](../language-reference/domain-modifiers/AAAA.md) [ALIAS](../language-reference/domain-modifiers/ALIAS.md) [CNAME](../language-reference/domain-modifiers/CNAME.md)
- [AKAMAIEDGEDNS](akamaiedgedns.md): [A](../language-reference/domain-modifiers/A.md) [AAAA](../language-reference/domain-modifiers/AAAA.md) [ALIAS](../language-reference/domain-modifiers/ALIAS.md) [CAA](../language-reference/domain-modifiers/CAA.md) [CNAME](../language-reference/domain-modifiers/CNAME.md) [LOC](../language-reference/domain-modifiers/LOC.md) [MX](../language-reference/domain-modifiers/MX.md) [NAPTR](../language-reference/domain-modifiers/NAPTR.md) [NS](../language-reference/domain-modifiers/NS.md) [PTR](../language-reference/domain-modifiers/PTR.md) [SRV](../language-reference/domain-modifiers/SRV.md) [SSHFP](../language-reference/domain-modifiers/SSHFP.md) [TLSA](../language-reference/domain-modifiers/TLSA.md) [TXT](../language-reference/domain-modifiers/TXT.md)
- [ALIDNS](alidns.md): [A](../language-reference/domain-modifiers/A.md) [AAAA](../language-reference/domain-modifiers/AAAA.md) [CAA](../language-reference/domain-modifiers/CAA.md) [CNAME](../language-reference/domain-modifiers/CNAME.md) [MX](../language-reference/domain-modifiers/MX.md) [NS](../language-reference/domain-modifiers/NS.md) [SRV](../language-reference/domain-modifiers/SRV.md) [TXT](../language-reference/domain-modifiers/TXT.md)
- [AUTODNS](autodns.md): [A](../language-reference/domain-modifiers/A.md) [AAAA](../language-reference/domain-modifiers/AAAA.md) [ALIAS](../language-reference/domain-modifiers/ALIAS.md) [CAA](../language-reference/domain-modifiers/CAA.md) [CNAME](../language-reference/domain-modifiers/CNAME.md) [MX](../language-reference/domain-modifiers/MX.md) [NS](../language-reference/domain-modifiers/NS.md) [PTR](../language-reference/domain-modifiers/PTR.md) [SRV](../language-reference/domain-modifiers/SRV.md) [TXT](../language-reference/domain-modifiers/TXT.md)
- [AXFRDDNS](axfrddns.md): [A](../language-reference/domain-modifiers/A.md) [AAAA](../language-reference/domain-modifiers/AAAA.md) [CAA](../language-reference/domain-modifiers/CAA.md) [CNAME](../language-reference/domain-modifiers/CNAME.md) [DHCID](../language-reference/domain-modifiers/DHCID.md) [DNAME](../language-reference/domain-modifiers/DNAME.md) [DS](../language-reference/domain-modifiers/DS.md) [HTTPS](../language-reference/domain-modifiers/HTTPS.md) [LOC](../language-reference/domain-modifiers/LOC.md) [MX](../language-reference/domain-modifiers/MX.md) [NAPTR](../language-reference/domain-modifiers/NAPTR.md) [NS](../language-reference/domain-modifiers/NS.md) [OPENPGPKEY](../language-reference/domain-modifiers/OPENPGPKEY.md) [PTR](../language-reference/domain-modifiers/PTR.md) [SMIMEA](../language-reference/domain-modifiers/SMIMEA.md) [SRV](../language-reference/domain-modifiers/SRV.md) [SSHFP](../language-reference/domain-modifiers/SSHFP.md) [SVCB](../language-reference/domain-modifiers/SVCB.md) [TLSA](../language-reference/domain-modifiers/TLSA.md) [TXT](../language-reference/domain-modifiers/TXT.md)
- [AZURE_DNS](azuredns.md): [A](../language-reference/domain-modifiers/A.md) [AAAA](../language-reference/domain-modifiers/AAAA.md) [CAA](../language-reference/domain-modifiers/CAA.md) [CNAME](../language-reference/domain-modifiers/CNAME.md) [MX](../language-reference/domain-modifiers/MX.md) [NS](../language-reference/domain-modifiers/NS.md) [PTR](../language-reference/domain-modifiers/PTR.md) [SRV](../language-reference/domain-modifiers/SRV.md) [TXT](../language-reference/domain-modifiers/TXT.md)
  - Not supported: [ALIAS](../language-reference/domain-modifiers/ALIAS.md) (Azure DNS does not provide a generic ALIAS functionality. Use AZURE_ALIAS instead.)
- [AZURE_PRIVATE_DNS](azureprivatedns.md): [A](../language-reference/domain-modifiers/A.md) [AAAA](../language-reference/domain-modifiers/AAAA.md) [CNAME](../language-reference/domain-modifiers/CNAME.md) [MX](../language-reference/domain-modifiers/MX.md) [PTR](../language-reference/domain-modifiers/PTR.md) [SRV](../language-reference/domain-modifiers/SRV.md) [TXT](../language-reference/domain-modifiers/TXT.md)
  - Not supported: [ALIAS](../language-reference/domain-modifiers/ALIAS.md) (Azure DNS does not provide a generic ALIAS functionality. Use AZURE_ALIAS instead.)
  - Not supported: [CAA](../language-reference/domain-modifiers/CAA.md) (Azure Private DNS does not support CAA records)
- [BIND](bind.md): [A](../language-reference/domain-modifiers/A.md) [AAAA](../language-reference/domain-modifiers/AAAA.md) [ALIAS](../language-reference/domain-modifiers/ALIAS.md) [CAA](../language-reference/domain-modifiers/CAA.md) [CNAME](../language-reference/domain-modifiers/CNAME.md) [CSYNC](../language-reference/domain-modifiers/CSYNC.md) [DHCID](../language-reference/domain-modifiers/DHCID.md) [DNAME](../language-reference/domain-modifiers/DNAME.md) [DNSKEY](../language-reference/domain-modifiers/DNSKEY.md) [DS](../language-reference/domain-modifiers/DS.md) [HTTPS](../language-reference/domain-modifiers/HTTPS.md) [LOC](../language-reference/domain-modifiers/LOC.md) [MX](../language-reference/domain-modifiers/MX.md) [NAPTR](../language-reference/domain-modifiers/NAPTR.md) [NS](../language-reference/domain-modifiers/NS.md) [OPENPGPKEY](../language-reference/domain-modifiers/OPENPGPKEY.md) [PTR](../language-reference/domain-modifiers/PTR.md) [RP](../language-reference/domain-modifiers/RP.md) [SMIMEA](../language-reference/domain-modifiers/SMIMEA.md) [SOA](../language-reference/domain-modifiers/SOA.md) [SRV](../language-reference/domain-modifiers/SRV.md) [SSHFP](../language-reference/domain-modifiers/SSHFP.md) [SVCB](../language-reference/domain-modifiers/SVCB.md) [TLSA](../language-reference/domain-modifiers/TLSA.md) [TXT](../language-reference/domain-modifiers/TXT.md)
- [BUNNY_DNS](bunnydns.md): [A](../language-reference/domain-modifiers/A.md) [AAAA](../language-reference/domain-modifiers/AAAA.md) [ALIAS](../language-reference/domain-modifiers/ALIAS.md) [CAA](../language-reference/domain-modifiers/CAA.md) [CNAME](../language-reference/domain-modifiers/CNAME.md) [HTTPS](../language-reference/domain-modifiers/HTTPS.md) [MX](../language-reference/domain-modifiers/MX.md) [NS](../language-reference/domain-modifiers/NS.md) [PTR](../language-reference/domain-modifiers/PTR.md) [SRV](../language-reference/domain-modifiers/SRV.md) [SVCB](../language-reference/domain-modifiers/SVCB.md) [TLSA](../language-reference/domain-modifiers/TLSA.md) [TXT](../language-reference/domain-modifiers/TXT.md)
- [CLOUDFLAREAPI](cloudflareapi.md): [A](../language-reference/domain-modifiers/A.md) [AAAA](../language-reference/domain-modifiers/AAAA.md) [ALIAS](../language-reference/domain-modifiers/ALIAS.md) [CAA](../language-reference/domain-modifiers/CAA.md) [CNAME](../language-reference/domain-modifiers/CNAME.md) [DS](../language-reference/domain-modifiers/DS.md) [HTTPS](../language-reference/domain-modifiers/HTTPS.md) [LOC](../language-reference/domain-modifiers/LOC.md) [MX](../language-reference/domain-modifiers/MX.md) [NAPTR](../language-reference/domain-modifiers/NAPTR.md) [NS](../language-reference/domain-modifiers/NS.md) [PTR](../language-reference/domain-modifiers/PTR.md) [SRV](../language-reference/domain-modifiers/SRV.md) [SSHFP](../language-reference/domain-modifiers/SSHFP.md) [SVCB](../language-reference/domain-modifiers/SVCB.md) [TLSA](../language-reference/domain-modifiers/TLSA.md) [TXT](../language-reference/domain-modifiers/TXT.md)
- [CLOUDNS](cloudns.md): [A](../language-reference/domain-modifiers/A.md) [AAAA](../language-reference/domain-modifiers/AAAA.md) [ALIAS](../language-reference/domain-modifiers/ALIAS.md) [CAA](../language-reference/domain-modifiers/CAA.md) [CNAME](../language-reference/domain-modifiers/CNAME.md) [DNAME](../language-reference/domain-modifiers/DNAME.md) [LOC](../language-reference/domain-modifiers/LOC.md) [MX](../language-reference/domain-modifiers/MX.md) [NAPTR](../language-reference/domain-modifiers/NAPTR.md) [NS](../language-reference/domain-modifiers/NS.md) [PTR](../language-reference/domain-modifiers/PTR.md) [SRV](../language-reference/domain-modifiers/SRV.md) [SSHFP](../language-reference/domain-modifiers/SSHFP.md) [TLSA](../language-reference/domain-modifiers/TLSA.md) [TXT](../language-reference/domain-modifiers/TXT.md)
  - Unimplemented: [SOA](../language-reference/domain-modifiers/SOA.md) (Supported by cloudns at a separate API endpoint (/dns/modify-soa.json), not implemented yet)
  - Not supported: [DS](../language-reference/domain-modifiers/DS.md) (Not supported for root, only for children)
- [CLOUDPRESS](cloudpress.md): [A](../language-reference/domain-modifiers/A.md) [AAAA](../language-reference/domain-modifiers/AAAA.md) [CAA](../language-reference/domain-modifiers/CAA.md) [CNAME](../language-reference/domain-modifiers/CNAME.md) [MX](../language-reference/domain-modifiers/MX.md) [NS](../language-reference/domain-modifiers/NS.md) [PTR](../language-reference/domain-modifiers/PTR.md) [SRV](../language-reference/domain-modifiers/SRV.md) [TXT](../language-reference/domain-modifiers/TXT.md)
- [CNR](cnr.md): [A](../language-reference/domain-modifiers/A.md) [AAAA](../language-reference/domain-modifiers/AAAA.md) [ALIAS](../language-reference/domain-modifiers/ALIAS.md) [CAA](../language-reference/domain-modifiers/CAA.md) [CNAME](../language-reference/domain-modifiers/CNAME.md) [DHCID](../language-reference/domain-modifiers/DHCID.md) [DNAME](../language-reference/domain-modifiers/DNAME.md) [LOC](../language-reference/domain-modifiers/LOC.md) [MX](../language-reference/domain-modifiers/MX.md) [NAPTR](../language-reference/domain-modifiers/NAPTR.md) [NS](../language-reference/domain-modifiers/NS.md) [PTR](../language-reference/domain-modifiers/PTR.md) [SMIMEA](../language-reference/domain-modifiers/SMIMEA.md) [SRV](../language-reference/domain-modifiers/SRV.md) [SSHFP](../language-reference/domain-modifiers/SSHFP.md) [SVCB](../language-reference/domain-modifiers/SVCB.md) [TLSA](../language-reference/domain-modifiers/TLSA.md) [TXT](../language-reference/domain-modifiers/TXT.md)
  - Unimplemented: [DNSKEY](../language-reference/domain-modifiers/DNSKEY.md) (Ask for this feature.)
  - Unimplemented: [DS](../language-reference/domain-modifiers/DS.md) (Ask for this feature.)
  - Not supported: [HTTPS](../language-reference/domain-modifiers/HTTPS.md) (Managed via (Query|Add|Modify|Delete)WebFwd API call. Data not accessible via the resource records list. Hard to integrate this into DNSControl by that.)
  - Not supported: [SOA](../language-reference/domain-modifiers/SOA.md) (The SOA record is managed on the DNSZone directly. Data only accessible via StatusDNSZone Request, not via the resource records list. Hard to integrate this into DNSControl by that.)
- [CONSTELLIX](constellix.md): [A](../language-reference/domain-modifiers/A.md) [AAAA](../language-reference/domain-modifiers/AAAA.md) [ALIAS](../language-reference/domain-modifiers/ALIAS.md) [CAA](../language-reference/domain-modifiers/CAA.md) [CNAME](../language-reference/domain-modifiers/CNAME.md) [MX](../language-reference/domain-modifiers/MX.md) [NS](../language-reference/domain-modifiers/NS.md) [PTR](../language-reference/domain-modifiers/PTR.md) [SRV](../language-reference/domain-modifiers/SRV.md) [TXT](../language-reference/domain-modifiers/TXT.md)
  - Unimplemented: [SOA](../language-reference/domain-modifiers/SOA.md) (Constellix keeps the SOA on the zone object, not as a record)
- [CSCGLOBAL](cscglobal.md): [A](../language-reference/domain-modifiers/A.md) [AAAA](../language-reference/domain-modifiers/AAAA.md) [CAA](../language-reference/domain-modifiers/CAA.md) [CNAME](../language-reference/domain-modifiers/CNAME.md) [MX](../language-reference/domain-modifiers/MX.md) [NS](../language-reference/domain-modifiers/NS.md) [SRV](../language-reference/domain-modifiers/SRV.md) [TXT](../language-reference/domain-modifiers/TXT.md)
- [DESEC](desec.md): [A](../language-reference/domain-modifiers/A.md) [AAAA](../language-reference/domain-modifiers/AAAA.md) [CAA](../language-reference/domain-modifiers/CAA.md) [CNAME](../language-reference/domain-modifiers/CNAME.md) [DNSKEY](../language-reference/domain-modifiers/DNSKEY.md) [HTTPS](../language-reference/domain-modifiers/HTTPS.md) [LOC](../language-reference/domain-modifiers/LOC.md) [MX](../language-reference/domain-modifiers/MX.md) [NAPTR](../language-reference/domain-modifiers/NAPTR.md) [NS](../language-reference/domain-modifiers/NS.md) [OPENPGPKEY](../language-reference/domain-modifiers/OPENPGPKEY.md) [PTR](../language-reference/domain-modifiers/PTR.md) [SMIMEA](../language-reference/domain-modifiers/SMIMEA.md) [SRV](../language-reference/domain-modifiers/SRV.md) [SSHFP](../language-reference/domain-modifiers/SSHFP.md) [SVCB](../language-reference/domain-modifiers/SVCB.md) [TLSA](../language-reference/domain-modifiers/TLSA.md) [TXT](../language-reference/domain-modifiers/TXT.md)
  - Unimplemented: [ALIAS](../language-reference/domain-modifiers/ALIAS.md) (Apex aliasing is supported via new SVCB and HTTPS record types. For details, check the deSEC docs.)
- [DIGITALOCEAN](digitalocean.md): [A](../language-reference/domain-modifiers/A.md) [AAAA](../language-reference/domain-modifiers/AAAA.md) [CAA](../language-reference/domain-modifiers/CAA.md) [CNAME](../language-reference/domain-modifiers/CNAME.md) [MX](../language-reference/domain-modifiers/MX.md) [NS](../language-reference/domain-modifiers/NS.md) [SRV](../language-reference/domain-modifiers/SRV.md) [TXT](../language-reference/domain-modifiers/TXT.md)
  - Not supported: [ALIAS](../language-reference/domain-modifiers/ALIAS.md) (Digital Ocean documents that this is not supported.)
  - Not supported: [DHCID](../language-reference/domain-modifiers/DHCID.md) (Digital Ocean documents that this is not supported.)
  - Not supported: [DNAME](../language-reference/domain-modifiers/DNAME.md) (Digital Ocean documents that this is not supported.)
  - Not supported: [DNSKEY](../language-reference/domain-modifiers/DNSKEY.md) (Digital Ocean documents that this is not supported.)
  - Not supported: [DS](../language-reference/domain-modifiers/DS.md) (Digital Ocean documents that this is not supported.)
  - Not supported: [HTTPS](../language-reference/domain-modifiers/HTTPS.md) (Digital Ocean documents that this is not supported.)
  - Not supported: [NAPTR](../language-reference/domain-modifiers/NAPTR.md) (Digital Ocean documents that this is not supported.)
  - Not supported: [PTR](../language-reference/domain-modifiers/PTR.md) (Digital Ocean documents that this is not supported.)
  - Not supported: [SMIMEA](../language-reference/domain-modifiers/SMIMEA.md) (Digital Ocean documents that this is not supported.)
  - Not supported: [SOA](../language-reference/domain-modifiers/SOA.md) (Technically SOA is supported but in reality the API only permits updates to the TTL. That is insufficient for DNSControl to claim 'support')
  - Not supported: [SSHFP](../language-reference/domain-modifiers/SSHFP.md) (Digital Ocean documents that this is not supported.)
  - Not supported: [SVCB](../language-reference/domain-modifiers/SVCB.md) (Digital Ocean documents that this is not supported.)
  - Not supported: [TLSA](../language-reference/domain-modifiers/TLSA.md) (Digital Ocean documents that this is not supported.)
- [DNSCALE](dnscale.md): [A](../language-reference/domain-modifiers/A.md) [AAAA](../language-reference/domain-modifiers/AAAA.md) [ALIAS](../language-reference/domain-modifiers/ALIAS.md) [CAA](../language-reference/domain-modifiers/CAA.md) [CNAME](../language-reference/domain-modifiers/CNAME.md) [HTTPS](../language-reference/domain-modifiers/HTTPS.md) [MX](../language-reference/domain-modifiers/MX.md) [NS](../language-reference/domain-modifiers/NS.md) [PTR](../language-reference/domain-modifiers/PTR.md) [SRV](../language-reference/domain-modifiers/SRV.md) [SSHFP](../language-reference/domain-modifiers/SSHFP.md) [SVCB](../language-reference/domain-modifiers/SVCB.md) [TLSA](../language-reference/domain-modifiers/TLSA.md) [TXT](../language-reference/domain-modifiers/TXT.md)
- [DNSIMPLE](dnsimple.md): [A](../language-reference/domain-modifiers/A.md) [AAAA](../language-reference/domain-modifiers/AAAA.md) [ALIAS](../language-reference/domain-modifiers/ALIAS.md) [CAA](../language-reference/domain-modifiers/CAA.md) [CNAME](../language-reference/domain-modifiers/CNAME.md) [HTTPS](../language-reference/domain-modifiers/HTTPS.md) [MX](../language-reference/domain-modifiers/MX.md) [NAPTR](../language-reference/domain-modifiers/NAPTR.md) [NS](../language-reference/domain-modifiers/NS.md) [PTR](../language-reference/domain-modifiers/PTR.md) [SRV](../language-reference/domain-modifiers/SRV.md) [SSHFP](../language-reference/domain-modifiers/SSHFP.md) [SVCB](../language-reference/domain-modifiers/SVCB.md) [TLSA](../language-reference/domain-modifiers/TLSA.md) [TXT](../language-reference/domain-modifiers/TXT.md)
- [DNSMADEEASY](dnsmadeeasy.md): [A](../language-reference/domain-modifiers/A.md) [AAAA](../language-reference/domain-modifiers/AAAA.md) [ALIAS](../language-reference/domain-modifiers/ALIAS.md) [CAA](../language-reference/domain-modifiers/CAA.md) [CNAME](../language-reference/domain-modifiers/CNAME.md) [MX](../language-reference/domain-modifiers/MX.md) [NS](../language-reference/domain-modifiers/NS.md) [PTR](../language-reference/domain-modifiers/PTR.md) [SRV](../language-reference/domain-modifiers/SRV.md) [TXT](../language-reference/domain-modifiers/TXT.md)
- [DOMAINNAMESHOP](domainnameshop.md): [A](../language-reference/domain-modifiers/A.md) [AAAA](../language-reference/domain-modifiers/AAAA.md) [CAA](../language-reference/domain-modifiers/CAA.md) [CNAME](../language-reference/domain-modifiers/CNAME.md) [MX](../language-reference/domain-modifiers/MX.md) [NS](../language-reference/domain-modifiers/NS.md) [SRV](../language-reference/domain-modifiers/SRV.md) [TXT](../language-reference/domain-modifiers/TXT.md)
  - Unimplemented: [ALIAS](../language-reference/domain-modifiers/ALIAS.md) (Needs custom implementation)
  - Unimplemented: [DS](../language-reference/domain-modifiers/DS.md)
  - Unimplemented: [TLSA](../language-reference/domain-modifiers/TLSA.md) (Has support but no documentation. Needs to be investigated.)
  - Not supported: [NAPTR](../language-reference/domain-modifiers/NAPTR.md) (According to Domainnameshop this will probably never be supported)
  - Not supported: [PTR](../language-reference/domain-modifiers/PTR.md) (According to Domainnameshop this will probably never be supported)
  - Not supported: [SSHFP](../language-reference/domain-modifiers/SSHFP.md) (Might be supported in the future)
- [DYNU](dynu.md): [A](../language-reference/domain-modifiers/A.md) [AAAA](../language-reference/domain-modifiers/AAAA.md) [CAA](../language-reference/domain-modifiers/CAA.md) [CNAME](../language-reference/domain-modifiers/CNAME.md) [DHCID](../language-reference/domain-modifiers/DHCID.md) [DNAME](../language-reference/domain-modifiers/DNAME.md) [HTTPS](../language-reference/domain-modifiers/HTTPS.md) [LOC](../language-reference/domain-modifiers/LOC.md) [MX](../language-reference/domain-modifiers/MX.md) [NAPTR](../language-reference/domain-modifiers/NAPTR.md) [NS](../language-reference/domain-modifiers/NS.md) [OPENPGPKEY](../language-reference/domain-modifiers/OPENPGPKEY.md) [PTR](../language-reference/domain-modifiers/PTR.md) [RP](../language-reference/domain-modifiers/RP.md) [SMIMEA](../language-reference/domain-modifiers/SMIMEA.md) [SRV](../language-reference/domain-modifiers/SRV.md) [SSHFP](../language-reference/domain-modifiers/SSHFP.md) [SVCB](../language-reference/domain-modifiers/SVCB.md) [TLSA](../language-reference/domain-modifiers/TLSA.md) [TXT](../language-reference/domain-modifiers/TXT.md)
- [EXOSCALE](exoscale.md): [A](../language-reference/domain-modifiers/A.md) [AAAA](../language-reference/domain-modifiers/AAAA.md) [ALIAS](../language-reference/domain-modifiers/ALIAS.md) [CAA](../language-reference/domain-modifiers/CAA.md) [CNAME](../language-reference/domain-modifiers/CNAME.md) [MX](../language-reference/domain-modifiers/MX.md) [SRV](../language-reference/domain-modifiers/SRV.md) [TXT](../language-reference/domain-modifiers/TXT.md)
- [FORTIGATE](fortigate.md): [A](../language-reference/domain-modifiers/A.md) [AAAA](../language-reference/domain-modifiers/AAAA.md) [CNAME](../language-reference/domain-modifiers/CNAME.md)
  - Not supported: [PTR](../language-reference/domain-modifiers/PTR.md) (does not really support ARPA Zones and handles PTR records weirdly)
- [GANDI_V5](gandiv5.md): [A](../language-reference/domain-modifiers/A.md) [AAAA](../language-reference/domain-modifiers/AAAA.md) [ALIAS](../language-reference/domain-modifiers/ALIAS.md) [CAA](../language-reference/domain-modifiers/CAA.md) [CNAME](../language-reference/domain-modifiers/CNAME.md) [HTTPS](../language-reference/domain-modifiers/HTTPS.md) [LOC](../language-reference/domain-modifiers/LOC.md) [MX](../language-reference/domain-modifiers/MX.md) [NAPTR](../language-reference/domain-modifiers/NAPTR.md) [NS](../language-reference/domain-modifiers/NS.md) [OPENPGPKEY](../language-reference/domain-modifiers/OPENPGPKEY.md) [PTR](../language-reference/domain-modifiers/PTR.md) [RP](../language-reference/domain-modifiers/RP.md) [SRV](../language-reference/domain-modifiers/SRV.md) [SSHFP](../language-reference/domain-modifiers/SSHFP.md) [SVCB](../language-reference/domain-modifiers/SVCB.md) [TLSA](../language-reference/domain-modifiers/TLSA.md) [TXT](../language-reference/domain-modifiers/TXT.md)
  - Not supported: [DS](../language-reference/domain-modifiers/DS.md) (Only supports DS records at the apex)
- [GCLOUD](gcloud.md): [A](../language-reference/domain-modifiers/A.md) [AAAA](../language-reference/domain-modifiers/AAAA.md) [ALIAS](../language-reference/domain-modifiers/ALIAS.md) [CAA](../language-reference/domain-modifiers/CAA.md) [CNAME](../language-reference/domain-modifiers/CNAME.md) [HTTPS](../language-reference/domain-modifiers/HTTPS.md) [MX](../language-reference/domain-modifiers/MX.md) [NS](../language-reference/domain-modifiers/NS.md) [PTR](../language-reference/domain-modifiers/PTR.md) [SRV](../language-reference/domain-modifiers/SRV.md) [SSHFP](../language-reference/domain-modifiers/SSHFP.md) [SVCB](../language-reference/domain-modifiers/SVCB.md) [TLSA](../language-reference/domain-modifiers/TLSA.md) [TXT](../language-reference/domain-modifiers/TXT.md)
- [GCORE](gcore.md): [A](../language-reference/domain-modifiers/A.md) [AAAA](../language-reference/domain-modifiers/AAAA.md) [ALIAS](../language-reference/domain-modifiers/ALIAS.md) [CAA](../language-reference/domain-modifiers/CAA.md) [CNAME](../language-reference/domain-modifiers/CNAME.md) [HTTPS](../language-reference/domain-modifiers/HTTPS.md) [MX](../language-reference/domain-modifiers/MX.md) [NS](../language-reference/domain-modifiers/NS.md) [PTR](../language-reference/domain-modifiers/PTR.md) [SRV](../language-reference/domain-modifiers/SRV.md) [SVCB](../language-reference/domain-modifiers/SVCB.md) [TXT](../language-reference/domain-modifiers/TXT.md)
- [GIDINET](gidinet.md): [A](../language-reference/domain-modifiers/A.md) [AAAA](../language-reference/domain-modifiers/AAAA.md) [CNAME](../language-reference/domain-modifiers/CNAME.md) [MX](../language-reference/domain-modifiers/MX.md) [NS](../language-reference/domain-modifiers/NS.md) [SRV](../language-reference/domain-modifiers/SRV.md) [TXT](../language-reference/domain-modifiers/TXT.md)
  - Not supported: [CAA](../language-reference/domain-modifiers/CAA.md) (Only premium service)
- [GIGAHOST](gigahost.md): [A](../language-reference/domain-modifiers/A.md) [AAAA](../language-reference/domain-modifiers/AAAA.md) [ALIAS](../language-reference/domain-modifiers/ALIAS.md) [CAA](../language-reference/domain-modifiers/CAA.md) [CNAME](../language-reference/domain-modifiers/CNAME.md) [DNAME](../language-reference/domain-modifiers/DNAME.md) [MX](../language-reference/domain-modifiers/MX.md) [NAPTR](../language-reference/domain-modifiers/NAPTR.md) [NS](../language-reference/domain-modifiers/NS.md) [PTR](../language-reference/domain-modifiers/PTR.md) [SRV](../language-reference/domain-modifiers/SRV.md) [TXT](../language-reference/domain-modifiers/TXT.md)
- [HEDNS](hedns.md): [A](../language-reference/domain-modifiers/A.md) [AAAA](../language-reference/domain-modifiers/AAAA.md) [ALIAS](../language-reference/domain-modifiers/ALIAS.md) [CAA](../language-reference/domain-modifiers/CAA.md) [CNAME](../language-reference/domain-modifiers/CNAME.md) [HTTPS](../language-reference/domain-modifiers/HTTPS.md) [LOC](../language-reference/domain-modifiers/LOC.md) [MX](../language-reference/domain-modifiers/MX.md) [NAPTR](../language-reference/domain-modifiers/NAPTR.md) [NS](../language-reference/domain-modifiers/NS.md) [PTR](../language-reference/domain-modifiers/PTR.md) [RP](../language-reference/domain-modifiers/RP.md) [SRV](../language-reference/domain-modifiers/SRV.md) [SSHFP](../language-reference/domain-modifiers/SSHFP.md) [SVCB](../language-reference/domain-modifiers/SVCB.md) [TXT](../language-reference/domain-modifiers/TXT.md)
- [HETZNER_V2](hetznerv2.md): [A](../language-reference/domain-modifiers/A.md) [AAAA](../language-reference/domain-modifiers/AAAA.md) [CAA](../language-reference/domain-modifiers/CAA.md) [CNAME](../language-reference/domain-modifiers/CNAME.md) [DS](../language-reference/domain-modifiers/DS.md) [HTTPS](../language-reference/domain-modifiers/HTTPS.md) [MX](../language-reference/domain-modifiers/MX.md) [NS](../language-reference/domain-modifiers/NS.md) [PTR](../language-reference/domain-modifiers/PTR.md) [SRV](../language-reference/domain-modifiers/SRV.md) [SVCB](../language-reference/domain-modifiers/SVCB.md) [TLSA](../language-reference/domain-modifiers/TLSA.md) [TXT](../language-reference/domain-modifiers/TXT.md)
- [HOSTINGDE](hostingde.md): [A](../language-reference/domain-modifiers/A.md) [AAAA](../language-reference/domain-modifiers/AAAA.md) [ALIAS](../language-reference/domain-modifiers/ALIAS.md) [CAA](../language-reference/domain-modifiers/CAA.md) [CNAME](../language-reference/domain-modifiers/CNAME.md) [DS](../language-reference/domain-modifiers/DS.md) [MX](../language-reference/domain-modifiers/MX.md) [NS](../language-reference/domain-modifiers/NS.md) [PTR](../language-reference/domain-modifiers/PTR.md) [SOA](../language-reference/domain-modifiers/SOA.md) [SRV](../language-reference/domain-modifiers/SRV.md) [SSHFP](../language-reference/domain-modifiers/SSHFP.md) [TLSA](../language-reference/domain-modifiers/TLSA.md) [TXT](../language-reference/domain-modifiers/TXT.md)
- [HUAWEICLOUD](huaweicloud.md): [A](../language-reference/domain-modifiers/A.md) [AAAA](../language-reference/domain-modifiers/AAAA.md) [CAA](../language-reference/domain-modifiers/CAA.md) [CNAME](../language-reference/domain-modifiers/CNAME.md) [MX](../language-reference/domain-modifiers/MX.md) [NS](../language-reference/domain-modifiers/NS.md) [SRV](../language-reference/domain-modifiers/SRV.md) [TXT](../language-reference/domain-modifiers/TXT.md)
- [INFOBLOX](infoblox.md): [A](../language-reference/domain-modifiers/A.md) [AAAA](../language-reference/domain-modifiers/AAAA.md) [CAA](../language-reference/domain-modifiers/CAA.md) [CNAME](../language-reference/domain-modifiers/CNAME.md) [MX](../language-reference/domain-modifiers/MX.md) [PTR](../language-reference/domain-modifiers/PTR.md) [SRV](../language-reference/domain-modifiers/SRV.md) [TXT](../language-reference/domain-modifiers/TXT.md)
- [INFOMANIAK](infomaniak.md): [A](../language-reference/domain-modifiers/A.md) [AAAA](../language-reference/domain-modifiers/AAAA.md) [CAA](../language-reference/domain-modifiers/CAA.md) [CNAME](../language-reference/domain-modifiers/CNAME.md) [DNAME](../language-reference/domain-modifiers/DNAME.md) [DS](../language-reference/domain-modifiers/DS.md) [MX](../language-reference/domain-modifiers/MX.md) [NS](../language-reference/domain-modifiers/NS.md) [SRV](../language-reference/domain-modifiers/SRV.md) [SSHFP](../language-reference/domain-modifiers/SSHFP.md) [TLSA](../language-reference/domain-modifiers/TLSA.md) [TXT](../language-reference/domain-modifiers/TXT.md)
- [INWX](inwx.md): [A](../language-reference/domain-modifiers/A.md) [AAAA](../language-reference/domain-modifiers/AAAA.md) [ALIAS](../language-reference/domain-modifiers/ALIAS.md) [CAA](../language-reference/domain-modifiers/CAA.md) [CNAME](../language-reference/domain-modifiers/CNAME.md) [HTTPS](../language-reference/domain-modifiers/HTTPS.md) [MX](../language-reference/domain-modifiers/MX.md) [NAPTR](../language-reference/domain-modifiers/NAPTR.md) [NS](../language-reference/domain-modifiers/NS.md) [PTR](../language-reference/domain-modifiers/PTR.md) [SRV](../language-reference/domain-modifiers/SRV.md) [SSHFP](../language-reference/domain-modifiers/SSHFP.md) [SVCB](../language-reference/domain-modifiers/SVCB.md) [TLSA](../language-reference/domain-modifiers/TLSA.md) [TXT](../language-reference/domain-modifiers/TXT.md)
  - Unimplemented: [DS](../language-reference/domain-modifiers/DS.md) (DS records are only supported at the apex and require a different API call that hasn't been implemented yet.)
  - Unimplemented: [LOC](../language-reference/domain-modifiers/LOC.md)
- [JOKER](joker.md): [A](../language-reference/domain-modifiers/A.md) [AAAA](../language-reference/domain-modifiers/AAAA.md) [CAA](../language-reference/domain-modifiers/CAA.md) [CNAME](../language-reference/domain-modifiers/CNAME.md) [MX](../language-reference/domain-modifiers/MX.md) [NAPTR](../language-reference/domain-modifiers/NAPTR.md) [NS](../language-reference/domain-modifiers/NS.md) [SRV](../language-reference/domain-modifiers/SRV.md) [TXT](../language-reference/domain-modifiers/TXT.md)
- [LINODE](linode.md): [A](../language-reference/domain-modifiers/A.md) [AAAA](../language-reference/domain-modifiers/AAAA.md) [CAA](../language-reference/domain-modifiers/CAA.md) [CNAME](../language-reference/domain-modifiers/CNAME.md) [MX](../language-reference/domain-modifiers/MX.md) [NS](../language-reference/domain-modifiers/NS.md) [SRV](../language-reference/domain-modifiers/SRV.md) [TXT](../language-reference/domain-modifiers/TXT.md)
- [LOOPIA](loopia.md): [A](../language-reference/domain-modifiers/A.md) [AAAA](../language-reference/domain-modifiers/AAAA.md) [CAA](../language-reference/domain-modifiers/CAA.md) [CNAME](../language-reference/domain-modifiers/CNAME.md) [LOC](../language-reference/domain-modifiers/LOC.md) [MX](../language-reference/domain-modifiers/MX.md) [NAPTR](../language-reference/domain-modifiers/NAPTR.md) [NS](../language-reference/domain-modifiers/NS.md) [SRV](../language-reference/domain-modifiers/SRV.md) [SSHFP](../language-reference/domain-modifiers/SSHFP.md) [TLSA](../language-reference/domain-modifiers/TLSA.md) [TXT](../language-reference/domain-modifiers/TXT.md)
  - Not supported: [DS](../language-reference/domain-modifiers/DS.md) (Only supports DS records at the apex, only for .se and .nu domains; done automatically at back-end.)
  - Not supported: [SOA](../language-reference/domain-modifiers/SOA.md) (💩)
- [LUADNS](luadns.md): [A](../language-reference/domain-modifiers/A.md) [AAAA](../language-reference/domain-modifiers/AAAA.md) [ALIAS](../language-reference/domain-modifiers/ALIAS.md) [CAA](../language-reference/domain-modifiers/CAA.md) [CNAME](../language-reference/domain-modifiers/CNAME.md) [HTTPS](../language-reference/domain-modifiers/HTTPS.md) [MX](../language-reference/domain-modifiers/MX.md) [NS](../language-reference/domain-modifiers/NS.md) [OPENPGPKEY](../language-reference/domain-modifiers/OPENPGPKEY.md) [PTR](../language-reference/domain-modifiers/PTR.md) [SRV](../language-reference/domain-modifiers/SRV.md) [SSHFP](../language-reference/domain-modifiers/SSHFP.md) [TLSA](../language-reference/domain-modifiers/TLSA.md) [TXT](../language-reference/domain-modifiers/TXT.md)
- [MIKROTIK](mikrotik.md): [A](../language-reference/domain-modifiers/A.md) [AAAA](../language-reference/domain-modifiers/AAAA.md) [CNAME](../language-reference/domain-modifiers/CNAME.md) [MX](../language-reference/domain-modifiers/MX.md) [NS](../language-reference/domain-modifiers/NS.md) [SRV](../language-reference/domain-modifiers/SRV.md) [TXT](../language-reference/domain-modifiers/TXT.md)
- [MISAKA](misaka.md): [A](../language-reference/domain-modifiers/A.md) [AAAA](../language-reference/domain-modifiers/AAAA.md) [ALIAS](../language-reference/domain-modifiers/ALIAS.md) [CAA](../language-reference/domain-modifiers/CAA.md) [CNAME](../language-reference/domain-modifiers/CNAME.md) [DNAME](../language-reference/domain-modifiers/DNAME.md) [MX](../language-reference/domain-modifiers/MX.md) [NS](../language-reference/domain-modifiers/NS.md) [PTR](../language-reference/domain-modifiers/PTR.md) [SRV](../language-reference/domain-modifiers/SRV.md) [SSHFP](../language-reference/domain-modifiers/SSHFP.md) [TLSA](../language-reference/domain-modifiers/TLSA.md) [TXT](../language-reference/domain-modifiers/TXT.md)
  - Not supported: [SOA](../language-reference/domain-modifiers/SOA.md) (Managed by Misaka.io)
- [MITTWALD](mittwald.md): [A](../language-reference/domain-modifiers/A.md) [AAAA](../language-reference/domain-modifiers/AAAA.md) [CAA](../language-reference/domain-modifiers/CAA.md) [CNAME](../language-reference/domain-modifiers/CNAME.md) [MX](../language-reference/domain-modifiers/MX.md) [SRV](../language-reference/domain-modifiers/SRV.md) [TXT](../language-reference/domain-modifiers/TXT.md)
- [MYTHICBEASTS](mythicbeasts.md): [A](../language-reference/domain-modifiers/A.md) [AAAA](../language-reference/domain-modifiers/AAAA.md) [CAA](../language-reference/domain-modifiers/CAA.md) [CNAME](../language-reference/domain-modifiers/CNAME.md) [MX](../language-reference/domain-modifiers/MX.md) [NS](../language-reference/domain-modifiers/NS.md) [PTR](../language-reference/domain-modifiers/PTR.md) [SRV](../language-reference/domain-modifiers/SRV.md) [SSHFP](../language-reference/domain-modifiers/SSHFP.md) [TLSA](../language-reference/domain-modifiers/TLSA.md) [TXT](../language-reference/domain-modifiers/TXT.md)
- [NAMECHEAP](namecheap.md): [A](../language-reference/domain-modifiers/A.md) [AAAA](../language-reference/domain-modifiers/AAAA.md) [ALIAS](../language-reference/domain-modifiers/ALIAS.md) [CAA](../language-reference/domain-modifiers/CAA.md) [CNAME](../language-reference/domain-modifiers/CNAME.md) [MX](../language-reference/domain-modifiers/MX.md) [NS](../language-reference/domain-modifiers/NS.md) [SRV](../language-reference/domain-modifiers/SRV.md) [TXT](../language-reference/domain-modifiers/TXT.md)
- [NAMEDOTCOM](namedotcom.md): [A](../language-reference/domain-modifiers/A.md) [AAAA](../language-reference/domain-modifiers/AAAA.md) [ALIAS](../language-reference/domain-modifiers/ALIAS.md) [CNAME](../language-reference/domain-modifiers/CNAME.md) [MX](../language-reference/domain-modifiers/MX.md) [NS](../language-reference/domain-modifiers/NS.md) [SRV](../language-reference/domain-modifiers/SRV.md) [TXT](../language-reference/domain-modifiers/TXT.md)
  - Not supported: [PTR](../language-reference/domain-modifiers/PTR.md) (PTR records are not supported (See Link): https://www.name.com/support/articles/205188508-Reverse-DNS-records)
- [NETBIRD](netbird.md): [A](../language-reference/domain-modifiers/A.md) [AAAA](../language-reference/domain-modifiers/AAAA.md) [CNAME](../language-reference/domain-modifiers/CNAME.md)
- [NETCUP](netcup.md): [A](../language-reference/domain-modifiers/A.md) [AAAA](../language-reference/domain-modifiers/AAAA.md) [CAA](../language-reference/domain-modifiers/CAA.md) [CNAME](../language-reference/domain-modifiers/CNAME.md) [MX](../language-reference/domain-modifiers/MX.md) [SRV](../language-reference/domain-modifiers/SRV.md) [TLSA](../language-reference/domain-modifiers/TLSA.md) [TXT](../language-reference/domain-modifiers/TXT.md)
- [NETLIFY](netlify.md): [A](../language-reference/domain-modifiers/A.md) [AAAA](../language-reference/domain-modifiers/AAAA.md) [ALIAS](../language-reference/domain-modifiers/ALIAS.md) [CAA](../language-reference/domain-modifiers/CAA.md) [CNAME](../language-reference/domain-modifiers/CNAME.md) [MX](../language-reference/domain-modifiers/MX.md) [NS](../language-reference/domain-modifiers/NS.md) [SRV](../language-reference/domain-modifiers/SRV.md) [TXT](../language-reference/domain-modifiers/TXT.md)
- [NETNOD](netnod.md): [A](../language-reference/domain-modifiers/A.md) [AAAA](../language-reference/domain-modifiers/AAAA.md) [ALIAS](../language-reference/domain-modifiers/ALIAS.md) [CAA](../language-reference/domain-modifiers/CAA.md) [CNAME](../language-reference/domain-modifiers/CNAME.md) [CSYNC](../language-reference/domain-modifiers/CSYNC.md) [HTTPS](../language-reference/domain-modifiers/HTTPS.md) [MX](../language-reference/domain-modifiers/MX.md) [NAPTR](../language-reference/domain-modifiers/NAPTR.md) [NS](../language-reference/domain-modifiers/NS.md) [PTR](../language-reference/domain-modifiers/PTR.md) [SRV](../language-reference/domain-modifiers/SRV.md) [SSHFP](../language-reference/domain-modifiers/SSHFP.md) [SVCB](../language-reference/domain-modifiers/SVCB.md) [TLSA](../language-reference/domain-modifiers/TLSA.md) [TXT](../language-reference/domain-modifiers/TXT.md)
- [NEXDNS](nexdns.md): [A](../language-reference/domain-modifiers/A.md) [AAAA](../language-reference/domain-modifiers/AAAA.md) [ALIAS](../language-reference/domain-modifiers/ALIAS.md) [CAA](../language-reference/domain-modifiers/CAA.md) [CNAME](../language-reference/domain-modifiers/CNAME.md) [DNAME](../language-reference/domain-modifiers/DNAME.md) [MX](../language-reference/domain-modifiers/MX.md) [NS](../language-reference/domain-modifiers/NS.md) [PTR](../language-reference/domain-modifiers/PTR.md) [SRV](../language-reference/domain-modifiers/SRV.md) [TLSA](../language-reference/domain-modifiers/TLSA.md) [TXT](../language-reference/domain-modifiers/TXT.md)
  - Not supported: [DS](../language-reference/domain-modifiers/DS.md) (DS at the zone apex belongs in the parent zone)
  - Not supported: [SOA](../language-reference/domain-modifiers/SOA.md) (The SOA record is maintained by the platform)
- [NS1](ns1.md): [A](../language-reference/domain-modifiers/A.md) [AAAA](../language-reference/domain-modifiers/AAAA.md) [ALIAS](../language-reference/domain-modifiers/ALIAS.md) [CAA](../language-reference/domain-modifiers/CAA.md) [CNAME](../language-reference/domain-modifiers/CNAME.md) [DHCID](../language-reference/domain-modifiers/DHCID.md) [DNAME](../language-reference/domain-modifiers/DNAME.md) [DS](../language-reference/domain-modifiers/DS.md) [HTTPS](../language-reference/domain-modifiers/HTTPS.md) [MX](../language-reference/domain-modifiers/MX.md) [NAPTR](../language-reference/domain-modifiers/NAPTR.md) [NS](../language-reference/domain-modifiers/NS.md) [PTR](../language-reference/domain-modifiers/PTR.md) [SRV](../language-reference/domain-modifiers/SRV.md) [SVCB](../language-reference/domain-modifiers/SVCB.md) [TLSA](../language-reference/domain-modifiers/TLSA.md) [TXT](../language-reference/domain-modifiers/TXT.md)
- [OPENPROVIDER](openprovider.md): [A](../language-reference/domain-modifiers/A.md) [AAAA](../language-reference/domain-modifiers/AAAA.md) [CAA](../language-reference/domain-modifiers/CAA.md) [CNAME](../language-reference/domain-modifiers/CNAME.md) [MX](../language-reference/domain-modifiers/MX.md) [SRV](../language-reference/domain-modifiers/SRV.md) [TLSA](../language-reference/domain-modifiers/TLSA.md) [TXT](../language-reference/domain-modifiers/TXT.md)
  - Not supported: [DNSKEY](../language-reference/domain-modifiers/DNSKEY.md) (DNSSEC keys are managed by Openprovider)
  - Not supported: [SOA](../language-reference/domain-modifiers/SOA.md) (The SOA record is managed by Openprovider)
- [OPENWRT](openwrt.md): [A](../language-reference/domain-modifiers/A.md) [AAAA](../language-reference/domain-modifiers/AAAA.md) [CNAME](../language-reference/domain-modifiers/CNAME.md) [MX](../language-reference/domain-modifiers/MX.md) [SRV](../language-reference/domain-modifiers/SRV.md)
- [ORACLE](oracle.md): [A](../language-reference/domain-modifiers/A.md) [AAAA](../language-reference/domain-modifiers/AAAA.md) [ALIAS](../language-reference/domain-modifiers/ALIAS.md) [CAA](../language-reference/domain-modifiers/CAA.md) [CNAME](../language-reference/domain-modifiers/CNAME.md) [MX](../language-reference/domain-modifiers/MX.md) [NAPTR](../language-reference/domain-modifiers/NAPTR.md) [NS](../language-reference/domain-modifiers/NS.md) [PTR](../language-reference/domain-modifiers/PTR.md) [SRV](../language-reference/domain-modifiers/SRV.md) [SSHFP](../language-reference/domain-modifiers/SSHFP.md) [TLSA](../language-reference/domain-modifiers/TLSA.md) [TXT](../language-reference/domain-modifiers/TXT.md)
  - Unimplemented: [LOC](../language-reference/domain-modifiers/LOC.md)
- [OVH](ovh.md): [A](../language-reference/domain-modifiers/A.md) [AAAA](../language-reference/domain-modifiers/AAAA.md) [CAA](../language-reference/domain-modifiers/CAA.md) [CNAME](../language-reference/domain-modifiers/CNAME.md) [MX](../language-reference/domain-modifiers/MX.md) [NS](../language-reference/domain-modifiers/NS.md) [SRV](../language-reference/domain-modifiers/SRV.md) [SSHFP](../language-reference/domain-modifiers/SSHFP.md) [TLSA](../language-reference/domain-modifiers/TLSA.md) [TXT](../language-reference/domain-modifiers/TXT.md)
  - Unimplemented: [LOC](../language-reference/domain-modifiers/LOC.md)
- [PACKETFRAME](packetframe.md): [A](../language-reference/domain-modifiers/A.md) [AAAA](../language-reference/domain-modifiers/AAAA.md) [CNAME](../language-reference/domain-modifiers/CNAME.md) [MX](../language-reference/domain-modifiers/MX.md) [NS](../language-reference/domain-modifiers/NS.md) [PTR](../language-reference/domain-modifiers/PTR.md) [SRV](../language-reference/domain-modifiers/SRV.md) [TXT](../language-reference/domain-modifiers/TXT.md)
- [PORKBUN](porkbun.md): [A](../language-reference/domain-modifiers/A.md) [AAAA](../language-reference/domain-modifiers/AAAA.md) [ALIAS](../language-reference/domain-modifiers/ALIAS.md) [CAA](../language-reference/domain-modifiers/CAA.md) [CNAME](../language-reference/domain-modifiers/CNAME.md) [HTTPS](../language-reference/domain-modifiers/HTTPS.md) [MX](../language-reference/domain-modifiers/MX.md) [NS](../language-reference/domain-modifiers/NS.md) [SRV](../language-reference/domain-modifiers/SRV.md) [SSHFP](../language-reference/domain-modifiers/SSHFP.md) [SVCB](../language-reference/domain-modifiers/SVCB.md) [TLSA](../language-reference/domain-modifiers/TLSA.md) [TXT](../language-reference/domain-modifiers/TXT.md)
- [POWERDNS](powerdns.md): [A](../language-reference/domain-modifiers/A.md) [AAAA](../language-reference/domain-modifiers/AAAA.md) [ALIAS](../language-reference/domain-modifiers/ALIAS.md) [CAA](../language-reference/domain-modifiers/CAA.md) [CNAME](../language-reference/domain-modifiers/CNAME.md) [DHCID](../language-reference/domain-modifiers/DHCID.md) [DNAME](../language-reference/domain-modifiers/DNAME.md) [DNSKEY](../language-reference/domain-modifiers/DNSKEY.md) [DS](../language-reference/domain-modifiers/DS.md) [HTTPS](../language-reference/domain-modifiers/HTTPS.md) [MX](../language-reference/domain-modifiers/MX.md) [NAPTR](../language-reference/domain-modifiers/NAPTR.md) [NS](../language-reference/domain-modifiers/NS.md) [OPENPGPKEY](../language-reference/domain-modifiers/OPENPGPKEY.md) [PTR](../language-reference/domain-modifiers/PTR.md) [SOA](../language-reference/domain-modifiers/SOA.md) [SRV](../language-reference/domain-modifiers/SRV.md) [SSHFP](../language-reference/domain-modifiers/SSHFP.md) [SVCB](../language-reference/domain-modifiers/SVCB.md) [TLSA](../language-reference/domain-modifiers/TLSA.md) [TXT](../language-reference/domain-modifiers/TXT.md)
  - Unimplemented: [LOC](../language-reference/domain-modifiers/LOC.md) (Normalization within the PowerDNS API seems to be buggy, so disabled: https://github.com/PowerDNS/pdns/issues/10558)
- [REALTIMEREGISTER](realtimeregister.md): [A](../language-reference/domain-modifiers/A.md) [AAAA](../language-reference/domain-modifiers/AAAA.md) [ALIAS](../language-reference/domain-modifiers/ALIAS.md) [CAA](../language-reference/domain-modifiers/CAA.md) [CNAME](../language-reference/domain-modifiers/CNAME.md) [LOC](../language-reference/domain-modifiers/LOC.md) [MX](../language-reference/domain-modifiers/MX.md) [NAPTR](../language-reference/domain-modifiers/NAPTR.md) [NS](../language-reference/domain-modifiers/NS.md) [SRV](../language-reference/domain-modifiers/SRV.md) [SSHFP](../language-reference/domain-modifiers/SSHFP.md) [TLSA](../language-reference/domain-modifiers/TLSA.md) [TXT](../language-reference/domain-modifiers/TXT.md)
  - Not supported: [DS](../language-reference/domain-modifiers/DS.md) (Only for subdomains)
- [ROUTE53](route53.md): [A](../language-reference/domain-modifiers/A.md) [AAAA](../language-reference/domain-modifiers/AAAA.md) [CAA](../language-reference/domain-modifiers/CAA.md) [CNAME](../language-reference/domain-modifiers/CNAME.md) [HTTPS](../language-reference/domain-modifiers/HTTPS.md) [MX](../language-reference/domain-modifiers/MX.md) [NS](../language-reference/domain-modifiers/NS.md) [PTR](../language-reference/domain-modifiers/PTR.md) [SOA](../language-reference/domain-modifiers/SOA.md) [SRV](../language-reference/domain-modifiers/SRV.md) [SSHFP](../language-reference/domain-modifiers/SSHFP.md) [SVCB](../language-reference/domain-modifiers/SVCB.md) [TLSA](../language-reference/domain-modifiers/TLSA.md) [TXT](../language-reference/domain-modifiers/TXT.md)
  - Not supported: [ALIAS](../language-reference/domain-modifiers/ALIAS.md) (R53 does not provide a generic ALIAS functionality. Use R53_ALIAS instead.)
- [RWTH](rwth.md): [A](../language-reference/domain-modifiers/A.md) [AAAA](../language-reference/domain-modifiers/AAAA.md) [CAA](../language-reference/domain-modifiers/CAA.md) [CNAME](../language-reference/domain-modifiers/CNAME.md) [MX](../language-reference/domain-modifiers/MX.md) [NS](../language-reference/domain-modifiers/NS.md) [PTR](../language-reference/domain-modifiers/PTR.md) [SRV](../language-reference/domain-modifiers/SRV.md) [SSHFP](../language-reference/domain-modifiers/SSHFP.md) [TXT](../language-reference/domain-modifiers/TXT.md)
  - Unimplemented: [DS](../language-reference/domain-modifiers/DS.md) (DS records are only supported at the apex and require a different API call that hasn't been implemented yet.)
- [SAKURACLOUD](sakuracloud.md): [A](../language-reference/domain-modifiers/A.md) [AAAA](../language-reference/domain-modifiers/AAAA.md) [ALIAS](../language-reference/domain-modifiers/ALIAS.md) [CAA](../language-reference/domain-modifiers/CAA.md) [CNAME](../language-reference/domain-modifiers/CNAME.md) [HTTPS](../language-reference/domain-modifiers/HTTPS.md) [MX](../language-reference/domain-modifiers/MX.md) [NS](../language-reference/domain-modifiers/NS.md) [PTR](../language-reference/domain-modifiers/PTR.md) [SRV](../language-reference/domain-modifiers/SRV.md) [SVCB](../language-reference/domain-modifiers/SVCB.md) [TXT](../language-reference/domain-modifiers/TXT.md)
- [SCALEWAY](scaleway.md): [A](../language-reference/domain-modifiers/A.md) [AAAA](../language-reference/domain-modifiers/AAAA.md) [ALIAS](../language-reference/domain-modifiers/ALIAS.md) [CAA](../language-reference/domain-modifiers/CAA.md) [CNAME](../language-reference/domain-modifiers/CNAME.md) [DNAME](../language-reference/domain-modifiers/DNAME.md) [HTTPS](../language-reference/domain-modifiers/HTTPS.md) [MX](../language-reference/domain-modifiers/MX.md) [NAPTR](../language-reference/domain-modifiers/NAPTR.md) [NS](../language-reference/domain-modifiers/NS.md) [SRV](../language-reference/domain-modifiers/SRV.md) [SSHFP](../language-reference/domain-modifiers/SSHFP.md) [SVCB](../language-reference/domain-modifiers/SVCB.md) [TLSA](../language-reference/domain-modifiers/TLSA.md) [TXT](../language-reference/domain-modifiers/TXT.md)
- [SOFTLAYER](softlayer.md): [A](../language-reference/domain-modifiers/A.md) [AAAA](../language-reference/domain-modifiers/AAAA.md) [CNAME](../language-reference/domain-modifiers/CNAME.md) [MX](../language-reference/domain-modifiers/MX.md) [NS](../language-reference/domain-modifiers/NS.md) [SRV](../language-reference/domain-modifiers/SRV.md) [TXT](../language-reference/domain-modifiers/TXT.md)
- [SPACESHIP](spaceship.md): [A](../language-reference/domain-modifiers/A.md) [AAAA](../language-reference/domain-modifiers/AAAA.md) [ALIAS](../language-reference/domain-modifiers/ALIAS.md) [CAA](../language-reference/domain-modifiers/CAA.md) [CNAME](../language-reference/domain-modifiers/CNAME.md) [HTTPS](../language-reference/domain-modifiers/HTTPS.md) [MX](../language-reference/domain-modifiers/MX.md) [NS](../language-reference/domain-modifiers/NS.md) [PTR](../language-reference/domain-modifiers/PTR.md) [SRV](../language-reference/domain-modifiers/SRV.md) [SVCB](../language-reference/domain-modifiers/SVCB.md) [TLSA](../language-reference/domain-modifiers/TLSA.md) [TXT](../language-reference/domain-modifiers/TXT.md)
- [TENCENTDNS](tencentdns.md): [A](../language-reference/domain-modifiers/A.md) [AAAA](../language-reference/domain-modifiers/AAAA.md) [CAA](../language-reference/domain-modifiers/CAA.md) [CNAME](../language-reference/domain-modifiers/CNAME.md) [MX](../language-reference/domain-modifiers/MX.md) [NS](../language-reference/domain-modifiers/NS.md) [SRV](../language-reference/domain-modifiers/SRV.md) [TXT](../language-reference/domain-modifiers/TXT.md)
- [TRANSIP](transip.md): [A](../language-reference/domain-modifiers/A.md) [AAAA](../language-reference/domain-modifiers/AAAA.md) [ALIAS](../language-reference/domain-modifiers/ALIAS.md) [CAA](../language-reference/domain-modifiers/CAA.md) [CNAME](../language-reference/domain-modifiers/CNAME.md) [MX](../language-reference/domain-modifiers/MX.md) [NAPTR](../language-reference/domain-modifiers/NAPTR.md) [NS](../language-reference/domain-modifiers/NS.md) [SRV](../language-reference/domain-modifiers/SRV.md) [SSHFP](../language-reference/domain-modifiers/SSHFP.md) [TLSA](../language-reference/domain-modifiers/TLSA.md) [TXT](../language-reference/domain-modifiers/TXT.md)
- [UNIFI](unifi.md): [A](../language-reference/domain-modifiers/A.md) [AAAA](../language-reference/domain-modifiers/AAAA.md) [CNAME](../language-reference/domain-modifiers/CNAME.md) [MX](../language-reference/domain-modifiers/MX.md) [SRV](../language-reference/domain-modifiers/SRV.md) [TXT](../language-reference/domain-modifiers/TXT.md)
- [VERCEL](vercel.md): [A](../language-reference/domain-modifiers/A.md) [AAAA](../language-reference/domain-modifiers/AAAA.md) [ALIAS](../language-reference/domain-modifiers/ALIAS.md) [CAA](../language-reference/domain-modifiers/CAA.md) [CNAME](../language-reference/domain-modifiers/CNAME.md) [HTTPS](../language-reference/domain-modifiers/HTTPS.md) [MX](../language-reference/domain-modifiers/MX.md) [NS](../language-reference/domain-modifiers/NS.md) [SRV](../language-reference/domain-modifiers/SRV.md) [TXT](../language-reference/domain-modifiers/TXT.md)
- [VULTR](vultr.md): [A](../language-reference/domain-modifiers/A.md) [AAAA](../language-reference/domain-modifiers/AAAA.md) [CAA](../language-reference/domain-modifiers/CAA.md) [CNAME](../language-reference/domain-modifiers/CNAME.md) [MX](../language-reference/domain-modifiers/MX.md) [NS](../language-reference/domain-modifiers/NS.md) [SRV](../language-reference/domain-modifiers/SRV.md) [SSHFP](../language-reference/domain-modifiers/SSHFP.md) [TXT](../language-reference/domain-modifiers/TXT.md)
- [WEBSUPPORT](websupport.md): [A](../language-reference/domain-modifiers/A.md) [AAAA](../language-reference/domain-modifiers/AAAA.md) [CNAME](../language-reference/domain-modifiers/CNAME.md) [MX](../language-reference/domain-modifiers/MX.md) [SRV](../language-reference/domain-modifiers/SRV.md) [TXT](../language-reference/domain-modifiers/TXT.md)
  - Not supported: [ALIAS](../language-reference/domain-modifiers/ALIAS.md) (WebSupport's ANAME is apex-only and conflicts with other apex records.)
  - Not supported: [CAA](../language-reference/domain-modifiers/CAA.md) (The v2 API does not return CAA tag/flags on read, so records cannot be managed without churn.)

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
|[`CONSTELLIX`](constellix.md)|@ZeroClover|
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
|[`HETZNER_V2`](hetznerv2.md)|@das7pad|
|[`HOSTINGDE`](hostingde.md)|@membero|
|[`HUAWEICLOUD`](huaweicloud.md)|@huihuimoe|
|[`INFOMANIAK`](infomaniak.md)|@jbelien|
|[`INTERNETBS`](internetbs.md)|@pragmaton|
|[`INWX`](inwx.md)|@patschi|
|[`LINODE`](linode.md)|@koesie10|
|[`LOOPIA`](loopia.md)|@systemcrash|
|[`LUADNS`](luadns.md)|@riku22|
|[`MIKROTIK`](mikrotik.md)|@hedger|
|[`MISAKA`](misaka.md)|@ZeroClover|
|[`NAMECHEAP`](namecheap.md)|@willpower232|
|[`NETCUP`](netcup.md)|@kordianbruck|
|[`NETLIFY`](netlify.md)|@SphericalKat|
|[`NETNOD`](netnod.md)|@Netnod @vilhelmprytz|
|[`NS1`](ns1.md)|@costasd|
|[`OPENPROVIDER`](openprovider.md)|@nvanlaerebeke|
|[`OPENSRS`](opensrs.md)|@philhug|
|[`OPENWRT`](openwrt.md)|@huskyistaken|
|[`ORACLE`](oracle.md)|@kallsyms|
|[`OVH`](ovh.md)|@masterzen|
|[`PACKETFRAME`](packetframe.md)|@hamptonmoore|
|[`POWERDNS`](powerdns.md)|@jpbede|
|[`REALTIMEREGISTER`](realtimeregister.md)|@PJEilers|
|[`ROUTE53`](route53.md)|@tresni|
|[`RWTH`](rwth.md)|@MisterErwin|
| [`SAKURACLOUD`](sakuracloud.md) | @ttkzw |
| [`SCALEWAY`](scaleway.md) | @alessiopcc |
| [`SOFTLAYER`](softlayer.md) | @jamielennox |
| [`SPACESHIP`](spaceship.md) | @rootful |
| [`TENCENTDNS`](tencentdns.md) | @cylonchau |
| [`TRANSIP`](transip.md) | @blackshadev |
|[`VERCEL`](vercel.md)|@SukkaW|
|[`VULTR`](vultr.md)|@pgaskin|

### Requested providers

We have received requests for the following providers. If you would like to contribute code to support this provider, we'd be glad to help in any way.

*(The list below is sorted alphabetically.)*

* [1984 Hosting](https://github.com/DNSControl/dnscontrol/issues/1251) (#1251)
* [BookMyName](https://github.com/DNSControl/dnscontrol/issues/3451) (#3451)
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
* [SynergyWholesale](https://github.com/DNSControl/dnscontrol/issues/1605) (#1605)
* [UltraDNS by Neustar / CSCGlobal](https://github.com/DNSControl/dnscontrol/issues/1533) (#1533)
* [Yandex Cloud DNS](https://github.com/DNSControl/dnscontrol/issues/3737) (#3737)

#### Q: Why are the above GitHub issues marked "closed"?

A: Following [provider requests](../developer-info/provider-request.md), the request is closed once it is added to this list. If someone chooses to implement the provider, they re-open the issue.

#### Q: Would someone write a provider for me?

A: The maintainer of DNSControl does not write new providers.  New providers are contributed by the community.

DNSControl tries to make writing a provider as easy as possible.  DNSControl does most of the work for you, you only have to write code to authenticate, download DNS records, and perform create/modify/delete operations on those records. Please read the directions for [Writing new DNS providers](../advanced-features/writing-providers.md).  The DNS maintainers will gladly coach you through the process.
