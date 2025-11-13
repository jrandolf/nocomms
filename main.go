package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

type Config struct {
	Files        []string
	BatchSize    int
	Prompt       string
	ForceProcess bool
	CacheOnly    bool
}

type FileCache struct {
	ProcessedFiles map[string]time.Time `json:"processed_files"`
}

// ErrUnsupportedFileType is returned when a file type is not supported
type ErrUnsupportedFileType struct {
	Extension string
}

func (e *ErrUnsupportedFileType) Error() string {
	return fmt.Sprintf("unsupported file type: %s", e.Extension)
}

const cacheFileName = ".nocomms-cache.json"

// findGitRoot walks up the directory tree to locate the git repository root.
// This approach ensures cache files are stored at the repository level rather than
// scattered across subdirectories, providing consistent cache behavior regardless
// of where the tool is invoked within the repository.
func findGitRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("failed to get working directory: %w", err)
	}

	for {
		gitDir := filepath.Join(dir, ".git")
		if info, err := os.Stat(gitDir); err == nil && info.IsDir() {
			return dir, nil
		}

		parent := filepath.Dir(dir)
		// Reached filesystem root without finding .git directory
		if parent == dir {
			return "", fmt.Errorf("not in a git repository")
		}
		dir = parent
	}
}

func getCachePath() (string, error) {
	gitRoot, err := findGitRoot()
	if err != nil {
		return "", fmt.Errorf("failed to find git repository root: %w", err)
	}

	return filepath.Join(gitRoot, cacheFileName), nil
}

// toRelativePath converts absolute paths to git-root-relative paths for cache storage.
// Relative paths are used in the cache because they remain valid when the repository
// is moved or accessed from different mount points, making the cache portable.
func toRelativePath(absolutePath string) (string, error) {
	gitRoot, err := findGitRoot()
	if err != nil {
		return "", err
	}

	relPath, err := filepath.Rel(gitRoot, absolutePath)
	if err != nil {
		return "", fmt.Errorf("failed to make path relative: %w", err)
	}

	return relPath, nil
}

func toAbsolutePath(relativePath string) (string, error) {
	gitRoot, err := findGitRoot()
	if err != nil {
		return "", err
	}

	return filepath.Join(gitRoot, relativePath), nil
}

// isGitIgnored checks if a file is ignored by git using git check-ignore.
// This respects all .gitignore files in the repository hierarchy.
func isGitIgnored(filePath string) bool {
	cmd := exec.Command("git", "check-ignore", "-q", filePath)
	// check-ignore returns 0 if file is ignored, 1 if not ignored
	err := cmd.Run()
	return err == nil
}

func loadCache() (*FileCache, error) {
	cachePath, err := getCachePath()
	if err != nil {
		return nil, err
	}

	cache := &FileCache{
		ProcessedFiles: make(map[string]time.Time),
	}

	data, err := os.ReadFile(cachePath)
	if err != nil {
		// Missing cache file is not an error; initialize with empty cache
		if os.IsNotExist(err) {
			return cache, nil
		}
		return nil, fmt.Errorf("failed to read cache file: %w", err)
	}

	if err := json.Unmarshal(data, cache); err != nil {
		return nil, fmt.Errorf("failed to parse cache file: %w", err)
	}

	return cache, nil
}

func (c *FileCache) save() error {
	cachePath, err := getCachePath()
	if err != nil {
		return err
	}

	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal cache: %w", err)
	}

	if err := os.WriteFile(cachePath, data, 0o644); err != nil {
		return fmt.Errorf("failed to write cache file: %w", err)
	}

	return nil
}

// shouldProcess determines if a file needs processing by comparing modification times.
// Files are reprocessed only if modified after their last processing time, avoiding
// redundant Claude API calls and preserving rate limits.
func (c *FileCache) shouldProcess(filePath string) (bool, error) {
	info, err := os.Stat(filePath)
	if err != nil {
		return false, fmt.Errorf("failed to stat file: %w", err)
	}

	relPath, err := toRelativePath(filePath)
	if err != nil {
		return false, fmt.Errorf("failed to convert to relative path: %w", err)
	}

	lastProcessed, exists := c.ProcessedFiles[relPath]
	if !exists {
		return true, nil
	}

	// Process if file was modified after last processing
	return info.ModTime().After(lastProcessed), nil
}

