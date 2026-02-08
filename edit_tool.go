// edit_tool.go - MOST ADVANCED: Full coding agent with file editing
// This is the ultimate example, adding file editing to create a complete
// AI coding assistant that can explore, read, execute, AND modify code.
//
// Learning Path:
// 1. Start with chat.go - Basic request/response
// 2. Then read.go - Single tool (read_file)
// 3. Then list_files.go - Multiple tools (read_file + list_files)
// 4. Then bash_tool.go - Command execution (+ bash)
// 5. Now this file - File editing (+ edit_file) - FULL CODING AGENT!
//
// New Concepts vs bash_tool.go:
// - File editing/modification capability
// - Search and replace functionality
// - File creation with directory creation
// - Safety checks (unique match requirement)
// - Even MORE security considerations!
//
// ⚠️⚠️ EXTREME SECURITY WARNING ⚠️⚠️
// This is the MOST DANGEROUS example! It can:
// - Execute arbitrary commands
// - Modify ANY file on your system
// - Create new files and directories
// - Potentially destroy your code!
//
// 🛡️ ONLY USE IN A SAFE, CONTROLLED, SANDBOXED ENVIRONMENT! 🛡️
//
// Example Flow:
// 1. User: "What files are in this project?"
// 2. Claude uses list_files
// 3. User: "Add a comment to chat.go explaining the main function"
// 4. Claude reads chat.go, then uses edit_file to add the comment
// 5. User: "Run the tests"
// 6. Claude uses bash to execute "go test"
//
// This demonstrates a full "AI Coding Agent" - the type of system that
// powers tools like GitHub Copilot Workspace, Cursor, and other AI IDEs.

package main

import (
	"context"       // For managing API request contexts
	"encoding/json" // For parsing tool inputs/outputs as JSON
	"fmt"           // For formatted output to the console
	"log"           // For logging debug information
	"os"            // For file operations
	"os/exec"       // For executing bash commands
	"path"          // For path manipulation
	"path/filepath" // For walking directory trees
	"strings"       // For string manipulation

	"github.com/anthropics/anthropic-sdk-go" // Anthropic's official Go SDK
	"github.com/invopop/jsonschema"          // For generating JSON schemas from Go structs
)

func main() {
	// Use shared utility functions from common.go (same as all other examples)
	verbose := ParseFlags()
	SetupLogging(verbose)
	client := InitClient(verbose)
	getUserMessage := CreateInputReader()

	// NEW vs bash_tool.go: Added file editing tool
	// This creates the MOST POWERFUL agent in our examples:
	// - Read files (explore code)
	// - List directories (understand structure)
	// - Execute commands (run tests, build, etc.)
	// - Edit files (modify code!) ← NEW!
	//
	// ⚠️⚠️ EXTREME DANGER! Can modify your code! ⚠️⚠️
	tools := []ToolDefinition{
		ReadFileDefinition,  // Read file contents
		ListFilesDefinition, // List directory contents
		BashDefinition,      // Execute bash commands
		EditFileDefinition,  // Edit/create files (NEW!)
	}
	if verbose {
		log.Printf("⚠️  Initialized %d tools (including FILE EDITING)", len(tools))
	}

	// Create and run the coding agent
	agent := NewCodingAgent(&client, getUserMessage, tools, verbose)
	err := agent.Run(context.TODO())
	if err != nil {
		fmt.Printf("Error: %s\n", err.Error())
	}
}

// NewCodingAgent creates a full-featured AI coding agent
// This is the most advanced agent - it can read, explore, execute, AND edit code!
//
// Parameters:
//   - client: The Anthropic API client
//   - getUserMessage: Function to get user input
//   - tools: All 4 tools (read_file, list_files, bash, edit_file)
//   - verbose: Enable detailed logging
func NewCodingAgent(
	client *anthropic.Client,
	getUserMessage func() (string, bool),
	tools []ToolDefinition,
	verbose bool,
) *CodingAgent {
	return &CodingAgent{
		client:         client,
		getUserMessage: getUserMessage,
		tools:          tools,
		verbose:        verbose,
	}
}

