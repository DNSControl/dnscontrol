# creds.json

When DNSControl interacts with a provider, any API keys, credentials, or other configuration parameters required are stored in `creds.json`.   The file contains a set of key/value pairs for each configuration.  That is, since a provider can be used multiple times with different credentials, the file contains a section for each set of credentials.

{% hint style="success" %}
Need to populate `creds.json` for the first time, or add an entry for a
new provider? [`dnscontrol init`](init.md) generates the file
interactively, prompting for the exact fields the chosen provider
expects.
{% endhint %}

Here's a sample file:

{% code title="creds.json" %}
```json
{
  "cloudflare_tal": {
    "TYPE": "CLOUDFLAREAPI",
    "apikey": "REDACTED",
    "apiuser": "REDACTED"
  },
  "inside": {
    "TYPE": "BIND",
    "directory": "inzones",
    "filenameformat": "db_%T%?_%D"
  },
  "CNR": {
    "TYPE": "CNR",
    "apilogin": "$CNR_APILOGIN",
    "apipassword": "$CNR_APIPASSWORD",
    "debugmode": "$CNR_DEBUGMODE",
    "domain": "$CNR_DOMAIN"
  }
}
```
{% endcode %}

## Format

* Primary keys: (e.g. `cloudflare_tal`, `inside`, `CNR`)
  * ...refer to the first parameter in the `NewRegistrar()` or `NewDnsProvider()` functions in a `dnsconfig.js` file.
  * ...may include any printable character except colon (`:`)
  * Convention: all lower case, usually the name of the provider or the username at the provider or both.
* Subkeys: (e.g. `apikey`, `apiuser` and etc.)
  * ...are whatever the provider specifies.
  * ...can be credentials, secrets, or configuration settings. In the above examples the `inside` setting is configuration parameters for the BIND provider, not credentials.
  * A missing subkey is not an error. The value is the empty string.
* Values:
  * ...may include any JSON string value including the empty string.
  * If a subkey starts with `$`, it is taken as an env variable.  In the above example, `$CNR_APILOGIN` would be replaced by the value of the environment variable `CNR_APILOGIN` or the empty string if no such environment variable exists.

## The TYPE subkey

The special subkey "TYPE" is required in each `creds.json` entry. It indicates the provider type (NONE, CLOUDFLAREAPI, GCLOUD, etc).

## Error messages

### Missing

Message: `WARNING: For future compatibility, add this entry creds.json:...`

Message: `WARNING: For future compatibility, update the ... entry in creds.json by adding:...`

These messages indicates that this provider is not mentioned in `creds.json`.  In v4.0 all providers used in `dnsconfig.js` will require an entry in `creds.json`.

For a smooth transition, please update your `creds.json` file now.

Here is the minimal entry required:

{% code title="creds.json" %}
```json
{
  "entryName": {
    "TYPE": "FILL_IN"
  }
}
```
{% endcode %}

### cleanup

Message: `INFO: In dnsconfig.js New*(..., ...) can be simplified to New*(...)`

This message indicates that the same provider name is specified in `dnsconfig.js` and `creds.json` and offers a suggestion for reducing the redundancy.

The fix is to update `dnsconfig.js` as suggested in the error. Usually this is to simply remove the second parameter to the function.

Examples:

{% code title="dnsconfig.js" %}
```diff
-var REG_THING = NewRegistrar("thing", "THING");
+var REG_THING = NewRegistrar("thing");
```
{% endcode %}

{% code title="dnsconfig.js" %}
```diff
-var REG_THING = NewRegistrar("thing", "THING", { settings: "value" } );
+var REG_THING = NewRegistrar("thing", { settings: "value" } );
```
{% endcode %}

{% code title="dnsconfig.js" %}
```diff
-var DNS_MYGANDI = NewDnsProvider("mygandi", "GANDI_V5");
+var DNS_MYGANDI = NewDnsProvider("mygandi");
```
{% endcode %}

