var REG = NewRegistrar("Third-Party", "NONE");
var CF = NewDnsProvider("Cloudflare", "CLOUDFLAREAPI");

var BASE = IP("1.2.3.4");

D("foo.com", REG, DnsProvider(CF, 0),
    A("@", BASE),
    A("p1", BASE + 1),
    A("p255", BASE + 255),

    // An IP address with the highest bit set, to catch uint/int bugs.
    A("yyy", IP("190.2.3.4"))
);
