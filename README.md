# DNSControl

[![DNSControl/dnscontrol/build](https://github.com/DNSControl/dnscontrol/actions/workflows/pr_build.yml/badge.svg)](https://github.com/DNSControl/dnscontrol/actions/workflows/pr_build.yml)
[![Google Group](https://img.shields.io/badge/google%20group-chat-green.svg)](https://groups.google.com/forum/#!forum/dnscontrol-discuss)
[![PkgGoDev](https://pkg.go.dev/badge/github.com/DNSControl/dnscontrol)](https://pkg.go.dev/github.com/DNSControl/dnscontrol/v4)

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
var REG_NAMECOM = NewRegistrar("name.com");
var r53 = NewDnsProvider("r53")

D("example.com", REG_NAMECOM, DnsProvider(r53),
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

See [Getting Started](https://docs.dnscontrol.org/getting-started/getting-started) page on documentation site to get started!

## Supported Providers

DNSControl supports 64 DNS providers and registrars:

| | | | | |
| ----- | ----- | ----- | ----- | ----- |
| [AdGuard Home](https://docs.dnscontrol.org/provider/adguardhome) | [Akamai Edge DNS](https://docs.dnscontrol.org/provider/akamaiedgedns) | [Alibaba Cloud DNS](https://docs.dnscontrol.org/provider/alidns) | [AutoDNS](https://docs.dnscontrol.org/provider/autodns) | [AWS Route 53](https://docs.dnscontrol.org/provider/route53)¹ |
| [AXFR+DDNS](https://docs.dnscontrol.org/provider/axfrddns) | [Azure DNS](https://docs.dnscontrol.org/provider/azuredns) | [Azure Private DNS](https://docs.dnscontrol.org/provider/azureprivatedns) | [BIND](https://docs.dnscontrol.org/provider/bind) | [Bunny DNS](https://docs.dnscontrol.org/provider/bunnydns) |
| [CentralNic Reseller](https://docs.dnscontrol.org/provider/cnr)¹ | [Cloudflare](https://docs.dnscontrol.org/provider/cloudflareapi) | [ClouDNS](https://docs.dnscontrol.org/provider/cloudns) | [CSC Global](https://docs.dnscontrol.org/provider/cscglobal)¹ | [deSEC](https://docs.dnscontrol.org/provider/desec) |
| [DigitalOcean](https://docs.dnscontrol.org/provider/digitalocean) | [DNS Made Easy](https://docs.dnscontrol.org/provider/dnsmadeeasy) | [DNSOVERHTTPS](https://docs.dnscontrol.org/provider/dnsoverhttps)² | [DNScale](https://docs.dnscontrol.org/provider/dnscale) | [DNSimple](https://docs.dnscontrol.org/provider/dnsimple)¹ |
| [Domainnameshop](https://docs.dnscontrol.org/provider/domainnameshop) | [Dynadot](https://docs.dnscontrol.org/provider/dynadot)² | [easyname](https://docs.dnscontrol.org/provider/easyname)² | [Exoscale](https://docs.dnscontrol.org/provider/exoscale) | [Fortigate](https://docs.dnscontrol.org/provider/fortigate) |
| [Gandi](https://docs.dnscontrol.org/provider/gandiv5)¹ | [Gcore](https://docs.dnscontrol.org/provider/gcore) | [Gidinet](https://docs.dnscontrol.org/provider/gidinet)¹ | [Google DNS](https://docs.dnscontrol.org/provider/gcloud) | [Hetzner](https://docs.dnscontrol.org/provider/hetzner) |
| [hosting.de](https://docs.dnscontrol.org/provider/hostingde)¹ | [Huawei Cloud DNS](https://docs.dnscontrol.org/provider/huaweicloud) | [Hurricane Electric DNS](https://docs.dnscontrol.org/provider/hedns) | [Infomaniak](https://docs.dnscontrol.org/provider/infomaniak) | [Internet.bs](https://docs.dnscontrol.org/provider/internetbs)² |
| [INWX](https://docs.dnscontrol.org/provider/inwx)¹ | [Joker](https://docs.dnscontrol.org/provider/joker) | [Linode](https://docs.dnscontrol.org/provider/linode) | [Loopia](https://docs.dnscontrol.org/provider/loopia)¹ | [LuaDNS](https://docs.dnscontrol.org/provider/luadns) |
| Windows Server DNS | [MikroTik RouterOS](https://docs.dnscontrol.org/provider/mikrotik) | [Mythic Beasts](https://docs.dnscontrol.org/provider/mythicbeasts) | [Name.com](https://docs.dnscontrol.org/provider/namedotcom)¹ | [Namecheap](https://docs.dnscontrol.org/provider/namecheap)¹ |
| [Netcup](https://docs.dnscontrol.org/provider/netcup) | [Netlify](https://docs.dnscontrol.org/provider/netlify) | [Netnod](https://docs.dnscontrol.org/provider/netnod) | [NS1](https://docs.dnscontrol.org/provider/ns1) | [OpenSRS](https://docs.dnscontrol.org/provider/opensrs)² |
| [Oracle Cloud](https://docs.dnscontrol.org/provider/oracle) | [OVH](https://docs.dnscontrol.org/provider/ovh)¹ | [Packetframe](https://docs.dnscontrol.org/provider/packetframe) | [Porkbun](https://docs.dnscontrol.org/provider/porkbun)¹ | [PowerDNS](https://docs.dnscontrol.org/provider/powerdns) |
| [Realtime Register](https://docs.dnscontrol.org/provider/realtimeregister)¹ | [RWTH DNS-Admin](https://docs.dnscontrol.org/provider/rwth) | [Sakura Cloud](https://docs.dnscontrol.org/provider/sakuracloud) | [SoftLayer](https://docs.dnscontrol.org/provider/softlayer) | [Tencent Cloud DNS](https://docs.dnscontrol.org/provider/tencentdns)¹ |
| [TransIP](https://docs.dnscontrol.org/provider/transip) | [UniFi Network](https://docs.dnscontrol.org/provider/unifi) | [Vercel](https://docs.dnscontrol.org/provider/vercel) | [Vultr](https://docs.dnscontrol.org/provider/vultr) | [Netbird](https://docs.dnscontrol.org/provider/netbird) |
| [WebSupport](https://docs.dnscontrol.org/provider/websupport) | | | | |

¹also supports registrar functions
²registrar only

Stack Overflow uses this system to manage hundreds of domains
and subdomains across multiple registrars and DNS providers.

You can think of it as a DNS compiler.  The configuration files are
written in a DSL that looks a lot like JavaScript.  It is compiled
to an intermediate representation (IR).  Compiler back-ends use the
IR to update your DNS zones on services such as Route53, Cloudflare,
and Gandi, or systems such as BIND.

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

DNSControl can be installed via packages for macOS, Linux and Windows, or from source code. See the [official instructions](https://docs.dnscontrol.org/getting-started/getting-started#1-install-the-software).

## Via GitHub Actions (GHA)

See [dnscontrol-action](https://github.com/koenrh/dnscontrol-action) or [gacts/install-dnscontrol](https://github.com/gacts/install-dnscontrol).

## Deprecation warnings (updated 2025-11-21)

- **REV() will switch from RFC2317 to RFC4183 in v5.0.**  This is a breaking change. Warnings are output if your configuration is affected. No date has been announced for v5.0. See https://docs.dnscontrol.org/language-reference/top-level-functions/revcompat
- **NAMEDOTCOM, OPENSRS, and SOFTLAYER need maintainers!** These providers have no maintainer. Maintainers respond to PRs and fix bugs in a timely manner, and try to stay on top of protocol changes. Interested in being a hero and adopting them?  Contact tal at what exit dot org.

## Contributing

Want to contribute? Whether it's fixing a bug, adding a new DNS provider, or improving documentation, contributions are welcome. See [CONTRIBUTING.md](CONTRIBUTING.md) for how to get started.

## More info at our website

The website: [https://docs.dnscontrol.org/](https://docs.dnscontrol.org/)

The getting started guide: [https://docs.dnscontrol.org/getting-started/getting-started](https://docs.dnscontrol.org/getting-started/getting-started)

## Stargazers over time

[![Stargazers over time](https://starchart.cc/DNSControl/dnscontrol.svg?variant=adaptive)](https://starchart.cc/DNSControl/dnscontrol)
