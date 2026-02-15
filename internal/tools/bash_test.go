package tools

import (
	"encoding/json"
	"strings"
	"testing"
)

// Test 1: Bash should successfully execute a simple command
func TestBash_SuccessfulCommand(t *testing.T) {
	// Create JSON input for echo command
	input := BashInput{Command: "echo 'Hello, World!'"}
	inputJSON, err := json.Marshal(input)
	if err != nil {
		t.Fatalf("Failed to marshal input: %v", err)
	}

	// Call the Bash function
	result, err := Bash(inputJSON)

	// Verify no error occurred
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	// Verify the output
	expectedOutput := "Hello, World!"
	if result != expectedOutput {
		t.Errorf("Expected output %q, got %q", expectedOutput, result)
	}
}

// Test 2: Bash should handle failing commands gracefully
func TestBash_FailingCommand(t *testing.T) {
	// Create JSON input for a command that will fail
	input := BashInput{Command: "exit 1"}
	inputJSON, err := json.Marshal(input)
	if err != nil {
		t.Fatalf("Failed to marshal input: %v", err)
	}

	// Call the Bash function
	result, err := Bash(inputJSON)

	// Bash returns nil error even for failed commands (returns error message in result)
	if err != nil {
		t.Errorf("Expected no error (errors are in result), got: %v", err)
	}

	// Verify result contains error information
	if !strings.Contains(result, "Command failed") {
		t.Errorf("Expected error message in result, got: %q", result)
	}
}

// Test 3: Bash should handle commands with pipes
func TestBash_CommandWithPipes(t *testing.T) {
	// Create JSON input for a piped command
	input := BashInput{Command: "echo 'hello world' | tr 'a-z' 'A-Z'"}
	inputJSON, err := json.Marshal(input)
	if err != nil {
		t.Fatalf("Failed to marshal input: %v", err)
	}

	// Call the Bash function
	result, err := Bash(inputJSON)

	// Verify no error occurred
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	// Verify the output is uppercase
	expectedOutput := "HELLO WORLD"
	if result != expectedOutput {
		t.Errorf("Expected output %q, got %q", expectedOutput, result)
	}
}
