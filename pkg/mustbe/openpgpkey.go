package mustbe

import (
	"fmt"

	"github.com/DNSControl/dnscontrol/v5/pkg/transform"
)

func OpenPGPKey(a any) string {
	switch v := a.(type) {
	case string:
		k, err := transform.OPENPGPKEY(v)
		if err != nil {
			return fmt.Sprintf("Can not decode %q", v)
		}
		return k
	}
	panic("mustbe.OpenPGPKey: unhandled type")
}
