# nocomms

A Go CLI tool that removes comments from source files and processes them in parallel batches using Claude.

## Features

- Removes comments from multiple programming languages:
  - JavaScript (.js)
  - TypeScript (.ts, .tsx)
  - Go (.go)
  - Python (.py)
  - Rust (.rs)
  - Terraform (.tf, .tfvars)
- Processes files in configurable batch sizes
- Runs Claude commands in parallel for each batch
- Modifies files in place by removing comments before processing with Claude

## Installation

```bash
go build -o nocomms
```

Or install directly:

```bash
go install
```

## Usage

```bash
nocomms [flags] <files...>
```

### Flags

- `-prompt`: Prompt to send to Claude for each file (has a comprehensive default for adding thoughtful comments)
- `-batch-size`: Number of files to process in parallel per batch (default: 5)
- `-force`: Force reprocessing of all files, ignoring the timestamp cache
- `-cache-only`: Mark files as cached without processing them (useful for initializing the cache)

### Examples

Process a single file with default prompt (adds thoughtful comments):
```bash
nocomms main.go
```

Process multiple files with custom batch size:
```bash
nocomms -batch-size 3 *.js
```

Process all TypeScript files with a custom prompt:
```bash
nocomms -prompt "Review for type safety issues in {filename}" src/**/*.ts
```

Force reprocess all files (ignore cache):
```bash
nocomms -force *.go
```

Initialize cache for files without processing them:
```bash
nocomms -cache-only src/**/*.js
```

## How It Works

1. **Comment Removal**: The tool removes all comments from each file in place:
   - Line comments (`//` for JS/TS/Go/Rust, `#` for Python/Terraform)
   - Block comments (`/* */` for JS/TS/Go/Rust/Terraform)
   - Docstrings and multiline strings are preserved in Python
   - Nested block comments are handled in Rust
   - Heredocs are preserved in Terraform
   - Files are modified directly - make sure to commit your changes first!

2. **Whitespace Normalization**: After removing comments, sequences of more than 2 consecutive newlines are collapsed to exactly 2 newlines to prevent excessive blank lines.

3. **Timestamp Cache**: The tool maintains a cache (`.nocomms-cache.json`) in the git repository root to track file modification times. Files are only reprocessed if they've been modified since the last run. Use `-force` to bypass the cache.

4. **Batching**: Files are processed in groups of the specified batch size. Each batch is processed before moving to the next.

5. **Parallel Execution**: Within each batch, the Claude command is executed in parallel for all files:
   ```bash
   claude --dangerously-skip-permissions {PROMPT} {FILE}
   ```

   The `{filename}` placeholder in the prompt will be replaced with the actual file path.

6. **Code Formatting**: After Claude adds comments, the appropriate formatter is automatically run:
   - Go: `go fmt`
   - JavaScript/TypeScript: `biome format --write`
   - Python: `ruff format`
   - Rust: `rustfmt`
   - Terraform: `terraform fmt`

## File Type Detection

File types are detected by extension:
- `.js`, `.jsx` - JavaScript
- `.ts`, `.tsx` - TypeScript
- `.go` - Go
- `.py` - Python
- `.rs` - Rust
- `.tf`, `.tfvars` - Terraform

## Important Notes

**WARNING**: This tool modifies files in place! Comments are permanently removed from the original files before Claude processes them. Make sure to:
- Commit your changes to version control before running
- Have backups of your code
- Test on a small set of files first

**Cache**: The tool creates a `.nocomms-cache.json` file in the git repository root to track processed files. Add this to your `.gitignore` as it's machine-specific. Delete the cache file to force reprocessing of all files, or use the `-force` flag. Use `-cache-only` to mark files as already processed without actually running the tool on them (useful for initializing a cache on an existing codebase).

**Note**: The tool must be run from within a git repository, as it stores the cache at the repository root.

## Prerequisites

The following formatters must be installed and available in your PATH:
- Go: `go` (comes with Go installation)
- JavaScript/TypeScript: `biome` (install via `npm install -g @biomejs/biome`)
- Python: `ruff` (install via `pip install ruff`)
- Rust: `rustfmt` (comes with Rust installation)
- Terraform: `terraform` (install from https://www.terraform.io/downloads)

If a formatter is not installed, the tool will log a warning but continue processing.

## Error Handling

- Files that fail to process will show a warning but won't stop the entire operation
- Claude command failures are reported per file
- If all files fail to process, the tool exits with an error

## Use Case

This tool is designed for adding AI-generated comments to code. The workflow is:

1. Remove all existing comments from your files
2. Send the comment-free code to Claude with a prompt asking it to add thoughtful comments
3. Claude analyzes the code and adds meaningful comments explaining the "why" not the "what"

The default prompt is specifically crafted to generate high-quality, non-redundant comments that focus on explaining rationale, edge cases, and non-obvious behavior rather than restating what the code does.

## License

MIT
