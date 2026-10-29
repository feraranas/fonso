// openai.go - OpenAI Responses API provider for ChatGPT-plan OAuth tokens
package agent

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"

	"github.com/fernankiwi/fonso/internal/tools"
)

// DefaultOpenAIModel is used when --model is not given. Whether a given model
// is allowed on a ChatGPT plan is decided server-side; see the PR notes.
const DefaultOpenAIModel = "gpt-5.5"

const defaultInstructions = "You are Fonso, a coding agent running in the user's terminal. " +
	"Use the provided tools to inspect and change files and to run commands when that helps. " +
	"Be concise."

// TokenSource supplies OAuth credentials. Refresh is called once after a 401
// with the access token the server rejected.
type TokenSource interface {
	Token(ctx context.Context) (access, accountID string, err error)
	Refresh(ctx context.Context, stale string) (access, accountID string, err error)
}

// OpenAIConfig configures an OpenAIProvider.
type OpenAIConfig struct {
	URL          string // full Responses endpoint, e.g. https://api.openai.com/v1/responses
	Model        string
	Instructions string // optional; a default is used when empty
	Tokens       TokenSource
	HTTPClient   *http.Client // optional
	Verbose      bool
}

// OpenAIProvider talks to the Responses API over HTTP+SSE with store=false:
// the full history is resent on every request, as ChatGPT-plan routes require.
type OpenAIProvider struct{ cfg OpenAIConfig }

// NewOpenAIProvider builds a provider; Model and URL default sensibly.
func NewOpenAIProvider(cfg OpenAIConfig) *OpenAIProvider {
	if cfg.Model == "" {
		cfg.Model = DefaultOpenAIModel
	}
	if cfg.Instructions == "" {
		cfg.Instructions = defaultInstructions
	}
	if cfg.HTTPClient == nil {
		cfg.HTTPClient = &http.Client{} // streaming: no overall timeout, ctx governs
	}
	return &OpenAIProvider{cfg: cfg}
}

func (p *OpenAIProvider) Label() string     { return "ChatGPT" }
func (p *OpenAIProvider) ModelName() string { return p.cfg.Model }

type responsesTool struct {
	Type        string          `json:"type"`
	Name        string          `json:"name"`
	Description string          `json:"description,omitempty"`
	Parameters  json.RawMessage `json:"parameters"`
	Strict      bool            `json:"strict"`
}

// toResponsesTool maps a fonso tool to a Responses function tool.
func toResponsesTool(t tools.ToolDefinition) (responsesTool, error) {
	raw, err := json.Marshal(t.InputSchema)
	if err != nil {
		return responsesTool{}, err
	}
	schema := map[string]any{}
	if err := json.Unmarshal(raw, &schema); err != nil {
		return responsesTool{}, err
	}
	schema["type"] = "object"
	if schema["properties"] == nil {
		schema["properties"] = map[string]any{}
	}
	params, err := json.Marshal(schema)
	if err != nil {
		return responsesTool{}, err
	}
	return responsesTool{Type: "function", Name: t.Name, Description: t.Description, Parameters: params}, nil
}

func (p *OpenAIProvider) NewConversation(toolList []tools.ToolDefinition) Conversation {
	c := &openAIConversation{p: p}
	for _, t := range toolList {
		rt, err := toResponsesTool(t)
		if err != nil {
			c.toolErr = fmt.Errorf("tool %s: %w", t.Name, err)
			break
		}
		c.tools = append(c.tools, rt)
	}
	return c
}

type openAIConversation struct {
	p       *OpenAIProvider
	tools   []responsesTool
	toolErr error
	input   []json.RawMessage // history, replayed in full each request
}

func mustJSON(v any) json.RawMessage {
	b, err := json.Marshal(v)
	if err != nil {
		panic(err) // only called with plain maps/strings
	}
	return b
}

