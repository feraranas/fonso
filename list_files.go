// list_files.go - ADVANCED chat with multiple tools (read_file + list_files)
// This extends read.go by adding a second tool, demonstrating how to build
// an agent with multiple capabilities.
//
// Learning Path:
// 1. Start with chat.go - Basic request/response
// 2. Then read.go - Single tool (read_file)
// 3. Now this file - Multiple tools working together
//
// New Concepts vs read.go:
// - Multiple tool definitions in one agent
// - Claude can choose which tool to use based on the question
// - Tools can work together (list files, then read specific ones)
//
// Example Flow:
// 1. User: "What Go files are in this project?"
// 2. Claude uses list_files tool
// 3. User: "Read the chat.go file"
// 4. Claude uses read_file tool

package main

import (
	"context"       // For managing API request contexts
	"encoding/json" // For parsing tool inputs/outputs as JSON
	"fmt"           // For formatted output to the console
	"log"           // For logging debug information
	"os"            // For file operations
	"path/filepath" // For walking directory trees
	"strings"       // For string manipulation

	"github.com/anthropics/anthropic-sdk-go" // Anthropic's official Go SDK
	"github.com/invopop/jsonschema"          // For generating JSON schemas from Go structs
)

func main() {
	// Use shared utility functions from common.go (same as chat.go and read.go)
	verbose := ParseFlags()
	SetupLogging(verbose)
	client := InitClient(verbose)
	getUserMessage := CreateInputReader()

	// NEW vs read.go: Define MULTIPLE tools
	// Claude can choose which tool to use based on the user's question
	tools := []ToolDefinition{
		ReadFileDefinition,  // Read file contents
		ListFilesDefinition, // List directory contents
	}
	if verbose {
		log.Printf("Initialized %d tools", len(tools))
	}

	// Create and run the agent with multiple tool support
	agent := NewMultiToolAgent(&client, getUserMessage, tools, verbose)
	err := agent.Run(context.TODO())
	if err != nil {
		fmt.Printf("Error: %s\n", err.Error())
	}
}

// NewMultiToolAgent creates a new chat agent with multiple tool capabilities
// This is similar to ToolAgent from read.go, but demonstrates multiple tools
//
// Parameters:
//   - client: The Anthropic API client
//   - getUserMessage: Function to get user input
//   - tools: List of tools (read_file, list_files, etc.)
//   - verbose: Enable detailed logging
func NewMultiToolAgent(
	client *anthropic.Client,
	getUserMessage func() (string, bool),
	tools []ToolDefinition,
	verbose bool,
) *MultiToolAgent {
	return &MultiToolAgent{
		client:         client,
		getUserMessage: getUserMessage,
		tools:          tools,
		verbose:        verbose,
	}
}

// MultiToolAgent manages conversations with Claude using multiple tools
//
// Key Feature: Claude intelligently chooses which tool to use
// - "What files are here?" → uses list_files
// - "Read README.md" → uses read_file
// - Can use multiple tools in sequence for complex tasks
//
// This demonstrates how LLMs can be "agentic" - making decisions
// about which tools to use based on the context.
type MultiToolAgent struct {
	client         *anthropic.Client     // API client for Claude
	getUserMessage func() (string, bool) // Function to get user input
	tools          []ToolDefinition      // Available tools (multiple!)
	verbose        bool                  // Enable detailed logging
}

// Run starts the interactive chat session with multiple tool support
//
// Same structure as ToolAgent.Run() from read.go:
// - Outer loop: Get user input
// - Inner loop: Execute tools Claude requests
//
// The difference: Claude now has MORE tools to choose from!
func (a *MultiToolAgent) Run(ctx context.Context) error {
	// Initialize empty conversation history
	conversation := []anthropic.MessageParam{}

	if a.verbose {
		log.Println("Starting chat session with multiple tools enabled")
	}
	fmt.Println("Chat with Claude (use 'ctrl-c' to quit)")

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
		// Claude may use multiple tools in sequence to answer the question
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
// Same as read.go, but with multiple tools passed to Claude
// Claude will intelligently choose which tool(s) to use
func (a *MultiToolAgent) runInference(ctx context.Context, conversation []anthropic.MessageParam) (*anthropic.Message, error) {
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
		Tools:     anthropicTools, // Claude sees all available tools
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
// Same structure as in read.go - this is reusable across all tool-based agents
type ToolDefinition struct {
	Name        string                                      `json:"name"`
	Description string                                      `json:"description"`
	InputSchema anthropic.ToolInputSchemaParam              `json:"input_schema"`
	Function    func(input json.RawMessage) (string, error) // Takes JSON input, returns string result
}

// ReadFileDefinition allows Claude to read file contents
// Same tool as in read.go - tools are composable!
var ReadFileDefinition = ToolDefinition{
	Name:        "read_file",
	Description: "Read the contents of a given relative file path. Use this when you want to see what's inside a file. Do not use this with directory names.",
	InputSchema: ReadFileInputSchema,
	Function:    ReadFile,
}

// ListFilesDefinition allows Claude to list directory contents
// NEW tool - this is what makes this example different from read.go
var ListFilesDefinition = ToolDefinition{
	Name:        "list_files",
	Description: "List files and directories at a given path. If no path is provided, lists files in the current directory. Use this to explore the project structure or find files.",
	InputSchema: ListFilesInputSchema,
	Function:    ListFiles,
}

// ReadFileInput defines parameters for read_file tool
type ReadFileInput struct {
	Path string `json:"path" jsonschema_description:"The relative path of a file in the working directory."`
}

var ReadFileInputSchema = GenerateSchema[ReadFileInput]()

// ListFilesInput defines parameters for list_files tool
// Note: Path is optional (omitempty) - defaults to current directory
type ListFilesInput struct {
	Path string `json:"path,omitempty" jsonschema_description:"Optional relative path to list files from. Defaults to current directory if not provided."`
}

var ListFilesInputSchema = GenerateSchema[ListFilesInput]()

// ReadFile implements the read_file tool
// Same implementation as read.go
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
// This walks the directory tree and returns a JSON array of files/directories
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
		// Add more patterns here if needed (e.g., node_modules, .git)
		if info.IsDir() && (relPath == ".devenv" || strings.HasPrefix(relPath, ".devenv/")) {
			return filepath.SkipDir
		}

		// Add file/directory to list (skip the base directory itself)
		if relPath != "." {
			if info.IsDir() {
				// Add trailing slash to indicate it's a directory
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

	log.Printf("Successfully listed %d items in %s", len(files), dir)

	// Return as JSON array for easy parsing by Claude
	result, err := json.Marshal(files)
	if err != nil {
		return "", err
	}

	return string(result), nil
}

// GenerateSchema generates a JSON schema from a Go struct using generics
// Same helper function used in read.go - tools share common infrastructure
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
