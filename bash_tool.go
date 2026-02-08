// bash_tool.go - ADVANCED agent with command execution capability
// This extends list_files.go by adding a bash execution tool, creating
// a powerful agent that can explore AND execute commands.
//
// Learning Path:
// 1. Start with chat.go - Basic request/response
// 2. Then read.go - Single tool (read_file)
// 3. Then list_files.go - Multiple tools (read_file + list_files)
// 4. Now this file - Command execution (read_file + list_files + bash)
//
// New Concepts vs list_files.go:
// - Command execution tool (potentially dangerous!)
// - Security considerations when giving AI shell access
// - How to handle command errors and output
// - Building a more capable "agentic" system
//
// ⚠️  SECURITY WARNING:
// This tool allows Claude to execute arbitrary bash commands on your system!
// Only use in a safe, controlled environment. Never expose this to untrusted input.
//
// Example Flow:
// 1. User: "What Go files are here?"
// 2. Claude uses list_files tool
// 3. User: "Show me the current git status"
// 4. Claude uses bash tool with "git status"
// 5. User: "Read the README.md file"
// 6. Claude uses read_file tool

package main

import (
	"context"       // For managing API request contexts
	"encoding/json" // For parsing tool inputs/outputs as JSON
	"fmt"           // For formatted output to the console
	"log"           // For logging debug information
	"os"            // For file operations
	"os/exec"       // For executing bash commands
	"path/filepath" // For walking directory trees
	"strings"       // For string manipulation

	"github.com/anthropics/anthropic-sdk-go" // Anthropic's official Go SDK
	"github.com/invopop/jsonschema"          // For generating JSON schemas from Go structs
)

func main() {
	// Use shared utility functions from common.go
	verbose := ParseFlags()
	SetupLogging(verbose)
	client := InitClient(verbose)
	getUserMessage := CreateInputReader()

	// NEW vs list_files.go: Added bash execution tool
	// This creates a very powerful agent that can:
	// - Read files
	// - List directories
	// - Execute shell commands
	//
	// ⚠️  Use with caution! This can execute ANY command.
	tools := []ToolDefinition{
		ReadFileDefinition,  // Read file contents
		ListFilesDefinition, // List directory contents
		BashDefinition,      // Execute bash commands (NEW!)
	}
	if verbose {
		log.Printf("Initialized %d tools (including bash execution)", len(tools))
	}

	// Create and run the agent with command execution capability
	agent := NewBashAgent(&client, getUserMessage, tools, verbose)
	err := agent.Run(context.TODO())
	if err != nil {
		fmt.Printf("Error: %s\n", err.Error())
	}
}

// NewBashAgent creates a new chat agent with bash execution capability
// This is the most powerful agent in our examples - it can execute commands!
//
// Parameters:
//   - client: The Anthropic API client
//   - getUserMessage: Function to get user input
//   - tools: List of tools (read_file, list_files, bash)
//   - verbose: Enable detailed logging
func NewBashAgent(
	client *anthropic.Client,
	getUserMessage func() (string, bool),
	tools []ToolDefinition,
	verbose bool,
) *BashAgent {
	return &BashAgent{
		client:         client,
		getUserMessage: getUserMessage,
		tools:          tools,
		verbose:        verbose,
	}
}

// BashAgent manages conversations with Claude and can execute shell commands
//
// Evolution of Agents:
// - BasicAgent (chat.go): Simple conversation
// - ToolAgent (read.go): Can read files
// - MultiToolAgent (list_files.go): Can read files and list directories
// - BashAgent (this file): Can do all of the above + execute commands!
//
// Key Feature: Claude can now take ACTIONS on your system
// - "What's the git status?" → runs `git status`
// - "What Go version am I using?" → runs `go version`
// - "Create a new directory" → runs `mkdir dirname`
//
// This demonstrates a key concept in AI agents: giving them the ability
// to interact with and modify their environment, not just read from it.
type BashAgent struct {
	client         *anthropic.Client     // API client for Claude
	getUserMessage func() (string, bool) // Function to get user input
	tools          []ToolDefinition      // Available tools (including bash!)
	verbose        bool                  // Enable detailed logging
}

// Run starts the interactive chat session with bash execution support
//
// Same structure as MultiToolAgent.Run() from list_files.go:
// - Outer loop: Get user input
// - Inner loop: Execute tools Claude requests
//
// The difference: Claude can now execute shell commands, making it more "agentic"
func (a *BashAgent) Run(ctx context.Context) error {
	// Initialize empty conversation history
	conversation := []anthropic.MessageParam{}

	if a.verbose {
		log.Println("Starting chat session with bash execution enabled")
	}
	fmt.Println("Chat with Claude (use 'ctrl-c' to quit)")
	fmt.Println("⚠️  Warning: Bash execution is enabled. Claude can run commands on your system!")

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
		// Claude may use multiple tools in sequence (list files, read one, execute command, etc.)
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
		log.Println("Chat session ended")
	}
	return nil
}

// runInference makes an API call to Claude with the conversation and available tools
//
// Same as list_files.go, but with the bash tool included
// Claude will intelligently choose which tool to use based on the task
func (a *BashAgent) runInference(ctx context.Context, conversation []anthropic.MessageParam) (*anthropic.Message, error) {
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
		Tools:     anthropicTools, // Claude sees all available tools including bash
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
// Same structure as all previous examples - tools are composable!
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
// NEW tool - this is what makes this example powerful (and potentially dangerous!)
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

// ReadFileInput defines parameters for read_file tool
type ReadFileInput struct {
	Path string `json:"path" jsonschema_description:"The relative path of a file in the working directory."`
}

var ReadFileInputSchema = GenerateSchema[ReadFileInput]()

// ListFilesInput defines parameters for list_files tool
type ListFilesInput struct {
	Path string `json:"path,omitempty" jsonschema_description:"Optional relative path to list files from. Defaults to current directory if not provided."`
}

var ListFilesInputSchema = GenerateSchema[ListFilesInput]()

// BashInput defines parameters for bash tool
type BashInput struct {
	Command string `json:"command" jsonschema_description:"The bash command to execute. Should be a single command or pipeline."`
}

var BashInputSchema = GenerateSchema[BashInput]()

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

	log.Printf("Successfully listed %d files/directories in %s", len(files), dir)
	return string(result), nil
}

// Bash implements the bash tool - executes shell commands
//
// # NEW TOOL - Command Execution
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
//
// Example commands Claude might use:
// - "git status" - Check git status
// - "go version" - Check Go version
// - "ls -la" - List files with details
// - "cat file.txt" - Show file contents (though read_file is better)
// - "date" - Get current date/time
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
