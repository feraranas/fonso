// read.go - ADVANCED chat application with tool usage (function calling)
// This builds upon the basic chat in chat.go by adding tools (function calling).
//
// Learning Path:
// 1. First understand chat.go - Basic request/response pattern
// 2. Now this file (read.go) - How to add tools that Claude can call
//
// Key New Concepts vs chat.go:
// - ToolDefinition: Defines functions Claude can call
// - Nested Loop: One loop for user input, another for tool execution
// - Tool Results: Sending function results back to Claude
//
// Tool Usage Flow:
// 1. User asks: "What's in README.md?"
// 2. Claude responds with a tool_use request for read_file("README.md")
// 3. Program executes the tool and returns the file contents
// 4. Claude receives the results and provides a natural language answer

package main

import (
	"context"       // For managing API request contexts
	"encoding/json" // For parsing tool inputs/outputs as JSON
	"fmt"           // For formatted output to the console
	"log"           // For logging debug information
	"os"            // For file operations

	"github.com/anthropics/anthropic-sdk-go" // Anthropic's official Go SDK
	"github.com/invopop/jsonschema"          // For generating JSON schemas from Go structs
)

func main() {
	// Use shared utility functions from common.go (same as chat.go)
	verbose := ParseFlags()
	SetupLogging(verbose)
	client := InitClient(verbose)
	getUserMessage := CreateInputReader()

	// NEW: Define the tools available to Claude
	// Currently only has read_file tool, but you can add more tools here
	// (e.g., write_file, list_directory, execute_command, etc.)
	tools := []ToolDefinition{ReadFileDefinition}
	if verbose {
		log.Printf("Initialized %d tools", len(tools))
	}

	// Create and run the agent with tool support
	agent := NewToolAgent(&client, getUserMessage, tools, verbose)
	err := agent.Run(context.TODO())
	if err != nil {
		fmt.Printf("Error: %s\n", err.Error())
	}
}

// NewToolAgent creates a new chat agent with tool usage capabilities
// This extends the BasicAgent concept from chat.go by adding tool support.
//
// Parameters:
//   - client: The Anthropic API client
//   - getUserMessage: Function to get user input (same as BasicAgent)
//   - tools: NEW - List of tools that Claude can use (e.g., read_file)
//   - verbose: Enable detailed logging
func NewToolAgent(
	client *anthropic.Client,
	getUserMessage func() (string, bool),
	tools []ToolDefinition,
	verbose bool,
) *ToolAgent {
	return &ToolAgent{
		client:         client,
		getUserMessage: getUserMessage,
		tools:          tools,
		verbose:        verbose,
	}
}

// ToolAgent manages conversations with Claude and executes tools when requested
//
// Comparison with BasicAgent (chat.go):
// - BasicAgent: Simple linear loop (user input → Claude response → repeat)
// - ToolAgent: Nested loop with tool execution (user input → Claude response → tools → repeat)
//
// This agent is more advanced because it can:
// - Detect when Claude wants to use a tool
// - Execute the tool function
// - Send the results back to Claude
// - Continue the conversation with the tool results in a loop
type ToolAgent struct {
	client         *anthropic.Client     // API client for Claude
	getUserMessage func() (string, bool) // Function to get user input
	tools          []ToolDefinition      // Available tools Claude can use
	verbose        bool                  // Enable detailed logging
}

