package main

import (
	"strings"
)

func removeGoComments(content string) string {
	var result strings.Builder
	lines := strings.Split(content, "\n")

	// Track state across lines since Go supports multi-line raw strings and block comments
	inBlockComment := false
	inRawStringMultiline := false

	for i, line := range lines {
		// Handle continuation of multi-line raw string from previous line
		if inRawStringMultiline {
			if idx := strings.Index(line, "`"); idx != -1 {
				// Found the closing backtick - preserve content up to and including it
				result.WriteString(line[:idx+1])
				inRawStringMultiline = false
				// Continue processing remainder of line in case there's code after the raw string
				line = line[idx+1:]
			} else {
				// Still inside raw string - preserve entire line as-is
				result.WriteString(line)
				if i < len(lines)-1 {
					result.WriteString("\n")
				}
				continue
			}
		}

		// Handle continuation of block comment from previous line
		if inBlockComment {
			if idx := strings.Index(line, "*/"); idx != -1 {
				inBlockComment = false
				// Continue processing remainder of line after block comment ends
				line = line[idx+2:]
			} else {
				// Still inside block comment - preserve line structure but not content
				result.WriteString("\n")
				continue
			}
		}

		// Process line character by character to handle inline strings and comments
		var cleaned strings.Builder
		inString := false
		inRawString := false
		inRune := false
		escaped := false
		j := 0

		// Use runes instead of bytes to properly handle Unicode characters
		runes := []rune(line)

		for j < len(runes) {
			ch := runes[j]

			if escaped {
				cleaned.WriteRune(ch)
				escaped = false
				j++
				continue
			}

			// Track escape sequences to avoid treating escaped quotes as string delimiters
			if ch == '\\' && (inString || inRune) {
				cleaned.WriteRune(ch)
				escaped = true
				j++
				continue
			}

			// Handle raw strings (backtick-delimited) which don't support escape sequences
			if ch == '`' && !inString && !inRune {
				if !inRawString {
					inRawString = true
					cleaned.WriteRune(ch)

					// Look ahead to see if raw string closes on same line
					endIdx := strings.IndexRune(string(runes[j+1:]), '`')
					if endIdx == -1 {
						// Raw string spans multiple lines - set flag and preserve rest of line
						inRawStringMultiline = true
						cleaned.WriteString(string(runes[j+1:]))
						break
					}

					j++
					continue
				} else {
					inRawString = false
					cleaned.WriteRune(ch)
					j++
					continue
				}
			}

			// Handle rune literals (single-quoted characters)
			if ch == '\'' && !inString && !inRawString {
				if !inRune {
					inRune = true
				} else {
					inRune = false
				}
				cleaned.WriteRune(ch)
				j++
				continue
			}

			// Handle regular string literals (double-quoted)
			if ch == '"' && !inRawString && !inRune {
				if !inString {
					inString = true
				} else {
					inString = false
				}
				cleaned.WriteRune(ch)
				j++
				continue
			}

			// Preserve all content inside strings/runes without processing for comments
			if inString || inRawString || inRune {
				cleaned.WriteRune(ch)
				j++
				continue
			}

			// Detect block comments outside of strings
			if j+1 < len(runes) && runes[j] == '/' && runes[j+1] == '*' {
				inBlockComment = true

				// Check if block comment closes on same line
				if endIdx := strings.Index(string(runes[j+2:]), "*/"); endIdx != -1 {
					inBlockComment = false
					// Skip past the entire inline block comment
					j += endIdx + 4
					continue
				}

				// Block comment continues to next line
				break
			}

			// Detect line comments - everything after '//' is ignored
			if j+1 < len(runes) && runes[j] == '/' && runes[j+1] == '/' {
				break
			}

			cleaned.WriteRune(ch)
			j++
		}

		// Remove trailing whitespace but preserve the line structure
		trimmed := strings.TrimRight(cleaned.String(), " \t")
		result.WriteString(trimmed)

		// Preserve newlines except after the last line
		if i < len(lines)-1 {
			result.WriteString("\n")
		}
	}

	return result.String()
}
