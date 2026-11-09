// Test MIGADU_BUILDER: the apex, a label with a wildcard MX, D_EXTEND(), and
// TTL() at the call.

D(
    'foo.com',
    'reg',
    // Apex: verification, SPF and every default record.
    MIGADU_BUILDER('foo.com', {
        verify: 'wxyz5678',
        spf: true,
    })
);

// A subdomain of the zone: the DKIM targets embed the mail domain, and the
// wildcard MX lands below the label rather than at the apex.
D(
    'bar.com',
    'reg',
    MIGADU_BUILDER(
        'mail.bar.com', {
            label: 'mail',
            verify: 'abcd1234',
            wildcard: true,
            autoconfig: false,
            autodiscover: false,
            clientSRV: false,
        },
        TTL('1h')
    )
);

// D_EXTEND(): every record lands below the subdomain.
D_EXTEND(
    'sub.bar.com',
    MIGADU_BUILDER('sub.bar.com', {
        autoconfig: false,
        autodiscover: false,
        clientSRV: false,
    })
);
