package privatetypes

// TokensToArgs copies tokens, removing surrounding quotes from quoted strings.
// Two consecutive quote tokens represent an empty string.
func TokensToArgs(tokens []string) []string {
	var args []string
	for i := 0; i < len(tokens); i++ {
		if i+1 < len(tokens) && tokens[i] == `"` && tokens[i+1] == `"` {
			args = append(args, "")
			i++
		} else if ((i + 2) < len(tokens)) && tokens[i] == "\"" && tokens[i+2] == "\"" {
			args = append(args, tokens[i+1])
			i += 2
		} else {
			args = append(args, tokens[i])
		}
	}
	return args
}