// markProcessed records the file's current modification time, not the current time.
// This ensures the cache accurately reflects when the file content was last changed,
// preventing false cache misses if the file is touched but not modified.
func (c *FileCache) markProcessed(filePath string) error {
	info, err := os.Stat(filePath)
	if err != nil {
		return fmt.Errorf("failed to stat file: %w", err)
	}

	relPath, err := toRelativePath(filePath)
	if err != nil {
		return fmt.Errorf("failed to convert to relative path: %w", err)
	}

	c.ProcessedFiles[relPath] = info.ModTime()
	return nil
}

// getStagedFiles retrieves the list of staged files from git.
// These are files that have been added to the git staging area via git add.
func getStagedFiles() ([]string, error) {
	cmd := exec.Command("git", "diff", "--staged", "--name-only")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get staged files: %w", err)
	}

	// Split by newlines and filter out empty strings
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	files := make([]string, 0, len(lines))
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" {
			files = append(files, line)
		}
	}

	if len(files) == 0 {
		return nil, fmt.Errorf("no staged files found")
	}

	return files, nil
}

func main() {
	batchSize := flag.Int("batch-size", 24, "Number of files to process in parallel per batch")
	forceProcess := flag.Bool("force", false, "Force reprocessing of all files, ignoring cache")
	cacheOnly := flag.Bool("cache-only", false, "Mark files as cached without processing (useful for initialization)")
	staged := flag.Bool("staged", false, "Process only staged files from git")
	prompt := flag.String("prompt", `You are tasked with adding thoughtful, meaningful comments to the
{filename} ONLY. Do not modify any other files or suggest
changes to other files.

## Context Gathering for Complex Code
Before adding comments, thoroughly analyze the codebase context:
1. **Examine imports and dependencies** - Look at what external libraries, modules, or APIs this code relies on and their specific behaviors or constraints
2. **Study related files** - Consider how this code interacts with other parts of the system, especially interfaces, shared types, or utility functions
3. **Understand the broader architecture** - Identify the role this code plays in the overall system and any architectural patterns or constraints
4. **Research external context** - When you encounter unfamiliar APIs, frameworks, or domain-specific logic, reason about what additional context would help explain the code's purpose and constraints
5. **Identify complexity indicators** - Pay special attention to:
   - Complex algorithms or data structures that aren't immediately obvious
   - Error handling that suggests specific edge cases or failure modes
   - Performance optimizations that trade readability for efficiency
   - Security considerations or access control logic
   - Business logic that requires domain knowledge to understand

## Core Principles
1. **Focus on "Why", not "What"**: Comments should explain the RATIONALE and INTENT behind the code, not describe what it does. If a comment starts with "This code..." or "This function...", it's likely describing "what" instead of "why". Good comments answer questions like:
   - Why was this approach chosen over simpler alternatives?
   - Why must this edge case be handled this way?
   - Why is this performance optimization structured like this?
   - Why does this business rule require this specific logic?
   - Why is this seemingly unusual code actually the correct approach?

2. **Embrace Strategic Silence**: Most code is self-explanatory through good naming and should have NO comments. Only comment when there's a genuine gap between what the code appears to do and why it must do it that way. Ask: "Would a reasonably experienced developer understand both WHAT this code does AND WHY it must work this way without comments?"

3. **Target Only True Complexity**: Add comments ONLY for:
	- Language-specific subtleties (e.g., "Closure captures loop variable by reference, not value")
	- Business logic nuances (e.g., "Must check both user role AND subscription status for access")
	- Performance-critical sections with non-obvious optimizations (e.g., "O(n) vs O(n²) choice due to data size constraints")
	- Complex algorithms that aren't immediately clear (e.g., "Using A* search because Dijkstra's would be O(n²) for this graph density")
	- APIs that require careful usage to avoid errors (e.g., "Must call Close() to prevent resource leaks in this framework")
	- Code that appears unusual but is intentional (e.g., "Polling used instead of events because USB driver doesn't support async notifications")
	- External dependencies with specific constraints (e.g., "PostgreSQL JSONB used for schemaless data despite MongoDB alternative")
4. **Preserve Code Clarity**: If the code can be made clearer
through better naming rather than comments, note this but DO NOT
rename anything - only add comments to the existing code as-is.
5. **Improve Code Formatting**: Add appropriate newlines to improve
readability and logical grouping. Follow language-specific conventions:
	- Add blank lines between logical sections
	- Separate related but distinct operations with blank lines
	- Group related statements together without blank lines
	- Follow standard formatting conventions for the language

## What to Comment
- **Why** a particular approach was chosen over alternatives
- **Why** certain edge cases are handled in specific ways
- **Why** performance optimizations are structured as they are
- **Why** business rules require specific logic flow
- **Why** specific external APIs or frameworks are used and their constraints
- **Why** complex data structures or algorithms are implemented this way
- Assumptions that must hold true for the code to work correctly
- Side effects that aren't immediately obvious
- Relationships between distant parts of the code (e.g., callbacks
defined far from their usage)
- External context or domain knowledge required to understand the code

## What NOT to Comment
- **Any comment that starts with "This..."**: "This function...", "This code...", "This variable..." - these describe WHAT, not WHY
- **Obvious operations**: "Increment counter", "Return true if valid", "Loop through array"
- **Simple getters/setters**: "Get user name", "Set user age"
- **Standard patterns**: "Check if nil", "Handle error", "Validate input"
- **Trivial functions**: Functions with 1-3 lines that do exactly what their name suggests
- **Redundant explanations**: Comments that just rephrase what clear variable/function names already express
- **Well-known APIs**: Standard usage of fmt.Println, json.Unmarshal, http.Get, etc.
- **Self-documenting code**: Any code where good naming makes the purpose and logic crystal clear

## Output Format
Write to the same file with comments added in the
appropriate language-specific comment syntax AND improved formatting
with appropriate newlines. Preserve all existing code exactly as-is -
only add comments and improve whitespace/newline placement for better
readability.

Remember: **Strategic silence is golden.** Most code needs no comments when well-named. Comments should make future maintainers' lives easier by explaining the non-obvious, not burden them with noise. Only comment when there's a genuine gap between what the code appears to do and why it must work that specific way. When you encounter complex code that would benefit from external context, explain what additional context would be helpful for future maintainers.
`, "Prompt to send to Claude")

	flag.Parse()

	if *prompt == "" {
		fmt.Fprintln(os.Stderr, "Error: -prompt flag is required")
		flag.Usage()
		os.Exit(1)
	}

	var files []string
	var err error

	if *staged {
		// Get staged files from git when -staged flag is set
		files, err = getStagedFiles()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Found %d staged file(s)\n", len(files))
	} else {
		// Use command-line arguments when -staged flag is not set
		files = flag.Args()
		if len(files) == 0 {
			fmt.Fprintln(os.Stderr, "Error: No files provided. Use -staged flag or provide file paths as arguments")
			flag.Usage()
			os.Exit(1)
		}
	}

	// Convert all input paths to absolute paths upfront to ensure consistent
	// cache key generation and avoid ambiguity between relative path interpretations
	absoluteFiles := make([]string, 0, len(files))
	for _, file := range files {
		absPath, err := filepath.Abs(file)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: failed to resolve absolute path for %s: %v\n", file, err)
			os.Exit(1)
		}
		absoluteFiles = append(absoluteFiles, absPath)
	}

	config := Config{
		Files:        absoluteFiles,
		BatchSize:    *batchSize,
		Prompt:       *prompt,
		ForceProcess: *forceProcess,
		CacheOnly:    *cacheOnly,
	}

	if err := run(config); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func run(config Config) error {
	cache, err := loadCache()
	if err != nil {
		return fmt.Errorf("failed to load cache: %w", err)
	}

	// Cache-only mode allows initializing the cache without expensive processing,
	// useful for marking existing commented code as "already processed"
	if config.CacheOnly {
		fmt.Println("Cache-only mode: marking files as cached without processing")
		cachedCount := 0

		for _, file := range config.Files {
			// Skip gitignored files even in cache-only mode
			if isGitIgnored(file) {
				fmt.Printf("Skipping (gitignored): %s\n", file)
				continue
			}

			if err := cache.markProcessed(file); err != nil {
				fmt.Fprintf(os.Stderr, "Warning: failed to mark %s as cached: %v\n", file, err)
				continue
			}
			fmt.Printf("Cached: %s\n", file)
			cachedCount++
		}

		if cachedCount == 0 {
			return fmt.Errorf("no files were successfully cached")
		}

		if err := cache.save(); err != nil {
			return fmt.Errorf("failed to save cache: %w", err)
		}

		fmt.Printf("\nMarked %d files as cached\n", cachedCount)
		return nil
	}

	// Filter files before expensive Claude processing to avoid unnecessary API calls
	processedFiles := make([]string, 0, len(config.Files))
	skippedFiles := 0

	for _, file := range config.Files {
		// Skip gitignored files
		if isGitIgnored(file) {
			fmt.Printf("Skipping (gitignored): %s\n", file)
			skippedFiles++
			continue
		}

		shouldProcess := config.ForceProcess
		if !shouldProcess {
			var err error
			shouldProcess, err = cache.shouldProcess(file)
			if err != nil {
				// On cache check failure, err on the side of processing to ensure correctness
				fmt.Fprintf(os.Stderr, "Warning: failed to check cache for %s: %v\n", file, err)
				shouldProcess = true
			}
		}

		if !shouldProcess {
			fmt.Printf("Skipping (unchanged): %s\n", file)
			skippedFiles++
			continue
		}

		// Comment removal happens before Claude processing to provide clean input,
		// allowing Claude to focus on adding meaningful comments without existing noise
		if err := processFile(file); err != nil {
			// Check if this is an unsupported file type error
			var unsupportedErr *ErrUnsupportedFileType
			if errors.As(err, &unsupportedErr) {
				fmt.Printf("Skipping (unsupported): %s\n", file)
				skippedFiles++
				continue
			}
			// Other errors are warnings
			fmt.Fprintf(os.Stderr, "Warning: failed to process %s: %v\n", file, err)
			continue
		}

		processedFiles = append(processedFiles, file)
		fmt.Printf("Removed comments from: %s\n", file)
	}

	if len(processedFiles) == 0 {
		if skippedFiles > 0 {
			fmt.Printf("\nAll %d files are up to date (no changes needed)\n", skippedFiles)
			return nil
		}
		return fmt.Errorf("no files were successfully processed")
	}

	fmt.Printf("\nProcessing %d files in batches of %d...\n\n", len(processedFiles), config.BatchSize)

	if err := processBatches(processedFiles, config.BatchSize, config.Prompt, cache); err != nil {
		return err
	}

	return nil
}

