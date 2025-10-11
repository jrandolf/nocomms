package main

import (
	"testing"
)

func TestRemoveYAMLComments(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name: "single line comment",
			input: `key: value  # this is a comment
another: value`,
			expected: `key: value
another: value`,
		},
		{
			// YAML comments at start of line should leave empty line to maintain structure
			name: "comment at start of line",
			input: `# header comment
key: value
# another comment
nested:
  child: value`,
			expected: `
key: value

nested:
  child: value`,
		},
		{
			name: "string with hash in double quotes",
			input: `message: "# not a comment"
another: "value # still not"`,
			expected: `message: "# not a comment"
another: "value # still not"`,
		},
		{
			name: "string with hash in single quotes",
			input: `message: '# not a comment'
another: 'value # still not'`,
			expected: `message: '# not a comment'
another: 'value # still not'`,
		},
		{
			// Double-quoted strings use backslash escaping similar to most languages
			name: "escaped quotes in double-quoted string",
			input: `message: "He said \"hello\" # comment"
# another comment`,
			expected: `message: "He said \"hello\" # comment"
`,
		},
		{
			// Single-quoted strings in YAML use '' to escape a single quote, not backslash
			name: "escaped quotes in single-quoted string",
			input: `message: 'It''s working # comment'
# another comment`,
			expected: `message: 'It''s working # comment'
`,
		},
		{
			name: "mixed strings and comments",
			input: `# header
key: "string"  # comment
another: 'value'
# footer`,
			expected: `
key: "string"
another: 'value'
`,
		},
		{
			// Empty strings are valid YAML values and must be properly tracked
			name: "empty string",
			input: `empty: ""  # comment
also_empty: ''`,
			expected: `empty: ""
also_empty: ''`,
		},
		{
			name: "comment at end of file",
			input: `key: value
# final comment`,
			expected: `key: value
`,
		},
		{
			// When all content is comments, result should be empty lines matching structure
			name: "only comments",
			input: `# comment 1
# comment 2
# comment 3`,
			expected: `

`,
		},
		{
			// Backslash in double-quoted YAML strings is an escape character
			name: "backslash in double-quoted string",
			input: `path: "C:\\path\\to\\file"  # comment`,
			expected: `path: "C:\\path\\to\\file"`,
		},
		{
			// Single-quoted strings in YAML treat backslash as literal, not escape
			name: "backslash in single-quoted string",
			input: `path: 'C:\path\to\file'  # comment`,
			expected: `path: 'C:\path\to\file'`,
		},
		{
			// Complex YAML structure with nested objects and arrays
			name: "complex nested structure",
			input: `# Configuration file
app:
  name: myapp  # application name
  version: 1.0
  # database settings
  database:
    host: localhost  # db host
    port: 5432`,
			expected: `
app:
  name: myapp
  version: 1.0

  database:
    host: localhost
    port: 5432`,
		},
		{
			// YAML arrays with comments
			name: "arrays with comments",
			input: `# List of items
items:
  - name: item1  # first item
  - name: item2  # second item
  # - name: item3  # commented out`,
			expected: `
items:
  - name: item1
  - name: item2
`,
		},
		{
			// Inline arrays with comments
			name: "inline arrays",
			input: `ports: [8080, 8081, 8082]  # exposed ports
tags: ["dev", "staging"]  # environment tags`,
			expected: `ports: [8080, 8081, 8082]
tags: ["dev", "staging"]`,
		},
		{
			// Multi-line strings in YAML (using | or >) don't need quotes
			// but are separate YAML constructs, not handled by quote escaping
			name: "literal block scalar",
			input: `description: |
  This is a multi-line
  # this looks like a comment but is part of the string
  description
key: value  # actual comment`,
			expected: `description: |
  This is a multi-line

  description
key: value`,
		},
		{
			// Edge case: hash immediately after colon
			name: "hash after colon",
			input: `key:# comment
value: test`,
			expected: `key:
value: test`,
		},
		{
			// Edge case: multiple hashes in line
			name: "multiple hashes",
			input: `key: "value # with # hashes"  # real comment # more comment`,
			expected: `key: "value # with # hashes"`,
		},
		{
			// Boolean and numeric values with comments
			name: "boolean and numeric values",
			input: `enabled: true  # feature flag
count: 42  # answer
ratio: 3.14  # pi`,
			expected: `enabled: true
count: 42
ratio: 3.14`,
		},
		{
			// null values
			name: "null values",
			input: `optional: null  # not set
implicit:  # also null`,
			expected: `optional: null
implicit:`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := removeYAMLComments(tt.input)

			if result != tt.expected {
				t.Errorf("removeYAMLComments() failed\nInput:\n%s\n\nExpected:\n%s\n\nGot:\n%s", tt.input, tt.expected, result)
			}
		})
	}
}
