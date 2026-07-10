CentralNic Reseller (CNR), formerly known as RRPProxy, is a prominent provider of domain registration and DNS solutions. Trusted by individuals, service providers, and registrars around the world, CNR is recognized for its cutting-edge technology, exceptional performance, and reliable uptime.

Our advanced DNS expertise is integral to our offering. With CentralNic Reseller, you benefit from a leading DNS platform that features robust DNS automation, DNSSEC for enhanced security, and PremiumDNS via our Anycast Network. Additionally, our platform supports a comprehensive set of features, as detailed by DNSControl.

This is based on API documents found at [https://kb.centralnicreseller.com/api/api-commands/api-command-reference#cat-dynamicdns](https://kb.centralnicreseller.com/api/api-commands/api-command-reference#cat-dynamicdns)

## Configuration

To use this provider, add an entry to `creds.json` with `TYPE` set to `CNR` along with your CentralNic Reseller login data.

Example:

{% code title="creds.json" %}
```json
{
  "CNR": {
    "TYPE": "CNR",
    "apilogin": "your-cnr-account-id",
    "apipassword": "your-cnr-account-password",
    "apientity": "LIVE", // for the LIVE system; use "OTE" for the OT&E system
    // --- debugmode ---
    // "0" -> turned off (default)
    // "1" -> turned on, basic logging of the changes reflected as API command parameters for the CNR API
    // "2" -> turned on, most verbose level - showing the detailed CNR API communication
    "debugmode": "0"
  }
}
```
{% endcode %}

Here a working example for our OT&E System:

{% code title="creds.json" %}
```json
{
  "CNR": {
    "TYPE": "CNR",
    "apilogin": "YourUserName",
    "apipassword": "YourPassword",
    "apientity": "OTE",
    "debugmode": "0"
  }
}
```
{% endcode %}

{% hint style="info" %}
**NOTE**: The above credentials are known to the public.
{% endhint %}

With the above CentralNic Reseller entry in `creds.json`, you can run the integration tests or by specifying the data per environment vars as follows:

```shell
# Review the output.  Pick one domain and set CNR_DOMAIN.
export CNR_DOMAIN=yodream.com            # Pick a domain name.
export CNR_ENTITY=OTE
export CNR_UID=test.user
export CNR_PW=test.passw0rd
export CNR_DEBUGMODE=2
cd integrationTest              # NOTE: Not needed if already in that subdirectory
go test -v -args -verbose -profile CNR
```

## Usage

Fetch a list of all DNSZones:

```shell
dnscontrol get-zones --format=nameonly cnr CNR all
```

Here's an example DNS Configuration `dnsconfig.js` using our provider module. Even though it shows how you use us as Domain Registrar AND DNS Provider, we don't force you to do that. You are free to decide if you want to use both of our provider technology or just one of them.

{% code title="dnsconfig.js" %}
```javascript
var REG_CNR = NewRegistrar("CNR");
var DSP_CNR = NewDnsProvider("CNR");

// Set Default TTL for all RR to reflect our Backend API Default
// If you use additional DNS Providers, configure a default TTL
// per domain using the domain modifier DefaultTTL instead.
// also check this issue for [NAMESERVER TTL](https://github.com/DNSControl/dnscontrol/issues/176).
DEFAULTS(
    {"ns_ttl":"3600"},
    DefaultTTL(3600)
);

D("example.com", REG_CNR, DnsProvider(DSP_CNR),
    NAMESERVER("ns1.rrpproxy.net."),
    NAMESERVER("ns2.rrpproxy.net."),
    NAMESERVER("ns3.rrpproxy.net."),
    A("elk1", "10.190.234.178"),
    A("test", "56.123.54.12"),
);
```
{% endcode %}

## Metadata

This provider does not recognize any special metadata fields unique to CentralNic Reseller (CNR).

## get-zones

`dnscontrol get-zones` is implemented for this provider. The list includes both basic and premier zones.

## New domains

If a dnszone does not exist in your CNR account, DNSControl will *not* automatically add it with the `dnscontrol push` or `dnscontrol preview` command. You'll need to do that via the control panel manually or using the command `dnscontrol create-domains`. This is because it could lead to unwanted costs on customer-side that we want to avoid.

## Debug Mode

As shown in the configuration examples above, this can be activated on demand and it can be used to check the API commands send to our system. In general this is thought for our purpose to have an easy way to dive into issues. But if you're interested what's going on, feel free to activate it.
