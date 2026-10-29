package agent

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/fernankiwi/fonso/internal/tools"
)

// fakeProvider scripts replies and records what the agent sent back.
type fakeProvider struct {
	replies []Reply
	gotUser []string
	gotRes  [][]ToolResult
}

func (f *fakeProvider) Label() string     { return "Fake" }
func (f *fakeProvider) ModelName() string { return "fake-1" }
func (f *fakeProvider) NewConversation([]tools.ToolDefinition) Conversation {
	return f
}
func (f *fakeProvider) next() (Reply, error) {
	if len(f.replies) == 0 {
		return Reply{}, errors.New("no scripted reply left")
	}
	r := f.replies[0]
	f.replies = f.replies[1:]
	return r, nil
}
func (f *fakeProvider) UserMessage(_ context.Context, text string) (Reply, error) {
	f.gotUser = append(f.gotUser, text)
	return f.next()
}
func (f *fakeProvider) ToolResults(_ context.Context, rs []ToolResult) (Reply, error) {
	f.gotRes = append(f.gotRes, rs)
	return f.next()
}

func TestToolAgentLoopWithAnyProvider(t *testing.T) {
	echo := tools.ToolDefinition{
		Name: "echo",
		Function: func(in json.RawMessage) (string, error) {
			return "echoed:" + string(in), nil
		},
	}
	f := &fakeProvider{replies: []Reply{
		{Blocks: []Block{{Text: "thinking"}, {Call: &ToolCall{ID: "c1", Name: "echo", Input: json.RawMessage(`{"a":1}`)}}, {Call: &ToolCall{ID: "c2", Name: "missing", Input: json.RawMessage(`{}`)}}}},
		{Blocks: []Block{{Text: "done"}}},
	}}
	a := NewToolAgentWithProvider(f, scriptedInput("hi", ""), []tools.ToolDefinition{echo}, false, "")
	out := captureStdout(t, func() {
		if err := a.Run(context.Background()); err != nil {
			t.Fatal(err)
		}
	})
	if len(f.gotUser) != 1 || f.gotUser[0] != "hi" {
		t.Fatalf("user messages = %v", f.gotUser)
	}
	if len(f.gotRes) != 1 || len(f.gotRes[0]) != 2 {
		t.Fatalf("tool results = %+v", f.gotRes)
	}
	r0, r1 := f.gotRes[0][0], f.gotRes[0][1]
	if r0.CallID != "c1" || r0.Output != `echoed:{"a":1}` || r0.IsError {
		t.Errorf("result 0 = %+v", r0)
	}
	if r1.CallID != "c2" || !r1.IsError || !strings.Contains(r1.Output, "not found") {
		t.Errorf("result 1 = %+v", r1)
	}
	for _, want := range []string{"Chat with Fake", "Fake\x1b[0m: thinking", "Fake\x1b[0m: done"} {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q:\n%s", want, out)
		}
	}
}

func TestToolAgentPropagatesProviderError(t *testing.T) {
	f := &fakeProvider{}
	a := NewToolAgentWithProvider(f, scriptedInput("hi"), nil, false, "")
	var err error
	captureStdout(t, func() { err = a.Run(context.Background()) })
	if err == nil || !strings.Contains(err.Error(), "no scripted reply") {
		t.Fatalf("err = %v", err)
	}
}
