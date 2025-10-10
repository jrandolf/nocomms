package main

import (
	"testing"
)

func TestRemoveJSComments(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name: "single line comment",
			input: `const x = 5; // this is a comment
const y = 10;`,
			expected: `const x = 5;
const y = 10;`,
		},
		{
			name: "block comment",
			input: `const x = 5;
/* this is a
   block comment */
const y = 10;`,
			expected: `const x = 5;
const y = 10;`,
		},
		{
			name:     "inline block comment",
			input:    `const x = /* comment */ 5;`,
			expected: `const x =  5;`,
		},

		// String literal edge cases - critical to test because comment markers inside strings must be preserved
		{
			name: "string with comment-like content",
			input: `const str = "// not a comment";
const str2 = '/* also not */';`,
			expected: `const str = "// not a comment";
const str2 = '/* also not */';`,
		},
		{
			name:     "template literal with comment-like content",
			input:    "const str = `// not a comment\n/* still not */`;",
			expected: "const str = `// not a comment\n/* still not */`;",
		},
		{
			// Escaped quotes don't terminate the string, so comment markers after them are still inside the string
			name: "escaped quotes in string",
			input: `const str = "He said \"hello\" // comment";
// another comment`,
			expected: `const str = "He said \"hello\" // comment";`,
		},

		{
			name: "mixed comments and code",
			input: `// header comment
const x = 5; // inline comment
/* block
   comment */
const y = 10; /* inline block */ const z = 15;`,
			expected: `
const x = 5;
const y = 10;  const z = 15;`,
		},
		{
			name: "empty lines",
			input: `const x = 5;
// comment
const y = 10;`,
			expected: `const x = 5;
const y = 10;`,
		},
		{
			name: "comment at end of file",
			input: `const x = 5;
// final comment`,
			expected: `const x = 5;`,
		},
		{
			name: "only comments",
			input: `// comment 1
/* comment 2 */
// comment 3`,
			expected: `
`,
		},

		{
			// Backslashes in template literals don't escape quotes (unlike regular strings), so they need special handling
			name:     "backslash in template literal",
			input:    "const str = `path\\\\to\\\\file`; // comment",
			expected: "const str = `path\\\\to\\\\file`;",
		},

		{
			// JavaScript doesn't support nested block comments - the first */ closes the outermost /*
			// This test verifies the parser correctly handles this edge case
			name: "nested block comments not supported",
			input: `/* outer /* inner */ still in comment */
const x = 5;`,
			expected: ` still in comment */
const x = 5;`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := removeJSComments(tt.input)
			if result != tt.expected {
				t.Errorf("removeJSComments() failed\nInput:\n%s\n\nExpected:\n%s\n\nGot:\n%s", tt.input, tt.expected, result)
			}
		})
	}
}

func TestRemoveJSCommentsTypeScript(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name: "typescript interface",
			input: `// Interface definition
interface User {
  name: string; // user's name
  age: number; /* user's age */
}`,
			expected: `
interface User {
  name: string;
  age: number;
}`,
		},
		{
			// TypeScript generics use < and > which could be confused with comparison operators
			// This ensures the comment removal doesn't break on angle bracket syntax
			name: "typescript generics",
			input: `function map<T, U>(arr: T[]): U[] {
  // implementation
  return arr.map(/* ... */);
}`,
			expected: `function map<T, U>(arr: T[]): U[] {
  return arr.map();
}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := removeJSComments(tt.input)
			if result != tt.expected {
				t.Errorf("removeJSComments() failed for TypeScript\nInput:\n%s\n\nExpected:\n%s\n\nGot:\n%s", tt.input, tt.expected, result)
			}
		})
	}
}
