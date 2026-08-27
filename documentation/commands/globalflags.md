# Global Flags

These flags are global. They affect all subcommands.

```text
   --debug, -v        Enable detailed logging (default: false)
   --allow-fetch      Enable JS fetch(), dangerous on untrusted code! (default: false)
   --disableordering  Disables update reordering (default: false)
   --no-colors        Disable colors (default: false)
   --help, -h         show help
```

They must appear before the subcommand.

**Right:**

{% hint style="success" %}

```shell
dnscontrol --no-colors preview
```

{% endhint %}

**Wrong:**

{% hint style="danger" %}

```shell
dnscontrol preview --no-colors
```

{% endhint %}

* `-debug`
  * Enable debug output.  (The `-v` alias is the original name for this flag. That alias will go away eventually.)

* `--allow-fetch`
  * Enable the `fetch()` function in `dnsconfig.js` (or equivalent). It is disabled by default because it can be used for nefarious purposes. It is dangerous on untrusted code!  Enable it only if you trust all the people editing dnsconfig.js.

* `--disableordering`
  * Disables update reordering. Normally DNSControl re-orders the updates done by `push`. This is usually only used to work around bugs in the reordering code.

* `--no-colors`
  * Disable colors. See [Disabling Colors](colors.md) for details.
