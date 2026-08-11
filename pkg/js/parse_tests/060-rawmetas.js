// This tests whether or not metadata gets passed to RecordConfig v2.

D("bar.com", "none",
    RP("foo.bar.com", "user2.example.com.", "mytxt.example.com.", DISABLE_REPEATED_DOMAIN_CHECK),
    RP("bar.com", "user2.example.com.", "mytxt.example.com.", DISABLE_REPEATED_DOMAIN_CHECK),
)
