✅ Prerequisites

- Go 1.24.2+ or devenv (recommended for easy setup)
- An Anthropic API Key

🔧 Set Up Your Environment

`devenv shell`  # Loads everything you need

# Make sure Go is installed
`go mod tidy`

🔐 Add Your API Key

`export ANTHROPIC_API_KEY="your-api-key-here"`

### What can this agent do?
1. Talks to claude
2. **File Reader** — read code files [tool](./internal/tools/read_file.go)
3. **File Explorer** — list files in folders [tool](./internal/tools/list_files.go)
4. **Command Runner** — run shell commands [tool](./internal/tools/bash.go)
5. **File Editor** — modify files [tool](./internal/tools/edit.go)
6. **Code Search** — search your codebase with patterns [tool](./internal/tools/code_search.go)