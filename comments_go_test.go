package main

import (
	"testing"
)

func TestRemoveGoComments(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name: "single line comment",
			input: `package main
func main() {
	x := 5 // this is a comment
	y := 10
}`,
			expected: `package main
func main() {
	x := 5
	y := 10
}`,
		},
		{
			name: "block comment",
			input: `package main
/* This is a
   block comment */
func main() {}`,
			expected: `package main
func main() {}`,
		},
		{
			name:     "inline block comment",
			input:    `x := /* comment */ 5`,
			expected: `x :=  5`,
		},
		{
			// Critical test: ensures the parser distinguishes between comment syntax
			// inside string literals (which should be preserved) vs actual comments
			name: "string with comment-like content",
			input: `s := "// not a comment"
s2 := "/* also not */"`,
			expected: `s := "// not a comment"
s2 := "/* also not */"`,
		},
		{
			// Tests rune literal handling - single quotes use different parsing rules
			// than double quotes, and comment characters inside runes must be preserved
			name: "rune literals",
			input: `r := '/' // comment
r2 := '*'
r3 := '\''`,
			expected: `r := '/'
r2 := '*'
r3 := '\''`,
		},
		{
			// Ensures escaped quotes don't prematurely terminate string parsing,
			// which would incorrectly treat the remainder as code/comments
			name: "escaped quotes in string",
			input: `s := "He said \"hello\" // comment"
// another comment`,
			expected: `s := "He said \"hello\" // comment"
`,
		},
		{
			name: "mixed comments and code",
			input: `// Package comment
package main
/*
Multi-line
comment
*/
func main() {
	x := 5 // inline
	/* block */ y := 10
}`,
			expected: `
package main
func main() {
	x := 5
	 y := 10
}`,
		},
		{
			// Edge case: verifies trailing comments don't cause issues with EOF handling
			name: "comment at end of file",
			input: `package main
// final comment`,
			expected: `package main
`,
		},
		{
			// Tests escape sequences within runes - the parser must not confuse
			// the escaped character with comment syntax (e.g., '\n' contains 'n', not newline)
			name: "empty rune",
			input: `r := '\n' // newline rune
x := 5`,
			expected: `r := '\n'
x := 5`,
		},
	}

	// Range over slice creates a copy of the struct on each iteration
	for _, tt := range tests {
		// Parallel test execution requires capturing tt in closure scope
		// to avoid race conditions from loop variable reuse
		t.Run(tt.name, func(t *testing.T) {
			result := removeGoComments(tt.input)

			if result != tt.expected {
				t.Errorf("removeGoComments() failed\nInput:\n%s\n\nExpected:\n%s\n\nGot:\n%s", tt.input, tt.expected, result)
			}
		})
	}
}
