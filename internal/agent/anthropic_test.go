package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
	"github.com/fernankiwi/fonso/internal/tools"
)

// captureStdout runs fn and returns what it printed to stdout.
func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	old := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout = w
	done := make(chan string)
	go func() { b, _ := io.ReadAll(r); done <- string(b) }()
	fn()
	w.Close()
	os.Stdout = old
	return <-done
}

// scriptedInput feeds fixed lines to the agent then signals EOF.
func scriptedInput(lines ...string) func() (string, bool) {
	i := 0
	return func() (string, bool) {
		if i >= len(lines) {
			return "", false
		}
		i++
		return lines[i-1], true
	}
}

// TestAnthropicDefaultPathUnchanged pins the exact wire behaviour of the
// default Claude path through the public NewToolAgent constructor: model,
// max_tokens, tool schema, tool_result shape, and printed transcript. The same
// test file passes against the pre-change tree (see PR description).
func TestAnthropicDefaultPathUnchanged(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "note.txt")
	if err := os.WriteFile(target, []byte("hello from note"), 0o600); err != nil {
		t.Fatal(err)
	}

	var bodies []map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/messages" {
			t.Errorf("unexpected path %s", r.URL.Path)
		}
		if r.Header.Get("X-Api-Key") != "test-key" {
			t.Errorf("x-api-key header missing")
		}
		raw, _ := io.ReadAll(r.Body)
		var body map[string]any
		if err := json.Unmarshal(raw, &body); err != nil {
			t.Fatal(err)
		}
		bodies = append(bodies, body)
		w.Header().Set("Content-Type", "application/json")
		if len(bodies) == 1 {
			fmt.Fprintf(w, `{"id":"msg_1","type":"message","role":"assistant","model":"claude-3-7-sonnet-latest","stop_reason":"tool_use","usage":{"input_tokens":1,"output_tokens":1},
"content":[{"type":"text","text":"Let me look."},{"type":"tool_use","id":"toolu_1","name":"read_file","input":{"path":%q}}]}`, target)
			return
		}
		fmt.Fprint(w, `{"id":"msg_2","type":"message","role":"assistant","model":"claude-3-7-sonnet-latest","stop_reason":"end_turn","usage":{"input_tokens":1,"output_tokens":1},
"content":[{"type":"text","text":"The note says hello."}]}`)
	}))
	defer srv.Close()

	client := anthropic.NewClient(option.WithBaseURL(srv.URL), option.WithAPIKey("test-key"), option.WithMaxRetries(0))
	a := NewToolAgent(&client, scriptedInput("what is in the note?"), []tools.ToolDefinition{tools.ReadFileDefinition}, false, "")
	out := captureStdout(t, func() {
		if err := a.Run(context.Background()); err != nil {
			t.Fatal(err)
		}
	})

	if len(bodies) != 2 {
		t.Fatalf("want 2 requests, got %d", len(bodies))
	}
	for i, b := range bodies {
		if b["model"] != "claude-3-7-sonnet-latest" {
			t.Errorf("req %d model = %v", i, b["model"])
		}
		if b["max_tokens"] != float64(1024) {
			t.Errorf("req %d max_tokens = %v", i, b["max_tokens"])
		}
		ts, _ := b["tools"].([]any)
		if len(ts) != 1 || ts[0].(map[string]any)["name"] != "read_file" {
			t.Errorf("req %d tools = %v", i, b["tools"])
		}
	}
	msgs := bodies[1]["messages"].([]any)
	if len(msgs) != 3 {
		t.Fatalf("second request should carry user, assistant, tool_result; got %d messages", len(msgs))
	}
	last := msgs[2].(map[string]any)
	if last["role"] != "user" {
		t.Errorf("tool_result role = %v", last["role"])
	}
	tr := last["content"].([]any)[0].(map[string]any)
	if tr["type"] != "tool_result" || tr["tool_use_id"] != "toolu_1" || tr["content"].([]any)[0].(map[string]any)["text"] != "hello from note" {
		t.Errorf("unexpected tool_result: %v", tr)
	}
	for _, want := range []string{"Chat with Claude", "\x1b[93mClaude\x1b[0m: Let me look.", "\x1b[96mtool\x1b[0m: read_file(", "\x1b[92mresult\x1b[0m: hello from note", "\x1b[93mClaude\x1b[0m: The note says hello."} {
		if !strings.Contains(out, want) {
			t.Errorf("transcript missing %q\n%s", want, out)
		}
	}
}

func TestAnthropicModelAndMaxTokensOverride(t *testing.T) {
	var got map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&got)
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"id":"m","type":"message","role":"assistant","model":"x","stop_reason":"end_turn","usage":{"input_tokens":1,"output_tokens":1},"content":[{"type":"text","text":"ok"}]}`)
	}))
	defer srv.Close()
	client := anthropic.NewClient(option.WithBaseURL(srv.URL), option.WithAPIKey("k"), option.WithMaxRetries(0))

	p := NewAnthropicProvider(&client, false).WithModel("claude-test-model").WithMaxTokens(4096)
	if _, err := p.NewConversation(nil).UserMessage(context.Background(), "hi"); err != nil {
		t.Fatal(err)
	}
	if got["model"] != "claude-test-model" || got["max_tokens"] != float64(4096) || p.ModelName() != "claude-test-model" {
		t.Errorf("override not applied: %v", got)
	}

	// Zero values keep the defaults.
	p = NewAnthropicProvider(&client, false).WithModel("").WithMaxTokens(0)
	if _, err := p.NewConversation(nil).UserMessage(context.Background(), "hi"); err != nil {
		t.Fatal(err)
	}
	if got["model"] != "claude-3-7-sonnet-latest" || got["max_tokens"] != float64(1024) {
		t.Errorf("defaults changed: %v", got)
	}
}
