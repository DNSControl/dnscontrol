package rtypecontrol

import (
	"fmt"
	"strings"
)

// StringifyQuoted returns a string with each argument quoted.
func StringifyQuoted(args []any) string {
	if len(args) == 0 {
		return ""
	}

	var sb strings.Builder
	fmt.Fprintf(&sb, "%q", args[0])
	for _, arg := range args[1:] {
		fmt.Fprintf(&sb, " %q", arg)
	}
	return sb.String()
}
