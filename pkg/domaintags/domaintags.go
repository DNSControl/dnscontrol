package domaintags

import (
	"strings"

	"golang.org/x/net/idna"
)

// DomainNameVarieties stores the various forms of a domain name and tag.
type DomainNameVarieties struct {
	NameRaw     string // "originalinput.com" (name as input by the user (no tag))
	NameASCII   string // "punycode.com" (converted to punycode and downcase)
	NameUnicode string // "unicode.com" (converted to unicode, ASCII portions downcased)
	UniqueName  string // "punycode.com!tag" (canonical unique name with tag)
	DisplayName string // "canonical" or "canonical (unicode.com)" if unicode

	Tag     string // The tag portion of `example.com!tag`
	HasBang bool   // Was there a "!" in the input when creating this struct?

}

// MakeDomainNameVarieties turns the user-supplied name into the varioius forms.
// * .Tag: the domain tag (of "example.com!tag")
// * .NameRaw: how the user input the name in dnsconfig.js (no tag)
// * .NameASCII: punycode version, downcased
// * .NameUnicode: unicode version of the name, downcased.
// * .UniqueName: "example.com!tag" unique across the entire config.
// * .NameDisplay: "punycode.com!tag" or "punycode.com!tag (unicode.com)" if unicode.
func MakeDomainNameVarieties(n string) *DomainNameVarieties {
	var err error
	var tag, nameRaw, nameASCII, nameUnicode, uniqueName string
	var hasBang bool

	// Split tag from name.
	p := strings.SplitN(n, "!", 2)
	if len(p) == 2 {
		tag = p[1]
		hasBang = true
	} else {
		tag = ""
		hasBang = false
	}

	nameRaw = p[0]
	if strings.HasPrefix(n, nameRaw) {
		// Avoid pointless duplication.
		nameRaw = n[0:len(nameRaw)]
	}

	nameASCII, err = idna.ToASCII(nameRaw)
	if err != nil {
		nameASCII = nameRaw // Fallback to raw name on error.
	} else {
		nameASCII = strings.ToLower(nameASCII)
		// Avoid pointless duplication.
		if strings.HasPrefix(n, nameASCII) {
			// Avoid pointless duplication.
			nameASCII = n[0:len(nameASCII)]
		}
	}

	nameUnicode, err = idna.ToUnicode(nameASCII) // We use nameASCII since it is already lowercased.
	if err != nil {
		nameUnicode = nameRaw // Fallback to raw name on error.
	} else {
		// Avoid pointless duplication.
		if strings.HasPrefix(n, nameUnicode) {
			// Avoid pointless duplication.
			nameUnicode = n[0:len(nameUnicode)]
		}
	}

	if hasBang {
		uniqueName = nameASCII + "!" + tag
	} else {
		uniqueName = nameASCII
	}

	// Display this as "example.com" or "punycode.com (unicode.com)"
	display := Display(uniqueName, nameASCII, nameUnicode)

	return &DomainNameVarieties{
		NameRaw:     nameRaw,
		NameASCII:   nameASCII,
		NameUnicode: nameUnicode,
		UniqueName:  uniqueName,
		DisplayName: display,

		Tag:     tag,
		HasBang: hasBang,
	}
}

// Display constructs the string suitable for displaying to the user
// "example.com" or "punycode.com (unicode.com)"
// If we add a user-configurable display format, it will be implemented here.
func Display(canonical, nameASCII, nameUnicode string) string {
	if nameUnicode != nameASCII {
		return canonical + " (" + nameUnicode + ")"
	}
	return canonical
}
