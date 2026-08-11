package dnsrr

import (
	dnsv2 "codeberg.org/miekg/dns"
	dnsv1 "github.com/miekg/dns"
)

// RRv1tov2 converts github.com/miekg/dns (v1) RR to codeberg.org/miekg/dns (v2) RR.
// Typically used in providers that still use v1.
// Note: this function is not efficient. It converts via string representation.
// Use it until you are able to convert to v2 fully.
// Note: Panics on error.
func RRv1tov2(rrv1 dnsv1.RR) dnsv2.RR {
	rrv2, err := dnsv2.New(rrv1.String())
	if err != nil {
		panic("Failed to convert RR to v2: " + err.Error())
	}
	return rrv2
}

// RRv2tov1 converts codeberg.org/miekg/dns (v2) RR to github.com/miekg/dns (v1) RR.
// Typically used in providers that still use v1.
// Note: this function is not efficient. It converts via string representation.
// Use it until you are able to convert to v1 fully.
// Note: Panics on error.
func RRv2tov1(rrv2 dnsv2.RR) dnsv1.RR {
	rrv1, err := dnsv1.NewRR(rrv2.String())
	if err != nil {
		panic("Failed to convert RR to v1: " + err.Error())
	}
	return rrv1
}
