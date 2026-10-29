package main

import (
	"strings"
	"testing"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/fernankiwi/fonso/internal/agent"
	"github.com/fernankiwi/fonso/internal/auth/openai"
)

func failAnthropic(t *testing.T) func() *anthropic.Client {
	return func() *anthropic.Client {
		t.Fatal("anthropic client must not be built for this provider")
		return nil
	}
}

func TestDefaultProviderIsAnthropicClaude(t *testing.T) {
	built := false
	client := anthropic.NewClient()
	p, src, err := newProvider(providerOptions{}, func() *anthropic.Client { built = true; return &client })
	if err != nil || src != nil || !built {
		t.Fatalf("err=%v src=%v built=%v", err, src, built)
	}
	if _, ok := p.(*agent.AnthropicProvider); !ok || p.Label() != "Claude" || p.ModelName() != "claude-3-7-sonnet-latest" {
		t.Errorf("provider = %T %s %s", p, p.Label(), p.ModelName())
	}
}

func TestOpenAIProviderSelection(t *testing.T) {
	cfg := openai.Config{StorePath: t.TempDir() + "/auth.json"}
	p, src, err := newProvider(providerOptions{Name: providerOpenAI, OpenAIAuth: "official", OpenAIConfig: cfg}, failAnthropic(t))
	if err != nil || src == nil {
		t.Fatalf("err=%v src=%v", err, src)
	}
	if p.Label() != "ChatGPT" || p.ModelName() != agent.DefaultOpenAIModel {
		t.Errorf("provider = %s %s", p.Label(), p.ModelName())
	}
	p, _, err = newProvider(providerOptions{Name: providerOpenAI, Model: "gpt-5.4-mini", OpenAIAuth: "opencode", OpenAIConfig: cfg}, failAnthropic(t))
	if err != nil || p.ModelName() != "gpt-5.4-mini" {
		t.Errorf("model override: %v %v", p, err)
	}
	// Without a login the first token request tells the user to log in.
	if _, _, err := src.Token(t.Context()); err == nil || !strings.Contains(err.Error(), "--login") {
		t.Errorf("missing-login error = %v", err)
	}
}

func TestProviderFlagValidation(t *testing.T) {
	cases := []struct {
		name string
		opts providerOptions
		want string
	}{
		{"unknown provider", providerOptions{Name: "bogus"}, "unknown --provider"},
		{"max tokens with openai", providerOptions{Name: providerOpenAI, MaxTokens: 4096}, "--max-tokens is not supported"},
		{"negative max tokens", providerOptions{Name: providerAnthropic, MaxTokens: -1}, "must be positive"},
		{"bad auth flavour", providerOptions{Name: providerOpenAI, OpenAIAuth: "nope"}, "unknown --openai-auth"},
	}
	for _, tc := range cases {
		_, _, err := newProvider(tc.opts, failAnthropic(t))
		if err == nil || !strings.Contains(err.Error(), tc.want) {
			t.Errorf("%s: err = %v", tc.name, err)
		}
	}
}

func TestAnthropicOverridesFlowThrough(t *testing.T) {
	client := anthropic.NewClient()
	p, _, err := newProvider(providerOptions{Name: providerAnthropic, Model: "claude-test-model", MaxTokens: 2048}, func() *anthropic.Client { return &client })
	if err != nil || p.ModelName() != "claude-test-model" {
		t.Fatalf("provider = %v err=%v", p, err)
	}
}
