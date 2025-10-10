package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestCollapseExcessiveNewlines(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "no changes needed - single newline",
			input:    "line1\nline2\nline3",
			expected: "line1\nline2\nline3",
		},
		{
			name:     "collapse triple newlines to single",
			input:    "line1\n\n\nline2",
			expected: "line1\nline2",
		},
		{
			name:     "collapse quadruple newlines to single",
			input:    "line1\n\n\n\nline2",
			expected: "line1\nline2",
		},
		{
			name:     "multiple sequences to collapse",
			input:    "line1\n\n\nline2\n\n\n\nline3",
			expected: "line1\nline2\nline3",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "only newlines",
			input:    "\n\n\n\n",
			expected: "\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := collapseExcessiveNewlines(tt.input)
			if result != tt.expected {
				t.Errorf("collapseExcessiveNewlines() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestFindGitRoot(t *testing.T) {
	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get current directory: %v", err)
	}
	defer os.Chdir(originalDir)

	gitRoot, err := findGitRoot()
	if err != nil {
		t.Skipf("not in a git repository, skipping test: %v", err)
	}

	gitDir := filepath.Join(gitRoot, ".git")
	if info, err := os.Stat(gitDir); err != nil || !info.IsDir() {
		t.Errorf("findGitRoot() returned %q, but .git directory not found", gitRoot)
	}

	// Test that findGitRoot works from a subdirectory
	if err := os.Chdir(gitRoot); err == nil {
		tempDir := filepath.Join(gitRoot, "temp_test_dir")
		if err := os.Mkdir(tempDir, 0755); err == nil {
			defer os.RemoveAll(tempDir)

			if err := os.Chdir(tempDir); err == nil {
				rootFromSubdir, err := findGitRoot()
				if err != nil {
					t.Errorf("findGitRoot() from subdirectory failed: %v", err)
				}
				if rootFromSubdir != gitRoot {
					t.Errorf("findGitRoot() from subdirectory = %q, want %q", rootFromSubdir, gitRoot)
				}
			}
		}
	}
}

func TestPathConversion(t *testing.T) {
	gitRoot, err := findGitRoot()
	if err != nil {
		t.Skipf("not in a git repository, skipping test: %v", err)
	}

	tests := []struct {
		name         string
		absolutePath string
		wantRelative string
	}{
		{
			name:         "file in root",
			absolutePath: filepath.Join(gitRoot, "main.go"),
			wantRelative: "main.go",
		},
		{
			name:         "file in subdirectory",
			absolutePath: filepath.Join(gitRoot, "src", "utils.go"),
			wantRelative: filepath.Join("src", "utils.go"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			relPath, err := toRelativePath(tt.absolutePath)
			if err != nil {
				t.Errorf("toRelativePath() error = %v", err)
				return
			}
			if relPath != tt.wantRelative {
				t.Errorf("toRelativePath() = %q, want %q", relPath, tt.wantRelative)
			}

			// Verify round-trip conversion works correctly
			absPath, err := toAbsolutePath(relPath)
			if err != nil {
				t.Errorf("toAbsolutePath() error = %v", err)
				return
			}
			if absPath != tt.absolutePath {
				t.Errorf("toAbsolutePath() = %q, want %q", absPath, tt.absolutePath)
			}
		})
	}
}

func TestFileCacheSaveLoad(t *testing.T) {
	gitRoot, err := findGitRoot()
	if err != nil {
		t.Skipf("not in a git repository, skipping test: %v", err)
	}

	tempCache := filepath.Join(gitRoot, ".nocomms-test-cache.json")
	defer os.Remove(tempCache)

	// Truncate to second precision because JSON serialization loses subsecond precision
	cache := &FileCache{
		ProcessedFiles: map[string]time.Time{
			"main.go":      time.Now().Add(-1 * time.Hour).Truncate(time.Second),
			"src/utils.go": time.Now().Add(-30 * time.Minute).Truncate(time.Second),
		},
	}

	data, err := json.MarshalIndent(cache, "", "  ")
	if err != nil {
		t.Fatalf("json.MarshalIndent() error = %v", err)
	}
	if err := os.WriteFile(tempCache, data, 0644); err != nil {
		t.Fatalf("os.WriteFile() error = %v", err)
	}

	loadedData, err := os.ReadFile(tempCache)
	if err != nil {
		t.Fatalf("os.ReadFile() error = %v", err)
	}

	loadedCache := &FileCache{
		ProcessedFiles: make(map[string]time.Time),
	}
	if err := json.Unmarshal(loadedData, loadedCache); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}

	if len(loadedCache.ProcessedFiles) != len(cache.ProcessedFiles) {
		t.Errorf("loaded cache has %d files, want %d", len(loadedCache.ProcessedFiles), len(cache.ProcessedFiles))
	}

	for path := range cache.ProcessedFiles {
		if _, exists := loadedCache.ProcessedFiles[path]; !exists {
			t.Errorf("loaded cache missing file: %s", path)
		}
	}
}