func (c *openAIConversation) UserMessage(ctx context.Context, text string) (Reply, error) {
	c.input = append(c.input, mustJSON(map[string]any{"type": "message", "role": "user", "content": []map[string]string{{"type": "input_text", "text": text}}}))
	return c.infer(ctx)
}

func (c *openAIConversation) ToolResults(ctx context.Context, results []ToolResult) (Reply, error) {
	for _, r := range results {
		c.input = append(c.input, mustJSON(map[string]any{"type": "function_call_output", "call_id": r.CallID, "output": r.Output}))
	}
	return c.infer(ctx)
}

type responsesRequest struct {
	Model             string            `json:"model"`
	Instructions      string            `json:"instructions"`
	Input             []json.RawMessage `json:"input"`
	Tools             []responsesTool   `json:"tools,omitempty"`
	ToolChoice        string            `json:"tool_choice,omitempty"`
	ParallelToolCalls bool              `json:"parallel_tool_calls"`
	Store             bool              `json:"store"`
	Stream            bool              `json:"stream"`
	Include           []string          `json:"include"`
}

func (c *openAIConversation) infer(ctx context.Context) (Reply, error) {
	if c.toolErr != nil {
		return Reply{}, c.toolErr
	}
	// Deliberately no max_output_tokens: ChatGPT-plan routes reject it.
	body, err := json.Marshal(responsesRequest{
		Model:             c.p.cfg.Model,
		Instructions:      c.p.cfg.Instructions,
		Input:             c.input,
		Tools:             c.tools,
		ToolChoice:        toolChoice(c.tools),
		ParallelToolCalls: true,
		Store:             false,
		Stream:            true,
		Include:           []string{"reasoning.encrypted_content"},
	})
	if err != nil {
		return Reply{}, err
	}
	if c.p.cfg.Verbose {
		log.Printf("Making Responses API call with model: %s, %d tools, %d input items", c.p.cfg.Model, len(c.tools), len(c.input))
	}

	access, account, err := c.p.cfg.Tokens.Token(ctx)
	if err != nil {
		return Reply{}, err
	}
	resp, err := c.post(ctx, body, access, account)
	if err != nil {
		return Reply{}, err
	}
	if resp.StatusCode == http.StatusUnauthorized {
		resp.Body.Close()
		access, account, err = c.p.cfg.Tokens.Refresh(ctx, access)
		if err != nil {
			return Reply{}, err
		}
		if resp, err = c.post(ctx, body, access, account); err != nil {
			return Reply{}, err
		}
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return Reply{}, httpError(resp)
	}

	reply, items, err := parseResponsesStream(resp.Body)
	if err != nil {
		return Reply{}, err
	}
	c.input = append(c.input, items...)
	if c.p.cfg.Verbose {
		log.Printf("API call successful, %d blocks received", len(reply.Blocks))
	}
	return reply, nil
}

func toolChoice(ts []responsesTool) string {
	if len(ts) == 0 {
		return ""
	}
	return "auto"
}

func (c *openAIConversation) post(ctx context.Context, body []byte, access, account string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.p.cfg.URL, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+access)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "text/event-stream")
	req.Header.Set("User-Agent", "fonso")
	if account != "" {
		req.Header.Set("ChatGPT-Account-Id", account)
	}
	return c.p.cfg.HTTPClient.Do(req)
}

// httpError surfaces status plus the server's error code/message. Bodies are
// diagnostic text from OpenAI and never contain our credentials.
func httpError(resp *http.Response) error {
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 8<<10))
	var e struct {
		Detail json.RawMessage `json:"detail"`
		Error  struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	_ = json.Unmarshal(raw, &e)
	msg := e.Error.Message
	if msg == "" && len(e.Detail) > 0 {
		msg = strings.Trim(string(e.Detail), `"`)
	}
	return responsesFailure(fmt.Sprintf("Responses API returned HTTP %d", resp.StatusCode), e.Error.Code, msg)
}

