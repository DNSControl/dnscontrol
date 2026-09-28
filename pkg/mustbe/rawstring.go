package mustbe

import (
	"fmt"
	"strings"
)

func RawString(a any) string {
	switch v := a.(type) {
	case string:
		return v
	}
	return fmt.Sprintf("%s", a)

}

func ToUpperRawString(a any) string {
	switch v := a.(type) {
	case string:
		return strings.ToUpper(v)
	}
	return strings.ToUpper(fmt.Sprintf("%s", a))
}

// ToLowerRawString downcases a hex-encoded string so that comparisons do not
// need to be case-aware. Used for DS.Digest, SSHFP.FingerPrint and
// TLSA.Certificate; see pkg/rdatafields.KindHexLower.
func ToLowerRawString(a any) string {
	switch v := a.(type) {
	case string:
		return strings.ToLower(v)
	}
	return strings.ToLower(fmt.Sprintf("%s", a))
}
