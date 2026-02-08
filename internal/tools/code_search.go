// code_search.go - Code search tool implementation using ripgrep
package tools

import (
	"encoding/json"
	"fmt"
	"log"
	"os/exec"
	"strings"
)

// CodeSearchInput defines parameters for code_search tool
type CodeSearchInput struct {
	Pattern       string `json:"pattern" jsonschema_description:"The search pattern or regex to look for in code."`
	Path          string `json:"path,omitempty" jsonschema_description:"Optional path to search in (file or directory). Defaults to current directory."`
	FileType      string `json:"file_type,omitempty" jsonschema_description:"Optional file extension to limit search to (e.g., 'go', 'js', 'py', 'md')."`
	CaseSensitive bool   `json:"case_sensitive,omitempty" jsonschema_description:"Whether the search should be case sensitive. Defaults to false (case-insensitive)."`
}

// CodeSearchInputSchema is the generated JSON schema for CodeSearchInput
var CodeSearchInputSchema = GenerateSchema[CodeSearchInput]()

// CodeSearchDefinition allows Claude to search code using ripgrep
//
// Why This Matters:
// - Finding function definitions in large codebases
// - Locating variable usage across files
// - Searching for patterns (error handling, TODOs, etc.)
// - Understanding code structure quickly
var CodeSearchDefinition = ToolDefinition{
	Name: "code_search",
	Description: `Search for code patterns using ripgrep (rg) - a fast code search tool.

Use this to find:
- Function definitions (e.g., "func FunctionName")
- Variable usage (e.g., "myVariable")
- Code patterns (e.g., "if err", "TODO", "FIXME")
- Text in files

You can filter by file type (e.g., 'go', 'js', 'py') and control case sensitivity.
Results are limited to first 50 matches to keep output manageable.`,
	InputSchema: CodeSearchInputSchema,
	Function:    CodeSearch,
}

// CodeSearch implements the code_search tool - search code with ripgrep!
//
// How it works:
// 1. Parse search parameters (pattern, path, file_type, case_sensitive)
// 2. Build ripgrep (rg) command with appropriate flags
// 3. Execute the search
// 4. Parse and limit results (max 50 matches)
// 5. Return formatted results to Claude
func CodeSearch(input json.RawMessage) (string, error) {
	codeSearchInput := CodeSearchInput{}
	err := json.Unmarshal(input, &codeSearchInput)
	if err != nil {
		return "", err
	}

	if codeSearchInput.Pattern == "" {
		return "", fmt.Errorf("pattern is required for code search")
	}

	log.Printf("Searching for pattern: %s", codeSearchInput.Pattern)

	// Build ripgrep command with flags
	args := []string{"rg", "--line-number", "--with-filename", "--color=never"}

	// Add case sensitivity flag (default is case-insensitive)
	if !codeSearchInput.CaseSensitive {
		args = append(args, "--ignore-case")
	}

	// Add file type filter if specified
	// Examples: go, js, py, md, rs, java, etc.
	if codeSearchInput.FileType != "" {
		args = append(args, "--type", codeSearchInput.FileType)
	}

	// Add the search pattern
	args = append(args, codeSearchInput.Pattern)

	// Add path to search (defaults to current directory)
	if codeSearchInput.Path != "" {
		args = append(args, codeSearchInput.Path)
	} else {
		args = append(args, ".")
	}

	log.Printf("Executing ripgrep with args: %v", args[1:]) // Skip 'rg' for cleaner log

	// Execute ripgrep
	cmd := exec.Command(args[0], args[1:]...)
	output, err := cmd.Output()

	// ripgrep returns exit code 1 when no matches are found
	// This is NOT an error - just means the pattern wasn't found
	if err != nil {
		if exitError, ok := err.(*exec.ExitError); ok && exitError.ExitCode() == 1 {
			log.Printf("No matches found for pattern: %s", codeSearchInput.Pattern)
			return "No matches found", nil
		}
		log.Printf("Ripgrep command failed: %v", err)
		return "", fmt.Errorf("search failed: %w", err)
	}

	result := strings.TrimSpace(string(output))
	lines := strings.Split(result, "\n")

	log.Printf("Found %d matches for pattern: %s", len(lines), codeSearchInput.Pattern)

	// Limit output to prevent overwhelming responses
	// 50 matches is usually enough for Claude to understand the context
	if len(lines) > 50 {
		result = strings.Join(lines[:50], "\n") + fmt.Sprintf("\n... (showing first 50 of %d matches)", len(lines))
		log.Printf("Truncated results to first 50 matches")
	}

	return result, nil
}
