# Nightingale MCP Server

[![Go](https://img.shields.io/badge/Go-1.21+-00ADD8?style=flat&logo=go)](https://go.dev/)
[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](https://github.com/n9e/n9e-mcp-server/blob/main/LICENSE)
[![MCP](https://img.shields.io/badge/MCP-Compatible-green.svg)](https://modelcontextprotocol.io/)

An MCP (Model Context Protocol) server for [Nightingale](https://github.com/ccfos/nightingale) monitoring system. This server enables AI assistants to interact with Nightingale APIs for alert management, monitoring, and observability tasks through natural language.

## Key Use Cases

- **Alert Management**: Query active and historical alerts, view alert rules and subscriptions
- **Target Monitoring**: Browse and search monitored hosts/targets, analyze target status
- **Incident Response**: Create and manage alert mutes/silences, notification rules, and event pipelines
- **Team Collaboration**: Query users, teams, and business groups

## Quick Start

### 1. Get an API Token

1. Log in to your Nightingale web interface
2. Navigate to **Personal Settings** > **Profile** > **Token Management**
3. Create a new token with appropriate permissions

### 2. Configure MCP Client

Add to your MCP client config (e.g., `~/.cursor/mcp.json` or `~/.opencode/mcp.json`):

```json
{
  "mcpServers": {
    "nightingale": {
      "command": "npx",
      "args": ["-y", "@n9e/n9e-mcp-server", "stdio"],
      "env": {
        "N9E_TOKEN": "your-api-token",
        "N9E_BASE_URL": "http://your-n9e-server:17000"
      }
    }
  }
}
```

### 3. Restart Your MCP Client

## Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `N9E_TOKEN` | Nightingale API token (required) | - |
| `N9E_BASE_URL` | Nightingale API base URL | `http://localhost:17000` |
| `N9E_READ_ONLY` | Disable write operations | `false` |
| `N9E_TOOLSETS` | Enabled toolsets (comma-separated) | `all` |

## Example Prompts

- "Show me all critical alerts from the last 24 hours"
- "What alerts are currently firing?"
- "List all monitored targets that have been down for more than 5 minutes"
- "Create a mute rule for service=api alerts for the next 2 hours"

## Documentation

For full documentation and available tools, see the [GitHub repository](https://github.com/n9e/n9e-mcp-server).

## License

Apache License 2.0
