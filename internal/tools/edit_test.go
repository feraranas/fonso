package tools

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Test 1: EditFile should successfully replace text in a file
func TestEditFile_SuccessfulEdit(t *testing.T) {
	// Create a temporary file with test content
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	originalContent := "Hello, World!\nThis is a test."

	err := os.WriteFile(testFile, []byte(originalContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Create JSON input to replace "World" with "Universe"
	input := EditFileInput{
		Path:   testFile,
		OldStr: "World",
		NewStr: "Universe",
	}
	inputJSON, err := json.Marshal(input)
	if err != nil {
		t.Fatalf("Failed to marshal input: %v", err)
	}

	// Call the EditFile function
	result, err := EditFile(inputJSON)

	// Verify no error occurred
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	// Verify result is "OK"
	if result != "OK" {
		t.Errorf("Expected result 'OK', got %q", result)
	}

	// Read the file and verify the change
	newContent, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatalf("Failed to read modified file: %v", err)
	}

	expectedContent := "Hello, Universe!\nThis is a test."
	if string(newContent) != expectedContent {
		t.Errorf("Expected content %q, got %q", expectedContent, string(newContent))
	}
}

// Test 2: EditFile should create a new file when old_str is empty
func TestEditFile_CreateNewFile(t *testing.T) {
	// Create a temporary directory
	tmpDir := t.TempDir()
	newFile := filepath.Join(tmpDir, "newfile.txt")

	// Create JSON input with empty old_str to create new file
	input := EditFileInput{
		Path:   newFile,
		OldStr: "",
		NewStr: "This is a new file!",
	}
	inputJSON, err := json.Marshal(input)
	if err != nil {
		t.Fatalf("Failed to marshal input: %v", err)
	}

	// Call the EditFile function
	result, err := EditFile(inputJSON)

	// Verify no error occurred
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	// Verify result indicates file creation
	if result != "Successfully created file "+newFile {
		t.Errorf("Expected creation message, got %q", result)
	}

	// Read the file and verify the content
	content, err := os.ReadFile(newFile)
	if err != nil {
		t.Fatalf("Failed to read new file: %v", err)
	}

	expectedContent := "This is a new file!"
	if string(content) != expectedContent {
		t.Errorf("Expected content %q, got %q", expectedContent, string(content))
	}
}

// Test 3: EditFile should return error when old_str matches multiple times
func TestEditFile_MultipleMatches(t *testing.T) {
	// Create a temporary file with duplicate text
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	originalContent := "test\ntest\ntest"

	err := os.WriteFile(testFile, []byte(originalContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Create JSON input to replace "test" (which appears 3 times)
	input := EditFileInput{
		Path:   testFile,
		OldStr: "test",
		NewStr: "replaced",
	}
	inputJSON, err := json.Marshal(input)
	if err != nil {
		t.Fatalf("Failed to marshal input: %v", err)
	}

	// Call the EditFile function
	_, err = EditFile(inputJSON)

	// Verify an error occurred
	if err == nil {
		t.Errorf("Expected error for multiple matches, got nil")
	}

	// Verify error message mentions multiple matches
	if err != nil && !strings.Contains(err.Error(), "found 3 times") {
		t.Errorf("Expected error about multiple matches, got: %v", err)
	}
}

// Test 4: EditFile should create nested directories when creating a new file
func TestEditFile_CreateNestedDirectories(t *testing.T) {
	// Create a temporary directory
	tmpDir := t.TempDir()
	nestedFile := filepath.Join(tmpDir, "a", "b", "c", "newfile.txt")

	// Create JSON input with empty old_str to create new file in nested dirs
	input := EditFileInput{
		Path:   nestedFile,
		OldStr: "",
		NewStr: "Nested file content",
	}
	inputJSON, err := json.Marshal(input)
	if err != nil {
		t.Fatalf("Failed to marshal input: %v", err)
	}

	// Call the EditFile function
	result, err := EditFile(inputJSON)

	// Verify no error occurred
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	// Verify result indicates file creation
	if !strings.Contains(result, "Successfully created file") {
		t.Errorf("Expected creation message, got %q", result)
	}

	// Read the file and verify the content
	content, err := os.ReadFile(nestedFile)
	if err != nil {
		t.Fatalf("Failed to read nested file: %v", err)
	}

	expectedContent := "Nested file content"
	if string(content) != expectedContent {
		t.Errorf("Expected content %q, got %q", expectedContent, string(content))
	}

	// Verify directories were created
	if _, err := os.Stat(filepath.Join(tmpDir, "a", "b", "c")); os.IsNotExist(err) {
		t.Errorf("Expected nested directories to be created")
	}
}