// CodingAgent - A full AI coding assistant
//
// Evolution of Agents (Complete Journey):
// 1. BasicAgent (chat.go): Just conversation
// 2. ToolAgent (read.go): + Read files
// 3. MultiToolAgent (list_files.go): + List directories
// 4. BashAgent (bash_tool.go): + Execute commands
// 5. CodingAgent (this file): + Edit files = FULL CODING AGENT!
//
// This represents a complete AI coding assistant that can:
// - Understand your codebase (list + read)
// - Execute commands (test, build, run)
// - Modify code (add features, fix bugs, refactor)
//
// This is essentially what powers modern AI coding tools!
//
// ⚠️ Security Implications:
// - Can read any file (privacy concern)
// - Can execute any command (security risk)
// - Can modify/delete code (data loss risk)
// - Should NEVER be exposed to untrusted input
// - Should ALWAYS require user confirmation for edits
// - Should run in isolated/sandboxed environments
type CodingAgent struct {
	client         *anthropic.Client     // API client for Claude
	getUserMessage func() (string, bool) // Function to get user input
	tools          []ToolDefinition      // All 4 tools!
	verbose        bool                  // Enable detailed logging
}

// Run starts the interactive coding session
//
// Same structure as BashAgent.Run() from bash_tool.go:
// - Outer loop: Get user input
// - Inner loop: Execute tools Claude requests
//
// The difference: Claude can now MODIFY files, not just read them!
//
// Example conversation:
// User: "Add error handling to the ReadFile function"
// Claude:
//  1. Reads the file to see current code
//  2. Identifies what needs to change
//  3. Uses edit_file to make the modification
//  4. Confirms the change was successful
func (a *CodingAgent) Run(ctx context.Context) error {
	// Initialize empty conversation history
	conversation := []anthropic.MessageParam{}

	if a.verbose {
		log.Println("Starting coding agent session with full edit capabilities")
	}
	fmt.Println("🤖 AI Coding Agent (use 'ctrl-c' to quit)")
	fmt.Println("⚠️⚠️ WARNING: This agent can MODIFY FILES on your system! ⚠️⚠️")
	fmt.Println("🛡️  Only use in a safe, version-controlled environment!")
	fmt.Println()

	// Outer loop: Get user input
	for {
		fmt.Print("\u001b[94mYou\u001b[0m: ")
		userInput, ok := a.getUserMessage()
		if !ok {
			if a.verbose {
				log.Println("User input ended, breaking from chat loop")
			}
			break
		}

		// Skip empty messages
		if userInput == "" {
			if a.verbose {
				log.Println("Skipping empty message")
			}
			continue
		}

		if a.verbose {
			log.Printf("User input received: %q", userInput)
		}

		// Add user message to conversation
		userMessage := anthropic.NewUserMessage(anthropic.NewTextBlock(userInput))
		conversation = append(conversation, userMessage)

		if a.verbose {
			log.Printf("Sending message to Claude, conversation length: %d", len(conversation))
		}

		// Get Claude's initial response
		message, err := a.runInference(ctx, conversation)
		if err != nil {
			if a.verbose {
				log.Printf("Error during inference: %v", err)
			}
			return err
		}
		conversation = append(conversation, message.ToParam())

		// Inner loop: Keep processing until Claude stops using tools
		// Claude may use multiple tools in sequence:
		// Example: list files → read specific file → edit that file → run tests
		for {
			// Collect all tool uses and their results from this response
			var toolResults []anthropic.ContentBlockParamUnion
			var hasToolUse bool

			if a.verbose {
				log.Printf("Processing %d content blocks from Claude", len(message.Content))
			}

			// Process each content block in Claude's response
			for _, content := range message.Content {
				switch content.Type {
				case "text":
					// Regular text response - display to user
					fmt.Printf("\u001b[93mClaude\u001b[0m: %s\n", content.Text)

				case "tool_use":
					// Claude wants to use a tool!
					hasToolUse = true
					toolUse := content.AsToolUse()
					if a.verbose {
						log.Printf("Tool use detected: %s with input: %s", toolUse.Name, string(toolUse.Input))
					}
					// Display the tool call in cyan
					fmt.Printf("\u001b[96mtool\u001b[0m: %s(%s)\n", toolUse.Name, string(toolUse.Input))

					// Find and execute the requested tool
					var toolResult string
					var toolError error
					var toolFound bool

					// Search through our available tools
					for _, tool := range a.tools {
						if tool.Name == toolUse.Name {
							if a.verbose {
								log.Printf("Executing tool: %s", tool.Name)
							}
							// Execute the tool function with the provided input
							toolResult, toolError = tool.Function(toolUse.Input)

							// Display result in green
							fmt.Printf("\u001b[92mresult\u001b[0m: %s\n", toolResult)
							if toolError != nil {
								// Display error in red
								fmt.Printf("\u001b[91merror\u001b[0m: %s\n", toolError.Error())
							}
							if a.verbose {
								if toolError != nil {
									log.Printf("Tool execution failed: %v", toolError)
								} else {
									log.Printf("Tool execution successful, result length: %d chars", len(toolResult))
								}
							}
							toolFound = true
							break
						}
					}

					// Handle case where tool wasn't found
					if !toolFound {
						toolError = fmt.Errorf("tool '%s' not found", toolUse.Name)
						fmt.Printf("\u001b[91merror\u001b[0m: %s\n", toolError.Error())
					}

					// Create a tool result block to send back to Claude
					if toolError != nil {
						toolResults = append(toolResults, anthropic.NewToolResultBlock(toolUse.ID, toolError.Error(), true))
					} else {
						toolResults = append(toolResults, anthropic.NewToolResultBlock(toolUse.ID, toolResult, false))
					}
				}
			}

			// If Claude didn't request any tools, we're done with this turn
			if !hasToolUse {
				break
			}

			// Send all tool results back to Claude
			if a.verbose {
				log.Printf("Sending %d tool results back to Claude", len(toolResults))
			}
			toolResultMessage := anthropic.NewUserMessage(toolResults...)
			conversation = append(conversation, toolResultMessage)

			// Get Claude's response after receiving the tool results
			message, err = a.runInference(ctx, conversation)
			if err != nil {
				if a.verbose {
					log.Printf("Error during followup inference: %v", err)
				}
				return err
			}
			conversation = append(conversation, message.ToParam())

			if a.verbose {
				log.Printf("Received followup response with %d content blocks", len(message.Content))
			}

			// Loop continues - check if Claude wants to use more tools
		}
	}

	if a.verbose {
		log.Println("Coding agent session ended")
	}
	return nil
}

