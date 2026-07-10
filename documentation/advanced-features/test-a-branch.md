### Test A Branch

Instructions for testing DNSControl at a particular PR or branch.

Assumptions:
* `/THE/PATH` -- Change this to the full path to where your dnsconfig.js and other files are located.
* `INSERT_BRANCH_HERE` -- The branch you want to test.  The branch associated with a PR is listed on [https://github.com/DNSControl/dnscontrol/branches](https://github.com/DNSControl/dnscontrol/branches).

## Using Docker

Using Docker assures you're using the latest version of Go and doesn't require you to install anything on your machine, other than Docker!

```shell
docker run -it -v /THE/PATH:/dns golang
git clone -b INSERT_BRANCH_HERE --single-branch https://github.com/DNSControl/dnscontrol.git
cd dnscontrol
go install
```

```shell
cd /dns
dnscontrol preview
```

If you want to run the integration tests, follow the
[Integration Tests](integration-tests.md) document
as usual. The directory to be in is `/go/dnscontrol/integrationTest`.

```shell
cd /go/dnscontrol/integrationTest
go test -v -args -verbose -profile INSERT_PROVIDER_NAME -start 1 -end 3
```

Change `INSERT_PROVIDER_NAME` to the name of your provider (`BIND`, `ROUTE53`, `GCLOUD`, etc.)

## Not using Docker

Step 1: Install Go

[https://go.dev/dl/](https://go.dev/dl/)

Step 2: Check out the software

```shell
git clone -b INSERT_BRANCH_HERE --single-branch https://github.com/DNSControl/dnscontrol.git
cd dnscontrol
go install
```

```shell
cd /THE/PATH
dnscontrol preview
```

Step 3: Clean up

`go install` put the `dnscontrol` program in your `$HOME/bin` directory. You probably want to remove it.

```shell
rm -i $HOME/bin/dnscontrol
```

