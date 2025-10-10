package main

import (
	"testing"
)

func TestRemovePythonComments(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name: "single line comment",
			input: `x = 5  # this is a comment
y = 10`,
			expected: `x = 5
y = 10`,
		},
		{
			// Empty line preserved at start because Python comment removal must maintain line structure
			// for any leading comments to avoid shifting code positioning
			name: "comment at start of line",
			input: `# header comment
x = 5
# another comment
y = 10`,
			expected: `
x = 5
y = 10`,
		},
		{
			name: "string with hash",
			input: `s = "# not a comment"
s2 = '# also not'`,
			expected: `s = "# not a comment"
s2 = '# also not'`,
		},
		{
			// Python's triple-quoted strings can span multiple lines and contain ANY characters
			// including # symbols, which must not be treated as comments
			name: "multiline string triple quotes",
			input: `s = """This is a
multiline string
# not a comment"""
x = 5`,
			expected: `s = """This is a
multiline string
# not a comment"""
x = 5`,
		},
		{
			// Python allows both """ and ''' for multiline strings - both must be handled identically
			name: "multiline string single quotes",
			input: `s = '''This is a
multiline string
# not a comment'''
x = 5`,
			expected: `s = '''This is a
multiline string
# not a comment'''
x = 5`,
		},
		{
			// Docstrings are semantically documentation but syntactically strings,
			// so they should be preserved even though they describe the code
			name: "docstring",
			input: `def foo():
    """
    This is a docstring
    # not a comment
    """
    x = 5`,
			expected: `def foo():
    """
    This is a docstring
    # not a comment
    """
    x = 5`,
		},
		{
			// Triple quotes can also be used inline (single line), and comments after them
			// on the same line should still be removed
			name:     "inline multiline string",
			input:    `x = """single line""" # comment`,
			expected: `x = """single line"""`,
		},
		{
			// Escaped quotes inside strings don't terminate the string, so # after them
			// is still inside the string and not a comment
			name: "escaped quotes in string",
			input: `s = "He said \"hello\" # comment"
# another comment`,
			expected: `s = "He said \"hello\" # comment"`,
		},
		{
			name: "mixed strings and comments",
			input: `# header
x = "string"  # comment
y = 'another'
# footer`,
			expected: `
x = "string"
y = 'another'`,
		},
		{
			// Empty strings are still strings and must be properly tracked
			// to avoid treating subsequent # as inside a string
			name: "empty string",
			input: `s = ""  # comment
s2 = ''`,
			expected: `s = ""
s2 = ''`,
		},
		{
			name: "comment at end of file",
			input: `x = 5
# final comment`,
			expected: `x = 5`,
		},
		{
			// When all content is comments, result should be empty lines matching the structure,
			// testing the edge case of no actual code
			name: "only comments",
			input: `# comment 1
# comment 2
# comment 3`,
			expected: `
`,
		},
		{
			// Backslashes in Python strings are escape sequences, but \\ is a literal backslash,
			// not an escape for the next character - important for Windows paths
			name:     "backslash in string",
			input:    `s = "path\\to\\file"  # comment`,
			expected: `s = "path\\to\\file"`,
		},
		{
			// f-strings (formatted string literals) are still strings despite the f prefix,
			// and # inside them is not a comment
			name: "hash in f-string",
			input: `s = f"value: {x}"  # comment
s2 = f"# not a comment"`,
			expected: `s = f"value: {x}"
s2 = f"# not a comment"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := removePythonComments(tt.input)

			if result != tt.expected {
				t.Errorf("removePythonComments() failed\nInput:\n%s\n\nExpected:\n%s\n\nGot:\n%s", tt.input, tt.expected, result)
			}
		})
	}
}
