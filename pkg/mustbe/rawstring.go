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