// runInference makes an API call to Claude with the conversation and available tools
//
// Same as bash_tool.go, but with the edit_file tool included
// This gives Claude FULL coding capabilities
func (a *CodingAgent) runInference(ctx context.Context, conversation []anthropic.MessageParam) (*anthropic.Message, error) {
	// Convert our tool definitions to Anthropic's format
	anthropicTools := []anthropic.ToolUnionParam{}
	for _, tool := range a.tools {
		anthropicTools = append(anthropicTools, anthropic.ToolUnionParam{
			OfTool: &anthropic.ToolParam{
				Name:        tool.Name,
				Description: anthropic.String(tool.Description),
				InputSchema: tool.InputSchema,
			},
		})
	}

	if a.verbose {
		log.Printf("Making API call to Claude with model: %s and %d tools", anthropic.ModelClaude3_7SonnetLatest, len(anthropicTools))
	}

	// Make the API call with all tools included
	message, err := a.client.Messages.New(ctx, anthropic.MessageNewParams{
		Model:     anthropic.ModelClaude3_7SonnetLatest,
		MaxTokens: int64(1024),
		Messages:  conversation,
		Tools:     anthropicTools, // Claude sees ALL tools including edit_file!
	})

	if a.verbose {
		if err != nil {
			log.Printf("API call failed: %v", err)
		} else {
			log.Printf("API call successful, response received")
		}
	}

	return message, err
}

