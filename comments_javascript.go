package main

import (
	"strings"
)

func removeJSComments(content string) string {
	var result strings.Builder
	lines := strings.Split(content, "\n")

	// Track state across lines since comments and template literals can span multiple lines
	inBlockComment := false
	inTemplateLiteralMultiline := false

	for i, line := range lines {
		// Handle continuation of multiline template literals - must preserve all content
		// including comment-like syntax (e.g., `text // not a comment` or `text /* still not */`)
		if inTemplateLiteralMultiline {
			result.WriteString(line)

			if strings.Contains(line, "`") {
				// Check if this backtick actually closes the template literal
				// Must count preceding backslashes to detect escaped backticks
				for idx := 0; idx < len(line); idx++ {
					if line[idx] == '`' {
						escapeCount := 0
						for k := idx - 1; k >= 0 && line[k] == '\\'; k-- {
							escapeCount++
						}

						// Even number of backslashes means the backtick is NOT escaped
						if escapeCount%2 == 0 {
							inTemplateLiteralMultiline = false
							break
						}
					}
				}
			}

			if i < len(lines)-1 {
				result.WriteString("\n")
			}
			continue
		}

		// Handle continuation of block comments from previous lines
		if inBlockComment {
			if idx := strings.Index(line, "*/"); idx != -1 {
				inBlockComment = false
				// Process remainder of line after comment closes
				line = line[idx+2:]
			} else {
				// Entire line is still inside block comment, preserve newline structure
				result.WriteString("\n")
				continue
			}
		}
		// Character-by-character parsing state for this line
		var cleaned strings.Builder
		inString := false
		inTemplateLiteral := false
		stringChar := rune(0)  // Track which quote type started the string (' or ")
		escaped := false

		j := 0
		runes := []rune(line)

		for j < len(runes) {
			ch := runes[j]

			// Escaped characters are always literal, never syntax
			if escaped {
				cleaned.WriteRune(ch)
				escaped = false
				j++
				continue
			}

			// Backslash starts escape sequence within strings/templates
			if ch == '\\' && (inString || inTemplateLiteral) {
				cleaned.WriteRune(ch)
				escaped = true
				j++
				continue
			}

			// Handle string literals (' and ") but not when inside template literals
			// (template literals can contain unescaped quotes)
			if !inTemplateLiteral && (ch == '"' || ch == '\'') {
				if !inString {
					inString = true
					stringChar = ch
				} else if ch == stringChar {
					inString = false
					stringChar = 0
				}
				cleaned.WriteRune(ch)
				j++
				continue
			}
			// Template literals (backticks) require lookahead to determine if they're single or multiline
			if ch == '`' && !inString {
				if !inTemplateLiteral {
					inTemplateLiteral = true
					endIdx := -1

					// Look ahead to find closing backtick on same line
					for k := j + 1; k < len(runes); k++ {
						if runes[k] == '`' {
							escapeCount := 0
							for m := k - 1; m >= 0 && runes[m] == '\\'; m-- {
								escapeCount++
							}

							// Even escapes = unescaped backtick = actual close
							if escapeCount%2 == 0 {
								endIdx = k
								break
							}
						}
					}

					// No closing backtick on this line = multiline template literal
					// Must preserve remainder of line and switch to multiline mode
					if endIdx == -1 {
						inTemplateLiteralMultiline = true
						cleaned.WriteString(string(runes[j:]))
						break
					}
				} else {
					inTemplateLiteral = false
				}
				cleaned.WriteRune(ch)
				j++
				continue
			}

			// Inside strings or template literals, preserve everything (including comment syntax)
			if inString || inTemplateLiteral {
				cleaned.WriteRune(ch)
				j++
				continue
			}
			// Block comment start - check if it closes on same line
			if j+1 < len(runes) && runes[j] == '/' && runes[j+1] == '*' {
				inBlockComment = true

				// Optimize single-line block comments by skipping over them immediately
				if endIdx := strings.Index(string(runes[j+2:]), "*/"); endIdx != -1 {
					inBlockComment = false
					j += endIdx + 4  // Skip past the entire comment including */
					continue
				}

				// Comment extends beyond this line
				break
			}

			// Line comment - rest of line is a comment
			if j+1 < len(runes) && runes[j] == '/' && runes[j+1] == '/' {
				break
			}

			cleaned.WriteRune(ch)
			j++
		}

		// Only write cleaned line if we're not entering multiline template mode
		// (multiline template content was already written above)
		if !inTemplateLiteralMultiline {
			// Remove trailing whitespace but preserve line structure
			trimmed := strings.TrimRight(cleaned.String(), " \t")
			result.WriteString(trimmed)
		}

		if i < len(lines)-1 {
			result.WriteString("\n")
		}
	}

	return result.String()
}
