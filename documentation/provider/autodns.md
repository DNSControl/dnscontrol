## Configuration

To use this provider, add an entry to `creds.json` with `TYPE` set to `AUTODNS` along with
[username, password and a context](https://help.internetx.com/display/APIXMLEN/Authentication#Authentication-AuthenticationviaCredentials(username/password/context)).

Example:

{% code title="creds.json" %}
```json
{
  "autodns": {
    "TYPE": "AUTODNS",
    "username": "autodns.service-account@example.com",
    "password": "[***]",
    "context": "33004"
  }
}
```
{% endcode %}

### Including sub-user zones

By default DNSControl only sees zones owned directly by the configured user. If
you authenticate as a master/admin user and want `get-zones` (and the `all`
keyword) to also include zones owned by sub-users — the same optional "include
subusers" toggle the AutoDNS web UI offers — set `children` to `"true"`:

{% code title="creds.json" %}
```json
{
  "autodns": {
    "TYPE": "AUTODNS",
    "username": "autodns.service-account@example.com",
    "password": "[***]",
    "context": "33004",
    "children": "true"
  }
}
```
{% endcode %}

## Usage

An example configuration:

{% code title="dnsconfig.js" %}
```javascript
var REG_NONE = NewRegistrar("none");
var DSP_AUTODNS = NewDnsProvider("autodns");

D("example.com", REG_NONE, DnsProvider(DSP_AUTODNS),
    A("test", "1.2.3.4"),
);
```
{% endcode %}
