package models

import (
	"github.com/gobwas/glob"
)

// UnmanagedConfig describes an IGNORE() rule.
// NB(tlim): This is called "Unmanaged" because the original design
// was to call this function UNMANAGED(). However we then realized
// that we could repurpose IGNORE() without any compatibility issues.
// NB(tlim): TechDebt: UnmanagedConfig and DebugUnmanagedConfig should
// be moved to pkg/diff2/handsoff.go and the following fields could be
// unexported: LabelGlob, RTypeMap, and TargetGlob
type UnmanagedConfig struct {
	// Glob pattern for matching labels.
	LabelPattern string        `json:"label_pattern,omitempty"`
	LabelGlob    *glob.Pattern `json:"-"` // Compiled version

	// Comma-separated list of DNS Resource Types.
	RTypePattern string              `json:"rType_pattern,omitempty"`
	RTypeMap     map[string]struct{} `json:"-"` // map of RTypes or len()=0 for all

	// Glob pattern for matching targets.
	TargetPattern string        `json:"target_pattern,omitempty"`
	TargetGlob    *glob.Pattern `json:"-"` // Compiled version
}

// Uncomment to use:
// // DebugUnmanagedConfig returns a string version of an []*UnmanagedConfig for debugging purposes.
// func DebugUnmanagedConfig(uc []*UnmanagedConfig) string {
// 	if len(uc) == 0 {
// 		return "UnmanagedConfig{}"
// 	}

// 	var buf bytes.Buffer
// 	b := &buf

// 	fmt.Fprint(b, "UnmanagedConfig{\n")
// 	for i, c := range uc {
// 		fmt.Fprintf(b, "%00d: (%v, %+v, %v)\n",
// 			i,
// 			c.LabelGlob,
// 			c.RTypeMap,
// 			c.TargetGlob,
// 		)
// 	}
// 	fmt.Fprint(b, "}")

// 	return b.String()
// }
