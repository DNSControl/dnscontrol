var REG_LEGACY = NewRegistrar("reg", "NONE");
var DNS_LEGACY = NewDnsProvider("dns", "BIND", {
    settings: "value"
});

D("example.com", REG_LEGACY, DnsProvider(DNS_LEGACY, 2),
    A("@", "192.0.2.1"),
);
