// bash.go - Bash command execution tool implementation
package tools

import (
	"encoding/json"
	"fmt"
	"log"
	"os/exec"
	"strings"
)

// BashInput defines parameters for bash tool
type BashInput struct {
	Command string `json:"command" jsonschema_description:"The bash command to execute. Should be a single command or pipeline."`
}

// BashInputSchema is the generated JSON schema for BashInput
var BashInputSchema = GenerateSchema[BashInput]()

// BashDefinition allows Claude to execute bash commands
//
// Security Note: This tool executes arbitrary commands. In production:
// - Add command whitelisting (only allow safe commands)
// - Add user confirmation before execution
// - Run in a sandboxed environment
// - Log all executed commands
// - Set timeouts to prevent hanging commands
var BashDefinition = ToolDefinition{
	Name:        "bash",
	Description: "Execute a bash command and return its output. Use this to run shell commands like 'git status', 'ls -la', 'go version', etc. Be careful with destructive commands.",
	InputSchema: BashInputSchema,
	Function:    Bash,
}

// Bash implements the bash tool - executes shell commands
//
// How it works:
// 1. Parse the command from JSON input
// 2. Execute it with `bash -c "command"`
// 3. Capture both stdout and stderr (CombinedOutput)
// 4. Return output to Claude
//
// Error Handling:
// - If command fails, we return the error AND output (useful for debugging)
// - We trim whitespace from output for cleaner results
// - We log all executions for security auditing
func Bash(input json.RawMessage) (string, error) {
	bashInput := BashInput{}
	err := json.Unmarshal(input, &bashInput)
	if err != nil {
		return "", err
	}

	log.Printf("Executing bash command: %s", bashInput.Command)

	// Execute the command using bash -c
	// This allows pipes, redirects, and other shell features
	cmd := exec.Command("bash", "-c", bashInput.Command)

	// CombinedOutput captures both stdout and stderr
	output, err := cmd.CombinedOutput()

	if err != nil {
		// Command failed, but we still return the output (useful for Claude to understand the error)
		log.Printf("Bash command failed: %s, error: %v", bashInput.Command, err)
		return fmt.Sprintf("Command failed with error: %s\nOutput: %s", err.Error(), string(output)), nil
	}

	log.Printf("Bash command succeeded: %s (output: %d bytes)", bashInput.Command, len(output))

	// Return trimmed output (removes trailing newlines)
	return strings.TrimSpace(string(output)), nil
}
