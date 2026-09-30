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

// ToLowerRawString upcases a field so that comparisons do not need to be
// case-aware.
func ToUpperRawString(a any) string {
	switch v := a.(type) {
	case string:
		return strings.ToUpper(v)
	}
	return strings.ToUpper(fmt.Sprintf("%s", a))
}

// ToLowerRawString downcases a field so that comparisons do not need to be
// case-aware.
func ToLowerRawString(a any) string {
	switch v := a.(type) {
	case string:
		return strings.ToLower(v)
	}
	return strings.ToLower(fmt.Sprintf("%s", a))
}
