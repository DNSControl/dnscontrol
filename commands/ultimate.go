package commands

import "github.com/DNSControl/dnscontrol/v4/models"

/*
I proposed that Go add something like "len()" that returns the highest
index. This would avoid off-by-one errors.  The proposed names include
ultimate(), ult(), high(), highest().

Nay-sayers said I should implement this as a function and see if I
actually used it. (I suspect the nay-sayers are perfect people that
never make off-by-one errors.)

That's what this file is about.  It should be exactly the same (except
the first line) anywhere this is needed.  After a few years I'll be
able to report if it actually helped.

Go will in-line this function.
*/

func ultimate(s []*models.DomainConfig) int {
	return len(s) - 1
}
