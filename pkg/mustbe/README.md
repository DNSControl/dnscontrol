# MustBe

`mustbe.*` functions process an argument of any type and convert it to the required type.

- It must do any IDNA (PunyCode) conversions. Only ASCII is stored in fields that hold hostnames.
- Any hostnames must be extended to their FQDN, including a trailing dot.
- Any validation steps must be done here. Any range checking, keyword restrictions, etc. should be done here.
- If a field is case insensitive, we convert it to lowercase (hostnames) or uppercase (base64 or similar) so that future comparisons can simply use `==`.

In short, by the time any data makes it to into the struct, it must be valid. No later validation is done.  (That's not entirely true. `pkg/normalize/validate` still does some validation/normalization but once all DNS record types have a `Make*()` function, we'll be removing validation steps from `pkg/normalize/validate`.  That is, in the future, `pkg/normalize/validate` will do per-zone validation only, all per-record validations will be done in the `Make*()` functions.
