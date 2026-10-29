// tool.go - ToolAgent implementation (supports all tool-based agents)
package agent

import (
	"context"
	"fmt"
	"log"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/fernankiwi/fonso/internal/tools"
)

// ToolAgent manages conversations with a model and executes tools when requested
//
// Comparison with BasicAgent:
// - BasicAgent: Simple linear loop (user input → Claude response → repeat)
// - ToolAgent: Nested loop with tool execution (user input → model response → tools → repeat)
//
// This agent is more advanced because it can:
// - Detect when the model wants to use a tool
// - Execute the tool function
// - Send the results back to the model
// - Continue the conversation with the tool results in a loop
//
// The model backend is a Provider (Anthropic by default, or the OpenAI
// Responses API); the loop below is identical for both.
type ToolAgent struct {
	provider       Provider               // Model backend
	getUserMessage func() (string, bool)  // Function to get user input
	tools          []tools.ToolDefinition // Available tools the model can use
	verbose        bool                   // Enable detailed logging
	welcomeMsg     string                 // Welcome message to display
}

// NewToolAgent creates a new chat agent with tool usage capabilities backed by
// Claude. It is a thin wrapper over NewToolAgentWithProvider so existing
// commands keep working unchanged.
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
	return NewToolAgentWithProvider(NewAnthropicProvider(client, verbose), getUserMessage, toolList, verbose, welcomeMsg)
}

// NewToolAgentWithProvider creates a ToolAgent that talks to any Provider.
func NewToolAgentWithProvider(
	provider Provider,
	getUserMessage func() (string, bool),
	toolList []tools.ToolDefinition,
	verbose bool,
	welcomeMsg string,
) *ToolAgent {
	if welcomeMsg == "" {
		welcomeMsg = "Chat with " + provider.Label() + " (use 'ctrl-c' to quit)"
	}
	return &ToolAgent{
		provider:       provider,
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
// 2. The model responds (may include tool calls)
// 3. Agent executes tools and sends results back (inner loop)
// 4. The model processes results and responds again (inner loop continues)
// 5. Repeat step 3-4 until the model has no more tools to use
// 6. Back to step 1 for next user input
func (a *ToolAgent) Run(ctx context.Context) error {
	conversation := a.provider.NewConversation(a.tools)
	label := a.provider.Label()

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
			log.Printf("Sending message to %s (model %s)", label, a.provider.ModelName())
		}

		// Get the model's initial response
		reply, err := conversation.UserMessage(ctx, userInput)
		if err != nil {
			if a.verbose {
				log.Printf("Error during inference: %v", err)
			}
			return err
		}

		// Inner loop: Keep processing until the model stops using tools
		for {
			var toolResults []ToolResult

			if a.verbose {
				log.Printf("Processing %d content blocks from %s", len(reply.Blocks), label)
			}

			for _, block := range reply.Blocks {
				if block.Call == nil {
					// Regular text response - display to user
					fmt.Printf("\u001b[93m%s\u001b[0m: %s\n", label, block.Text)
					continue
				}

				toolUse := block.Call
				if a.verbose {
					log.Printf("Tool use detected: %s with input: %s", toolUse.Name, string(toolUse.Input))
				}
				// Display the tool call in cyan
				fmt.Printf("\u001b[96mtool\u001b[0m: %s(%s)\n", toolUse.Name, string(toolUse.Input))

				// Find and execute the requested tool
				var toolResult string
				var toolError error
				var toolFound bool

				for _, tool := range a.tools {
					if tool.Name == toolUse.Name {
						if a.verbose {
							log.Printf("Executing tool: %s", tool.Name)
						}
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

				// Include the tool's ID so the model knows which request this result is for
				if toolError != nil {
					toolResults = append(toolResults, ToolResult{CallID: toolUse.ID, Output: toolError.Error(), IsError: true})
				} else {
					toolResults = append(toolResults, ToolResult{CallID: toolUse.ID, Output: toolResult})
				}
			}

			// If the model didn't request any tools, we're done with this turn
			if len(toolResults) == 0 {
				break
			}

			// Send all tool results back; the model will respond (possibly with more tool requests)
			if a.verbose {
				log.Printf("Sending %d tool results back to %s", len(toolResults), label)
			}
			reply, err = conversation.ToolResults(ctx, toolResults)
			if err != nil {
				if a.verbose {
					log.Printf("Error during followup inference: %v", err)
				}
				return err
			}

			if a.verbose {
				log.Printf("Received followup response with %d content blocks", len(reply.Blocks))
			}
		}
	}

	if a.verbose {
		log.Println("Chat session ended")
	}
	return nil
}
