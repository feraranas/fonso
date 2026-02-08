// tool.go - ToolAgent implementation (supports all tool-based agents)
package agent

import (
	"context"
	"fmt"
	"log"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/fernankiwi/fonso/internal/tools"
)

// ToolAgent manages conversations with Claude and executes tools when requested
//
// Comparison with BasicAgent:
// - BasicAgent: Simple linear loop (user input → Claude response → repeat)
// - ToolAgent: Nested loop with tool execution (user input → Claude response → tools → repeat)
//
// This agent is more advanced because it can:
// - Detect when Claude wants to use a tool
// - Execute the tool function
// - Send the results back to Claude
// - Continue the conversation with the tool results in a loop
type ToolAgent struct {
	client         *anthropic.Client           // API client for Claude
	getUserMessage func() (string, bool)       // Function to get user input
	tools          []tools.ToolDefinition      // Available tools Claude can use
	verbose        bool                        // Enable detailed logging
	welcomeMsg     string                      // Welcome message to display
}

// NewToolAgent creates a new chat agent with tool usage capabilities
// This extends the BasicAgent concept by adding tool support.
//
// Parameters:
//   - client: The Anthropic API client
//   - getUserMessage: Function to get user input (same as BasicAgent)
//   - tools: List of tools that Claude can use (e.g., read_file)
//   - verbose: Enable detailed logging
//   - welcomeMsg: Optional welcome message (empty string uses default)
func NewToolAgent(
	client *anthropic.Client,
	getUserMessage func() (string, bool),
	toolList []tools.ToolDefinition,
	verbose bool,
	welcomeMsg string,
) *ToolAgent {
	if welcomeMsg == "" {
		welcomeMsg = "Chat with Claude (use 'ctrl-c' to quit)"
	}
	return &ToolAgent{
		client:         client,
		getUserMessage: getUserMessage,
		tools:          toolList,
		verbose:        verbose,
		welcomeMsg:     welcomeMsg,
	}
}

// Run starts the interactive chat session with tool support
//
// Key Difference from BasicAgent.Run():
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
	fmt.Println(a.welcomeMsg)

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
// Key Difference from BasicAgent.runInference():
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