func TestFileCacheMarkProcessed(t *testing.T) {
	gitRoot, err := findGitRoot()
	if err != nil {
		t.Skipf("not in a git repository, skipping test: %v", err)
	}

	cache := &FileCache{
		ProcessedFiles: make(map[string]time.Time),
	}

	testFile := filepath.Join(gitRoot, "main.go")
	if _, err := os.Stat(testFile); err != nil {
		t.Skipf("main.go not found, skipping test")
	}

	if err := cache.markProcessed(testFile); err != nil {
		t.Fatalf("markProcessed() error = %v", err)
	}

	// Verify that markProcessed stores relative paths, not absolute
	if _, exists := cache.ProcessedFiles["main.go"]; !exists {
		t.Errorf("markProcessed() did not store file with relative path 'main.go'")
		t.Logf("cache contents: %+v", cache.ProcessedFiles)
	}
}

func TestFileCacheShouldProcess(t *testing.T) {
	gitRoot, err := findGitRoot()
	if err != nil {
		t.Skipf("not in a git repository, skipping test: %v", err)
	}

	testFile := filepath.Join(gitRoot, "main.go")
	if _, err := os.Stat(testFile); err != nil {
		t.Skipf("main.go not found, skipping test")
	}

	tests := []struct {
		name           string
		setupCache     func() *FileCache
		expectedResult bool
	}{
		{
			name: "file not in cache - should process",
			setupCache: func() *FileCache {
				return &FileCache{
					ProcessedFiles: make(map[string]time.Time),
				}
			},
			expectedResult: true,
		},
		{
			name: "file in cache with old timestamp - should process",
			setupCache: func() *FileCache {
				return &FileCache{
					ProcessedFiles: map[string]time.Time{
						"main.go": time.Now().Add(-24 * time.Hour),
					},
				}
			},
			expectedResult: true,
		},
		{
			name: "file in cache with future timestamp - should not process",
			setupCache: func() *FileCache {
				// Future timestamp indicates file hasn't been modified since last processing
				return &FileCache{
					ProcessedFiles: map[string]time.Time{
						"main.go": time.Now().Add(24 * time.Hour),
					},
				}
			},
			expectedResult: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cache := tt.setupCache()
			result, err := cache.shouldProcess(testFile)
			if err != nil {
				t.Errorf("shouldProcess() error = %v", err)
				return
			}
			if result != tt.expectedResult {
				t.Errorf("shouldProcess() = %v, want %v", result, tt.expectedResult)
			}
		})
	}
}

func TestCacheJSONFormat(t *testing.T) {
	cache := &FileCache{
		ProcessedFiles: map[string]time.Time{
			"main.go":      time.Date(2025, 10, 10, 10, 30, 0, 0, time.UTC),
			"src/utils.go": time.Date(2025, 10, 10, 10, 31, 0, 0, time.UTC),
		},
	}

	data, err := json.MarshalIndent(cache, "", "  ")
	if err != nil {
		t.Fatalf("json.MarshalIndent() error = %v", err)
	}

	var loadedCache FileCache
	if err := json.Unmarshal(data, &loadedCache); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}

	// Verify cache only contains relative paths to ensure portability across machines
	for path := range loadedCache.ProcessedFiles {
		if filepath.IsAbs(path) {
			t.Errorf("cache contains absolute path: %s (should be relative)", path)
		}
	}
}
