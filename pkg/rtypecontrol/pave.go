package rtypecontrol

import (
	"fmt"
	"math"
	"strconv"
)

// PaveArgs converts each arg to its desired type, or returns an error if conversion fails or if the number of arguments is wrong.
// argTypes is a string where each rune specifies the desired type of the arg in the same position:
// 's': string (will convert other types to string using %v)
// 'b': uint8 (will convert strings, truncate floats, etc)
// 'w': uint16 (will convert strings, truncate floats, etc)
// FUTURE 'd': uint32 (will convert strings, truncate floats, etc)
// FUTURE 'q': uint64 (will convert strings, truncate floats, etc)
// FUTURE: Uppercase runes for signed types.
func PaveArgs(args []any, argTypes string) error {
	if len(args) != len(argTypes) {
		return fmt.Errorf("wrong number of arguments. Expected %v, got %v", len(argTypes), len(args))
	}

	for i, at := range argTypes {
		arg := args[i]
		switch at {

		case 's':
			if _, ok := arg.(string); ok {
				args[i] = arg.(string)
			} else {
				args[i] = fmt.Sprintf("%v", arg)
			}

		case 'b': // uint8
			switch v := arg.(type) {
			case uint8:
				// already correct type
			case uint16:
				if v > math.MaxUint8 {
					return fmt.Errorf("value %q overflows uint8", arg)
				}
				args[i] = uint8(v)
			case int16:
				if v < 0 || v > math.MaxUint8 {
					return fmt.Errorf("value %q overflows uint8", arg)
				}
				args[i] = uint8(v)
			case uint:
				if v > math.MaxUint8 {
					return fmt.Errorf("value %q overflows uint8", arg)
				}
				args[i] = uint8(v)
			case int:
				if v < 0 || v > math.MaxUint8 {
					return fmt.Errorf("value %q overflows uint8", arg)
				}
				args[i] = uint8(v)
			case float64:
				if v < 0 || v > math.MaxUint8 {
					return fmt.Errorf("value %q overflows uint8", arg)
				}
				args[i] = uint8(v)
			case string:
				ni, err := strconv.ParseUint(arg.(string), 10, 8)
				if err != nil {
					return fmt.Errorf("value %q is not a number (uint8 wanted)", arg)
				}
				args[i] = uint8(ni)
			default:
				return fmt.Errorf("value %q is type %T, expected uint8", arg, arg)
			}

		case 'w': // uint16
			switch v := arg.(type) {
			case uint8:
				args[i] = uint16(v)
			case uint16:
				// already correct type
			case int16:
				if v < 0 {
					return fmt.Errorf("value %q overflows uint8", arg)
				}
				args[i] = uint16(v)
			case uint:
				if v > math.MaxUint16 {
					return fmt.Errorf("value %q overflows uint8", arg)
				}
				args[i] = uint16(v)
			case int:
				if v < 0 || v > math.MaxUint16 {
					return fmt.Errorf("value %q overflows uint8", arg)
				}
				args[i] = uint16(v)
			case float64:
				if v < 0 || v > math.MaxUint16 {
					return fmt.Errorf("value %q overflows uint8", arg)
				}
				args[i] = uint16(v)
			case string:
				ni, err := strconv.ParseUint(arg.(string), 10, 16)
				if err != nil {
					return fmt.Errorf("value %q is not a number (uint16 wanted)", arg)
				}
				args[i] = uint16(ni)
			default:
				return fmt.Errorf("value %q is type %T, expected uint16", arg, arg)
			}

		default:
			return fmt.Errorf("unknown argType rune: %q", at)
		}
	}

	return nil
}