// ToolDefinition defines a function that Claude can call
// Same structure as all previous examples - maximum reusability!
type ToolDefinition struct {
	Name        string                                      `json:"name"`
	Description string                                      `json:"description"`
	InputSchema anthropic.ToolInputSchemaParam              `json:"input_schema"`
	Function    func(input json.RawMessage) (string, error) // Takes JSON input, returns string result
}

// ReadFileDefinition allows Claude to read file contents
// Reused from previous examples
var ReadFileDefinition = ToolDefinition{
	Name:        "read_file",
	Description: "Read the contents of a given relative file path. Use this when you want to see what's inside a file. Do not use this with directory names.",
	InputSchema: ReadFileInputSchema,
	Function:    ReadFile,
}

// ListFilesDefinition allows Claude to list directory contents
// Reused from list_files.go
var ListFilesDefinition = ToolDefinition{
	Name:        "list_files",
	Description: "List files and directories at a given path. If no path is provided, lists files in the current directory. Use this to explore the project structure or find files.",
	InputSchema: ListFilesInputSchema,
	Function:    ListFiles,
}

// BashDefinition allows Claude to execute bash commands
// Reused from bash_tool.go
var BashDefinition = ToolDefinition{
	Name:        "bash",
	Description: "Execute a bash command and return its output. Use this to run shell commands like 'git status', 'go test', 'go build', etc. Be careful with destructive commands.",
	InputSchema: BashInputSchema,
	Function:    Bash,
}

// EditFileDefinition allows Claude to edit/create files
// NEW tool - this is what makes this a full coding agent!
//
// How it works:
// - Searches for `old_str` in the file
// - Replaces it with `new_str`
// - Requires EXACTLY one match (for safety)
// - Can create new files if old_str is empty
//
// Safety features:
// - old_str must be unique (prevents accidental multiple changes)
// - old_str and new_str must be different
// - Creates directories as needed
// - Returns clear error messages
//
// ⚠️ Production Considerations:
// - Add backup/undo functionality
// - Require user confirmation before edits
// - Show diffs before applying
// - Integrate with version control (git)
// - Add file size limits
// - Whitelist which files can be edited
var EditFileDefinition = ToolDefinition{
	Name: "edit_file",
	Description: `Make edits to a text file.

Replaces 'old_str' with 'new_str' in the given file. 'old_str' and 'new_str' MUST be different from each other.

The 'old_str' must appear EXACTLY ONCE in the file for safety (to prevent accidental multiple replacements).

If the file doesn't exist and old_str is empty, a new file will be created with new_str as content.`,
	InputSchema: EditFileInputSchema,
	Function:    EditFile,
}

// Input type definitions for each tool

type ReadFileInput struct {
	Path string `json:"path" jsonschema_description:"The relative path of a file in the working directory."`
}

var ReadFileInputSchema = GenerateSchema[ReadFileInput]()

type ListFilesInput struct {
	Path string `json:"path,omitempty" jsonschema_description:"Optional relative path to list files from. Defaults to current directory if not provided."`
}

var ListFilesInputSchema = GenerateSchema[ListFilesInput]()

type BashInput struct {
	Command string `json:"command" jsonschema_description:"The bash command to execute. Should be a single command or pipeline."`
}

var BashInputSchema = GenerateSchema[BashInput]()

type EditFileInput struct {
	Path   string `json:"path" jsonschema_description:"The path to the file to edit or create."`
	OldStr string `json:"old_str" jsonschema_description:"Text to search for - must match exactly and must only have one match. Can be empty to create a new file."`
	NewStr string `json:"new_str" jsonschema_description:"Text to replace old_str with, or content for a new file."`
}

var EditFileInputSchema = GenerateSchema[EditFileInput]()

// Tool implementation functions

