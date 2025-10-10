package main

import (
	"strings"
)

func removePythonComments(content string) string {
	var result strings.Builder
	lines := strings.Split(content, "\n")

	// Track multiline string state across lines since Python's triple-quoted strings
	// can span multiple lines and must not be treated as comment delimiters
	inMultilineString := false
	multilineDelim := ""

	for i, line := range lines {
		if inMultilineString {
			result.WriteString(line)

			// Check if the closing delimiter appears on this line
			if idx := strings.Index(line, multilineDelim); idx != -1 {
				inMultilineString = false
				multilineDelim = ""
			}

			if i < len(lines)-1 {
				result.WriteString("\n")
			}
			continue
		}

		var cleaned strings.Builder
		inString := false
		stringDelim := rune(0)
		escaped := false
		j := 0
		runes := []rune(line)

		for j < len(runes) {
			ch := runes[j]

			if escaped {
				cleaned.WriteRune(ch)
				escaped = false
				j++
				continue
			}

			if ch == '\\' && inString {
				cleaned.WriteRune(ch)
				escaped = true
				j++
				continue
			}

			// Check for triple-quoted strings (''' or """) which can span multiple lines
			// Must check before single quote handling to avoid treating ''' as three separate quotes
			if !inString && j+2 < len(runes) {
				if (j+2 < len(runes) && runes[j] == '\'' && runes[j+1] == '\'' && runes[j+2] == '\'') ||
					(j+2 < len(runes) && runes[j] == '"' && runes[j+1] == '"' && runes[j+2] == '"') {
					multilineDelim = string(runes[j : j+3])

					// Check if the triple-quoted string closes on the same line
					if endIdx := strings.Index(string(runes[j+3:]), multilineDelim); endIdx != -1 {
						cleaned.WriteString(string(runes[j : j+3+endIdx+3]))
						j += 3 + endIdx + 3
						multilineDelim = ""
						continue
					}

					// String continues to next line - preserve rest of line and set multiline state
					inMultilineString = true
					cleaned.WriteString(string(runes[j:]))
					break
				}
			}

			if (ch == '"' || ch == '\'') && !inString {
				inString = true
				stringDelim = ch
				cleaned.WriteRune(ch)
				j++
				continue
			}

			if inString && ch == stringDelim {
				inString = false
				stringDelim = 0
				cleaned.WriteRune(ch)
				j++
				continue
			}

			if inString {
				cleaned.WriteRune(ch)
				j++
				continue
			}

			// '#' outside of strings marks the start of a comment - discard rest of line
			if ch == '#' {
				break
			}

			cleaned.WriteRune(ch)
			j++
		}

		if !inMultilineString {
			// Remove trailing whitespace to avoid leaving empty spaces where comments were
			trimmed := strings.TrimRight(cleaned.String(), " \t")
			result.WriteString(trimmed)
		}

		if i < len(lines)-1 {
			result.WriteString("\n")
		}
	}

	return result.String()
}
