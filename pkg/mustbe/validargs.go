package mustbe

import "fmt"

func ValidArgs(a any) {
	switch a.(type) {
	case string:
		// no op
	default:
		fmt.Printf("DEBUG: ValidArgs %T [%v]\n", a, a)
	}
}
