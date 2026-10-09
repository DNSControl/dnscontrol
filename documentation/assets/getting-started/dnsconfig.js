/*
   dnsconfig.js: dnscontrol configuration file for ORGANIZATION NAME.
*/

// Domains:

D("example.com", REGISTRAR("none"), SERVICE("bind"),
    A("@", "1.2.3.4")
);
