D("example.com", "none",

    TXT("mytxt", "Do not call me on my phone"),

    // Test at the apex two ways:
    RP("@", "user.example.com.", "mytxt.example.com."),
    RP("example.com.", "user2.example.com.", "mytxt.example.com."),

    // Test the default TTL
    RP("aaa300", "user.example.com.", "mytxt.example.com."),

    // Test DefaultTTL()
    DefaultTTL(1111),
    RP("bbb1", "user.example.com.", "mytxt.example.com."),

    // Test TTL()
    RP("ccc2", "user.example.com.", "mytxt.example.com.", TTL(2222)),

    // Test the default TTL
    RP("ddd1", "user.example.com.", "mytxt.example.com."),

    // Test a second DefaultTTL()
    DefaultTTL(3333),
    RP("eee3", "user.example.com.", "mytxt.example.com."),
);
