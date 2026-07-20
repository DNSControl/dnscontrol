package bind

import (
	"bytes"
	"fmt"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/DNSControl/dnscontrol/v4/pkg/domaintags"
)

// makeFileName uses format to generate a zone's filename.  See the.
func makeFileName(format string, ff domaintags.DomainNameVarieties) string {
	//fmt.Printf("DEBUG: makeFileName(%q, %+v)\n", format, ff)
	if format == "" {
		panic("BUG: makeFileName called with null format")
	}

	var b bytes.Buffer

	tokens := strings.Split(format, "")
	lastpos := len(tokens) - 1
	for pos := 0; pos < len(tokens); pos++ {
		tok := tokens[pos]

		if tok != "%" {
			b.WriteString(tok)
			continue
		}
		if pos == lastpos {
			b.WriteString("%(format may not end in %)")
			continue
		}
		pos++
		tok = tokens[pos]
		switch tok {

		// v4.28 names
		case "r": // NameRaw     "originalinput.com" (i for input)
			b.WriteString(ff.NameRaw)
		case "a": // NameASCII   "punycode.com" (a for ascii)
			b.WriteString(ff.NameASCII)
		case "u": // NameUnicode "unicode.com" (u for unicode)
			b.WriteString(ff.NameUnicode)
		case "c": // UniqueName  "punycode.com!tag" or "punycode.com" if no tag (c for canonical)
			b.WriteString(ff.UniqueName)
		case "f": //
			b.WriteString(ff.NameASCII)
			if ff.Tag != "" {
				b.WriteString("_")
				b.WriteString(ff.Tag)
			}
		case "F": //
			if ff.Tag != "" {
				b.WriteString(ff.Tag)
				b.WriteString("_")
			}
			b.WriteString(ff.NameASCII)
		case "T": // Tag         The tag portion of `example.com!tag`
			b.WriteString(ff.Tag)

		case "%":
			b.WriteString("%")
		case "?":
			if pos == lastpos {
				b.WriteString("%(format may not end in %?)")
				continue
			}
			pos++
			tok = tokens[pos]
			if ff.Tag != "" {
				b.WriteString(tok)
			}

		// Legacy names kept for compatibility
		case "U": // the domain name as specified in `D()`
			b.WriteString(strings.ToLower(ff.NameRaw))
			if ff.Tag != "" {
				b.WriteString("!")
				b.WriteString(ff.Tag)
			}
		case "D": // domain (without tag) as specified in D() (no IDN conversion, but downcased)
			b.WriteString(strings.ToLower(ff.NameRaw))

		default:
			fmt.Fprintf(&b, "%%(unknown %%verb %%%s)", tok)
		}
	}

	// fmt.Printf("DEBUG: makeFileName returns= %q\n", b.String())
	return b.String()
}

// extractZonesFromFilenames extracts the zone names from a list of filenames
// based on the format string used to create the files. It is mathematically
// impossible to do this correctly for all format strings, but typical format
// strings are supported.
func extractZonesFromFilenames(format string, names []string) []string {
	var zones []string

	// Generate a regex that will extract the zonename from a filename.
	extractor, err := makeExtractor(format)
	if err != nil {
		// Give up. Return the list of filenames.
		return names
	}
	re := regexp.MustCompile(extractor)

	//
	for _, n := range names {
		_, file := filepath.Split(n)
		l := re.FindStringSubmatch(file)
		// l[1:] is a list of matches and null strings.  Pick the first non-null string.
		if len(l) > 1 {
			for _, s := range l[1:] {
				if s != "" {
					zones = append(zones, s)
					break
				}
			}
		}
	}
	return zones
}

// makeExtractor generates a regex that extracts domain names from filenames.
// format specifies the format string used by makeFileName to generate such
// filenames. It is mathematically impossible to do this correctly for all
// format strings, but typical format strings are supported.
func makeExtractor(format string) (string, error) {
	// The algorithm works as follows.

	// We generate a regex that is A or A|B.
	// A is the regex that works if tag is non-null.
	// B is the regex that assumes tags are "".
	// If no tag-related verbs are used, A is sufficient.
	// If a tag-related verb is used, we append | and generate B, which does
	// Each % verb is turned into an appropriate subexpression based on pass.

	// NB: This is some rather fancy CS stuff just to make the
	// "get-zones all" command work for BIND.  That's a lot of work for
	// a feature that isn't going to be used very often, if at all.
	// Therefore if this ever becomes a maintenance bother, we can just
	// replace this with something more simple. For example, the
	// creds.json file could specify the regex and humans can specify
	// the Extractor themselves. Or, just remove this feature from the
	// BIND driver.

	var b bytes.Buffer

	tokens := strings.Split(format, "")
	lastpos := len(tokens) - 1

	generateB := false
	for pass := range []int{0, 1} {
		for pos := 0; pos < len(tokens); pos++ {
			tok := tokens[pos]

			if tok == "." {
				// dots are escaped
				b.WriteString(`\.`)
				continue
			}
			if tok != "%" {
				// ordinary runes are passed unmodified.
				b.WriteString(tok)
				continue
			}
			if pos == lastpos {
				return ``, fmt.Errorf("format may not end in %%: %q", format)
			}

			// Process % verbs

			// Move to the next token, which is the verb name: D, U, etc.
			pos++
			tok = tokens[pos]
			switch tok {
			case "D", "a", "u", "r":
				b.WriteString(`(.*)`)
			case "T":
				if pass == 0 {
					// On the second pass, nothing is generated.
					b.WriteString(`.*`)
				}
			case "U", "c":
				if pass == 0 {
					b.WriteString(`(.*)!.+`)
				} else {
					b.WriteString(`(.*)`)
				}
				generateB = true
			case "?":
				if pos == lastpos {
					return ``, fmt.Errorf("format may not end in %%?: %q", format)
				}
				// Move to the next token, the tag-only char.
				pos++
				tok = tokens[pos]
				if pass == 0 {
					// On the second pass, nothing is generated.
					b.WriteString(tok)
				}
				generateB = true
			default:
				return ``, fmt.Errorf("unknown %%verb %%%s: %q", tok, format)
			}
		}

		// At the end of the first pass determine if we need the second pass.
		if pass == 0 {
			if generateB {
				// We had a %? token. Now repeat the process
				// but generate an "or" that assumes no tags.
				b.WriteString(`|`)
			} else {
				break
			}
		}
	}

	return b.String(), nil
}
