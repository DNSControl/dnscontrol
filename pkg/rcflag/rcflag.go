package rcflag

import "fmt"

type Flags struct {
	SrvWeirdSplit bool
	TxtDontParse  bool
}

type FlagType int

const (
	SrvWeirdSplit FlagType = iota
	TxtDontParse
)

// ProcessForNewRecordConfig divides args into args and flags.
func ProcessForNewRecordConfig(args []any) ([]any, Flags) {
	var a []any

	f := Flags{}
	for _, fl := range args {
		switch fl {
		case SrvWeirdSplit:
			f.SrvWeirdSplit = true
		case TxtDontParse:
			f.TxtDontParse = true
		default:
			a = append(a, fl)
		}
	}

	return a, f
}

// ProcessForNewRecordConfigParse parses rcflagList into a Flags struct.
func ProcessForNewRecordConfigParse(rcflagList []any) Flags {
	f := Flags{}
	for _, fl := range rcflagList {
		switch fl {
		case SrvWeirdSplit:
			f.SrvWeirdSplit = true
		case TxtDontParse:
			f.TxtDontParse = true
		default:
			panic(fmt.Sprintf("No such flag: %T %v", fl, fl))
		}
	}
	return f
}
