package models

import "fmt"

var dotwarned = map[string]bool{}

// WarnNameserverDot prints a warning about issue 491 never more than once.
func WarnNameserverDot(p, w string) {
	if dotwarned[p] {
		return
	}
	fmt.Printf("Warning: provider %s could be improved. See https://github.com/DNSControl/dnscontrol/issues/491 (%s)\n", p, w)
	dotwarned[p] = true
}
