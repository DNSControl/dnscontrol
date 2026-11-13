package privatetypes

import "strings"

// TokensToArgs copies tokens, removing surrounding quotes from quoted strings
// and decoding their zonefile escapes (\" \\ \DDD).
// Two consecutive quote tokens represent an empty string.
func TokensToArgs(tokens []string) []string {
	var args []string
	for i := 0; i < len(tokens); i++ {
		if i+1 < len(tokens) && tokens[i] == `"` && tokens[i+1] == `"` {
			args = append(args, "")
			i++
		} else if ((i + 2) < len(tokens)) && tokens[i] == "\"" && tokens[i+2] == "\"" {
			args = append(args, unescape(tokens[i+1]))
			i += 2
		} else {
			args = append(args, tokens[i])
		}
	}
	return args
}

// unescape decodes the zonefile escapes in the contents of a quoted string:
// \DDD is the byte with that decimal value, \X is the literal character X.
func unescape(s string) string {
	if !strings.Contains(s, `\`) {
		return s
	}
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		if s[i] != '\\' || i+1 >= len(s) {
			b.WriteByte(s[i])
			continue
		}
		if i+3 < len(s) && isDigit(s[i+1]) && isDigit(s[i+2]) && isDigit(s[i+3]) {
			if n := int(s[i+1]-'0')*100 + int(s[i+2]-'0')*10 + int(s[i+3]-'0'); n <= 255 {
				b.WriteByte(byte(n))
				i += 3
				continue
			}
		}
		i++
		b.WriteByte(s[i])
	}
	return b.String()
}

func isDigit(c byte) bool { return c >= '0' && c <= '9' }
