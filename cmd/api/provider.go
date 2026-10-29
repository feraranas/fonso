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
	Model      string // OpenAI model slug; not supported for anthropic yet
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
		if opts.Model != "" {
			return nil, nil, fmt.Errorf("--model is only supported with --provider %s", providerOpenAI)
		}
		return agent.NewAnthropicProvider(newAnthropic(), opts.Verbose), nil, nil
	case providerOpenAI:
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
