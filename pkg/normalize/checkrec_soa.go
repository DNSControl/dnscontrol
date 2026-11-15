package normalize

import (
	"errors"
)

// checkSoa checks the elements of an SOA.
// FIXME(tlim): Move this to MakeSOA(). (Note to self: API-downloaded items aren't run through validate).
func checkSoa(expire uint32, minttl uint32, refresh uint32, retry uint32, mbox string) error {
	if expire <= 0 {
		return errors.New("SOA Expire must be > 0")
	}
	if minttl <= 0 {
		return errors.New("SOA Minimum TTL must be > 0")
	}
	if refresh <= 0 {
		return errors.New("SOA Refresh must be > 0")
	}
	if retry <= 0 {
		return errors.New("SOA Retry must be > 0")
	}
	if mbox == "" {
		return errors.New("SOA MBox must be specified")
	}
	return nil
}
