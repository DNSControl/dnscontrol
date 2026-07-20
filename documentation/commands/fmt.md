# fmt

This is a stand-alone utility to pretty-format your `dnsconfig.js` configuration file.

```shell
NAME:
   dnscontrol fmt - [BETA] Format and prettify a given file

USAGE:
   dnscontrol fmt [command options] [arguments...]

CATEGORY:
   utility

OPTIONS:
   --input value, -i value   Input file (default: "dnsconfig.js")
   --output value, -o value  Output file (default: "dnsconfig.js")
   --verbose, -v             Output the filename
   --help, -h                show help
```

{% hint style="warning" %}
**Warning** This is a beta feature. In the future it may be replaced by a call
to an external program such as [Prettier](https://github.com/prettier/prettier)
which will have different formatting style.
{% endhint %}

The `fmt` subcommand formats and prettifies a dnsconfig.js file.

By default `dnsconfig.js` is read, reformatted, and (if there are no changes) rewritten. It is not rewritten if there are no changes to preserve the file's timestamp.

By default the command is silent if no changes were made. Add `-v` to always output the filename. (Prior to v2.28.3 the filename was always output.)

Changes:

```shell
$ dnscontrol fmt
dnsconfig.js (formatted)
$
```

No changes, no output:

```shell
$ dnscontrol fmt
$
```

No changes, `-v`:

```shell
$ dnscontrol fmt -v
dnsconfig.js (unchanged)
$
```

# Using `fmt` as a filter

`fmt` can also work as a filter by setting the input or output to `""` in which case stdin or stdout is used, respectively.  When the output is stdout, the filename is never output.

```shell
$ dnscontrol fmt -o "" >new-dnsconfig.js
```

# Safety

The **safest** use of this feature involves making a backup first:

```shell
$ cp dnsconfig.js dnsconfig.js.BACKUP
$ dnscontrol fmt -i dnsconfig.js.BACKUP -o dnsconfig.js
dnsconfig.js (formatted)
$
```

Alternatively use Git as your backup mechanism:

```shell
git commit -m'snapshot dnsconfig.js' dnsconfig.js
dnscontrol fmt
git diff -- dnsconfig.js
```
