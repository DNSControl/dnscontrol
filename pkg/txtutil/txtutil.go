package txtutil

// ToChunks returns the string as chunks of 255-octet strings (the last string being the remainder).
func ToChunks(s string) []string {
	return splitChunks(s, 255)
}

func splitChunks(buf string, lim int) []string {
	var chunk string
	chunks := make([]string, 0, len(buf)/lim+1)
	for len(buf) >= lim {
		chunk, buf = buf[:lim], buf[lim:]
		chunks = append(chunks, chunk)
	}
	if len(buf) > 0 {
		chunks = append(chunks, buf)
	}
	return chunks
}
