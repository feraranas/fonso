// anthropic.go - Anthropic Messages API provider (the default backend)
package agent

import (
	"context"
	"log"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/fernankiwi/fonso/internal/tools"
)

// AnthropicProvider talks to Claude through the Messages API.
type AnthropicProvider struct {
	client  *anthropic.Client
	verbose bool
}

// NewAnthropicProvider wraps an existing Anthropic client.
func NewAnthropicProvider(client *anthropic.Client, verbose bool) *AnthropicProvider {
	return &AnthropicProvider{client: client, verbose: verbose}
}

func (p *AnthropicProvider) Label() string     { return "Claude" }
func (p *AnthropicProvider) ModelName() string { return string(anthropic.ModelClaude3_7SonnetLatest) }

func (p *AnthropicProvider) NewConversation(toolList []tools.ToolDefinition) Conversation {
	anthropicTools := []anthropic.ToolUnionParam{}
	for _, tool := range toolList {
		anthropicTools = append(anthropicTools, anthropic.ToolUnionParam{
			OfTool: &anthropic.ToolParam{
				Name:        tool.Name,
				Description: anthropic.String(tool.Description),
				InputSchema: tool.InputSchema,
			},
		})
	}
	return &anthropicConversation{p: p, tools: anthropicTools}
}

type anthropicConversation struct {
	p       *AnthropicProvider
	tools   []anthropic.ToolUnionParam
	history []anthropic.MessageParam
}

func (c *anthropicConversation) UserMessage(ctx context.Context, text string) (Reply, error) {
	c.history = append(c.history, anthropic.NewUserMessage(anthropic.NewTextBlock(text)))
	return c.infer(ctx)
}

func (c *anthropicConversation) ToolResults(ctx context.Context, results []ToolResult) (Reply, error) {
	blocks := make([]anthropic.ContentBlockParamUnion, 0, len(results))
	for _, r := range results {
		blocks = append(blocks, anthropic.NewToolResultBlock(r.CallID, r.Output, r.IsError))
	}
	c.history = append(c.history, anthropic.NewUserMessage(blocks...))
	return c.infer(ctx)
}

func (c *anthropicConversation) infer(ctx context.Context) (Reply, error) {
	if c.p.verbose {
		log.Printf("Making API call to Claude with model: %s and %d tools", anthropic.ModelClaude3_7SonnetLatest, len(c.tools))
	}
	message, err := c.p.client.Messages.New(ctx, anthropic.MessageNewParams{
		Model:     anthropic.ModelClaude3_7SonnetLatest,
		MaxTokens: int64(1024),
		Messages:  c.history,
		Tools:     c.tools,
	})
	if c.p.verbose {
		if err != nil {
			log.Printf("API call failed: %v", err)
		} else {
			log.Printf("API call successful, response received")
		}
	}
	if err != nil {
		return Reply{}, err
	}
	c.history = append(c.history, message.ToParam())

	var reply Reply
	for _, content := range message.Content {
		switch content.Type {
		case "text":
			reply.Blocks = append(reply.Blocks, Block{Text: content.Text})
		case "tool_use":
			tu := content.AsToolUse()
			reply.Blocks = append(reply.Blocks, Block{Call: &ToolCall{ID: tu.ID, Name: tu.Name, Input: tu.Input}})
		}
	}
	return reply, nil
}
