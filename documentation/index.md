{% hint style="info" %}
<span style="font-size: 21px; font-weight: 200;">DNSControl is an <a href="https://docs.dnscontrol.org/developer-info/opinions">opinionated</a> platform for seamlessly managing your DNS configuration across any number of DNS hosts, both in the cloud or in your own infrastructure. It manages all of the domains for the Stack Overflow network, and can do the same for you!</span>
{% endhint %}

# Try It

Want to jump right in? Run [`dnscontrol init`](commands/init.md) for an interactive wizard that scaffolds `creds.json` and `dnsconfig.js` in under a minute, follow our [quick start tutorial](getting-started/getting-started.md) on a new domain, or [migrate](getting-started/migrating.md) an existing one. Read the [JavaScript DSL reference](language-reference/js.md) for more info.

# Use It

Take advantage of the advanced features. Use macros and variables for easier updates. Upload your zones to [multiple DNS providers](provider/index.md).

{% hint style="success" %}
* Maintain your DNS data in a high-level DSL, with macros and variables for easier updates.
* Super extensible! Plug-in architecture makes adding new DNS providers and Registrars easy!
* Eliminate vendor lock-in. Switch DNS providers easily, any time, with full fidelity.
* Reduce points of failure: Easily maintain dual DNS providers and easily drop one that is down.
* Supports 60+ [DNS Providers](provider/index.md) including [BIND](provider/bind.md), [AWS Route 53](provider/route53.md), [Google DNS](provider/gcloud.md), and [name.com](provider/namedotcom.md).
* [Apply CI/CD principles](advanced-features/ci-cd-gitlab.md) to DNS: Unit-tests, system-tests, automated deployment.
* All the benefits of Git (or any VCS) for your DNS zone data. View history. Accept PRs.
* Optimize DNS with [SPF optimizer](language-reference/domain-modifiers/SPF_BUILDER.md). Detect too many lookups. Flatten includes.
* Runs on Linux, Windows, Mac, or any operating system supported by Go.
* Enable/disable Cloudflare proxying (the "orange cloud" button) directly from your DNSControl files.
* [Assign an IP address to a constant](getting-started/examples.md#variables-for-common-ip-addresses) and use the variable name throughout the configuration. Need to change the IP address globally? Just change the variable and "recompile".
* Keep similar domains in sync with transforms, [macros](getting-started/examples.md#macro-for-repeated-records), and variables.
{% endhint %}

# Get Involved

Join our [mailing list](https://groups.google.com/g/dnscontrol-discuss). We make it easy to contribute by using [GitHub](https://github.com/DNSControl/dnscontrol), you can make code changes with confidence thanks to extensive integration tests. The project is [newbie-friendly](https://everythingsysadmin.com/2017/08/go-get-up-to-speed.html) so jump right in!
