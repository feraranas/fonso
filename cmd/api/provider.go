// provider.go - Builds the model provider selected by --provider/--model
package main

import (
	"fmt"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/fernankiwi/fonso/internal/agent"
	"github.com/fernankiwi/fonso/internal/auth/openai"
)

const (
	providerAnthropic = "anthropic"
	providerOpenAI    = "openai-chatgpt"
)

type providerOptions struct {
	Name       string // anthropic | openai-chatgpt
	Model      string // model slug; empty = provider default
	MaxTokens  int64  // anthropic only; 0 = default (1024)
	OpenAIAuth string // official | opencode
	Verbose    bool
	// OpenAIConfig is the auth/endpoint configuration; tests inject stub URLs here.
	OpenAIConfig openai.Config
}

func openAIMode(s string) (openai.Mode, error) {
	switch openai.Mode(s) {
	case openai.ModeOfficial, openai.ModeOpenCode:
		return openai.Mode(s), nil
	}
	return "", fmt.Errorf("unknown --openai-auth %q (want official or opencode)", s)
}

// newProvider returns the agent.Provider for opts. newAnthropic is only called
// for the anthropic provider, so the OpenAI path never needs ANTHROPIC_API_KEY.
func newProvider(opts providerOptions, newAnthropic func() *anthropic.Client) (agent.Provider, *openai.Source, error) {
	switch opts.Name {
	case providerAnthropic, "":
		if opts.MaxTokens < 0 {
			return nil, nil, fmt.Errorf("--max-tokens must be positive")
		}
		return agent.NewAnthropicProvider(newAnthropic(), opts.Verbose).WithModel(opts.Model).WithMaxTokens(opts.MaxTokens), nil, nil
	case providerOpenAI:
		if opts.MaxTokens != 0 {
			return nil, nil, fmt.Errorf("--max-tokens is not supported with --provider %s (the ChatGPT-plan Responses route rejects max_output_tokens)", providerOpenAI)
		}
		mode, err := openAIMode(opts.OpenAIAuth)
		if err != nil {
			return nil, nil, err
		}
		cfg := opts.OpenAIConfig
		cfg.Mode = mode
		src := openai.NewSource(cfg)
		p := agent.NewOpenAIProvider(agent.OpenAIConfig{
			URL:     cfg.InferenceEndpoint(),
			Model:   opts.Model,
			Tokens:  src,
			Verbose: opts.Verbose,
		})
		return p, src, nil
	}
	return nil, nil, fmt.Errorf("unknown --provider %q (want %s or %s)", opts.Name, providerAnthropic, providerOpenAI)
}