func responsesFailure(prefix, code, msg string) error {
	out := prefix
	if code != "" {
		out += " (" + code + ")"
	}
	if msg != "" {
		out += ": " + msg
	}
	switch code {
	case "subscription_sharing_usage_limit_exceeded":
		out += " — your ChatGPT plan limit for this app is used up; see https://chatgpt.com/settings/usage"
	case "subscription_sharing_user_not_eligible":
		out += " — ChatGPT plan usage is not available for this account/workspace"
	}
	return errors.New(out)
}

type sseEvent struct {
	Type     string          `json:"type"`
	Item     json.RawMessage `json:"item"`
	Response struct {
		Error *struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
		IncompleteDetails *struct {
			Reason string `json:"reason"`
		} `json:"incomplete_details"`
	} `json:"response"`
	Code    string `json:"code"`
	Message string `json:"message"`
}

type outputItem struct {
	Type      string `json:"type"`
	Name      string `json:"name"`
	CallID    string `json:"call_id"`
	Arguments string `json:"arguments"`
	Content   []struct {
		Type    string `json:"type"`
		Text    string `json:"text"`
		Refusal string `json:"refusal"`
	} `json:"content"`
}

// parseResponsesStream reads SSE events through a terminal event. Output
// items are collected from response.output_item.done, in order, and returned
// raw so they can be replayed verbatim as history.
func parseResponsesStream(r io.Reader) (Reply, []json.RawMessage, error) {
	var (
		reply     Reply
		items     []json.RawMessage
		completed bool
		data      strings.Builder
	)
	handle := func(payload string) error {
		if payload == "" || payload == "[DONE]" {
			return nil
		}
		var ev sseEvent
		if err := json.Unmarshal([]byte(payload), &ev); err != nil {
			return fmt.Errorf("bad stream event: %w", err)
		}
		switch ev.Type {
		case "response.output_item.done":
			var it outputItem
			if err := json.Unmarshal(ev.Item, &it); err != nil {
				return err
			}
			items = append(items, append(json.RawMessage(nil), ev.Item...))
			switch it.Type {
			case "message":
				for _, part := range it.Content {
					switch part.Type {
					case "output_text":
						reply.Blocks = append(reply.Blocks, Block{Text: part.Text})
					case "refusal":
						reply.Blocks = append(reply.Blocks, Block{Text: part.Refusal})
					}
				}
			case "function_call":
				args := json.RawMessage(it.Arguments)
				if strings.TrimSpace(it.Arguments) == "" {
					args = json.RawMessage(`{}`)
				}
				reply.Blocks = append(reply.Blocks, Block{Call: &ToolCall{ID: it.CallID, Name: it.Name, Input: args}})
			}
		case "response.completed":
			completed = true
		case "response.failed":
			if e := ev.Response.Error; e != nil {
				return responsesFailure("response failed", e.Code, e.Message)
			}
			return errors.New("response failed")
		case "response.incomplete":
			reason := "unknown"
			if d := ev.Response.IncompleteDetails; d != nil && d.Reason != "" {
				reason = d.Reason
			}
			return fmt.Errorf("response incomplete (%s)", reason)
		case "error":
			return responsesFailure("stream error", ev.Code, ev.Message)
		}
		return nil
	}

	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 0, 64<<10), 16<<20)
	for sc.Scan() {
		line := sc.Text()
		switch {
		case line == "":
			if err := handle(data.String()); err != nil {
				return Reply{}, nil, err
			}
			data.Reset()
		case strings.HasPrefix(line, "data:"):
			if data.Len() > 0 {
				data.WriteByte('\n')
			}
			data.WriteString(strings.TrimPrefix(strings.TrimPrefix(line, "data:"), " "))
		}
	}
	if err := sc.Err(); err != nil {
		return Reply{}, nil, fmt.Errorf("reading stream: %w", err)
	}
	if err := handle(data.String()); err != nil {
		return Reply{}, nil, err
	}
	if !completed {
		return Reply{}, nil, errors.New("stream ended without response.completed")
	}
	return reply, items, nil
}
