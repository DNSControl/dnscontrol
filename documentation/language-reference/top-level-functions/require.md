---
name: require
parameters:
  - path
ts_ignore: true
---

`require(...)` loads the specified JavaScript, JSON, or JSON5 file, allowing to split your configuration across multiple files.

A better name for this function might be "include".

If the supplied `path` string ends with `.js`, the file is interpreted as JavaScript code, almost as though its contents had been included in the currently-executing file.  If  the path string ends with `.json` or `.json5` (case insensitive), `require()` returns the `JSON.parse()` of the file's contents.

If the path string begins with a `./`, it is interpreted relative to the currently-loading file (which may not be the file where the `require()` statement is, if called within a function). Otherwise it is interpreted relative to the program's working directory at the time of the call.

### Example 1: Simple

In this example, we separate our macros in one file, and put groups of domains in 3 other files. The result is a cleaner separation of code vs. domains.

{% code title="dnsconfig.js" %}
```javascript
require("lib/macros.json");

require("domains/main.json");
require("domains/parked.json");
require("domains/otherstuff.json");
```
{% endcode %}

### Example 2: Complex

Here's a more complex example:

{% code title="dnsconfig.js" %}
```javascript
require("kubernetes/clusters.js");

D("example.com", REG_MY_PROVIDER, DnsProvider(DSP_MY_PROVIDER),
    IncludeKubernetes(),
);
```
{% endcode %}

{% code title="kubernetes/clusters.js" %}
```javascript
require("./clusters/prod.js");
require("./clusters/dev.js");

function IncludeKubernetes() {
    return [includeK8Sprod(), includeK8Sdev()];
}
```
{% endcode %}

{% code title="kubernetes/clusters/prod.js" %}
```javascript
function includeK8Sprod() {
    return [
        // ...
    ];
}
```
{% endcode %}

{% code title="kubernetes/clusters/dev.js" %}
```javascript
function includeK8Sdev() {
    return [
        // ...
    ];
}
```
{% endcode %}

### Example 3: JSON and JSON5

Requiring JSON files initializes variables:

{% code title="dnsconfig.js" %}
```javascript
var domains = require("./domain-ip-map.json")

var REG_MY_PROVIDER = NewRegistrar("none");
var DSP_MY_DNSSERVER = NewDnsProvider("none");

for (var domain in domains) {
    D(domain, REG_MY_PROVIDER, DSP_MY_DNSSERVER,
        A("@", domains[domain])
    );
}
```
{% endcode %}

{% code title="domain-ip-map.json" %}
```javascript
{
    "example.com": "1.1.1.1",
    "other-example.com": "5.5.5.5"
}
```
{% endcode %}

JSON5 works the same way, but the filename ends in `.json5`. (Note: JSON5 features are supported whether the filename ends with `.json` or `.json5`. However please don't rely on JSON5 features in a `.json` file as this may change some day.)

# Notes

`require()` is *much* closer to PHP's `include()` function than it is to node's `require()`.

Node's `require()` only includes a file once. In contrast, DNSControl's `require()` is actually an imperative command to load the file and execute the code or parse the data from it.  For example if two files both `require("./tools.js")`, then it will be loaded twice, whereas in node.js it would only be loaded once.
