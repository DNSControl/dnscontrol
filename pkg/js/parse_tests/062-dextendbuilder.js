// Test if a builder such as LOC() abides by the .SubDomain convention. If this
// works for one builder, it should work for them all.

var REG = NewRegistrar("Third-Party", "NONE");
var DNS = NewDnsProvider("Cloudflare", "CLOUDFLAREAPI");

// Test the name matching algorithm

D("domain.tld", REG, DnsProvider(DNS), );

D("sub.domain.tld", REG, DnsProvider(DNS), );


// Should match domain.tld
D_EXTEND("domain.tld",
    // loctest1
    LOC("loctest1", 42, 21, 54, "N", 71, 6, 18, "W", -24, 30, 0, 0),
);

// Should match domain.tld
D_EXTEND("ssub.domain.tld",
    // loctest1.ssub
    LOC("loctest2", 42, 21, 54, "N", 71, 6, 18, "W", -24, 30, 0, 0),
);
