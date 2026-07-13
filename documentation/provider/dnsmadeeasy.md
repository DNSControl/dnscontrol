## Configuration

To use this provider, add an entry to `creds.json` with `TYPE` set to `DNSMADEEASY`
along with your `api_key` and `secret_key`. More info about authentication can be found in [DNS Made Easy API docs](https://api-docs.dnsmadeeasy.com/).

Example:

{% code title="creds.json" %}

```json
{
  "dnsmadeeasy": {
    "TYPE": "DNSMADEEASY",
    "api_key": "1c1a3c91-4770-4ce7-96f4-54c0eb0e457a",
    "secret_key": "e2268cde-2ccd-4668-a518-8aa8757a65a0"
  }
}
```

{% endcode %}

## Records

ALIAS/ANAME records are supported.

This provider does not support HTTPRED records.

SPF records are ignored by this provider. Use TXT records instead.

## Metadata

This provider does not recognize any special metadata fields unique to DNS Made Easy.

## Usage

An example configuration:

{% code title="dnsconfig.js" %}

```javascript
var REG_NONE = NewRegistrar('none');
var DSP_DNSMADEEASY = NewDnsProvider('dnsmadeeasy');

D('example.com', REG_NONE, DnsProvider(DSP_DNSMADEEASY), A('test', '1.2.3.4'));
```

{% endcode %}

## Activation

You can generate your `api_key` and `secret_key` in [Control Panel](https://cp.dnsmadeeasy.com/) in Account Information in Config menu.

API is only available for Business plan and higher plans.

## Caveats

### Global Traffic Director

Global Traffic Director feature is not supported.

## Development

### DNS Made Easy sandbox environment

Sandbox control panel is available at [https://sandbox.dnsmadeeasy.com/](sandbox.dnsmadeeasy.com). To generate sandbox API credentials, sign up for a free trial and go to [Account Information](https://sandbox.dnsmadeeasy.com/account/info) in Config menu.

Set `sandbox` key to a non-empty value in credentials JSON alongside `TYPE`, `api_key` and `secret_key` to make all API calls against DNS Made Easy sandbox environment. Details in [DNS Made Easy API documentation](https://api-docs.dnsmadeeasy.com/).

### Debugging

Set `DNSMADEEASY_DEBUG_HTTP` environment variable to `1` to dump all API calls made by this provider.

```bash
export DNSMADEEASY_DEBUG_HTTP=1
```

### Integration testing

For integration testing sandbox environment is used automatically. See `DNSMADEEASY` in `integrationTest/profiles.json`.

Run integration tests against DNS Made Easy sandbox environment. Increased test timeout is needed due to DNS Made Easy rate limits. Example:

```bash
export DNSMADEEASY_DOMAIN=dnscontroltest.com
export DNSMADEEASY_API_KEY=4aede38d-f8b4-41d7-9712-374336624046
export DNSMADEEASY_SECRET_KEY=fcf5d4f8-765e-4c90-a79e-0369d332a890

cd integrationTest
go test -v -test.timeout 2h -args -verbose -profile DNSMADEEASY
```
