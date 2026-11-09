/*
   dnsconfig.js: dnscontrol configuration file for ORGANIZATION NAME.
*/

// Domains:

D("example.com", REGISTRAR(), SERVICE("bind"),
    A("@", "1.2.3.4")
);
