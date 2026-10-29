// provider.go - Model-provider abstraction used by ToolAgent
package agent

import (
	"context"
	"encoding/json"

	"github.com/fernankiwi/fonso/internal/tools"
)

// Block is one ordered piece of a model reply: either text or a tool call.
type Block struct {
	Text string
	Call *ToolCall
}

// ToolCall is a model request to run one of the registered tools.
type ToolCall struct {
	ID    string
	Name  string
	Input json.RawMessage
}

// ToolResult is the outcome of running a ToolCall, sent back to the model.
type ToolResult struct {
	CallID  string
	Output  string
	IsError bool
}

// Reply is what a model produced for one request.
type Reply struct {
	Blocks []Block
}

// Provider creates conversations against one model backend.
type Provider interface {
	// Label is the speaker name printed before model text (e.g. "Claude").
	Label() string
	// ModelName is the model id, for verbose logs.
	ModelName() string
	// NewConversation starts an empty conversation that can call toolList.
	NewConversation(toolList []tools.ToolDefinition) Conversation
}

// Conversation holds the history for one session. Providers append both sides
// of every exchange themselves, so ToolAgent only talks in Blocks.
type Conversation interface {
	// UserMessage sends user text and returns the model's reply.
	UserMessage(ctx context.Context, text string) (Reply, error)
	// ToolResults sends results for the previous reply's tool calls.
	ToolResults(ctx context.Context, results []ToolResult) (Reply, error)
}