func processFile(inputPath string) error {
	content, err := os.ReadFile(inputPath)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	ext := filepath.Ext(inputPath)
	var cleaned string

	switch ext {
	case ".js", ".ts", ".jsx", ".tsx":
		cleaned = removeJSComments(string(content))
	case ".go":
		cleaned = removeGoComments(string(content))
	case ".py":
		cleaned = removePythonComments(string(content))
	case ".rs":
		cleaned = removeRustComments(string(content))
	case ".tf", ".tfvars":
		cleaned = removeTerraformComments(string(content))
	case ".yaml", ".yml":
		cleaned = removeYAMLComments(string(content))
	default:
		// Return special error type to indicate unsupported file should be skipped
		return &ErrUnsupportedFileType{Extension: ext}
	}

	if err := os.WriteFile(inputPath, []byte(cleaned), 0o644); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}

func processBatches(files []string, batchSize int, prompt string, cache *FileCache) error {
	for i := 0; i < len(files); i += batchSize {
		end := min(i+batchSize, len(files))
		batch := files[i:end]

		fmt.Printf("Processing batch %d/%d (%d files)...\n", (i/batchSize)+1, (len(files)+batchSize-1)/batchSize, len(batch))

		if err := processBatch(batch, prompt); err != nil {
			return fmt.Errorf("batch processing failed: %w", err)
		}

		// Cache updates happen after each successful batch to prevent data loss
		// if processing is interrupted partway through
		for _, file := range batch {
			if err := cache.markProcessed(file); err != nil {
				fmt.Fprintf(os.Stderr, "Warning: failed to update cache for %s: %v\n", file, err)
			}
		}

		// Cache save failures are warnings rather than errors because processing succeeded;
		// worst case is redundant work on next run
		if err := cache.save(); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to save cache: %v\n", err)
		}
	}

	return nil
}

