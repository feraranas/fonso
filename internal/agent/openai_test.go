package agent

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/fernankiwi/fonso/internal/tools"
)

type fakeTokens struct {
	access, account string
	refreshed       atomic.Int32
	staleSeen       string
	err             error
}

func (f *fakeTokens) Token(context.Context) (string, string, error) {
	return f.access, f.account, f.err
}
func (f *fakeTokens) Refresh(_ context.Context, stale string) (string, string, error) {
	f.refreshed.Add(1)
	f.staleSeen = stale
	f.access = "fresh-token"
	return f.access, f.account, nil
}

func sse(w http.ResponseWriter, events ...string) {
	w.Header().Set("Content-Type", "text/event-stream")
	for _, e := range events {
		var t struct {
			Type string `json:"type"`
		}
		_ = json.Unmarshal([]byte(e), &t)
		fmt.Fprintf(w, "event: %s\ndata: %s\n\n", t.Type, e)
	}
}

func itemDone(item string) string {
	return `{"type":"response.output_item.done","output_index":0,"item":` + item + `}`
}

const completed = `{"type":"response.completed","response":{"id":"resp_1","status":"completed"}}`

type recorded struct {
	header http.Header
	body   map[string]any
}

func TestOpenAIToolLoopTwoSteps(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "note.txt")
	if err := os.WriteFile(target, []byte("hello from note"), 0o600); err != nil {
		t.Fatal(err)
	}

	var reqs []recorded
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/backend/responses" {
			t.Errorf("unexpected %s %s", r.Method, r.URL.Path)
		}
		raw, _ := io.ReadAll(r.Body)
		var body map[string]any
		if err := json.Unmarshal(raw, &body); err != nil {
			t.Fatal(err)
		}
		reqs = append(reqs, recorded{r.Header.Clone(), body})
		if len(reqs) == 1 {
			args, _ := json.Marshal(map[string]string{"path": target})
			call, _ := json.Marshal(map[string]any{"type": "function_call", "id": "fc_1", "call_id": "call_1", "name": "read_file", "arguments": string(args)})
			sse(w,
				`{"type":"response.created","response":{"id":"resp_1"}}`,
				itemDone(`{"type":"reasoning","id":"rs_1","encrypted_content":"enc","summary":[]}`),
				`{"type":"response.output_text.delta","delta":"ignored"}`,
				itemDone(string(call)),
				completed)
			return
		}
		sse(w,
			itemDone(`{"type":"message","id":"msg_1","role":"assistant","content":[{"type":"output_text","text":"The note says hello."}]}`),
			completed)
	}))
	defer srv.Close()

	tokens := &fakeTokens{access: "tok-1", account: "acct-9"}
	p := NewOpenAIProvider(OpenAIConfig{URL: srv.URL + "/backend/responses", Model: "gpt-test", Tokens: tokens})
	a := NewToolAgentWithProvider(p, scriptedInput("what is in the note?"), []tools.ToolDefinition{tools.ReadFileDefinition}, false, "")
	out := captureStdout(t, func() {
		if err := a.Run(context.Background()); err != nil {
			t.Fatal(err)
		}
	})

	if len(reqs) != 2 {
		t.Fatalf("want 2 requests, got %d", len(reqs))
	}
	for i, r := range reqs {
		if got := r.header.Get("Authorization"); got != "Bearer tok-1" {
			t.Errorf("req %d Authorization = %q", i, got)
		}
		if got := r.header.Get("ChatGPT-Account-Id"); got != "acct-9" {
			t.Errorf("req %d ChatGPT-Account-Id = %q", i, got)
		}
		b := r.body
		if b["model"] != "gpt-test" || b["store"] != false || b["stream"] != true {
			t.Errorf("req %d model/store/stream = %v/%v/%v", i, b["model"], b["store"], b["stream"])
		}
		if s, _ := b["instructions"].(string); s == "" {
			t.Errorf("req %d has no instructions", i)
		}
		if _, ok := b["max_output_tokens"]; ok {
			t.Errorf("req %d must not send max_output_tokens", i)
		}
		ts := b["tools"].([]any)
		tool := ts[0].(map[string]any)
		params := tool["parameters"].(map[string]any)
		if tool["type"] != "function" || tool["name"] != "read_file" || params["type"] != "object" || params["properties"].(map[string]any)["path"] == nil {
			t.Errorf("req %d tool mapping = %v", i, tool)
		}
	}

	in1 := reqs[0].body["input"].([]any)
	if len(in1) != 1 || in1[0].(map[string]any)["role"] != "user" {
		t.Errorf("first input = %v", in1)
	}
	in2 := reqs[1].body["input"].([]any)
	if len(in2) != 4 {
		t.Fatalf("second request input has %d items, want user, reasoning, function_call, function_call_output: %v", len(in2), in2)
	}
	wantTypes := []string{"message", "reasoning", "function_call", "function_call_output"}
	for i, w := range wantTypes {
		if got := in2[i].(map[string]any)["type"]; got != w {
			t.Errorf("input[%d].type = %v, want %s", i, got, w)
		}
	}
	fco := in2[3].(map[string]any)
	if fco["call_id"] != "call_1" || fco["output"] != "hello from note" {
		t.Errorf("function_call_output = %v", fco)
	}
	if in2[1].(map[string]any)["encrypted_content"] != "enc" {
		t.Errorf("reasoning item must be replayed verbatim: %v", in2[1])
	}
	if !strings.Contains(out, "Chat with ChatGPT") || !strings.Contains(out, "\x1b[96mtool\x1b[0m: read_file(") ||
		!strings.Contains(out, "\x1b[92mresult\x1b[0m: hello from note") || !strings.Contains(out, "\x1b[93mChatGPT\x1b[0m: The note says hello.") {
		t.Errorf("transcript:\n%s", out)
	}
	if strings.Contains(out, "tok-1") {
		t.Errorf("token leaked into output")
	}
}

