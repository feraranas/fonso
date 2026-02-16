#!/bin/bash
# Test script to verify MCP tools are available

echo "Testing MCP integration..."
echo ""
echo "Commands to try:"
echo "  - 'list all available tools'"
echo "  - 'use the hello tool with my name'"
echo "  - 'what time is it using the timestamp tool'"
echo ""

# Start the agent with verbose logging
./bin/api --verbose
