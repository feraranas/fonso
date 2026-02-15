package tools

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Test 1: CodeSearch should find pattern in files
func TestCodeSearch_FindPattern(t *testing.T) {
	// Create a temporary directory with test files
	tmpDir := t.TempDir()

	// Create test files with searchable content
	os.WriteFile(filepath.Join(tmpDir, "test1.txt"), []byte("Hello World\nTest pattern"), 0644)
	os.WriteFile(filepath.Join(tmpDir, "test2.txt"), []byte("Another file\nwith pattern"), 0644)

	// Create JSON input to search for "pattern"
	input := CodeSearchInput{
		Pattern: "pattern",
		Path:    tmpDir,
	}
	inputJSON, err := json.Marshal(input)
	if err != nil {
		t.Fatalf("Failed to marshal input: %v", err)
	}

	// Call the CodeSearch function
	result, err := CodeSearch(inputJSON)

	// Verify no error occurred
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	// Verify results contain matches from both files
	if !strings.Contains(result, "test1.txt") {
		t.Errorf("Expected test1.txt in results, got: %q", result)
	}
	if !strings.Contains(result, "test2.txt") {
		t.Errorf("Expected test2.txt in results, got: %q", result)
	}
	if !strings.Contains(result, "pattern") {
		t.Errorf("Expected 'pattern' in results, got: %q", result)
	}
}

// Test 2: CodeSearch should handle no matches
func TestCodeSearch_NoMatches(t *testing.T) {
	// Create a temporary directory with test files
	tmpDir := t.TempDir()
	os.WriteFile(filepath.Join(tmpDir, "test.txt"), []byte("Hello World"), 0644)

	// Create JSON input to search for non-existent pattern
	input := CodeSearchInput{
		Pattern: "nonexistent",
		Path:    tmpDir,
	}
	inputJSON, err := json.Marshal(input)
	if err != nil {
		t.Fatalf("Failed to marshal input: %v", err)
	}

	// Call the CodeSearch function
	result, err := CodeSearch(inputJSON)

	// Verify no error occurred
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	// Verify result indicates no matches
	if result != "No matches found" {
		t.Errorf("Expected 'No matches found', got: %q", result)
	}
}

// Test 3: CodeSearch should filter by file type
func TestCodeSearch_FileTypeFilter(t *testing.T) {
	// Create a temporary directory with different file types
	tmpDir := t.TempDir()
	os.WriteFile(filepath.Join(tmpDir, "test.go"), []byte("func main() {}"), 0644)
	os.WriteFile(filepath.Join(tmpDir, "test.txt"), []byte("func main() {}"), 0644)
	os.WriteFile(filepath.Join(tmpDir, "test.md"), []byte("func main() {}"), 0644)

	// Create JSON input to search only in .go files
	input := CodeSearchInput{
		Pattern:  "func",
		Path:     tmpDir,
		FileType: "go",
	}
	inputJSON, err := json.Marshal(input)
	if err != nil {
		t.Fatalf("Failed to marshal input: %v", err)
	}

	// Call the CodeSearch function
	result, err := CodeSearch(inputJSON)

	// Verify no error occurred
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	// Verify only .go file is in results
	if !strings.Contains(result, "test.go") {
		t.Errorf("Expected test.go in results, got: %q", result)
	}
	if strings.Contains(result, "test.txt") {
		t.Errorf("Expected test.txt NOT in results, got: %q", result)
	}
	if strings.Contains(result, "test.md") {
		t.Errorf("Expected test.md NOT in results, got: %q", result)
	}
}

// Test 4: CodeSearch should respect case sensitivity
func TestCodeSearch_CaseSensitivity(t *testing.T) {
	// Create a temporary directory with test content
	tmpDir := t.TempDir()
	os.WriteFile(filepath.Join(tmpDir, "test.txt"), []byte("UPPER lower MiXeD"), 0644)

	// Test case-insensitive search (default)
	inputInsensitive := CodeSearchInput{
		Pattern:       "upper",
		Path:          tmpDir,
		CaseSensitive: false,
	}
	inputJSON, err := json.Marshal(inputInsensitive)
	if err != nil {
		t.Fatalf("Failed to marshal input: %v", err)
	}

	result, err := CodeSearch(inputJSON)
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	// Should find "UPPER" when case-insensitive
	if !strings.Contains(result, "UPPER") {
		t.Errorf("Expected to find 'UPPER' with case-insensitive search, got: %q", result)
	}

	// Test case-sensitive search
	inputSensitive := CodeSearchInput{
		Pattern:       "upper",
		Path:          tmpDir,
		CaseSensitive: true,
	}
	inputJSON, err = json.Marshal(inputSensitive)
	if err != nil {
		t.Fatalf("Failed to marshal input: %v", err)
	}

	result, err = CodeSearch(inputJSON)
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	// Should NOT find anything when case-sensitive (file has "UPPER" not "upper")
	if result != "No matches found" {
		t.Errorf("Expected no matches with case-sensitive search for 'upper', got: %q", result)
	}
}

// Test 5: CodeSearch should limit results to 50 matches
func TestCodeSearch_ResultLimit(t *testing.T) {
	// Create a temporary directory
	tmpDir := t.TempDir()

	// Create a file with many matching lines (more than 50)
	var manyLines strings.Builder
	for i := 0; i < 60; i++ {
		manyLines.WriteString("match\n")
	}
	os.WriteFile(filepath.Join(tmpDir, "test.txt"), []byte(manyLines.String()), 0644)

	// Create JSON input to search for "match"
	input := CodeSearchInput{
		Pattern: "match",
		Path:    tmpDir,
	}
	inputJSON, err := json.Marshal(input)
	if err != nil {
		t.Fatalf("Failed to marshal input: %v", err)
	}

	// Call the CodeSearch function
	result, err := CodeSearch(inputJSON)

	// Verify no error occurred
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	// Verify result contains truncation message
	if !strings.Contains(result, "showing first 50") {
		t.Errorf("Expected truncation message for 50+ matches, got: %q", result)
	}
}
