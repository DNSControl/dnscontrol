var REG_LEGACY = NewRegistrar("reg", "NONE");
var DNS_LEGACY = NewDnsProvider("dns", "BIND");

D("example.com!legacy", REG_LEGACY, DnsProvider(DNS_LEGACY, 2),
    A("@", "192.0.2.1"),
    TXT("note", "same configuration"),
);

D("example.com!modern", REGISTRAR("reg"), SERVICE("dns", 2),
    A("@", "192.0.2.1"),
    TXT("note", "same configuration"),
);
