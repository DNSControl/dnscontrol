// Test M365_BUILDER: the apex, a label, D_EXTEND(), the DKIM targets Microsoft
// assigns to domains created since May 2025, DANE, and TTL() at the call.

D(
    'domain.tld',
    'reg',
    // Apex: derived domainGUID, derived legacy DKIM targets, every record on.
    M365_BUILDER(
        'domain.tld', {
            initialDomain: 'contoso.onmicrosoft.com',
            verificationToken: 'MS=ms12345678',
            skypeForBusiness: true,
            mdm: true,
        },
        TTL('1h')
    ),
    // A subdomain of the same zone, with the DKIM targets copied from the
    // Microsoft 365 admin center.
    M365_BUILDER('test.domain.tld', {
        label: 'test',
        dkimSelector1Target: 'selector1-test-domain-tld._domainkey.contoso.n-v1.dkim.mail.microsoft',
        dkimSelector2Target: 'selector2-test-domain-tld._domainkey.contoso.n-v1.dkim.mail.microsoft',
    })
);

// DANE: the MX target comes from Enable-DnssecForVerifiedDomain, so no token is
// derived and no initialDomain is needed.
D(
    'dane.tld',
    'reg',
    M365_BUILDER('dane.tld', {
        mxTarget: 'dane-tld.o-v1.mx.microsoft',
        dkimSelector1Target: 'selector1-dane-tld._domainkey.contoso.o-v1.dkim.mail.microsoft',
        dkimSelector2Target: 'selector2-dane-tld._domainkey.contoso.o-v1.dkim.mail.microsoft',
    })
);

// D_EXTEND(): every record lands below the subdomain.
D_EXTEND(
    'ssub.domain.tld',
    M365_BUILDER('ssub.domain.tld', {
        initialDomain: 'contoso.onmicrosoft.com',
    })
);
