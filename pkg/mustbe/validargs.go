package mustbe

import "fmt"

// ValidArgs panics if args looks like a variadic call that lost its spread
// operator somewhere up the call chain.
//
// Many Make*() functions are declared `func(..., args ...any)` and forward to
// another `...any` function. The forwarding call must use `f(..., args...)`.
// If the `...` is omitted, the inner function sees args == []any{ []any{...} }
// — a single element whose value is itself the original []any. The individual
// values are then never validated as strings/ints/etc., and the bug surfaces
// far from its cause.
//
// ValidArgs catches exactly that shape: len(args) == 1 AND args[0] is a []any.
// We panic rather than return an error because the only way to produce this
// shape is a programmer mistake at a call site; tests will surface it loudly.
func ValidArgs(args []any) {
	if len(args) == 1 {
		if inner, ok := args[0].([]any); ok {
			panic(fmt.Sprintf(
				"mustbe.ValidArgs: variadic args were not spread (missing `...`); "+
					"got a single []any of length %d: %+v",
				len(inner), inner))
		}
	}
}

/*
Claude recommends:

Alternative strategies worth considering

1. Static analysis instead of runtime. A small go vet-style analyzer (or even a golangci-lint custom rule) could flag f(..., args) where f is variadic and args is []any in scope. Catches the bug at compile time, no runtime cost, no test coverage required. More upfront work but stronger guarantee.
2. Replace args ...any with args []any everywhere. Forces callers to write args (no ...) and removes the ambiguity entirely. Bigger refactor; the DSL ergonomics at the JS-glue layer may make this awkward.
3. Keep ValidArgs but also stamp a unique sentinel type. E.g. wrap user-supplied slices in a type Args []any so any nested Args inside Args is obviously wrong. More invasive than the current approach without much more catching power.

I'd keep the runtime check (cheap, narrowly targeted, already wired in) and add the vet analyzer later if these bugs keep recurring.
*/
