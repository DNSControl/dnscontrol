### Integration Tests

This is a simple framework for testing dns providers by making real requests.

There is a sequence of changes that are defined in the test file that are run against your chosen provider.

For each step, it will run the config once and expect changes. It will run it again and expect no changes. This should give us much higher confidence that providers will work in real life.

## Configuration

`profiles.json` should have an object for each provider type under test. This is identical to the json expected in `creds.json` for dnscontrol, except it also has a "domain" field specified for the domain to test. The domain does not even need to be registered for most providers. Note that `profiles.json` expects environment variables to be specified with the relevant info.

## Running a test

1. The integration tests need a test domain to run on. All the records of this domain will be deleted!
2. Define all environment variables expected for the provider you wish to run.
3. run `cd integrationTest && go test -v -args -profile $NAME` where $NAME is the name of the provider you wish to run.

Example:

```shell
egrep ROUTE53 profiles.json
```

```text
    "KeyId": "$ROUTE53_KEY_ID",
    "SecretKey": "$ROUTE53_KEY",
    "domain": "$ROUTE53_DOMAIN"
```

```shell
export ROUTE53_KEY_ID="redacted"
export ROUTE53_KEY="also redacted"
export ROUTE53_DOMAIN="testdomain.tld"
```

```shell
cd integrationTest              # NOTE: Not needed if already in that subdirectory
go test -v -args -verbose -profile ROUTE53
```

The `-start` and `-end` flags allow you to run just a portion of the tests.

```shell
go test -v -args -verbose -profile ROUTE53 -start 16
go test -v -args -verbose -profile ROUTE53 -end 5
go test -v -args -verbose -profile ROUTE53 -start 16 -end 20
```

The `start` and `end` flags are both inclusive (i.e. `-start 16 -end 20` will run `[16, 17, 18, 19, 20]`).

For some providers it may be necessary to increase the test timeout using `-test`. The default is 10 minutes.  `0` is "no limit".  Typical Go durations work too (`1h` for 1 hour, etc).

```shell
go test -timeout 0 -v -args -verbose -profile CLOUDNS
```

FYI: The order of the flags matters.  Flags native to the Go testing suite (`-timeout` and `-v`) must come before flags that are part of the DNSControl integration tests (`-verbose`, `-profile`). Yeah, that sucks and is confusing.

The actual tests are in the file `integrationTest/integration_test.go`.  The tests are in a little language which can be used to describe just about any interaction with the API.  Look for the comment `START HERE` or the line `func makeTests` for instructions.

{% hint style="warning" %}
**WARNING**: THE RECORDS IN THE TEST DOMAIN WILL BE DELETED.  Only use
a domain that is not used in production. Some providers have a way
to run tests on domains that aren't registered (often a test
environment or a side-effect of the company not being a registrar).
In other cases we use a domain we squat on, or we register a domain
called `dnscontrol-$provider.com` just for testing.
{% endhint %}

{% hint style="info" %}
**ProTip**: If you run these tests frequently (and we hope you do), you
should create a script that you can `source` to set these
variables. Be careful not to check this script into Git since it
contains credentials.
{% endhint %}
