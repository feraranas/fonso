Fonso Coding Agent

✅ Prerequisites

- Go 1.24.2+ or devenv (recommended for easy setup)
- An Anthropic API Key

🔧 Set Up Your Environment

`devenv shell`  # Loads everything you need

# Make sure Go is installed
`go mod tidy`

🔐 Add Your API Key

`export ANTHROPIC_API_KEY="your-api-key-here"`

### Use your ChatGPT plan instead of an API key (`cmd/api` only)

`cmd/api` can run its agent loop on your ChatGPT subscription through OAuth. The default stays Anthropic.

```bash
# 1. Sign in once. Opens your browser, listens on 127.0.0.1:1455, saves
#    ~/.config/fonso/openai-auth.json (mode 0600). Tokens are never printed.
go run ./cmd/api --provider openai-chatgpt --login

# 2. Run the agent on your ChatGPT plan (optionally pick a model).
go run ./cmd/api --provider openai-chatgpt
go run ./cmd/api --provider openai-chatgpt --model gpt-5.4-mini
```

Notes:
- The access token is refreshed automatically (it lasts about an hour). If the refresh token is revoked or expires, re-run step 1.
- `--openai-auth official` (default) uses OpenAI's documented "Sign in with ChatGPT" flow for open-source clients (dynamic client registration, public Responses API). Which models you can use is decided by your plan; usage counts against your ChatGPT limits (see https://chatgpt.com/settings/usage).
- `--openai-auth opencode` reuses opencode's/Codex's client id and the `chatgpt.com/backend-api/codex` endpoint. fonso is not a registered client for that id, so treat it as unsanctioned and use the official mode unless you have a reason not to.
- `--model` also works with the default Anthropic provider, and `--max-tokens N` raises its 1024-token response limit. `--max-tokens` is rejected for `openai-chatgpt`, whose route does not accept `max_output_tokens`.

### What can this agent do?
1. Talks to claude
2. **File Reader** — read code files [tool](./internal/tools/read_file.go)
3. **File Explorer** — list files in folders [tool](./internal/tools/list_files.go)
4. **Command Runner** — run shell commands [tool](./internal/tools/bash.go)
5. **File Editor** — modify files [tool](./internal/tools/edit.go)
6. **Code Search** — search your codebase with patterns [tool](./internal/tools/code_search.go)
