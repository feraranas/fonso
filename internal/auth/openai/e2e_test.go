package openai

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/fernankiwi/fonso/internal/agent"
	"github.com/fernankiwi/fonso/internal/tools"
)

// End to end against stubs only: log in, let the token expire, then run the
// agent loop. The access token must be refreshed *before* the /responses
// request, and that request must carry the refreshed Bearer (and, in opencode
// mode, ChatGPT-Account-Id).
func TestLoginExpireRefreshThenToolLoop(t *testing.T) {
	for _, mode := range []Mode{ModeOfficial, ModeOpenCode} {
		t.Run(string(mode), func(t *testing.T) {
			issuer := newStub(t)
			target := filepath.Join(t.TempDir(), "note.txt")
			if err := os.WriteFile(target, []byte("hello from note"), 0o600); err != nil {
				t.Fatal(err)
			}

			var order []string
			var gotAuth, gotAccount []string
			var inputs [][]any
			codex := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				issuer.mu.Lock()
				order = append(order, fmt.Sprintf("responses(refreshes=%d)", issuer.refreshCount))
				issuer.mu.Unlock()
				gotAuth = append(gotAuth, r.Header.Get("Authorization"))
				gotAccount = append(gotAccount, r.Header.Get("ChatGPT-Account-Id"))
				var body map[string]any
				raw, _ := io.ReadAll(r.Body)
				_ = json.Unmarshal(raw, &body)
				inputs = append(inputs, body["input"].([]any))
				w.Header().Set("Content-Type", "text/event-stream")
				if len(inputs) == 1 {
					args, _ := json.Marshal(map[string]string{"path": target})
					call, _ := json.Marshal(map[string]any{"type": "function_call", "call_id": "call_1", "name": "read_file", "arguments": string(args)})
					fmt.Fprintf(w, "data: {\"type\":\"response.output_item.done\",\"item\":%s}\n\ndata: {\"type\":\"response.completed\",\"response\":{}}\n\n", call)
					return
				}
				fmt.Fprint(w, "data: {\"type\":\"response.output_item.done\",\"item\":{\"type\":\"message\",\"role\":\"assistant\",\"content\":[{\"type\":\"output_text\",\"text\":\"done\"}]}}\n\ndata: {\"type\":\"response.completed\",\"response\":{}}\n\n")
			}))
			defer codex.Close()

			var out strings.Builder
			cfg := testConfig(t, issuer, mode, &out, nil)
			cfg.OpenBrowser = browserFollowing(t, nil)
			cfg.InferenceURL = codex.URL + "/responses"
			if _, err := Login(t.Context(), cfg); err != nil {
				t.Fatal(err)
			}

			// Time passes: the stored access token is now expired.
			rec, err := LoadRecord(cfg.StorePath)
			if err != nil {
				t.Fatal(err)
			}
			rec.ExpiresAt = time.Now().Add(-time.Minute)
			if err := SaveRecord(cfg.StorePath, rec); err != nil {
				t.Fatal(err)
			}

			src := NewSource(cfg)
			provider := agent.NewOpenAIProvider(agent.OpenAIConfig{URL: cfg.InferenceEndpoint(), Model: "gpt-test", Tokens: src})
			conv := provider.NewConversation([]tools.ToolDefinition{tools.ReadFileDefinition})

			reply, err := conv.UserMessage(t.Context(), "read the note")
			if err != nil {
				t.Fatal(err)
			}
			call := reply.Blocks[0].Call
			if call == nil || call.Name != "read_file" {
				t.Fatalf("reply = %+v", reply)
			}
			result, err := tools.ReadFile(call.Input)
			if err != nil {
				t.Fatal(err)
			}
			reply, err = conv.ToolResults(t.Context(), []agent.ToolResult{{CallID: call.ID, Output: result}})
			if err != nil || reply.Blocks[0].Text != "done" {
				t.Fatalf("final reply = %+v, %v", reply, err)
			}

			if issuer.refreshCount != 1 {
				t.Errorf("refresh count = %d, want exactly 1", issuer.refreshCount)
			}
			if order[0] != "responses(refreshes=1)" {
				t.Errorf("request order = %v: refresh must happen before the first /responses call", order)
			}
			wantTok := "Bearer access-2" // access-1 came from login, access-2 from the refresh
			for i, a := range gotAuth {
				if a != wantTok {
					t.Errorf("request %d Authorization = %q, want %q", i, a, wantTok)
				}
			}
			wantAccount := ""
			if mode == ModeOpenCode {
				wantAccount = "acct-42"
			}
			for i, a := range gotAccount {
				if a != wantAccount {
					t.Errorf("request %d ChatGPT-Account-Id = %q, want %q", i, a, wantAccount)
				}
			}
			last := inputs[1][len(inputs[1])-1].(map[string]any)
			if last["type"] != "function_call_output" || last["output"] != "hello from note" {
				t.Errorf("function_call_output = %v", last)
			}
		})
	}
}
