package mustbe

import (
	"fmt"

	"github.com/DNSControl/dnscontrol/v5/pkg/transform"
)

// OpenPGPKey returns the key re-encoded as base64. The input may be hex or base64.
func OpenPGPKey(a any) (string, error) {
	switch v := a.(type) {
	case string:
		return transform.OPENPGPKEY(v)
	}
	return "", fmt.Errorf("not a valid OPENPGPKEY: %v", a)
}
