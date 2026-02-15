package tools

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// Test 1: ReadFile should successfully read an existing file
func TestReadFile_Success(t *testing.T) {
	// Create a temporary file with test content
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	testContent := "Hello, World!"

	err := os.WriteFile(testFile, []byte(testContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Create JSON input for the tool
	input := ReadFileInput{Path: testFile}
	inputJSON, err := json.Marshal(input)
	if err != nil {
		t.Fatalf("Failed to marshal input: %v", err)
	}

	// Call the ReadFile function
	result, err := ReadFile(inputJSON)

	// Verify no error occurred
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	// Verify the content matches
	if result != testContent {
		t.Errorf("Expected content %q, got %q", testContent, result)
	}
}

// Test 2: ReadFile should return an error for non-existent file
func TestReadFile_NonExistentFile(t *testing.T) {
	// Use a path that doesn't exist
	nonExistentFile := "/tmp/this-file-does-not-exist-12345.txt"

	// Create JSON input for the tool
	input := ReadFileInput{Path: nonExistentFile}
	inputJSON, err := json.Marshal(input)
	if err != nil {
		t.Fatalf("Failed to marshal input: %v", err)
	}

	// Call the ReadFile function
	result, err := ReadFile(inputJSON)

	// Verify an error occurred
	if err == nil {
		t.Errorf("Expected an error for non-existent file, got nil")
	}

	// Verify result is empty on error
	if result != "" {
		t.Errorf("Expected empty result on error, got %q", result)
	}
}

// Test 3: ReadFile should handle invalid JSON input
func TestReadFile_InvalidJSON(t *testing.T) {
	// Create invalid JSON input
	invalidJSON := json.RawMessage(`{"invalid": json}`)

	// Call the ReadFile function - should panic due to unmarshal error
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("Expected panic for invalid JSON, but didn't panic")
		}
	}()

	ReadFile(invalidJSON)
}