func TestOpenAIToolErrorBecomesFunctionCallOutput(t *testing.T) {
	var second []any
	n := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n++
		if n == 1 {
			sse(w, itemDone(`{"type":"function_call","call_id":"c","name":"nope","arguments":""}`), completed)
			return
		}
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		second = body["input"].([]any)
		sse(w, itemDone(`{"type":"message","role":"assistant","content":[{"type":"output_text","text":"ok"}]}`), completed)
	}))
	defer srv.Close()
	p := NewOpenAIProvider(OpenAIConfig{URL: srv.URL, Tokens: &fakeTokens{access: "t"}})
	a := NewToolAgentWithProvider(p, scriptedInput("go"), nil, false, "")
	captureStdout(t, func() {
		if err := a.Run(context.Background()); err != nil {
			t.Fatal(err)
		}
	})
	last := second[len(second)-1].(map[string]any)
	if last["type"] != "function_call_output" || !strings.Contains(last["output"].(string), "not found") {
		t.Errorf("last input = %v", last)
	}
}

func TestOpenAIRefreshesOnceOn401(t *testing.T) {
	var auths []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auths = append(auths, r.Header.Get("Authorization"))
		if len(auths) == 1 {
			w.WriteHeader(http.StatusUnauthorized)
			fmt.Fprint(w, `{"detail":"nope"}`)
			return
		}
		sse(w, itemDone(`{"type":"message","role":"assistant","content":[{"type":"output_text","text":"hi"}]}`), completed)
	}))
	defer srv.Close()
	tokens := &fakeTokens{access: "stale-token"}
	conv := NewOpenAIProvider(OpenAIConfig{URL: srv.URL, Tokens: tokens}).NewConversation(nil)
	reply, err := conv.UserMessage(context.Background(), "hello")
	if err != nil {
		t.Fatal(err)
	}
	if len(reply.Blocks) != 1 || reply.Blocks[0].Text != "hi" {
		t.Errorf("reply = %+v", reply)
	}
	if tokens.refreshed.Load() != 1 || tokens.staleSeen != "stale-token" {
		t.Errorf("refresh calls=%d stale=%q", tokens.refreshed.Load(), tokens.staleSeen)
	}
	if len(auths) != 2 || auths[0] != "Bearer stale-token" || auths[1] != "Bearer fresh-token" {
		t.Errorf("auth headers = %v", auths)
	}
}

func TestOpenAIErrors(t *testing.T) {
	cases := []struct {
		name   string
		handle func(w http.ResponseWriter)
		want   string
	}{
		{"usage limit via response.failed", func(w http.ResponseWriter) {
			sse(w, `{"type":"response.failed","response":{"error":{"code":"subscription_sharing_usage_limit_exceeded","message":"limit"}}}`)
		}, "chatgpt.com/settings/usage"},
		{"http 429 body", func(w http.ResponseWriter) {
			w.WriteHeader(429)
			fmt.Fprint(w, `{"error":{"code":"subscription_sharing_usage_limit_exceeded","message":"limit"}}`)
		}, "HTTP 429 (subscription_sharing_usage_limit_exceeded)"},
		{"admission detail body", func(w http.ResponseWriter) {
			w.WriteHeader(403)
			fmt.Fprint(w, `{"detail":"region not permitted"}`)
		}, "region not permitted"},
		{"incomplete", func(w http.ResponseWriter) {
			sse(w, `{"type":"response.incomplete","response":{"incomplete_details":{"reason":"max_output_tokens"}}}`)
		}, "incomplete (max_output_tokens)"},
		{"stream cut before completion", func(w http.ResponseWriter) {
			sse(w, itemDone(`{"type":"message","role":"assistant","content":[{"type":"output_text","text":"partial"}]}`))
		}, "without response.completed"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { tc.handle(w) }))
			defer srv.Close()
			conv := NewOpenAIProvider(OpenAIConfig{URL: srv.URL, Tokens: &fakeTokens{access: "t"}}).NewConversation(nil)
			_, err := conv.UserMessage(context.Background(), "x")
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("err = %v, want containing %q", err, tc.want)
			}
		})
	}
}

func TestOpenAITokenErrorStopsBeforeAnyRequest(t *testing.T) {
	hit := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { hit = true }))
	defer srv.Close()
	want := errors.New("login required")
	conv := NewOpenAIProvider(OpenAIConfig{URL: srv.URL, Tokens: &fakeTokens{err: want}}).NewConversation(nil)
	if _, err := conv.UserMessage(context.Background(), "x"); !errors.Is(err, want) || hit {
		t.Fatalf("err=%v hit=%v", err, hit)
	}
}

func TestResponsesToolSchemaFromMCPStyleTool(t *testing.T) {
	// A tool with no properties still yields a valid object schema.
	rt, err := toResponsesTool(tools.ToolDefinition{Name: "ping", Description: "d"})
	if err != nil {
		t.Fatal(err)
	}
	var p map[string]any
	_ = json.Unmarshal(rt.Parameters, &p)
	if p["type"] != "object" || p["properties"] == nil {
		t.Errorf("parameters = %s", rt.Parameters)
	}
}
