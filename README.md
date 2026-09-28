# DNSControl

[![DNSControl/dnscontrol/build](https://github.com/DNSControl/dnscontrol/actions/workflows/pr_build.yml/badge.svg)](https://github.com/DNSControl/dnscontrol/actions/workflows/pr_build.yml)
[![Google Group](https://img.shields.io/badge/google%20group-chat-green.svg)](https://groups.google.com/forum/#!forum/dnscontrol-discuss)
[![PkgGoDev](https://pkg.go.dev/badge/github.com/DNSControl/dnscontrol)](https://pkg.go.dev/github.com/DNSControl/dnscontrol/v5)

[DNSControl](https://docs.dnscontrol.org/) is Infrastructure as Code for DNS.
It includes a full-featured configuration language (Javascript-compatible),
plus "plug-ins" that speak to DNS provider APIs such as AWS Route 53,
Cloudflare, and Gandi. It can send the same DNS records to multiple providers.
It runs anywhere Go runs (Linux, macOS, Windows). The provider model is
extensible, so more providers can be added.

## An Example

`dnsconfig.js`:

```js
// define our registrar and providers
var REG_NAMECOM = NewRegistrar("ndc_main");
var DSP_ROUTE53 = NewDnsProvider("r53_main");

D("example.com", REG_NAMECOM, DnsProvider(DSP_ROUTE53),
  A("@", "1.2.3.4"),
  CNAME("www","@"),
  MX("@",5,"mail.myserver.com."),
  A("test", "5.6.7.8")
)
```

Running `dnscontrol preview` will talk to the providers (here name.com as registrar and route 53 as the dns host), and determine what changes need to be made.

Running `dnscontrol push` will make those changes with the provider and my dns records will be correctly updated.

The easiest way to run DNSControl is to use the Docker container:

```shell
docker run --rm -it -v "$(pwd):/dns"  ghcr.io/dnscontrol/dnscontrol preview
```

## Getting Started

The quickest way to start is [`dnscontrol init`](https://docs.dnscontrol.org/commands/init). The interactive wizard asks for your DNS provider and registrar, verifies your credentials and writes a working `creds.json` and `dnsconfig.js`, including the records that already exist in your zones.

See the [Getting Started](https://docs.dnscontrol.org/getting-started/getting-started) page on the documentation site for the full walkthrough.

Want full "GitOps" control of your DNS data? Clone the [dns-config](https://github.com/DNSControl/dns-config) starter repo to get started!

## Supported Providers

<!-- provider-table-start -->

DNSControl supports 74 DNS providers and registrars:

| | | | | |
| ----- | ----- | ----- | ----- | ----- |
| [`ADGUARDHOME`](https://docs.dnscontrol.org/provider/adguardhome) | [`AKAMAIEDGEDNS`](https://docs.dnscontrol.org/provider/akamaiedgedns) | [`ALIDNS`](https://docs.dnscontrol.org/provider/alidns) | [`AUTODNS`](https://docs.dnscontrol.org/provider/autodns)¹ | [`AXFRDDNS`](https://docs.dnscontrol.org/provider/axfrddns) |
| [`AZURE_DNS`](https://docs.dnscontrol.org/provider/azuredns) | [`AZURE_PRIVATE_DNS`](https://docs.dnscontrol.org/provider/azureprivatedns) | [`BIND`](https://docs.dnscontrol.org/provider/bind) | [`BUNNY_DNS`](https://docs.dnscontrol.org/provider/bunnydns) | [`CLOUDFLAREAPI`](https://docs.dnscontrol.org/provider/cloudflareapi) |
| [`CLOUDNS`](https://docs.dnscontrol.org/provider/cloudns)¹ | [`CNR`](https://docs.dnscontrol.org/provider/cnr)¹ | [`CSCGLOBAL`](https://docs.dnscontrol.org/provider/cscglobal)¹ | [`DESEC`](https://docs.dnscontrol.org/provider/desec) | [`DIGITALOCEAN`](https://docs.dnscontrol.org/provider/digitalocean) |
| [`DNSCALE`](https://docs.dnscontrol.org/provider/dnscale) | [`DNSIMPLE`](https://docs.dnscontrol.org/provider/dnsimple)¹ | [`DNSMADEEASY`](https://docs.dnscontrol.org/provider/dnsmadeeasy) | [`DNSOVERHTTPS`](https://docs.dnscontrol.org/provider/dnsoverhttps)² | [`DOMAINNAMESHOP`](https://docs.dnscontrol.org/provider/domainnameshop) |
| [`DYNADOT`](https://docs.dnscontrol.org/provider/dynadot)² | [`DYNU`](https://docs.dnscontrol.org/provider/dynu) | [`EASYNAME`](https://docs.dnscontrol.org/provider/easyname)² | [`EXOSCALE`](https://docs.dnscontrol.org/provider/exoscale) | [`FORTIGATE`](https://docs.dnscontrol.org/provider/fortigate) |
| [`GANDI_V5`](https://docs.dnscontrol.org/provider/gandiv5)¹ | [`GCLOUD`](https://docs.dnscontrol.org/provider/gcloud) | [`GCORE`](https://docs.dnscontrol.org/provider/gcore) | [`GIDINET`](https://docs.dnscontrol.org/provider/gidinet)¹ | [`GIGAHOST`](https://docs.dnscontrol.org/provider/gigahost) |
| [`HEDNS`](https://docs.dnscontrol.org/provider/hedns) | [`HETZNER_V2`](https://docs.dnscontrol.org/provider/hetznerv2) | [`HOSTINGDE`](https://docs.dnscontrol.org/provider/hostingde)¹ | [`HUAWEICLOUD`](https://docs.dnscontrol.org/provider/huaweicloud) | [`INFOBLOX`](https://docs.dnscontrol.org/provider/infoblox) |
| [`INFOMANIAK`](https://docs.dnscontrol.org/provider/infomaniak) | [`INTERNETBS`](https://docs.dnscontrol.org/provider/internetbs)² | [`INWX`](https://docs.dnscontrol.org/provider/inwx)¹ | [`JOKER`](https://docs.dnscontrol.org/provider/joker) | [`LINODE`](https://docs.dnscontrol.org/provider/linode) |
| [`LOOPIA`](https://docs.dnscontrol.org/provider/loopia)¹ | [`LUADNS`](https://docs.dnscontrol.org/provider/luadns) | [`MIKROTIK`](https://docs.dnscontrol.org/provider/mikrotik) | [`MITTWALD`](https://docs.dnscontrol.org/provider/mittwald) | [`MYTHICBEASTS`](https://docs.dnscontrol.org/provider/mythicbeasts) |
| [`NAMECHEAP`](https://docs.dnscontrol.org/provider/namecheap)¹ | [`NAMEDOTCOM`](https://docs.dnscontrol.org/provider/namedotcom)¹ | [`NETBIRD`](https://docs.dnscontrol.org/provider/netbird) | [`NETCUP`](https://docs.dnscontrol.org/provider/netcup) | [`NETLIFY`](https://docs.dnscontrol.org/provider/netlify) |
| [`NETNOD`](https://docs.dnscontrol.org/provider/netnod) | [`NEXDNS`](https://docs.dnscontrol.org/provider/nexdns) | [`NS1`](https://docs.dnscontrol.org/provider/ns1) | [`OPENPROVIDER`](https://docs.dnscontrol.org/provider/openprovider) | [`OPENSRS`](https://docs.dnscontrol.org/provider/opensrs)² |
| [`OPENWRT`](https://docs.dnscontrol.org/provider/openwrt) | [`ORACLE`](https://docs.dnscontrol.org/provider/oracle) | [`OVH`](https://docs.dnscontrol.org/provider/ovh)¹ | [`PACKETFRAME`](https://docs.dnscontrol.org/provider/packetframe) | [`PORKBUN`](https://docs.dnscontrol.org/provider/porkbun)¹ |
| [`POWERDNS`](https://docs.dnscontrol.org/provider/powerdns) | [`REALTIMEREGISTER`](https://docs.dnscontrol.org/provider/realtimeregister)¹ | [`ROUTE53`](https://docs.dnscontrol.org/provider/route53)¹ | [`RWTH`](https://docs.dnscontrol.org/provider/rwth) | [`SAKURACLOUD`](https://docs.dnscontrol.org/provider/sakuracloud) |
| [`SCALEWAY`](https://docs.dnscontrol.org/provider/scaleway) | [`SOFTLAYER`](https://docs.dnscontrol.org/provider/softlayer) | [`SPACESHIP`](https://docs.dnscontrol.org/provider/spaceship)¹ | [`TENCENTDNS`](https://docs.dnscontrol.org/provider/tencentdns)¹ | [`TRANSIP`](https://docs.dnscontrol.org/provider/transip) |
| [`UNIFI`](https://docs.dnscontrol.org/provider/unifi) | [`VERCEL`](https://docs.dnscontrol.org/provider/vercel) | [`VULTR`](https://docs.dnscontrol.org/provider/vultr) | [`WEBSUPPORT`](https://docs.dnscontrol.org/provider/websupport) |  |

¹also supports registrar functions
²registrar only

<!-- provider-table-end -->

## Benefits

- **Less error-prone** than editing a BIND zone file.
- **More reproducible**  than clicking buttons on a web portal.
- **Easily switch between DNS providers:**  The DNSControl language is
  vendor-agnostic.  If you use it to maintain your DNS zone records,
  you can switch between DNS providers easily. In fact, DNSControl
  will upload your DNS records to multiple providers, which means you
  can test one while switching to another. We've switched providers 3
  times in three years and we've never lost a DNS record.
- **Apply CI/CD principles to DNS!**  StackOverflow maintains their
  DNSControl configurations in Git and use our CI system to roll out
  changes.  Keeping DNS information in a VCS means we have full
  history.  Using CI enables us to include unit-tests and
  system-tests.  Remember when you forgot to include a "." at the end
  of an MX record?  We haven't had that problem since we included a
  test to make sure Tom doesn't make that mistake... again.
- **Adopt (GitOps) PR-based updates.**  Allow developers to send updates as PRs,
  which you can review before you approve.
- **Variables save time!**  Assign an IP address to a constant and use the
  variable name throughout the file. Need to change the IP address
  globally? Just change the variable and "recompile."
- **Macros!**  Define your SPF records, MX records, or other repeated data
  once and re-use them for all domains.
- **Control Cloudflare from a single source of truth.**  Enable/disable
  Cloudflare proxying (the "orange cloud" button) directly from your
  DNSControl files.
- **Keep similar domains in sync** with transforms and other features.  If
  one domain is supposed to be a filtered version of another, this is
  easy to set up.
- **It is extendable!**  All the DNS providers are written as plugins.
  Writing new plugins is very easy.

## Installation

DNSControl can be installed via packages for macOS, Linux and Windows, or from source code. See the [official instructions](https://docs.dnscontrol.org/getting-started/getting-started#id-1.-install-the-software).

## Via GitHub Actions (GHA)

The official GitHub Action is: [github.com/dnscontrol/dnscontrol-action](https://github.com/dnscontrol/dnscontrol-action)

Others have been created such as:

* [github.com/metabrainz/dnscontrol-action](https://github.com/metabrainz/dnscontrol-action)
* [github.com/gacts/install-dnscontrol](https://github.com/gacts/install-dnscontrol)

## Deprecation warnings (updated 2026-08-17)

- **REV() will switch from RFC2317 to RFC4183 sometime after v5.0 is released.** This is a breaking change. Warnings are output if your configuration is affected. See https://docs.dnscontrol.org/language-reference/top-level-functions/revcompat
- **NAMEDOTCOM, OPENSRS, and SOFTLAYER need maintainers!** These providers have no maintainer. Maintainers respond to PRs and fix bugs in a timely manner, and try to stay on top of protocol changes. Interested in being a hero and adopting them?  Contact tal at what exit dot org.

## Contributing

Want to contribute? Whether it's fixing a bug, adding a new DNS provider, or improving documentation, contributions are welcome. See [CONTRIBUTING.md](CONTRIBUTING.md) for how to get started.

## More info at our website

The website: [https://docs.dnscontrol.org/](https://docs.dnscontrol.org/)

The getting started guide: [https://docs.dnscontrol.org/getting-started/getting-started](https://docs.dnscontrol.org/getting-started/getting-started)

## Stargazers over time

[![Stargazers over time](https://starchart.cc/DNSControl/dnscontrol.svg?variant=adaptive)](https://starchart.cc/DNSControl/dnscontrol)
