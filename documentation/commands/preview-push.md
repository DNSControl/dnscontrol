# preview/push

`preview` reads the dnsconfig.js file (or equivalent), determines what changes are to be made, and prints them. `push` is the same but executes the changes.

```shell
NAME:
   dnscontrol preview - read live configuration and identify changes to be made, without applying them

USAGE:
   dnscontrol preview [command options] [arguments...]

CATEGORY:
   main

OPTIONS:
   --config value                                             File containing dns config in javascript DSL (default: "dnsconfig.js")
   --dev                                                      Use helpers.js from disk instead of embedded copy (default: false)
   --variable value, -v value [ --variable value, -v value ]  Add variable that is passed to JS
   --ir value                                                 Read IR (json) directly from this file. Do not process DSL at all
   --creds value                                              Provider credentials JSON file (or !program to execute program that outputs json) (default: "creds.json")
   --providers value                                          Providers to enable (comma separated list); default is all. Can exclude individual providers from default by adding '"_exclude_from_defaults": "true"' to the credentials file for a provider
   --domains value                                            Comma separated list of domain names to include
   --notify                                                   set to true to send notifications to configured destinations (default: false)
   --expect-no-changes                                        set to true for non-zero return code if there are changes (default: false)
   --no-populate                                              Use this flag to not auto-create non-existing zones at the provider (default: false)
   --full                                                     Add headings, providers names, notifications of no changes, etc (default: false)
   --bindserial value                                         Force BIND serial numbers to this value (for reproducibility) (default: 0)
   --report value                                             Generate a JSON-formatted report of the number of changes.
   --help, -h                                                 show help
```

* `--config name`
 * Specifies the name of the main configuration file, normally `dnsconfig.js`.

* `--creds name`
 * Specifies the name of the credentials file, normally `creds.json`. Typically the file is read. If the executable bit is set, the file is executed and the output is used as the configuration. See [creds.json][creds-json.md] for details.

* `--providers name,name2`
 * Specifies a comma-separated list of providers to enable. The default is all providers. A provider can opt out of being in the default list by `"_exclude_from_defaults": "true"` to the credentials entry for that provider. In that case, the provider will only be activated if it is included in `--providers`.

* `--domains value`
 * Specifies a comma-separated list of domains to include. Example: `--domains example.com,myexample.net`
 * Domains may include a wildcard at the beginning. For example, `--domains example.com,*.in-addr.arpa` would include `example.com` plus all IPv4 reverse lookup domains.
 * Matching includes tags. If the domains are `example.com!foo` and `example.com!bar`, then `--domains example.com!foo` would match the first one, and `--domains example.com` will not match either.
 * A wildcard tag is permitted and indicates all configured tags of that domain should be selected. Example: `--domains=example.com!*` would match `example.com!foo` and `example.com!bar` but not `example.com`.
 * If `--domains` is not specified, the default is all domains.
 * NOTE: An empty tag is considered equivalent to the untagged domain. For example, `--domains=example.com!` will match `example.com` and `example.com!`

* `--v foo=bar`
 * Sets the variable `foo` to the value `bar` prior to interpreting the configuration file. Multiple `-v` options can be used.

* `--notify`
 * Enables sending notifications to the destinations configured in `creds.json`.

* `--dev`
 * Developer mode. Normally `helpers.js` is embedded in the dnscontrol executable. With this flag, the local file `helpers.js` is read instead.

* `--expect-no-changes`
 * If set, a non-zero exit code is returned if there are changes. Normally DNSControl sets the exit code based on whether or not there were protocol errors or other reasons the program can not continue. With this flag set, the exit code indicates if any changes were required. This is typically used with `preview` to allow scripts to determine if changes would happen if `push` was used. For example, one might want to run `dnscontrol preview --expect-no-changes` daily to determine if changes have been made to a domain outside of DNSControl.

* `--no-populate`
 * Do not auto-create non-existing zones at the provider. Normally non-existent zones are automatically created at a provider (unless the provider does not implement zone creation). This flag disables that feature.

* `--full`
 * Add headings, providers names, notifications of no changes, etc. to the output. Normally the output of `preview`/`push` is extremely brief. This makes the output more verbose. Useful for debugging.

* `--bindserial value`
 * Force BIND serial numbers to this value. Normally the BIND provider generates SOA serial numbers automatically. This flag forces the serial number generator to output the value specified for all domains. This is generally used for reproducibility in testing pipelines.

* `--cmode value`
 * Concurrency mode. See below.

* `--report name`
 * Write a machine-parseable report of corrections to the file named `name`. If no name is specified, no report is generated. See [JSON Reports](../advanced-features/json-reports.md)

## cmode

The `preview`/`push` commands begin with a data-gathering phase that collects current configuration from providers and zones. This collection can be done sequentially or concurrently. Concurrently is significantly faster. However since concurrent mode is newer, not all providers have been tested and certified as being compatible with this mode. Therefore the `--cmode` flag can be used to control concurrency.

The `--cmode` value may be one of the following:

* `legacy` -- Use the older, sequential code. All data is gathered sequentially. This option is removed as of release v4.16.
* `concurrent` -- Gathering is done either sequentially or concurrently depending on whether the provider is marked as having been tested to run concurrently.
* `none` -- All providers are run sequentially. This is the safest mode. It can be used if a concurrency bug is discovered.
* `all` -- This is unsafe. It runs all providers concurrently, even the ones that have not be validated to run concurrently. It is generally only used for demonstrating bugs.

The default value of `--cmode` will change over time:

* v4.14: `--cmode legacy`
* v4.15: `--cmode concurrent`
* v4.16: The `--cmode legacy` option was removed, along with the old serial code.

## ppreview/ppush

{% hint style="warning" %}
`ppreview`/`ppush` are the same as `preview`/`push` with `--cmode concurrent` instead.
These commands were for testing and were removed in v4.16.
{% endhint %}