// Run starts the interactive chat session with tool support
//
// Key Difference from BasicAgent.Run() in chat.go:
// - BasicAgent has ONE loop (get input → send → display response)
// - ToolAgent has TWO NESTED loops (outer: get input, inner: tool execution)
//
// The agentic loop:
// 1. User asks a question (outer loop)
// 2. Claude responds (may include tool_use requests)
// 3. Agent executes tools and sends results back (inner loop)
// 4. Claude processes results and responds again (inner loop continues)
// 5. Repeat step 3-4 until Claude has no more tools to use
// 6. Back to step 1 for next user input
func (a *ToolAgent) Run(ctx context.Context) error {
	// Initialize empty conversation history
	conversation := []anthropic.MessageParam{}

	if a.verbose {
		log.Println("Starting chat session with tools enabled")
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
		// This is the "agentic loop" where Claude can use multiple tools in sequence
		for {
			// Collect all tool uses and their results from this response
			var toolResults []anthropic.ContentBlockParamUnion
			var hasToolUse bool

			if a.verbose {
				log.Printf("Processing %d content blocks from Claude", len(message.Content))
			}

			// Process each content block in Claude's response
			// Claude can return multiple types: text messages and tool_use requests
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
					// Include the tool's ID so Claude knows which request this result is for
					if toolError != nil {
						// is_error=true indicates the tool execution failed
						toolResults = append(toolResults, anthropic.NewToolResultBlock(toolUse.ID, toolError.Error(), true))
					} else {
						// is_error=false indicates success
						toolResults = append(toolResults, anthropic.NewToolResultBlock(toolUse.ID, toolResult, false))
					}
				}
			}

			// If Claude didn't request any tools, we're done with this turn
			if !hasToolUse {
				break
			}

			// Send all tool results back to Claude
			// Claude will process these results and respond (possibly with more tool requests)
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
// Key Difference from BasicAgent.runInference() in chat.go:
// - BasicAgent: Simple API call with just messages
// - ToolAgent: API call includes tool definitions (the Tools parameter)
//
// By including tools in the request, Claude knows:
// - What functions it can call
// - What parameters each function accepts
// - When to use each tool based on the descriptions
func (a *ToolAgent) runInference(ctx context.Context, conversation []anthropic.MessageParam) (*anthropic.Message, error) {
	// Convert our tool definitions to Anthropic's format
	anthropicTools := []anthropic.ToolUnionParam{}
	for _, tool := range a.tools {
		anthropicTools = append(anthropicTools, anthropic.ToolUnionParam{
			OfTool: &anthropic.ToolParam{
				Name:        tool.Name,                          // e.g., "read_file"
				Description: anthropic.String(tool.Description), // Explains what the tool does
				InputSchema: tool.InputSchema,                   // JSON schema defining required parameters
			},
		})
	}

	if a.verbose {
		log.Printf("Making API call to Claude with model: %s and %d tools", anthropic.ModelClaude3_7SonnetLatest, len(anthropicTools))
	}

	// Make the API call with tools included
	message, err := a.client.Messages.New(ctx, anthropic.MessageNewParams{
		Model:     anthropic.ModelClaude3_7SonnetLatest,
		MaxTokens: int64(1024),
		Messages:  conversation,
		Tools:     anthropicTools, // This enables Claude to use the tools
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
// Each tool has:
// - Name: The identifier Claude uses to call it
// - Description: Tells Claude when and how to use this tool
// - InputSchema: JSON schema describing the parameters
// - Function: The actual Go function to execute
type ToolDefinition struct {
	Name        string                                      `json:"name"`
	Description string                                      `json:"description"`
	InputSchema anthropic.ToolInputSchemaParam              `json:"input_schema"`
	Function    func(input json.RawMessage) (string, error) // Takes JSON input, returns string result
}

// ReadFileDefinition is the tool that allows Claude to read files
// When Claude receives a question like "What's in README.md?", it can:
// 1. Decide to call read_file tool
// 2. Provide the path parameter: {"path": "README.md"}
// 3. Receive the file contents as the result
// 4. Use the contents to answer the user's question
var ReadFileDefinition = ToolDefinition{
	Name:        "read_file",
	Description: "Read the contents of a given relative file path. Use this when you want to see what's inside a file. Do not use this with directory names.",
	InputSchema: ReadFileInputSchema, // Defines that it takes a "path" parameter
	Function:    ReadFile,            // Points to the actual implementation
}

// ReadFileInput defines the parameters for the read_file tool
// The struct tags are used to generate the JSON schema:
// - json:"path" - The parameter name in JSON
// - jsonschema_description - Description shown to Claude about this parameter
type ReadFileInput struct {
	Path string `json:"path" jsonschema_description:"The relative path of a file in the working directory."`
}

// Generate the JSON schema from the struct
// This schema tells Claude exactly what format to use when calling the tool
var ReadFileInputSchema = GenerateSchema[ReadFileInput]()

// ReadFile is the actual implementation of the read_file tool
// It receives JSON input, parses it, reads the file, and returns the contents
func ReadFile(input json.RawMessage) (string, error) {
	// Parse the JSON input to extract the path parameter
	readFileInput := ReadFileInput{}
	err := json.Unmarshal(input, &readFileInput)
	if err != nil {
		panic(err)
	}

	log.Printf("Reading file: %s", readFileInput.Path)

	// Read the file from the filesystem
	content, err := os.ReadFile(readFileInput.Path)
	if err != nil {
		// If file doesn't exist or can't be read, return error
		log.Printf("Failed to read file %s: %v", readFileInput.Path, err)
		return "", err
	}

	log.Printf("Successfully read file %s (%d bytes)", readFileInput.Path, len(content))

	// Return file contents as a string
	return string(content), nil
}

// GenerateSchema generates a JSON schema from a Go struct using generics
// This is a helper function that automatically creates the schema Claude needs
// to understand what parameters a tool accepts
//
// Example: GenerateSchema[ReadFileInput]() produces a schema like:
//
//	{
//	  "type": "object",
//	  "properties": {
//	    "path": {
//	      "type": "string",
//	      "description": "The relative path of a file in the working directory."
//	    }
//	  },
//	  "required": ["path"]
//	}
func GenerateSchema[T any]() anthropic.ToolInputSchemaParam {
	reflector := jsonschema.Reflector{
		AllowAdditionalProperties: false, // Strict: only allow defined properties
		DoNotReference:            true,  // Inline all definitions instead of using $ref
	}
	var v T

	// Use reflection to generate the schema from the struct
	schema := reflector.Reflect(v)

	return anthropic.ToolInputSchemaParam{
		Properties: schema.Properties, // The field definitions from the struct
	}
}
