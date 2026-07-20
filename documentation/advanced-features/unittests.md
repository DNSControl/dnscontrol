# Unit Testing DNS Data

## Built-in Tests

DNSControl performs a number of tests during the validation stage. You can find them in `pkg/normalize/validate.go`.

## External tests

Tests specific to your environment may be added as external tests. Output the intermediate representation as a JSON file and perform tests on this data.

Output the intermediate representation:

```shell
dnscontrol print-ir --out foo.json --pretty
```

{% hint style="info" %}
**NOTE**: The `--pretty` flag is optional.
{% endhint %}

Here is a sample test written in `bash` using the [jq](https://stedolan.github.io/jq/) command.  This fails if the number of MX records in the `stackex.com` domain is not exactly 5:

    COUNTMX=$(jq --raw-output <foo.json '.domains[] | select(.name == "stackex.com") | .records[] | select(.type == "MX") | .target' | wc -l)
    echo COUNT=:"$COUNTMX":
    if [[ "$COUNTMX" -eq "5" ]]; then
      echo GOOD
    else
      echo BAD
    fi

## Future directions

Manipulating JSON data is difficult. If you implement ways to make it easier, we'd gladly accept contributions.
