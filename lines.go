package main

import "strings"

// parseLines splits content into text (terminator stripped, used for
// comparison) and raw (terminator preserved, used for output) lines.
func parseLines(content []byte) (texts, raws []string) {
	s := string(content)
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] != '\n' {
			continue
		}
		raw := s[start : i+1]
		raws = append(raws, raw)
		texts = append(texts, trimTerminator(raw))
		start = i + 1
	}
	if start < len(s) {
		raw := s[start:]
		raws = append(raws, raw)
		texts = append(texts, trimTerminator(raw))
	}
	return texts, raws
}

func trimTerminator(line string) string {
	line = strings.TrimSuffix(line, "\n")
	line = strings.TrimSuffix(line, "\r")
	return line
}

// markKept reports which original lines still appear, in order, in edited. It
// returns false when edited contains a line that cannot be matched in order,
// meaning the user added or modified a line rather than only deleting them.
func markKept(original, edited []string) (kept []bool, ok bool) {
	kept = make([]bool, len(original))
	next := 0
	for _, line := range edited {
		for next < len(original) && original[next] != line {
			next++
		}
		if next == len(original) {
			return nil, false
		}
		kept[next] = true
		next++
	}
	return kept, true
}
