package main

import (
	"strings"
)

func removeYAMLComments(content string) string {
	var result strings.Builder
	lines := strings.Split(content, "\n")

	// YAML comments work like Python - # outside strings marks comment to end of line
	// YAML supports single and double quotes with different escaping rules
	for i, line := range lines {
		var cleaned strings.Builder
		inString := false
		stringDelim := rune(0)
		escaped := false
		runes := []rune(line)

		for j := 0; j < len(runes); j++ {
			ch := runes[j]

			if escaped {
				cleaned.WriteRune(ch)
				escaped = false
				continue
			}

			// In YAML double-quoted strings, backslash escapes the next character
			// Single-quoted strings use '' to escape a single quote, not backslash
			if ch == '\\' && inString && stringDelim == '"' {
				cleaned.WriteRune(ch)
				escaped = true
				continue
			}

			// Handle quote escaping in single-quoted strings ('' represents a single ')
			if inString && stringDelim == '\'' && ch == '\'' && j+1 < len(runes) && runes[j+1] == '\'' {
				cleaned.WriteRune(ch)
				cleaned.WriteRune(runes[j+1])
				j++
				continue
			}

			if (ch == '"' || ch == '\'') && !inString {
				inString = true
				stringDelim = ch
				cleaned.WriteRune(ch)
				continue
			}

			if inString && ch == stringDelim {
				inString = false
				stringDelim = 0
				cleaned.WriteRune(ch)
				continue
			}

			if inString {
				cleaned.WriteRune(ch)
				continue
			}

			// '#' outside of strings marks the start of a comment - discard rest of line
			if ch == '#' {
				break
			}

			cleaned.WriteRune(ch)
		}

		// Remove trailing whitespace to avoid leaving empty spaces where comments were
		trimmed := strings.TrimRight(cleaned.String(), " \t")
		result.WriteString(trimmed)

		if i < len(lines)-1 {
			result.WriteString("\n")
		}
	}

	return result.String()
}
