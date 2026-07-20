# Getting Started

## 1. Install the software

Choose one of the following installation methods:

1. [Homebrew](#homebrew)
2. [Docker](#docker)
3. [GitHub binaries](#binaries)
4. [GitHub source](#source)

### Homebrew

On macOS (or Linux) you can install it using [Homebrew](https://brew.sh).

```shell
brew trust --formula dnscontrol/tap/dnscontrol
brew install DNSControl/tap/dnscontrol
```

### Docker

You can use DNSControl locally using the Docker image from [Docker hub](https://hub.docker.com/r/dnscontrol/dnscontrol/) or [GitHub Container Registry](https://github.com/dnscontrol/dnscontrol/pkgs/container/dnscontrol) and the command below.

```shell
docker run --rm -it -v "$(pwd):/dns"  ghcr.io/dnscontrol/dnscontrol preview
```

### Binaries

Download binaries from [GitHub](https://github.com/DNSControl/dnscontrol/releases/latest) for Linux (binary, tar, RPM, DEB), FreeBSD (tar), Windows (exec, ZIP) for 32-bit, 64-bit, and ARM.

### Source

DNSControl can be built from source with Go version 1.18 or higher.

The `go install` command will download the source, compile it, and install `dnscontrol` in your `$GOBIN` directory.

To install, simply run

```shell
go install github.com/DNSControl/dnscontrol/v4@latest
```

To download the source

```shell
git clone https://github.com/DNSControl/dnscontrol
```

If these don't work, more info is in [#805](https://github.com/DNSControl/dnscontrol/issues/805).

## 1.1. Shell Completion

Shell completion is available for `zsh` and `bash`.

### zsh

Add `eval "$(dnscontrol shell-completion zsh)"` to your `~/.zshrc` file.

This requires completion to be enabled in zsh. A good tutorial for this is available at [The Valuable Dev](https://thevaluable.dev/zsh-completion-guide-examples/) <sup>[[archived](https://web.archive.org/web/20231015083946/https://thevaluable.dev/zsh-completion-guide-examples/)]</sup>.

### bash

Add `eval "$(dnscontrol shell-completion bash)"` to your `~/.bashrc` file.

This requires the `bash-completion` package to be installed. See [scop/bash-completion](https://github.com/scop/bash-completion/) for instructions.

## 2. Create a place for the config files

Create a directory where you'll store your configuration files. We highly recommend storing these files in a Git repo, but for simple tests anything will do.

Create a subdirectory called `zones` in the same directory as the configuration files.  (`mkdir zones`).  `zones` is where the BIND provider writes the zonefiles it creates. Even if you don't use BIND for DNS service, it is useful for testing.

## 3. Create the initial `dnsconfig.js`

`dnsconfig.js` is the main configuration and defines providers, DNS domains, and so on.

{% hint style="success" %}
**Quick path:** run `dnscontrol init` instead. The interactive wizard asks for your DNS provider and registrar, prompts for the right `creds.json` fields and writes a working starter `dnsconfig.js` for you. The rest of this section is the manual equivalent. See the [init command reference](../commands/init.md) for details.
{% endhint %}

Start your `dnsconfig.js` file by downloading [dnsconfig.js](https://github.com/DNSControl/dnscontrol/blob/main/documentation/assets/getting-started/dnsconfig.js) and renaming it.

The file looks like:

{% code title="dnsconfig.js" %}
```javascript
var REG_NONE = NewRegistrar("none");
var DNS_BIND = NewDnsProvider("bind");

D("example.com", REG_NONE, DnsProvider(DNS_BIND),
    A("@", "1.2.3.4"),    // "@" means the domain's apex.
);
```
{% endcode %}

Modify this file to match your particular providers and domains. See [the DNSConfig docs](../language-reference/js.md) and [the provider docs](../provider/index.md) for more details.

Create a file called `creds.json` for storing provider configurations (API tokens and other account information). For example, to use both name.com and Cloudflare, you would have:

{% code title="creds.json" %}
```json
{
  "cloudflare": {                               // The provider name used in dnsconfig.js
    "TYPE": "CLOUDFLAREAPI",                    // The provider type identifier
    "accountid": "your-cloudflare-account-id",  // credentials
    "apitoken": "your-cloudflare-api-token"     // credentials
  },
  "namecom": {                                  // The provider name used in dnsconfig.js
    "TYPE": "NAMEDOTCOM",                       // The provider type identifier
    "apikey": "key",                            // credentials
    "apiuser": "username"                       // credentials
  },
  "none": { "TYPE": "NONE" }                    // The no-op provider
}
```
{% endcode %}

Note: Do **not** store your `creds.json` file in Git unencrypted. That is unsafe. Add `creds.json` to your `.gitignore` file as a precaution.  This file should be encrypted using something like [git-crypt](https://www.agwa.name/projects/git-crypt).

There are 2 types of providers:

A "Registrar" is with whom you register the domain.  Start with `NONE`, which is a provider that never talks to or updates the registrar.  You can define your registrar later when you want to use advanced features.

A "DnsProvider" is the service that actually provides DNS service (port 53) and may be the same or a different registrar. Even if both your Registrar and DnsProvider are the same company, two different definitions must be included in `dnsconfig.js`.

## 4. Create the initial `creds.json`

`creds.json` stores credentials and a few global settings. It is only needed if any providers require credentials (API keys, usernames, passwords, etc.).

Start your `creds.json` file by downloading [creds.json](https://github.com/DNSControl/dnscontrol/blob/main/documentation/assets/getting-started/creds.json) and renaming it.

The file looks like:

{% code title="creds.json" %}
```json
{
  "bind": {
    "TYPE": "BIND"
  },
  "r53_accountname": {
    "TYPE": "ROUTE53",
    "KeyId": "change_to_your_keyid",
    "SecretKey": "change_to_your_secretkey"
  }
}
```
{% endcode %}

Ignore the `r53_accountname` section.  It is a placeholder and will be ignored. You can use it later when you define your first set of API credentials.

Note that `creds.json` is a JSON file. JSON is very strict about commas and other formatting.  There are a few different ways to check for typos:

Python:

```shell
python -m json.tool creds.json
```

jq:

```shell
jq . < creds.json
```

FYI: `creds.json` fields can be read from an environment variable. The field must begin with a `$` followed by the variable name. No other text. For example:

{% code title="creds.json" %}
```json
{
  "apikey": "$GANDI_V5_APIKEY"
}
```
{% endcode %}

## 5. Test the sample files

Before you edit the sample files, verify that the system is working.

First run `dnscontrol preview` and ensure it completes without error(s).  The preview command is the "dry run" mode that shows only what changes need to be made and never makes any actual changes. It will use APIs if needed to find out what DNS entries currently exist.

(All output assumes the `--full` flag)

It should look something like this:

```shell
dnscontrol preview
```
```text
Initialized 1 registrars and 1 dns service providers.
******************** Domain: example.com
----- Getting nameservers from: bind
----- DNS Provider: bind... 1 correction
#1: GENERATE_ZONEFILE: example.com
 (2 records)

----- Registrar: none
Done. 1 corrections.
```

Next, run `dnscontrol push` to actually make the changes. In this case, the change will be to create a zone file where one didn't previously exist.

{% hint style="warn" %}
**Nervous?**  The first "push" can be nervous-making. How to migrate a zone
safely?  It's best to loop through iterations of `preview` and editing
`dnsconfig.js` until no changes are listed.
Once you see the preview is clean (or at least making the changes you desire),
doing a `push` is safe.  (Though, as with any tool, backups are recommended. Even a screenshot of your DNS provider's web portal is better than nothing!)
{% endhint %}

```shell
dnscontrol push
```
```text
Initialized 1 registrars and 1 dns service providers.
******************** Domain: example.com
----- Getting nameservers from: bind
----- DNS Provider: bind... 1 correction
#1: GENERATE_ZONEFILE: example.com
 (2 records)

CREATING ZONEFILE: zones/example.com.zone
SUCCESS!
----- Registrar: none
Done. 1 corrections.
```

## 6. Make a change

Try making a change to `dnsconfig.js`. For example, change the IP address of in `A("@", "1.2.3.4")` or add an additional A record.

In our case, we changed the IP address to 10.10.10.10. Previewing our change looks like this:

```shell
dnscontrol preview
```
```text
Initialized 1 registrars and 1 dns service providers.
******************** Domain: example.com
----- Getting nameservers from: bind
----- DNS Provider: bind... 1 correction
#1: GENERATE_ZONEFILE: example.com
MODIFY A example.com: (1.2.3.4 300) -> (10.10.10.10 300)

----- Registrar: none
Done. 1 corrections.
```

Notice that it read the old zone file and was able to produce a "diff" between the old `A` record and the new one.  If the zonefile didn't exist, the output would look different because the zone file was being created from scratch.

Run `dnscontrol push` to see the system generate a new zone file.

Other providers use an API to do updates. In those cases the individual changes will translate into API calls that update the specific records.

Take a look at the `zones/example.com.zone` file.  It should look like:

```text
$TTL 300
@                IN SOA   DEFAULT_NOT_SET. DEFAULT_NOT_SET. 1 3600 600 604800 1440
                 IN A     10.10.10.10
```

You can change the "DEFAULT_NOT_SET" text by following the documentation for the [BIND provider](../provider/bind.md) to set the "master" and "mbox" settings.  Try that now.

## 7. Use your own domains

Now that we know the system is working for test data, try controlling a real domain (or a test domain if you have one).

Set up the provider:  Add the providers's definition to `dnsconfig.js` and list any credentials in `creds.json`.  Each provider is different. See [the provider docs](../provider/index.md) for specifics.

Edit the domain: Add the `D()` entry for the domain, or repurpose the `example.com` domain.  Add individual `A()`, `MX()` and other records as needed.  Remember that the first parameter to `D()` is always a Registrar.

Run `dnscontrol preview` to test your work. It may take a few tries to list all the DNS records that make up the domain.  When `preview` shows no changes required, then you know you are at record parity.

The [Migrating](migrating.md) doc has advice about converting from other systems. You can manually create the `D()` statements, or you can generate them automatically using the [dnscontrol get-zones](../commands/get-zones.md) command to import the zone from (most) providers and output it as code that can be added to `dnsconfig.js` and used with very little modification.

Now you can make changes to the domain(s)  and run `dnscontrol preview`

{% hint style="warn" %}
**get-zones is not perfect** It is intended to be "a decent first draft", only
requiring minimal editing. Please submit a [bug report](https://github.com/DNSControl/dnscontrol/issues/new?template=bug_report.md) if you have suggestions
on how it can be improved.
{% endhint %}

## 8. Production Advice

If you are going to use this in production, we highly recommend the following:

* Store the configuration files in Git.
* Encrypt the `creds.json` file before storing it in Git. Do NOT store API keys or other credentials without encrypting them.
* Use a CI/CD tool like [GitLab](../advanced-features/ci-cd-gitlab.md), Jenkins, CircleCI, [GitHub Actions](https://github.com/DNSControl/dnscontrol#via-github-actions-gha), etc. to automatically push DNS changes.
* Join the DNSControl community. File [issues](https://github.com/DNSControl/dnscontrol/issues) and [PRs](https://github.com/DNSControl/dnscontrol/pulls).
