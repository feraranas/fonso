// chat.go - A simple command-line chat application that communicates with Claude AI
// This is the BASIC example showing fundamental conversation flow without any tools.
//
// Learning Path:
// 1. Start here (chat.go) - Learn basic request/response with Claude
// 2. Then see read.go - Learn how to add tool usage (function calling)
//
// This program creates an interactive chat session where users can send messages
// and receive responses from the Claude AI model via the Anthropic API.

package main

import (
	"context" // For managing API request contexts
	"fmt"     // For formatted output to the console
	"log"     // For logging debug information

	"github.com/anthropics/anthropic-sdk-go" // Anthropic's official Go SDK
)

func main() {
	// Use shared utility functions from common.go
	verbose := ParseFlags()
	SetupLogging(verbose)
	client := InitClient(verbose)
	getUserMessage := CreateInputReader()

	// Create and run the basic chat agent (no tools)
	agent := NewBasicAgent(&client, getUserMessage, verbose)
	err := agent.Run(context.TODO())
	if err != nil {
		fmt.Printf("Error: %s\n", err.Error())
	}
}

// NewBasicAgent creates a new basic chat agent (without tool support)
// Parameters:
//   - client: The Anthropic API client for making requests
//   - getUserMessage: A function that retrieves the next user message from input
//   - verbose: Whether to enable detailed logging
func NewBasicAgent(client *anthropic.Client, getUserMessage func() (string, bool), verbose bool) *BasicAgent {
	return &BasicAgent{
		client:         client,
		getUserMessage: getUserMessage,
		verbose:        verbose,
	}
}

// BasicAgent manages a simple chat conversation with Claude
// This is the foundational agent that demonstrates core concepts:
// - Maintaining conversation history
// - Making API calls to Claude
// - Displaying responses
//
// Key limitation: Cannot use tools (see ToolAgent in read.go for that)
type BasicAgent struct {
	client         *anthropic.Client     // API client for communicating with Claude
	getUserMessage func() (string, bool) // Function to get user input
	verbose        bool                  // Flag for enabling detailed logging
}

// Run starts the interactive chat session
// This is the main loop that demonstrates the basic conversation pattern:
// 1. Get user input
// 2. Send it to Claude
// 3. Display Claude's response
// 4. Repeat until user quits (Ctrl-C or Ctrl-D)
//
// Note: This is a simple linear conversation. See read.go for the advanced
// version with tool usage that requires a nested loop for tool execution.
func (a *BasicAgent) Run(ctx context.Context) error {
	// Initialize an empty conversation history
	// This slice stores all messages (both user and assistant) in order
	conversation := []anthropic.MessageParam{}

	if a.verbose {
		log.Println("Starting chat session")
	}
	fmt.Println("Chat with Claude (use 'ctrl-c' to quit)")

	// Main chat loop - runs until user exits
	for {
		// Print colored prompt: "\u001b[94m" = blue color, "\u001b[0m" = reset color
		fmt.Print("\u001b[94mYou\u001b[0m: ")

		// Get the next line of user input
		userInput, ok := a.getUserMessage()
		if !ok {
			// Input stream ended (user pressed Ctrl-D or piped input finished)
			if a.verbose {
				log.Println("User input ended, breaking from chat loop")
			}
			break
		}

		// Skip empty messages - just continue to next iteration
		if userInput == "" {
			if a.verbose {
				log.Println("Skipping empty message")
			}
			continue
		}

		if a.verbose {
			log.Printf("User input received: %q", userInput)
		}

		// Create a new user message and add it to the conversation history
		// NewTextBlock wraps the text in a content block structure
		userMessage := anthropic.NewUserMessage(anthropic.NewTextBlock(userInput))
		conversation = append(conversation, userMessage)

		if a.verbose {
			log.Printf("Sending message to Claude, conversation length: %d", len(conversation))
		}

		// Send the entire conversation to Claude and get a response
		message, err := a.runInference(ctx, conversation)
		if err != nil {
			if a.verbose {
				log.Printf("Error during inference: %v", err)
			}
			return err
		}

		// Add Claude's response to the conversation history
		// ToParam() converts the Message to a MessageParam for the next API call
		conversation = append(conversation, message.ToParam())

		if a.verbose {
			log.Printf("Received response from Claude with %d content blocks", len(message.Content))
		}

		// Display Claude's response to the user
		// A response can have multiple content blocks (text, images, etc.)
		for _, content := range message.Content {
			switch content.Type {
			case "text":
				// Print text response in yellow color
				fmt.Printf("\u001b[93mClaude\u001b[0m: %s\n", content.Text)
			}
		}
	}

	if a.verbose {
		log.Println("Chat session ended")
	}
	return nil
}

// runInference makes an API call to Claude with the current conversation
// This is the basic version without tool support.
//
// Parameters:
//   - ctx: Context for the API request (for timeouts, cancellation, etc.)
//   - conversation: The full conversation history (all messages so far)
//
// Returns:
//   - *anthropic.Message: Claude's response message
//   - error: Any error that occurred during the API call
func (a *BasicAgent) runInference(ctx context.Context, conversation []anthropic.MessageParam) (*anthropic.Message, error) {
	if a.verbose {
		log.Printf("Making API call to Claude with model: %s", anthropic.ModelClaude3_7SonnetLatest)
	}

	// Make the API call to Claude
	message, err := a.client.Messages.New(ctx, anthropic.MessageNewParams{
		Model:     anthropic.ModelClaude3_7SonnetLatest, // Use the latest Claude 3.7 Sonnet model
		MaxTokens: int64(1024),                          // Maximum tokens in the response (limits response length)
		Messages:  conversation,                         // Send the entire conversation for context
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
