# Configuration

To use this provider, add an entry to `creds.json` with `TYPE` set to
`BUNNY_DNS` along with your
[Bunny API Key](https://dash.bunny.net/account/settings).

Example:

{% code title="creds.json" %}

```json
{
    "bunny_dns": {
        "TYPE": "BUNNY_DNS",
        "api_key": "your-bunny-api-key"
    }
}
```

{% endcode %}

You can also use environment variables:

```shell
export BUNNY_DNS_API_KEY=XXXXXXXXX
```

{% code title="creds.json" %}

```json
{
    "bunny_dns": {
        "TYPE": "BUNNY_DNS",
        "api_key": "$BUNNY_DNS_API_KEY"
    }
}
```

{% endcode %}

## Metadata

This provider does not recognize any special metadata fields unique to Bunny
DNS.

## Limitations

### Records

#### CNAME / ALIAS

Bunny supports adding CNAME records for the root domain (`@`), but within
DNSControl the `ALIAS` record should be used.

```javascript
D('example.com', REG_NONE, DnsProvider(DSP_BUNNY_DNS),
    ALIAS('@', 'example2.com'),
);
```

## Usage

An example configuration:

{% code title="dnsconfig.js" %}

```javascript
var REG_NONE = NewRegistrar('none');
var DSP_BUNNY_DNS = NewDnsProvider('bunny_dns');

D('example.com', REG_NONE, DnsProvider(DSP_BUNNY_DNS), A('test', '1.2.3.4'));
```

{% endcode %}

# Activation

DNSControl depends on the
[Bunny API](https://docs.bunny.net/reference/bunnynet-api-overview) to manage
your DNS records. You will need to generate an
[API key](https://dash.bunny.net/account/settings) to use this provider.

## New domains

If a domain does not exist in your Bunny account, DNSControl will automatically
add it with the `push` command.

## Custom record types

DNSControl supports only the custom record types listed below for Bunny DNS.
Other Bunny-specific types (such as Script or Flatten) are not supported and
will be ignored by DNSControl and left as-is.

### Redirect

You can configure Bunny's Redirect type with `BUNNY_DNS_RDR`:

{% code title="dnsconfig.js" %}

```javascript
    BUNNY_DNS_RDR("@", "https://foo.bar"),
```

{% endcode %}

### Pull Zone (PZ)

You can configure Bunny's Pull Zone type with `BUNNY_DNS_PZ`. The target is the
Pull Zone ID:

{% code title="dnsconfig.js" %}

```javascript
    BUNNY_DNS_PZ("@", 12345),
```

{% endcode %}

## Caveats

- Bunny DNS does not support dual-hosting or configuring custom TTLs for NS
  records on the zone apex.
- While custom nameservers are properly recognized by this provider, it is
  currently not possible to configure them.
