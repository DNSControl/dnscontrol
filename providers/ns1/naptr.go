package ns1

import (
	"errors"
	"fmt"
	"strings"
)

func ns1NAPTRAnswer(ans string) (string, error) {
	// NS1 doesn't quote a missing parameter properly. Therefore we look for 2
	// spaces and assume there is a missing item.
	ans = strings.ReplaceAll(ans, "  ", ` "" `)
	return naptrAssureQuotedFields(ans)
}

// naptrAssureQuotedFields makes sure that NAPTR flags, service, and regexp
// fields are RFC1035-quoted. NS1 returns these without quotes.
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
		return "", errors.New("empty flag")
	case 1:
		if flag[0] == '"' {
			return "", fmt.Errorf("invalid flag: %q", flag)
		}
		return `"` + flag + `"`, nil
	case 2:
		if flag == `""` {
			return "", errors.New("empty flag")
		}
		if flag[0] == '"' {
			return "", errors.New("unclosed quote")
		}
		if flag[1] == '"' {
			return "", errors.New("unopened quote")
		}
		return `"` + flag + `"`, nil
	}

	last := len(flag) - 1
	if flag[0] == '"' && flag[last] == '"' {
		return flag, nil
	}
	if flag[0] == '"' {
		return "", errors.New("unclosed quote")
	}
	if flag[last] == '"' {
		return "", errors.New("unopened quote")
	}
	return `"` + flag + `"`, nil
}
