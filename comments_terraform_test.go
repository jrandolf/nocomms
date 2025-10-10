package main

import (
	"strings"
	"testing"
)

func TestRemoveTerraformComments(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name: "hash line comment",
			input: `resource "aws_instance" "example" {
  # This is a comment
  ami = "ami-123456"
}`,
			expected: "resource \"aws_instance\" \"example\" {\n  \n  ami = \"ami-123456\"\n}",
		},
		{
			name: "double slash line comment",
			input: `variable "name" {
  // This is a comment
  type = string
}`,
			expected: "variable \"name\" {\n  \n  type = string\n}",
		},
		{
			name: "block comment",
			input: `resource "aws_instance" "web" {
  /* This is a
     block comment */
  ami = "ami-123"
}`,
			expected: "resource \"aws_instance\" \"web\" {\n  \n  ami = \"ami-123\"\n}",
		},
		{
			name: "inline hash comment",
			input: `ami = "ami-123" # inline comment
instance_type = "t2.micro"`,
			expected: "ami = \"ami-123\" \ninstance_type = \"t2.micro\"",
		},
		{
			name: "inline block comment",
			input: `ami = "ami-123" /* inline */ instance_type = "t2.micro"`,
			expected: `ami = "ami-123"  instance_type = "t2.micro"`,
		},
		{
			name: "string with comment-like content",
			input: `description = "This is # not a comment"
name = "test // also not"`,
			expected: `description = "This is # not a comment"
name = "test // also not"`,
		},
		{
			name: "string with block comment-like content",
			input: `description = "This /* is */ not removed"`,
			expected: `description = "This /* is */ not removed"`,
		},
		{
			name: "heredoc with comments inside",
			input: `user_data = <<EOF
#!/bin/bash
# This comment inside heredoc should be preserved
echo "Hello"
EOF`,
			expected: `user_data = <<EOF
#!/bin/bash
# This comment inside heredoc should be preserved
echo "Hello"
EOF`,
		},
		{
			name: "indented heredoc",
			input: `user_data = <<-EOF
  # This should be preserved
  echo "test"
  EOF`,
			expected: `user_data = <<-EOF
  # This should be preserved
  echo "test"
  EOF`,
		},
		{
			name: "mixed comments",
			input: `# Header comment
resource "aws_instance" "web" {
  ami = "ami-123" # inline
  /* block
     comment */
  instance_type = "t2.micro" // double slash
}`,
			expected: "\nresource \"aws_instance\" \"web\" {\n  ami = \"ami-123\" \n  \n  instance_type = \"t2.micro\" \n}",
		},
		{
			name: "escaped quotes in string",
			input: `description = "He said \"hello\" # comment"
# another comment`,
			expected: "description = \"He said \\\"hello\\\" # comment\"\n",
		},
		{
			name: "empty lines and only comments",
			input: `# Comment 1

# Comment 2
resource "test" "example" {}`,
			expected: `


resource "test" "example" {}`,
		},
		{
			name: "comment at end of file",
			input: `resource "test" "example" {}
# final comment`,
			expected: `resource "test" "example" {}
`,
		},
		{
			name: "multiple heredocs",
			input: `user_data = <<EOF
# Preserved
EOF
# Removed comment
metadata = <<-METADATA
  # Also preserved
  METADATA`,
			expected: `user_data = <<EOF
# Preserved
EOF

metadata = <<-METADATA
  # Also preserved
  METADATA`,
		},
		{
			name: "nested strings",
			input: `value = "outer \"inner # not comment\" end"`,
			expected: `value = "outer \"inner # not comment\" end"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := removeTerraformComments(tt.input)
			if result != tt.expected {
				t.Errorf("removeTerraformComments() failed\n"+
					"Input:\n%s\n"+
					"Expected:\n%s\n"+
					"Got:\n%s",
					tt.input, tt.expected, result)
			}
		})
	}
}

func TestRemoveTerraformCommentsPreservesCode(t *testing.T) {
	// Ensure that removing comments doesn't break valid Terraform code
	input := `terraform {
  required_version = ">= 1.0"
}

provider "aws" {
  region = "us-west-2"
}

resource "aws_instance" "web" {
  ami           = "ami-123456"
  instance_type = "t2.micro"

  tags = {
    Name = "WebServer"
  }
}

output "instance_id" {
  value = aws_instance.web.id
}`

	result := removeTerraformComments(input)

	// Check that key elements are still present
	requiredElements := []string{
		"terraform",
		"provider",
		"resource",
		"aws_instance",
		"ami-123456",
		"t2.micro",
		"output",
	}

	for _, element := range requiredElements {
		if !strings.Contains(result, element) {
			t.Errorf("removeTerraformComments() removed required element: %s", element)
		}
	}
}
