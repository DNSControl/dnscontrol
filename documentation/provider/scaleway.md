# Configuration

To use this provider, add an entry to `creds.json` with `TYPE` set to `SCALEWAY`
along with a Scaleway API key (access key + secret key). You can generate one
in the Scaleway console under **Identity and Access Management → API Keys**.

Example:

{% code title="creds.json" %}
```json
{
  "scaleway": {
    "TYPE": "SCALEWAY",
    "access_key": "SCWXXXXXXXXXXXXXXXXX",
    "secret_key": "11111111-2222-3333-4444-555555555555"
  }
}
```
{% endcode %}

The `project_id` field is optional. It scopes the client to a specific Scaleway
project; if omitted, the project associated with the API key is used:

{% code title="creds.json" %}
```json
{
  "scaleway": {
    "TYPE": "SCALEWAY",
    "access_key": "$SCALEWAY_ACCESS_KEY",
    "secret_key": "$SCALEWAY_SECRET_KEY",
    "project_id": "$SCALEWAY_PROJECT_ID"
  }
}
```
{% endcode %}

## Metadata

This provider does not recognize any special metadata fields unique to Scaleway.

## Usage

An example configuration:

{% code title="dnsconfig.js" %}
```javascript
var REG_NONE = NewRegistrar("none");
var DSP_SCALEWAY = NewDnsProvider("scaleway");

D("example.com", REG_NONE, DnsProvider(DSP_SCALEWAY),
    A("test", "1.2.3.4"),
    MX("@", 10, "mail.example.com."),
);
```
{% endcode %}

# Activation

DNSControl uses the [Scaleway Domains & DNS API](https://www.scaleway.com/en/developers/api/domains-and-dns/)
to manage records. You need a Scaleway account with a DNS zone already created
in the console — this provider does not create zones.

## New domains

Zones must be created via the Scaleway console or API before they can be
managed by DNSControl.
