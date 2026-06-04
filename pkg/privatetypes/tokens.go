package privatetypes

// TokensToArgs copies tokens. If a token is a quote ("\""), it is followed by the string to be copied followed by another quote. The quote is skipped.
func TokensToArgs(tokens []string) []string {
	var args []string
	for i := 0; i < len(tokens); i++ {
		if ((i + 2) < len(tokens)) && tokens[i] == "\"" && tokens[i+2] == "\"" {
			args = append(args, tokens[i+1])
			i += 2
		} else {
			args = append(args, tokens[i])
		}
	}
	return args
}
