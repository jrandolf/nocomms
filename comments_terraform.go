package main

import (
	"strings"
)

// removeTerraformComments removes line comments (#, //) and block comments (/* */)
// from Terraform code while preserving strings and avoiding comment-like content
// within string literals.
func removeTerraformComments(code string) string {
	var result strings.Builder
	runes := []rune(code)
	i := 0

	for i < len(runes) {
		// Check for double-quoted string
		if runes[i] == '"' {
			result.WriteRune(runes[i])
			i++
			for i < len(runes) {
				result.WriteRune(runes[i])
				if runes[i] == '\\' && i+1 < len(runes) {
					// Skip escaped character
					i++
					result.WriteRune(runes[i])
					i++
					continue
				}
				if runes[i] == '"' {
					i++
					break
				}
				i++
			}
			continue
		}

		// Check for heredoc (<<EOF or <<-EOF)
		if i+2 < len(runes) && runes[i] == '<' && runes[i+1] == '<' {
			// Handle heredoc
			heredocStart := i
			i += 2

			// Check for optional - (indented heredoc)
			if i < len(runes) && runes[i] == '-' {
				i++
			}

			// Read the delimiter
			var delimiter strings.Builder
			for i < len(runes) && (isAlphanumeric(runes[i]) || runes[i] == '_') {
				delimiter.WriteRune(runes[i])
				i++
			}

			if delimiter.Len() > 0 {
				// Write the heredoc start
				for j := heredocStart; j < i; j++ {
					result.WriteRune(runes[j])
				}

				// Copy everything until we find the delimiter on its own line
				delimiterStr := delimiter.String()
				for i < len(runes) {
					// Read the line
					var line strings.Builder
					for i < len(runes) && runes[i] != '\n' {
						line.WriteRune(runes[i])
						i++
					}

					// Write the line
					result.WriteString(line.String())

					// Check if this line is the delimiter
					if strings.TrimSpace(line.String()) == delimiterStr {
						if i < len(runes) {
							result.WriteRune(runes[i]) // Write the newline
							i++
						}
						break
					}

					// Write newline if present
					if i < len(runes) && runes[i] == '\n' {
						result.WriteRune(runes[i])
						i++
					}
				}
				continue
			}
		}

		// Check for # line comment
		if runes[i] == '#' {
			// Skip until end of line
			for i < len(runes) && runes[i] != '\n' {
				i++
			}
			continue
		}

		// Check for // line comment
		if i+1 < len(runes) && runes[i] == '/' && runes[i+1] == '/' {
			// Skip until end of line
			for i < len(runes) && runes[i] != '\n' {
				i++
			}
			continue
		}

		// Check for /* block comment */
		if i+1 < len(runes) && runes[i] == '/' && runes[i+1] == '*' {
			i += 2
			// Skip until */
			for i < len(runes) {
				if i+1 < len(runes) && runes[i] == '*' && runes[i+1] == '/' {
					i += 2
					break
				}
				i++
			}
			continue
		}

		// Regular character
		result.WriteRune(runes[i])
		i++
	}

	return result.String()
}

func isAlphanumeric(r rune) bool {
	return (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9')
}
