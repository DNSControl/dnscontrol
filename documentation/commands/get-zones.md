# get-zones

DNSControl has a stand-alone utility that will contact a provider, download the records of one or more zones, and output them to a file in a variety of formats.

`get-zones` relies on command line parameters and `creds.json` exclusively. It does not use `dnsconfig.js`. This is to assist bootstrapping a new system.

## Use case 1: Bootstrapping a new system

If you are moving a DNS zone from a provider to DNSControl, this command will do most of the work for you by downloading the records and writing them out in `dnsconfig.js` format. It is intended to be "a decent first draft", only requiring minimal editing.

Use `--format=djs` or `--format=js` (djs is recommended; djs format is a comma-leading formatting style for lists, sometimes also called Haskell style).

Minor editing is required. Not all record formats are supported. SOA records are commented out, since most providers do not support it. BIND supports it, but requires the data to be entered as meta data.

The `NAMESERVER()` command is generated commented out. This is usually not needed as DNSControl can get more accurate information via the API. Remove the comments only to override the DNS service provider.

## Use case 2: Generating BIND ZONE files

The `--format=zone` generates BIND-style zonefiles. Pseudo records not supported by BIND are generated as comments.

This format is useful when moving zonedata between providers, since the format is relatively universal.

This format is also useful for generating backups of DNS zones. Unlike making a backup of the `dnsconfig.js`, this is the raw records, which may be useful.

## Use case 3: TAB separated values

The goal of `--format=tsv` is to provide a high-fidelity format that is easy enough to parse with `awk`.

## Use case 4: List zones

If a provider supports it, `--format=nameonly` lists the names of the zones at the provider.

## Syntax

```shell
dnscontrol get-zones [command options] credkey zone [...]

--creds value   Provider credentials JSON file (default: "creds.json")
--format value  Output format: js djs zone tsv nameonly (default: "zone")
--out value     Instead of stdout, write to this file
--ttl value     Default TTL (0 picks the zone's most common TTL) (default: 0)

ARGUMENTS:
credkey:  The name used in creds.json (first parameter to NewDnsProvider() in dnsconfig.js)
zone:     One or more zones (domains) to download; or "all".
```

The provider type is read from the `TYPE` field in `creds.json`. For backwards compatibility, you may still specify the provider name explicitly as a second argument (e.g. `dnscontrol get-zones my_route53 ROUTE53 example.com`), but this is deprecated.

```shell
FORMATS:
--format=js        dnsconfig.js format (not perfect, just a decent first draft)
--format=djs       js with disco commas (leading commas)
--format=zone      BIND zonefile format
--format=tsv       TAB separated value (useful for AWK)
--format=nameonly  Just print the zone names

The columns in `--format=tsv` are:

    FQDN (the label with the domain)
    ShortName (just the label, "@" if it is the naked domain)
    TTL
    Record Type (A, AAAA, CNAME, etc.)
    Target and arguments (quoted like in a zonefile)
    Either empty or a comma-separated list of properties like "cloudflare_proxy=true"

The `--ttl` flag only applies to zone/js/djs formats.
```

## Examples

```shell
dnscontrol get-zones my_route53 example.com
dnscontrol get-zones my_gandi example.com other.com
dnscontrol get-zones my_cloudflare all
dnscontrol get-zones --format=tsv my_bind example.com
dnscontrol get-zones --format=djs --out=draft.js my_gcloud example.com
```

Read a zonefile, generate a JS file, then use the JS file to see how different it is from the zonefile:

```shell
dnscontrol get-zones --format=djs --out=foo.djs my_bind example.com
dnscontrol preview --config foo.js
```

# Developer Notes

This command is not implemented for all providers.

To add this to a provider:

**Step 1. Document the feature**

In the `*provider.go` file, change the setting to implemented.

{% code title="provider.go" %}
```diff
-providers.CanGetZones: providers.Unimplemented(),
+providers.CanGetZones: providers.Can(),
```
{% endcode %}

**Step 2. Update the docs**

```shell
go generate
```

**Step 3. Implement the `GetZoneRecords` function**

Find the `GetZoneRecords` function in the `*Provider.go` file.

It currently returns `fmt.Errorf("not implemented")`.

Instead, it should gather the records from the provider and return them as a list of RecordConfig structs.

The code to do that already exists in `GetDomainCorrections`. You should extract it into its own function (`GetZoneRecords`), rather than having it be buried in the middle of `GetDomainCorrections`. `GetDomainCorrections` should call `GetZoneRecords`.

Once that is done the `get-zone` subcommand should work.

**Step 4. Optionally implement the `ListZones` function**

If the `ListZones` function is implemented, the "all" special case will be activated. In this case, listing a single zone named `all` will query the provider for the list of zones.

(Technically what is happening is by implementing the `ListZones` function, you are completing the `ZoneLister` interface for that provider.)

Implementing the `ListZones` function also activates the `check-creds` subcommand for that provider. Please add to the provider documentation a list of error messages that people might see if the credentials are invalid. See `documentation/provider/gcloud.md` for examples.