// ReadFile implements the read_file tool
// Same implementation as previous examples
func ReadFile(input json.RawMessage) (string, error) {
	readFileInput := ReadFileInput{}
	err := json.Unmarshal(input, &readFileInput)
	if err != nil {
		panic(err)
	}

	log.Printf("Reading file: %s", readFileInput.Path)

	content, err := os.ReadFile(readFileInput.Path)
	if err != nil {
		log.Printf("Failed to read file %s: %v", readFileInput.Path, err)
		return "", err
	}

	log.Printf("Successfully read file %s (%d bytes)", readFileInput.Path, len(content))
	return string(content), nil
}

// ListFiles implements the list_files tool
// Same implementation as list_files.go
func ListFiles(input json.RawMessage) (string, error) {
	listFilesInput := ListFilesInput{}
	err := json.Unmarshal(input, &listFilesInput)
	if err != nil {
		panic(err)
	}

	// Default to current directory if no path specified
	dir := "."
	if listFilesInput.Path != "" {
		dir = listFilesInput.Path
	}

	log.Printf("Listing files in directory: %s", dir)

	var files []string

	// Walk the directory tree
	err = filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Get relative path from the base directory
		relPath, err := filepath.Rel(dir, path)
		if err != nil {
			return err
		}

		// Skip certain directories
		if info.IsDir() && (relPath == ".devenv" || strings.HasPrefix(relPath, ".devenv/")) {
			return filepath.SkipDir
		}

		// Add file/directory to list
		if relPath != "." {
			if info.IsDir() {
				files = append(files, relPath+"/")
			} else {
				files = append(files, relPath)
			}
		}
		return nil
	})

	if err != nil {
		log.Printf("Failed to list files in %s: %v", dir, err)
		return "", err
	}

	// Return as JSON array
	result, err := json.Marshal(files)
	if err != nil {
		return "", err
	}

	log.Printf("Successfully listed %d files in %s", len(files), dir)
	return string(result), nil
}

// Bash implements the bash tool - executes shell commands
// Same implementation as bash_tool.go
func Bash(input json.RawMessage) (string, error) {
	bashInput := BashInput{}
	err := json.Unmarshal(input, &bashInput)
	if err != nil {
		return "", err
	}

	log.Printf("Executing bash command: %s", bashInput.Command)

	// Execute the command using bash -c
	cmd := exec.Command("bash", "-c", bashInput.Command)

	// CombinedOutput captures both stdout and stderr
	output, err := cmd.CombinedOutput()

	if err != nil {
		// Command failed, but we still return the output
		log.Printf("Bash command failed: %v", err)
		return fmt.Sprintf("Command failed with error: %s\nOutput: %s", err.Error(), string(output)), nil
	}

	log.Printf("Bash command executed successfully, output length: %d chars", len(output))

	// Return trimmed output
	return strings.TrimSpace(string(output)), nil
}

