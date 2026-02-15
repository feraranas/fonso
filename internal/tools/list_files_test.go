package tools

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Test 1: ListFiles should successfully list files in a directory
func TestListFiles_Success(t *testing.T) {
	// Create a temporary directory with test files
	tmpDir := t.TempDir()

	// Create some test files
	os.WriteFile(filepath.Join(tmpDir, "file1.txt"), []byte("test"), 0644)
	os.WriteFile(filepath.Join(tmpDir, "file2.go"), []byte("test"), 0644)
	os.MkdirAll(filepath.Join(tmpDir, "subdir"), 0755)
	os.WriteFile(filepath.Join(tmpDir, "subdir", "file3.md"), []byte("test"), 0644)

	// Create JSON input for the tool
	input := ListFilesInput{Path: tmpDir}
	inputJSON, err := json.Marshal(input)
	if err != nil {
		t.Fatalf("Failed to marshal input: %v", err)
	}

	// Call the ListFiles function
	result, err := ListFiles(inputJSON)

	// Verify no error occurred
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	// Parse the JSON result
	var files []string
	err = json.Unmarshal([]byte(result), &files)
	if err != nil {
		t.Fatalf("Failed to unmarshal result: %v", err)
	}

	// Verify we got the expected files
	expectedFiles := []string{"file1.txt", "file2.go", "subdir/", "subdir/file3.md"}
	if len(files) != len(expectedFiles) {
		t.Errorf("Expected %d files, got %d: %v", len(expectedFiles), len(files), files)
	}

	// Check each expected file is in the result
	for _, expected := range expectedFiles {
		found := false
		for _, file := range files {
			if file == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected file %q not found in result", expected)
		}
	}

	// Verify directories have trailing slash
	for _, file := range files {
		if strings.HasSuffix(file, "/") {
			// This should be a directory
			fullPath := filepath.Join(tmpDir, strings.TrimSuffix(file, "/"))
			info, err := os.Stat(fullPath)
			if err != nil {
				t.Errorf("Directory marker %q points to non-existent path", file)
			} else if !info.IsDir() {
				t.Errorf("File %q has trailing slash but is not a directory", file)
			}
		}
	}
}

// Test 2: ListFiles should default to current directory when path is empty
func TestListFiles_DefaultCurrentDirectory(t *testing.T) {
	// Save current directory
	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current directory: %v", err)
	}

	// Create and change to a temporary directory
	tmpDir := t.TempDir()
	err = os.Chdir(tmpDir)
	if err != nil {
		t.Fatalf("Failed to change directory: %v", err)
	}
	defer os.Chdir(originalDir) // Restore on cleanup

	// Create a test file in current directory
	os.WriteFile("test.txt", []byte("test"), 0644)

	// Create JSON input with empty path
	input := ListFilesInput{Path: ""}
	inputJSON, err := json.Marshal(input)
	if err != nil {
		t.Fatalf("Failed to marshal input: %v", err)
	}

	// Call the ListFiles function
	result, err := ListFiles(inputJSON)

	// Verify no error occurred
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	// Parse the JSON result
	var files []string
	err = json.Unmarshal([]byte(result), &files)
	if err != nil {
		t.Fatalf("Failed to unmarshal result: %v", err)
	}

	// Verify we got the test file
	found := false
	for _, file := range files {
		if file == "test.txt" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("Expected test.txt in current directory listing, got: %v", files)
	}
}

// Test 3: ListFiles should skip .devenv directory
func TestListFiles_SkipsDevenvDirectory(t *testing.T) {
	// Create a temporary directory with .devenv folder
	tmpDir := t.TempDir()

	// Create .devenv directory with files
	devenvDir := filepath.Join(tmpDir, ".devenv")
	os.MkdirAll(devenvDir, 0755)
	os.WriteFile(filepath.Join(devenvDir, "should-be-skipped.txt"), []byte("test"), 0644)

	// Create a normal file
	os.WriteFile(filepath.Join(tmpDir, "normal.txt"), []byte("test"), 0644)

	// Create JSON input
	input := ListFilesInput{Path: tmpDir}
	inputJSON, err := json.Marshal(input)
	if err != nil {
		t.Fatalf("Failed to marshal input: %v", err)
	}

	// Call the ListFiles function
	result, err := ListFiles(inputJSON)

	// Verify no error occurred
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	// Parse the JSON result
	var files []string
	err = json.Unmarshal([]byte(result), &files)
	if err != nil {
		t.Fatalf("Failed to unmarshal result: %v", err)
	}

	// Verify .devenv is not in the results
	for _, file := range files {
		if strings.Contains(file, ".devenv") {
			t.Errorf("Expected .devenv directory to be skipped, but found: %q", file)
		}
	}

	// Verify normal.txt IS in the results
	found := false
	for _, file := range files {
		if file == "normal.txt" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("Expected normal.txt in listing, got: %v", files)
	}
}
