# Nightingale MCP Server

English | [中文](README_zh.md)

[![Go](https://img.shields.io/badge/Go-1.21+-00ADD8?style=flat&logo=go)](https://go.dev/)
[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](LICENSE)
[![MCP](https://img.shields.io/badge/MCP-Compatible-green.svg)](https://modelcontextprotocol.io/)

An MCP (Model Context Protocol) server for [Nightingale](https://github.com/ccfos/nightingale) monitoring system. This server enables AI assistants to interact with Nightingale APIs for alert management, monitoring, and observability tasks through natural language.

## Compatibility

- **Nightingale**: v8.0.0+

## Key Use Cases

- **Alert Management**: Query active and historical alerts, view alert rules and subscriptions
- **Target Monitoring**: Browse and search monitored hosts/targets, analyze target status
- **Incident Response**: Create and manage alert mutes, notification rules, and event pipelines
- **Team Collaboration**: Query users, teams, and business groups

## Quick Start

### 1. Get an API Token

1. Make sure `HTTP.TokenAuth` is enabled in `config.toml`:
  ```toml
    [HTTP.TokenAuth]
    Enable = true
  ```
2. Log in to your Nightingale web interface
3. Navigate to **Personal Settings** > **Profile** > **Token Management**
4. Create a new token with appropriate permissions

![image-20260205172354525](./doc/img/image-20260205172354525.png)

> **Security Note**: Store your API token securely. Never commit tokens to version control. Use environment variables or secure secret management.

### 2. Configure MCP Client

#### Cursor (stdio mode, default)

Add to your `~/.cursor/mcp.json`:

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

#### HTTP mode (optional)

To run the server over HTTP (MCP streamable transport, JSON only, no SSE), start the server with the `http` subcommand.

**Shared vs non-shared (HTTP only):**

- **`--shared=false`** (default): Token and base URL may be omitted at startup. Each client can provide `X-User-Token` and `X-N9e-Base-Url` in mcp.json so everyone uses their own Nightingale identity or instance. If you do set `N9E_TOKEN` and `N9E_BASE_URL` at startup, they act as defaults and clients can still override via headers.
- **`--shared=true`**: Startup **must** set `N9E_TOKEN` and `N9E_BASE_URL`. The server uses only this config; client headers `X-User-Token` and `X-N9e-Base-Url` are **ignored**. Use this when the MCP server is a shared org service and users must not override credentials.

```bash
# Non-shared: users supply token/URL in mcp.json (or you set defaults at startup)
n9e-mcp-server http --listen :8080

# Shared: one token/URL for all; require at startup, ignore client headers
N9E_TOKEN=xxx N9E_BASE_URL=https://n9e.example.com n9e-mcp-server http --listen :8080 --shared
```

**Cursor: connect to HTTP server**

If the server is already running in HTTP mode (e.g. on `http://localhost:8080`), add a URL-based entry to `~/.cursor/mcp.json` (no `command`/`args`; Cursor will use the streamable HTTP transport).

**Token:** You can use either source; you do **not** need to pass token in mcp.json if the server was started with `N9E_TOKEN`.

1. **Server startup only** – Set `N9E_TOKEN` when starting the server (e.g. `N9E_TOKEN=xxx ./n9e-mcp-server http`). All clients will use this token; no headers needed in Cursor.
2. **Client headers (optional)** – The client can send:
   - `X-User-Token`: use this token for N9e API calls instead of the startup token.
   - `X-N9e-Base-Url`: use this URL as the Nightingale API base (e.g. `https://n9e.other-env.com`) instead of the server's `N9E_BASE_URL`.
   So each user can point to a different Nightingale instance or use their own token (or both).

Example when the server **was** started with `N9E_TOKEN` (no header needed in Cursor):

```json
{
  "mcpServers": {
    "nightingale": {
      "url": "http://localhost:8080"
    }
  }
}
```

Example when passing token and/or base URL from Cursor (e.g. shared server, or different Nightingale instance):

```json
{
  "mcpServers": {
    "nightingale": {
      "url": "http://localhost:8080",
      "headers": {
        "X-User-Token": "your-nightingale-api-token",
        "X-N9e-Base-Url": "http://your-n9e-server:17000"
      }
    }
  }
}
```

You can omit either header: only `X-User-Token`, or only `X-N9e-Base-Url`, or both. The server falls back to startup `N9E_TOKEN` and `N9E_BASE_URL` when a header is not set. **If the server was started with `--shared`**, these headers are ignored and you must not rely on them.

If your HTTP server is behind a gateway that requires its own auth, add those headers as well (e.g. `Authorization: Bearer your-gateway-token`). The server uses the `X-User-Token` header only for calling the Nightingale API.
#### Docker

The official code uses the `stdio` protocol by default for inter-process communication. If you need to integrate with web-based LLM orchestration platforms like Dify or FastGPT that only support network calls (HTTP/SSE), we recommend using the provided Docker Compose solution. This deployment automatically introduces the `mcp-proxy` bridge to enable network protocol support.

**Deployment Steps:**

1. Clone this repository:

   ```bash
   git clone https://github.com/n9e/n9e-mcp-server.git
   cd n9e-mcp-server/docker
   ```

2. Modify the configuration in `docker-compose.yml`:

   - `MCP_VERSION`: (Optional) Used by the default Dockerfile to install `@n9e/n9e-mcp-server` from NPM during image build. Set this explicitly to `latest` or to a concrete version such as `0.1.1`. Do not leave it blank.
   - `N9E_BASE_URL`: Replace with your actual Nightingale API URL.
   - `N9E_TOKEN`: Replace with your generated API Token.

3. Start the service:

   ```bash
   # By default, it will install the specified version via NPM and start
   docker compose up -d --build
   ```

4. Configure the MCP plugin in Dify:

   - **Connection Type**: Streamable HTTP (SSE)
   - **URL**: `http://<Your-Server-IP>:8082/sse`

> **Developer Note**: If you have modified the source code and wish to build a local test image, change the `dockerfile` field in `docker-compose.yml` to `docker/Dockerfile.source`. This will copy the repository source into the image, build the Go server locally, and still start the server through the `mcp-proxy` bridge.

### 3. Restart Cursor or Other Client Processes to Use

## Available Tools

Tools split into two tiers:

- **read** — always available
- **write** — registered unless `--read-only` is set (covers create/update/import/clone/toggle)

The full default toolset list is `alerts, targets, datasource, mutes, busi_groups, notify_rules, alert_subscribes, event_pipelines, users, metrics, logs, dashboards, roles`. Restrict via `N9E_TOOLSETS`.

### `alerts`

| Tool | Tier | Description |
|---|---|---|
| `list_active_alerts` / `get_active_alert` | read | Active firing alerts |
| `list_history_alerts` / `get_history_alert` | read | Historical alerts |
| `list_alert_rules` / `get_alert_rule` | read | Alert rule listing |
| `create_alert_rule` / `update_alert_rule` | write | CRUD on a single rule |
| `import_alert_rules` / `import_prom_rules` | write | Bulk import (n9e JSON or Prometheus YAML) |
| `clone_alert_rules_to_bgs` | write | Copy rules into target business groups |
| `toggle_alert_rules` | write | Enable/disable a batch (uses `PUT .../fields`) |

### `dashboards`

| Tool | Tier | Description |
|---|---|---|
| `list_dashboards` / `get_dashboard` / `get_dashboard_pure` | read | List + metadata + full panel JSON |
| `create_dashboard` / `update_dashboard_meta` / `update_dashboard_panels` | write | CRUD splits meta vs panels |
| `set_dashboard_public` / `clone_dashboard` | write | Visibility + clone |

### `notify_rules`

| Tool | Tier | Description |
|---|---|---|
| `list_notify_rules` / `get_notify_rule` | read | Notification rules |
| `list_notify_channels` / `get_notify_channel` / `list_notify_templates` | read | Channel + template listing |
| `create_notify_rules` / `update_notify_rule` / `test_notify_rule` | write | Rule CRUD + dry-run test |
| `create_notify_channel` / `update_notify_channel` | write | Channel CRUD |
| `create_notify_template` / `update_notify_template` / `update_notify_template_content` | write | Template CRUD |

### `datasource`

| Tool | Tier | Description |
|---|---|---|
| `list_datasources` / `list_datasources_full` / `get_datasource` / `list_datasource_plugins` | read | Brief + full configs + supported plugins |
| `upsert_datasource` | write | Create or update; n9e validates connectivity inside upsert |
| `set_datasource_status` | write | Enable/disable batch |

### `users`

| Tool | Tier | Description |
|---|---|---|
| `list_users` / `get_user` / `list_user_groups` / `get_user_group` | read | |
| `create_user` / `update_user_profile` / `reset_user_password` | write | Profile.roles assigns RBAC roles (no separate endpoint) |
| `create_user_group` / `update_user_group` / `add_user_group_members` | write | Team management |

### `roles` (RBAC)

| Tool | Tier | Description |
|---|---|---|
| `list_roles` / `list_operations` / `list_role_operations` | read | |
| `create_role` / `update_role` / `bind_role_operations` | write | `bind_role_operations` replaces (not deltas) |

### `metrics` (Prometheus query proxy)

| Tool | Tier | Description |
|---|---|---|
| `query_instant` | read | `/api/v1/query` via `/api/n9e/proxy/{ds_id}/...` |
| `query_range` | read | Auto-adjusts `step` so result ≤ `max_points` (default 1000); flags `truncated:true` when downsampled |
| `query_label_values` / `query_series` | read | |

### `logs`

| Tool | Tier | Description |
|---|---|---|
| `query_logs` | read | Plugin-dispatched `/logs-query` (Loki/ES/OS); default `limit=200`; rejects time ranges > 7 days |
| `list_log_indices` / `list_log_fields` | read | ES (default) or OpenSearch (`engine=os`) |

### `mutes` / `alert_subscribes` / `event_pipelines` / `targets` / `busi_groups`

These keep their existing tools (see prior versions). `mutes` adds no new tools in this revision.

## Example Prompts

Once configured, you can interact with Nightingale using natural language:

- "Show me all critical alerts from the last 24 hours"
- "What alerts are currently firing?"
- "List all monitored targets that have been down for more than 5 minutes"
- "What alert rules are configured in business group 1?"
- "Create a mute rule for service=api alerts for the next 2 hours due to maintenance"
- "Show me the event pipeline execution history"
- "Who are the members of the ops team?"

## Configuration

### Modes

- **stdio** (default): MCP over stdin/stdout. Use with Cursor and other clients that spawn the server process.
- **http**: MCP over HTTP using the streamable transport (JSON request/response only, no SSE). Use `n9e-mcp-server http` and connect with a client that supports streamable HTTP (e.g. `StreamableClientTransport`).

### Environment Variables

| Variable | Flag | Description | Default |
|----------|------|-------------|---------|
| `N9E_TOKEN` | `--token` | Nightingale API token (required) | - |
| `N9E_BASE_URL` | `--base-url` | Nightingale API base URL | `http://localhost:17000` |
| `N9E_READ_ONLY` | `--read-only` | Disable write operations (only `read` tools registered) | `false` |
| `N9E_TOOLSETS` | `--toolsets` | Enabled toolsets (comma-separated) | `all` |
| `N9E_LISTEN` | `--listen` | HTTP mode: listen address | `:8080` |
| `N9E_SESSION_TIMEOUT` | `--session-timeout` | HTTP mode: idle session timeout (0 = no timeout) | `0` |

### Toolsets

By default, all toolsets are enabled. You can use the `--toolsets` flag or `N9E_TOOLSETS` environment variable to enable only the toolsets you need, reducing the number of tools exposed to the AI assistant and saving context window tokens.

Available toolsets: `alerts`, `targets`, `datasource`, `mutes`, `busi_groups`, `notify_rules`, `alert_subscribes`, `event_pipelines`, `users`, `metrics`, `logs`, `dashboards`, `roles`

For example, to enable only alert and target related tools:

```json
{
  "mcpServers": {
    "nightingale": {
      "command": "npx",
      "args": ["-y", "@n9e/n9e-mcp-server", "stdio"],
      "env": {
        "N9E_TOKEN": "your-api-token",
        "N9E_BASE_URL": "http://your-n9e-server:17000",
        "N9E_TOOLSETS": "alerts,targets"
      }
    }
  }
}
```

## License

Apache License 2.0

## Related Projects

- [Nightingale](https://github.com/ccfos/nightingale) - The enterprise-level cloud-native monitoring system
- [MCP Go SDK](https://github.com/modelcontextprotocol/go-sdk) - Official MCP SDK for Go
- [MCP Specification](https://modelcontextprotocol.io/) - Model Context Protocol specification
