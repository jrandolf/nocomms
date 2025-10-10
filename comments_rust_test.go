package main

import (
	"testing"
)

func TestRemoveRustComments(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name: "single line comment",
			input: `fn main() {
    let x = 5; // this is a comment
    let y = 10;
}`,
			expected: `fn main() {
    let x = 5;
    let y = 10;
}`,
		},
		{
			name: "block comment",
			input: `fn main() {
    /* this is a
       block comment */
    let x = 5;
}`,
			expected: `fn main() {
    let x = 5;
}`,
		},
		{
			// Rust allows nested block comments unlike C/C++ - this is a language-specific feature
			// that must be handled correctly to avoid leaving orphaned comment delimiters
			name: "nested block comments",
			input: `fn main() {
    /* outer /* inner */ still outer */
    let x = 5;
}`,
			expected: `fn main() {
    let x = 5;
}`,
		},
		{
			name:     "inline block comment",
			input:    `let x = /* comment */ 5;`,
			expected: `let x =  5;`,
		},
		{
			// Ensures comment syntax inside strings is preserved, not treated as actual comments
			name: "string with comment-like content",
			input: `let s = "// not a comment";
let s2 = "/* also not */";`,
			expected: `let s = "// not a comment";
let s2 = "/* also not */";`,
		},
		{
			// Raw strings (r"..." or r#"..."#) can contain unescaped quotes and comment-like text
			// The hash count in r#"..."# must match on both sides, requiring careful parsing
			name: "raw string",
			input: `let s = r"// not a comment";
let s2 = r#"/* also not */"#;`,
			expected: `let s = r"// not a comment";
let s2 = r#"/* also not */"#;`,
		},
		{
			// Char literals can contain comment delimiters ('/', '*') and must be distinguished
			// from division operators or actual comment starts
			name: "char literal",
			input: `let c = '/'; // comment
let c2 = '*';
let c3 = '\'';`,
			expected: `let c = '/';
let c2 = '*';
let c3 = '\'';`,
		},
		{
			// Escaped quotes within strings shouldn't terminate the string early
			name: "escaped quotes in string",
			input: `let s = "He said \"hello\" // comment";
// another comment`,
			expected: `let s = "He said \"hello\" // comment";`,
		},
		{
			name: "mixed comments and code",
			input: `// function comment
fn main() {
    /* block
       comment */
    let x = 5; // inline
    /* another */ let y = 10;
}`,
			expected: `
fn main() {
    let x = 5;
     let y = 10;
}`,
		},
		{
			// Doc comments (/// and //!) are syntactically regular comments in Rust
			// but have semantic meaning for documentation generation
			name: "doc comments treated as regular comments",
			input: `/// This is a doc comment
fn foo() {}
//! Module doc comment`,
			expected: `
fn foo() {}`,
		},
		{
			name: "comment at end of file",
			input: `fn main() {}
// final comment`,
			expected: `fn main() {}`,
		},
		{
			// Tests deeply nested block comments - Rust's nesting depth is unlimited
			// Each /* must be matched with its corresponding */ at the correct nesting level
			name: "multiple nested block comments",
			input: `/* level 1 /* level 2 /* level 3 */ level 2 */ level 1 */
let x = 5;`,
			expected: `
let x = 5;`,
		},
		{
			// The number of # symbols must match exactly: r##"..."## allows embedding r#"..."# inside
			// This tests the parser correctly counts hash symbols to find the string terminator
			name:     "raw string with multiple hashes",
			input:    `let s = r##"String with "quotes" and #hash"##; // comment`,
			expected: `let s = r##"String with "quotes" and #hash"##;`,
		},
		{
			// Backslash in char literals requires special handling - '\' is not a single char
			// but '\\' is (escaped backslash), testing escape sequence handling
			name: "char with backslash",
			input: `let c = '\\'; // backslash char
let c2 = '\n';`,
			expected: `let c = '\\';
let c2 = '\n';`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := removeRustComments(tt.input)

			if result != tt.expected {
				t.Errorf("removeRustComments() failed\nInput:\n%s\n\nExpected:\n%s\n\nGot:\n%s", tt.input, tt.expected, result)
			}
		})
	}
}
