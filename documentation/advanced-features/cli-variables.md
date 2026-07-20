# CLI variables

You can pass variables into your configuration from the command line using the `-v key=value` flag. There is also a mechanism called `CLI_DEFAULTS` which lets you easily set the defaults on variables that are otherwise controlled from the command line.

This gives you the opportunity to run different code when a value is passed.

## Passing variables

To pass a variable from CLI, just use the parameter `-v key=value` when using subcommands `preview` or `push`.

Example: `dnscontrol preview -v testKey=testValue`

This would set the variable with the name `testKey` and the value of `testValue` when processing `dnsconfig.js`

## Define defaults

The `CLI_DEFAULTS` feature is used to define default values for when a variable is not defined on the command line.

{% code title="dnsconfig.js" %}
```javascript
CLI_DEFAULTS({
    "variableName": "defaultValue",
});
```
{% endcode %}

You need to define this defaults just once in your `dnsconfig.js`. It should be defined **before** using it.

_Please keep in mind that accessing an undefined variable is an error. If it is not set on the command line nor in `CLI_DEFAULTS`, accessing the variable will fail._

## Example 1: Different IPs for internal/external DNS

In this example we have a number of variables which need to be set differently when `view=internal`.

In this configuration:

* `dnscontrol push` would generate the external (default) view.
* `dnscontrol push -v view=internal` would generate the internal view.

{% code title="dnsconfig.js" %}
```javascript
// See https://docs.dnscontrol.org/advanced-features/cli-variables
CLI_DEFAULTS({
    "view": "external",
});
if (view == "external") {
    // BIND view: external (192.168.0.0/16 addresses)
    var host01 = "192.168.0.16";
    var host02 = "192.168.0.17";
} else {
    // BIND view: internal (10.0.0.0/8 addresses)
    var host01 = "10.0.0.16";
    var host02 = "10.0.0.17";
}

/// ...much later...

D("example.com", REG_NAMECOM, DnsProvider(DNS_NAMECOM), DnsProvider(DNS_BIND),
    A("sitea", host01, TTL(1800)),
    A("siteb", host01, TTL(1800)),
    A("sitec", host02, TTL(1800)),
    A("sited", host02, TTL(1800)),
);
```
{% endcode %}

## Example 2: Different DNS records

In this example different code is run when `emergency=true`.  Normally `server12` is an A record but in an emergency it is a CNAME.

In this configuration:

* `dnscontrol push` would generate the normal configuration.
* `dnscontrol push -v emergency=true` would generate the emergency configuration.

{% code title="dnsconfig.js" %}
```javascript
// See https://docs.dnscontrol.org/advanced-features/cli-variables
CLI_DEFAULTS({
    "emergency": false,
});

// ...much later...

D("example.com", REG_EXAMPLE, DnsProvider(DNS_EXAMPLE),
    A("www", "10.10.10.10"),
);

if (emergency) {
    // Emergency mode: Configure A/B/C using CNAMEs to our alternate site.

    D_EXTEND("example.com",
        CNAME("a", "a.othersite"),
        CNAME("b", "b.othersite"),
        CNAME("c", "c.othersite"),
    );

} else {
    // Normal operation: Configure A/B/C using A records.

    D_EXTEND("example.com",
        A("a", "10.10.10.10"),
        A("b", "10.10.10.11"),
        A("c", "10.10.10.12"),
    );

}
```
{% endcode %}

### ProTips

The cli variables functionality permits you to create very complex and sophisticated configurations, but you shouldn't. Be nice to the next person that edits the file, who may not be as expert as yourself.

While there is no limit to the number of variables that can be set on the command line, doing so is annoying to the person using the tool.  It is better to set one variables which specifies a "mode".  This mode is then used to automatically set other variables. This way the user can determine the mode and the code can determine what to do in that mode. This is less error-prone and more testable.

In the first example, you'll see that one variable is used to set a mode which then determines many other variables.  This is done in one place, at the top of the file. Everything related to this is isolated to one place, thus easier to maintain. The rest of the file simply uses those variables.

In the second example, you'll see a boolean variable is set which selects which code will run different code. While the conditional code is not isolated to the top of the file, the conditional code is placed immediately after the domain.

In both examples, not setting any variables on the command line does something reasonable. If someone accidentally runs `dnscontrol push` without any variables, the behavior is correct (assuming we're not in emergency mode, which is unlikely).

