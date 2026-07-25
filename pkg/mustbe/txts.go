package mustbe

import (
	"bytes"
	"fmt"
)

func Txts(args ...any) []string {
	if len(args) == 0 {
		return []string{""}
	}
	// Join the parts using a string builder, assuring each part is a string (or converting it if needed)
	var sb bytes.Buffer
	for _, a := range args {
		fmt.Fprintf(&sb, "%v", a)
	}
	return splitChunks(sb.String(), 255)
}

func splitChunks(buf string, lim int) []string {
	if len(buf) == 0 {
		return []string{""}
	}
	if len(buf) <= lim {
		return []string{buf}
	}

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
