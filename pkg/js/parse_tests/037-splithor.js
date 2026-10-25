var REG = NewRegistrar("Third-Party", "NONE");
var DNS_MAIN = NewDnsProvider("otherconfig", "CLOUDFLAREAPI");
var DNS_INSIDE = NewDnsProvider("Cloudflare", "CLOUDFLAREAPI");
var DNS_OUTSIDE = NewDnsProvider("bind", "BIND");

D("example.com", REG, DnsProvider(DNS_MAIN),
    A("main", "3.3.3.3"),
);

D("example.com!inside", REG, DnsProvider(DNS_INSIDE),
    A("main", "1.1.1.1"),
);

D("example.com!outside", REG, DnsProvider(DNS_OUTSIDE),
    A("main", "8.8.8.8"),
);

D_EXTEND("example.com",
    A("www", "33.33.33.33"),
);

D_EXTEND("example.com!inside",
    A("main", "11.11.11.11"),
);

D("example.net", REG, DnsProvider(DNS_OUTSIDE),
    A("www", "203.0.113.1"),
);

D_EXTEND("example.net!",
    A("main", "203.0.113.12"),
);

D("example.net!inside", REG, DnsProvider(DNS_INSIDE),
    INCLUDE("example.net!"),
    A("main", "192.0.2.1"),
);

D("example.net!outside", REG, DnsProvider(DNS_OUTSIDE),
    INCLUDE("example.net"),
    A("main", "203.0.113.1"),
);

D("empty.example.net", REG, DnsProvider(DNS_OUTSIDE),
    A("www", "203.0.113.2"),
);

D_EXTEND("empty.example.net!",
    A("main", "203.0.113.22"),
);

D("example-b.net!", REG, DnsProvider(DNS_OUTSIDE),
    A("www", "203.0.113.1"),
);

D_EXTEND("example-b.net",
    A("main", "203.0.113.12"),
);

D_EXTEND("sub.example-b.net",
    A("main", "203.0.113.12"),
);

D_EXTEND("sub.empty.example.net!",
    A("main", "203.0.113.22"),
);
