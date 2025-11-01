# Browserbase MCP Server Setup Guide

This guide will help you set up and use the Browserbase MCP Server with your Fonso application.

## What is Browserbase MCP Server?

The Browserbase MCP Server allows AI models to control a browser through natural language commands. It provides:
- **AI-Powered Web Automation**: Control browsers using natural language
- **Screenshot Capture**: Take full-page or element-specific screenshots
- **Data Extraction**: Extract content from web pages intelligently
- **Stealth Mode**: Advanced anti-detection features for web scraping
- **Session Persistence**: Maintain authentication across sessions

## Prerequisites

1. **Browserbase Account**: Sign up at [browserbase.com](https://www.browserbase.com/)
2. **API Credentials**: Get your API key and Project ID from the Browserbase dashboard
3. **Anthropic API Key**: Required for Claude AI integration

## Installation

### Step 1: Install Dependencies

The Browserbase server is installed via npx, so no additional installation is required. However, make sure you have built your TypeScript code:

```bash
npm run build
```

### Step 2: Configure Environment Variables

Create a `.env` file in the project root with the following variables:

```bash
# Anthropic API Key (required for Claude AI)
ANTHROPIC_API_KEY=your_anthropic_api_key_here

# Browserbase Configuration
BROWSERBASE_API_KEY=your_browserbase_api_key_here
BROWSERBASE_PROJECT_ID=your_project_id_here  # Optional
```

**How to get your credentials:**
1. Go to [browserbase.com](https://www.browserbase.com/)
2. Create an account or sign in
3. Navigate to Settings → API Keys
4. Copy your API Key and Project ID

## Usage

### Quick Start

Run the MCP client with Browserbase server:

```bash
npm run mcp-browserbase
```

This will:
1. Build your TypeScript code
2. Start the MCP client
3. Connect to the Browserbase MCP server via npx
4. Begin an interactive chat session

### Example Queries

Once connected, you can use natural language commands like:

**Taking Screenshots:**
```
Take a screenshot of https://example.com
```

**Web Scraping:**
```
Go to https://news.ycombinator.com and extract the top 5 article titles
```

**Form Interactions:**
```
Navigate to https://example.com/search, type "AI automation" in the search box, and click submit
```

**Data Extraction:**
```
Visit https://github.com/trending and get me the top 3 trending repositories
```

## Available Tools

The Browserbase MCP Server provides the following tools:

1. **browserbase_create_session** - Create a new browser session
2. **browserbase_navigate** - Navigate to a URL
3. **browserbase_screenshot** - Capture screenshots
4. **browserbase_get_page_content** - Extract page content
5. **browserbase_click** - Click on elements
6. **browserbase_type** - Type text into inputs
7. **browserbase_close_session** - Close a browser session

## Advanced Configuration

### Custom Server Configuration

If you need more control, you can modify the `connectToServer` method in `src/client.ts` or create a configuration file:

```json
{
  "mcpServers": {
    "browserbase": {
      "command": "npx",
      "args": ["-y", "@browserbasehq/mcp-server-browserbase"],
      "env": {
        "BROWSERBASE_API_KEY": "${BROWSERBASE_API_KEY}",
        "BROWSERBASE_PROJECT_ID": "${BROWSERBASE_PROJECT_ID}"
      }
    }
  }
}
```

### Using with Other MCP Servers

You can also connect to other MCP servers by passing different commands:

```bash
# Python server
npm run mcp-client path/to/server.py

# JavaScript server
npm run mcp-client path/to/server.js

# NPX package
npm run mcp-client 'npx @package-name/mcp-server'
```

## Troubleshooting

### Common Issues

**Issue: "BROWSERBASE_API_KEY is not set"**
- Solution: Make sure your `.env` file exists and contains the correct API key

**Issue: "Connection timeout"**
- Solution: Check your internet connection and verify Browserbase service status

**Issue: "Invalid API key"**
- Solution: Verify your API key in the Browserbase dashboard

**Issue: "npx command not found"**
- Solution: Make sure Node.js and npm are installed and updated

### Debug Mode

To see more detailed logs, you can modify the client to enable verbose logging:

```bash
NODE_DEBUG=* npm run mcp-browserbase
```

## Security Best Practices

1. **Never commit `.env` files** - Keep your API keys secret
2. **Use environment-specific keys** - Different keys for dev/staging/production
3. **Rotate keys regularly** - Change API keys periodically
4. **Limit API key permissions** - Use read-only keys when possible

## Resources

- [Browserbase Documentation](https://docs.browserbase.com/)
- [Browserbase MCP Server GitHub](https://github.com/browserbase/mcp-server-browserbase)
- [MCP Protocol Documentation](https://modelcontextprotocol.info/)
- [Anthropic Claude API](https://docs.anthropic.com/)

## Support

For issues related to:
- **Browserbase MCP Server**: [GitHub Issues](https://github.com/browserbase/mcp-server-browserbase/issues)
- **Fonso App**: Contact the Fonso development team
- **MCP Protocol**: [MCP Documentation](https://modelcontextprotocol.info/)


