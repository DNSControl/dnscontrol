package models

import (
	"fmt"
	"strings"
)

// naptrAssureQuotedFields makes sure that the fields in an NAPTR record that
// require quotes are quoted.  This is mostly used to fix a bug in NS1 where the
// fields are returned without quotes.
func naptrAssureQuotedFields(contents string) (string, error) {
	fields := strings.Fields(contents)
	if len(fields) < 5 {
		return "", fmt.Errorf("not enough NAPTR fields: %q", contents)
	}

	flag, err := naptrAddQuotes(fields[2])
	if err != nil {
		return "", err
	}

	service, err := naptrAddQuotes(fields[3])
	if err != nil {
		return "", err
	}

	regex := fields[4]
	if regex != `""` { // empty regex is permitted.
		regex, err = naptrAddQuotes(fields[4])
		if err != nil {
			return "", err
		}
	}

	if fields[2] == flag && fields[3] == service && fields[4] == regex {
		return contents, nil
	}

	fields[2] = flag
	fields[3] = service
	fields[4] = regex
	return strings.Join(fields, " "), nil
}

func naptrAddQuotes(flag string) (string, error) {
	switch len(flag) {
	case 0:
		return "", fmt.Errorf("empty flag")
	case 1:
		if flag[0] == '"' {
			return "", fmt.Errorf("invalid flag: %q", flag)
		}
		return `"` + flag + `"`, nil
	case 2:
		if flag == `""` {
			return "", fmt.Errorf("empty flag")
		}
		if flag[0] == '"' {
			return "", fmt.Errorf("unclosed quote")
		}
		if flag[1] == '"' {
			return "", fmt.Errorf("unopened quote")
		}
		return `"` + flag + `"`, nil
	}

	last := len(flag) - 1
	if flag[0] == '"' && flag[last] == '"' {
		return flag, nil
	}
	if flag[0] == '"' {
		return "", fmt.Errorf("unclosed quote")
	}
	if flag[last] == '"' {
		return "", fmt.Errorf("unopened quote")
	}
	return `"` + flag + `"`, nil
}