// processBatch runs Claude in parallel for all files in a batch but waits for completion
// before returning. This controlled parallelism respects rate limits while maximizing
// throughput, unlike unbounded parallelism which could overwhelm the Claude API.
func processBatch(files []string, prompt string) error {
	var wg sync.WaitGroup
	errChan := make(chan error, len(files))

	for _, file := range files {
		wg.Add(1)
		// File parameter is passed to goroutine to avoid closure capture issues
		// where all goroutines would reference the final loop value
		go func(f string) {
			defer wg.Done()
			if err := runClaude(f, prompt); err != nil {
				errChan <- fmt.Errorf("%s: %w", f, err)
			}
		}(file)
	}

	wg.Wait()
	close(errChan)

	// Collect all errors rather than failing fast to provide complete feedback
	// on which files failed in the batch
	var errors []string
	for err := range errChan {
		errors = append(errors, err.Error())
	}

	if len(errors) > 0 {
		return fmt.Errorf("errors occurred:\n  %s", strings.Join(errors, "\n  "))
	}

	return nil
}

// runClaude formats before processing to ensure consistent code style,
// preventing Claude from being distracted by formatting issues
func runClaude(file, prompt string) error {
	fmt.Printf("  [%s] Running Claude...\n", filepath.Base(file))

	if err := formatFile(file); err != nil {
		// Formatter failures are warnings because formatting is a quality-of-life feature,
		// not critical to comment generation
		fmt.Fprintf(os.Stderr, "  [%s] Warning: formatter failed: %v\n", filepath.Base(file), err)
	} else {
		fmt.Printf("  [%s] Formatted\n", filepath.Base(file))
	}

	// bypassPermissions mode is required because Claude needs write access to modify files,
	// and interactive permission prompts would block batch processing
	cmd := exec.Command("claude", "--dangerously-skip-permissions", "--model", "haiku", "--permission-mode", "bypassPermissions", "-p", strings.Replace(prompt, "{filename}", file, 1))
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("claude command failed: %w", err)
	}

	if err := formatFile(file); err != nil {
		// Formatter failures are warnings because formatting is a quality-of-life feature,
		// not critical to comment generation
		fmt.Fprintf(os.Stderr, "  [%s] Warning: formatter failed: %v\n", filepath.Base(file), err)
	} else {
		fmt.Printf("  [%s] Formatted\n", filepath.Base(file))
	}

	fmt.Printf("  [%s] Completed\n", filepath.Base(file))
	return nil
}

func formatFile(file string) error {
	ext := filepath.Ext(file)
	var cmd *exec.Cmd

	switch ext {
	case ".go":
		cmd = exec.Command("go", "fmt", file)
	case ".js", ".ts", ".jsx", ".tsx":
		cmd = exec.Command("biome", "format", "--write", file)
	case ".py":
		cmd = exec.Command("ruff", "format", file)
	case ".rs":
		cmd = exec.Command("rustfmt", file)
	case ".tf", ".tfvars":
		cmd = exec.Command("terraform", "fmt", file)
	case ".yaml", ".yml":
		cmd = exec.Command("yamlfmt", file)
	default:
		// No formatter configured for this file type; skip silently
		return nil
	}

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("formatter command failed: %w (output: %s)", err, string(output))
	}

	return nil
}