// EditFile implements the edit_file tool - modify files!
//
// # NEW TOOL - File Editing
//
// How it works:
// 1. Parse the edit request (path, old_str, new_str)
// 2. Validate inputs (old_str != new_str, path not empty)
// 3. Read the file (or create if it doesn't exist)
// 4. Search for old_str - must find EXACTLY one match
// 5. Replace old_str with new_str
// 6. Write the modified content back to the file
//
// Safety Features:
// - Requires unique match: Prevents accidental multiple changes
// - Validates inputs: Prevents no-op edits
// - Clear error messages: Helps Claude understand what went wrong
// - Atomic writes: File is only modified if all checks pass
//
// Special Cases:
// - old_str empty + file doesn't exist = Create new file
// - old_str empty + file exists = Append to file
// - old_str not found = Error (helps Claude realize it needs to read the file first)
// - old_str found multiple times = Error (Claude needs to be more specific)
//
// Example Usage by Claude:
// 1. User: "Add error handling to the ReadFile function"
// 2. Claude reads the file to see current code
// 3. Claude identifies the exact code block to replace
// 4. Claude calls edit_file with:
//   - path: "read.go"
//   - old_str: "content, err := os.ReadFile(path)"
//   - new_str: "content, err := os.ReadFile(path)\nif err != nil {\n    return \"\", fmt.Errorf(\"failed to read file: %w\", err)\n}"
//
// 5. Function validates, finds one match, makes the replacement
// 6. Returns "OK" to confirm success
func EditFile(input json.RawMessage) (string, error) {
	editFileInput := EditFileInput{}
	err := json.Unmarshal(input, &editFileInput)
	if err != nil {
		return "", err
	}

	// Validate inputs
	if editFileInput.Path == "" || editFileInput.OldStr == editFileInput.NewStr {
		log.Printf("EditFile failed: invalid input parameters")
		return "", fmt.Errorf("invalid input parameters: path must not be empty and old_str must differ from new_str")
	}

	log.Printf("Editing file: %s (replacing %d chars with %d chars)", editFileInput.Path, len(editFileInput.OldStr), len(editFileInput.NewStr))

	// Try to read the file
	content, err := os.ReadFile(editFileInput.Path)
	if err != nil {
		// If file doesn't exist and old_str is empty, create a new file
		if os.IsNotExist(err) && editFileInput.OldStr == "" {
			log.Printf("File does not exist, creating new file: %s", editFileInput.Path)
			return createNewFile(editFileInput.Path, editFileInput.NewStr)
		}
		log.Printf("Failed to read file %s: %v", editFileInput.Path, err)
		return "", err
	}

	oldContent := string(content)

	// Special case: if old_str is empty, we're appending to the file
	var newContent string
	if editFileInput.OldStr == "" {
		newContent = oldContent + editFileInput.NewStr
		log.Printf("Appending to file: %s", editFileInput.Path)
	} else {
		// Count occurrences first to ensure we have exactly one match
		count := strings.Count(oldContent, editFileInput.OldStr)
		if count == 0 {
			log.Printf("EditFile failed: old_str not found in file %s", editFileInput.Path)
			return "", fmt.Errorf("old_str not found in file. Make sure to read the file first to see its exact content")
		}
		if count > 1 {
			log.Printf("EditFile failed: old_str found %d times in file %s, must be unique", count, editFileInput.Path)
			return "", fmt.Errorf("old_str found %d times in file, must be unique. Include more context in old_str to make it unique", count)
		}

		// Perform the replacement
		newContent = strings.Replace(oldContent, editFileInput.OldStr, editFileInput.NewStr, 1)
	}

	// Write the modified content back to the file
	err = os.WriteFile(editFileInput.Path, []byte(newContent), 0644)
	if err != nil {
		log.Printf("Failed to write file %s: %v", editFileInput.Path, err)
		return "", err
	}

	log.Printf("Successfully edited file %s", editFileInput.Path)
	return "OK", nil
}

// createNewFile creates a new file with the given content
// Also creates any necessary parent directories
func createNewFile(filePath, content string) (string, error) {
	log.Printf("Creating new file: %s (%d bytes)", filePath, len(content))

	// Create parent directories if needed
	dir := path.Dir(filePath)
	if dir != "." {
		log.Printf("Creating directory: %s", dir)
		err := os.MkdirAll(dir, 0755)
		if err != nil {
			log.Printf("Failed to create directory %s: %v", dir, err)
			return "", fmt.Errorf("failed to create directory: %w", err)
		}
	}

	// Create the file
	err := os.WriteFile(filePath, []byte(content), 0644)
	if err != nil {
		log.Printf("Failed to create file %s: %v", filePath, err)
		return "", fmt.Errorf("failed to create file: %w", err)
	}

	log.Printf("Successfully created file %s", filePath)
	return fmt.Sprintf("Successfully created file %s", filePath), nil
}

// GenerateSchema generates a JSON schema from a Go struct using generics
// Same helper function used in all tool-based examples
func GenerateSchema[T any]() anthropic.ToolInputSchemaParam {
	reflector := jsonschema.Reflector{
		AllowAdditionalProperties: false,
		DoNotReference:            true,
	}
	var v T

	schema := reflector.Reflect(v)

	return anthropic.ToolInputSchemaParam{
		Properties: schema.Properties,
	}
}
