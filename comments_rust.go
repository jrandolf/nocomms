package main

import (
	"strings"
)

func removeRustComments(content string) string {
	var result strings.Builder
	lines := strings.Split(content, "\n")

	// Rust allows nested block comments (/* /* nested */ */), so we must track depth
	inBlockComment := false
	blockCommentDepth := 0

	for i, line := range lines {
		// If we're inside a block comment from a previous line, continue processing it
		if inBlockComment {
			for idx := 0; idx < len(line); {
				if idx+1 < len(line) && line[idx] == '/' && line[idx+1] == '*' {
					blockCommentDepth++
					idx += 2
					continue
				}

				if idx+1 < len(line) && line[idx] == '*' && line[idx+1] == '/' {
					blockCommentDepth--

					// Only exit block comment when all nested levels are closed
					if blockCommentDepth == 0 {
						inBlockComment = false
						// Resume processing the rest of this line after the closing */
						line = line[idx+2:]
						break
					}
					idx += 2
					continue
				}
				idx++
			}

			// Entire line was inside block comment - preserve the newline structure
			if inBlockComment {
				result.WriteString("\n")
				continue
			}
		}

		var cleaned strings.Builder
		inString := false
		inRawString := false
		inChar := false
		escaped := false
		j := 0

		// Use runes instead of bytes to correctly handle multi-byte UTF-8 characters
		runes := []rune(line)

		for j < len(runes) {
			ch := runes[j]

			if escaped {
				cleaned.WriteRune(ch)
				escaped = false
				j++
				continue
			}

			if ch == '\\' && (inString || inChar) {
				cleaned.WriteRune(ch)
				escaped = true
				j++
				continue
			}

			// Rust raw strings: r"text", r#"text"#, r##"text"##, etc.
			// The number of # must match on both ends, making comment markers inside safe
			if !inString && !inChar && ch == 'r' && j+1 < len(runes) {
				hashCount := 0
				k := j + 1

				// Count the hash symbols to determine the delimiter
				for k < len(runes) && runes[k] == '#' {
					hashCount++
					k++
				}

				if k < len(runes) && runes[k] == '"' {
					inRawString = true
					cleaned.WriteString(string(runes[j : k+1]))
					j = k + 1

					// Look for closing delimiter with matching hash count
					delimiter := `"` + strings.Repeat("#", hashCount)
					if endIdx := strings.Index(string(runes[j:]), delimiter); endIdx != -1 {
						cleaned.WriteString(string(runes[j : j+endIdx+len(delimiter)]))
						j += endIdx + len(delimiter)
						inRawString = false
						continue
					}

					// Raw string extends beyond this line - capture rest and continue on next line
					cleaned.WriteString(string(runes[j:]))
					break
				}
			}

			if ch == '\'' && !inString && !inRawString {
				if !inChar {
					inChar = true
				} else {
					inChar = false
				}
				cleaned.WriteRune(ch)
				j++
				continue
			}

			if ch == '"' && !inRawString && !inChar {
				if !inString {
					inString = true
				} else {
					inString = false
				}
				cleaned.WriteRune(ch)
				j++
				continue
			}

			// Preserve all content inside strings and char literals
			if inString || inRawString || inChar {
				cleaned.WriteRune(ch)
				j++
				continue
			}

			// Handle block comments with nesting support
			if j+1 < len(runes) && runes[j] == '/' && runes[j+1] == '*' {
				inBlockComment = true
				blockCommentDepth = 1
				k := j + 2

				// Try to find the closing */ on this same line, tracking nesting depth
				for k < len(runes) {
					if k+1 < len(runes) && runes[k] == '/' && runes[k+1] == '*' {
						blockCommentDepth++
						k += 2
						continue
					}

					if k+1 < len(runes) && runes[k] == '*' && runes[k+1] == '/' {
						blockCommentDepth--

						if blockCommentDepth == 0 {
							inBlockComment = false
							// Continue processing code after the comment on this line
							j = k + 2
							break
						}
						k += 2
						continue
					}
					k++
				}

				// Block comment continues to next line - stop processing this line
				if inBlockComment {
					break
				}
				continue
			}

			// Line comments extend to end of line - nothing more to process
			if j+1 < len(runes) && runes[j] == '/' && runes[j+1] == '/' {
				break
			}

			cleaned.WriteRune(ch)
			j++
		}

		// Remove trailing whitespace but preserve the line structure
		trimmed := strings.TrimRight(cleaned.String(), " \t")
		result.WriteString(trimmed)

		if i < len(lines)-1 {
			result.WriteString("\n")
		}
	}

	return result.String()
}
