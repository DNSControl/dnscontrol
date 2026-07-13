package mustbe

import (
	"fmt"
	"strconv"
)

// Bool converts to a bool.
func Bool(a any) bool {
	switch v := a.(type) {
	case bool:
		return v
	case string:
		if v == "" {
			v = "false"
		}
		b, err := strconv.ParseBool(v)
		if err != nil {
			panic(fmt.Sprintf("Bool: invalid boolean string: %s", a))
		}
		return b
	}
	panic(fmt.Sprintf("Bool: unhandled type: %T", a))
}

// Bool converts to a string that is either `true` or `false`.
func BoolString(a any) string {
	return fmt.Sprintf("%t", Bool(a))
}