{% code title="dnsconfig.js" %}
```diff
-var DNS_MYGANDI = NewDnsProvider("mygandi", "GANDI_V5", { settings: "value" } );
+var DNS_MYGANDI = NewDnsProvider("mygandi", { settings: "value" } );
```
{% endcode %}

Use of the OLD format will trigger warnings with suggestions on how to adopt the NEW format.

### mismatch

Message: `ERROR: Mismatch found! creds.json entry ... has ... set to ... but dnsconfig.js specifies New*(..., ...)`

This indicates that the provider type specified in `creds.json` does not match the one specified in `dnsconfig.js` or on the command line.

The fix is to change one to match the other.

### fixcreds

Message: `ERROR: creds.json entry ... is missing ...: ...`

However no `TYPE` subkey was found in an entry in `creds.json`. In 3.16 forward, it is required if new-style `NewRegistrar()` or `NewDnsProvider()` was used. In 4.0 this is required.

The fix is to add a `TYPE` subkey to the `creds.json` entry.

### hyphen

Message: `ERROR: creds.json entry ... has invalid ... value ...`

This indicates that the type `-` was specified in a `TYPE` value in `creds.json`. There is no provider named `-` therefore that is invalid. Perhaps you meant to specify a `-` on a command-line tool?

The fix is to change the `TYPE` subkey entry in `creds.json` from `-` to a valid service provider identifier, as listed in [the service provider list](../provider/index.md).

## Using a different file name

The `--creds` flag allows you to specify a different file name.

* Normally the file is read as a JSON file.
* Do not end the filename with `.yaml` or `.yml` as some day we hope to support YAML.
* Rather than specifying a file, you can specify a program or shell command to be run. The output of the program/command must be valid JSON and will be read the same way.
  * If the name begins with `!`, the remainder of the name is taken to be a shell command or program to be run.
  * If the name is a file that is executable (chmod `+x` bit), it is taken as the command to be run (Linux/MacOS only).
  * Exceptions: The `x` bit is not checked if the filename ends with `.yaml`, `.yml` or `.json`.

### Example commands

Following commands would execute a program/script:

```shell
dnscontrol preview --creds !./creds.sh
dnscontrol preview --creds ./creds.sh
dnscontrol preview --creds creds.sh
dnscontrol preview --creds !creds.sh
dnscontrol preview --creds !/some/absolute/path/creds.sh
dnscontrol preview --creds /some/absolute/path/creds.sh
```

Following commands would execute a shell command:

```shell
dnscontrol preview --creds '!op inject -i creds.json.tpl'
```

This example requires the [1Password command-line tool](https://developer.1password.com/docs/cli/) but works with any shell command that returns a properly formatted `creds.json`. In this case, the 1Password CLI is used to inject the secrets from a 1Password vault, rather than storing them in environment variables. An example of a template file containing Linode and Cloudflare API credentials is available here: [creds.json](https://github.com/DNSControl/dnscontrol/blob/main/documentation/assets/1password/creds.json).

{% code title="creds.json" %}
```json
{
  "bind": {
    "TYPE": "BIND"
  },
  "cloudflare": {
    "TYPE": "CLOUDFLAREAPI",
    "apitoken": "op://Secrets/Cloudflare DNSControl/credential",
    "accountid": "op://Secrets/Cloudflare DNSControl/username"
  },
  "linode": {
    "TYPE": "LINODE",
    "token": "op://Secrets/Linode DNSControl/credential"
  }
}
```
{% endcode %}

## Don't store creds.json in a Git repo!

Do NOT store `creds.json` (or any secrets!) in a Git repository. That is not secure.

For example, storing the creds.json at the top of this document would be horribly insecure. Anyone with access to your Git repository *or the history* will know your apiuser is `REDACTED`. Removing secrets accidentally stored in Git is very difficult because you'll need to rewrite the repo history.

A better way is to use environment variables as in the `CNR` example above.  Use secure means to distribute the names and values of the environment variables.
