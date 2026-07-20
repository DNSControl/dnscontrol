# Migrating zones to DNSControl

This document explains how to migrate (convert) DNS zones from other systems to DNSControl's `dnsconfig.js` file.

This document assumes you have DNSControl set up and working on at least one zone. You should have a working `dnsconfig.js` file and `creds.json` file as explained in the [Getting Started](getting-started.md) doc, or by running [`dnscontrol init`](../commands/init.md) which generates both files interactively in under a minute.

## General advice

First, use the [Getting Started](getting-started.md) doc so that you have a working `dnsconfig.js` with at least one domain.

We recommend migrating one zone at a time. Start with a small, non-critical, zone first to learn the process. Convert larger, more important, zones as you gain confidence.

Experience has taught us that the best way to migrate a zone is to create an exact duplicate first. That is, convert the old DNS records with no changes. It is tempting to clean up the data as you do the migration... removing that old CNAME that nobody uses any more, or adding an A record you discovered was missing. Resist that temptation. If you make any changes it will be difficult to tell which changes were intentional and which are mistakes. During the migration you will know you are done when `dnscontrol preview` says there are no changes needed. At that point it is safe to do any cleanups.

## Create the first draft

Create the first draft of the `D()` statement either manually or automatically.

For a small domain you can probably create the `D()` statements by hand, possibly with your text editor's search and replace functions. However, where's the fun in that?

The `dnscontrol get-zones` subcommand [documented here](../commands/get-zones.md) can automate 90% of the conversion for you. It reads BIND-style zonefiles, or will use a providers API to gather the DNS records. It will then output the records in a variety of formats, including as a `D()` statement that is usually fairly complete. You may need to touch it up a bit, especially if you use pseudo record types in one provider that are not supported by another.

Example 1: Read a BIND zonefile

Most DNS Service Providers have an 'export to zonefile' feature.

```shell
dnscontrol get-zones --format=js bind BIND example.com
dnscontrol get-zones --format=js --out=draft.js bind BIND example.com
```

This will read the file `zones/example.com.zone`. By default the system uses directory `zones` and file suffix `.zone`. You can modify this default behaviour by configuring `creds.json` for BIND as described [here](../provider/bind.md).

Add the contents of `draft.js` to `dnsconfig.js` and edit it as needed.

Example 2: Read from a provider

This requires creating a `creds.json` file as described in [Getting Started](getting-started.md).

Suppose your `creds.json` file has the name `global_aws` for the provider `ROUTE53`. Your command would look like this:

```shell
dnscontrol get-zones --format=js global_aws ROUTE53 example.com
dnscontrol get-zones --format=js --out=draft.js global_aws ROUTE53 example.com
```

Add the contents of `draft.js` to `dnsconfig.js` and edit it as needed.

Run `dnscontrol preview` and see if it finds any differences. Edit dnsconfig.js until `dnscontrol preview` shows no errors and no changes to be made. This means the conversion of your old DNS data is correct.

`dnscontrol get-zones` makes a guess at what to do with NS records. An NS record at the apex is turned into a NAMESERVER() call, the rest are left as NS(). You probably want to check each of them for correctness.

Resist the temptation to clean up and old, obsolete, records or to add anything new. Experience has shown that making changes at this time leads to unhappy surprises, and people will blame DNSControl. Of course, once `dnscontrol preview` runs cleanly, you can do any kind of cleanups you want. In fact, they should be easier to do now that you are using DNSControl!

If `dnscontrol get-zones` could have done a better job, please [let us know](https://github.com/DNSControl/dnscontrol/issues)!

## Example workflow

Here is an example series of commands that would be used to convert a zone. Lines that start with `#` are comments.

```shell
# Note this command uses ">>" to append to dnsconfig.js. Do not use ">" as that will erase the existing file.
dnscontrol get-zones --format=js --out=draft.js bind BIND foo.com
cat >>dnsconfig.js draft.js # Append to dnsconfig.js
dnscontrol preview
vim dnsconfig.js
```

Repeat these two commands until all warnings/errors are resolved. When everything is as you wish, push the changes live:

```shell
dnscontrol push
```

Make any changes you do desire:

```shell
vim dnsconfig.js
dnscontrol preview
```

Repeat until all warnings/errors are resolved.

```shell
dnscontrol push
```
