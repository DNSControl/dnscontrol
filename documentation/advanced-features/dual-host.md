# Dual Host

The dual hosting feature of DNSControl provides for the ability to use multiple DNS providers simultaneously. Consult your provider docs to ensure that **both** of them support this feature.

✅  - A checkmark means "this has been tested, and the provider works with dual hosting".
❔  - The questionmark means "it hasn't been tested, safety unknown"
❌  - The red "X" means "this has been tested, and it does _not_ work currently".

## Source reference

[The source](https://github.com/DNSControl/dnscontrol/blob/cdbd54016f93140548d846842b0d7575603069c8/providers/capabilities.go#L93) states that this flag

>  provider allows full management of apex NS records, so we can safely dual-host with another provider
