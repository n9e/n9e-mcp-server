# Nightingale MCP Server

[English](README.md) | 中文

[![Go](https://img.shields.io/badge/Go-1.21+-00ADD8?style=flat&logo=go)](https://go.dev/)
[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](LICENSE)
[![MCP](https://img.shields.io/badge/MCP-Compatible-green.svg)](https://modelcontextprotocol.io/)

[Nightingale](https://github.com/ccfos/nightingale) 的 MCP Server。此 MCP Server 允许 AI 助手通过自然语言与夜莺 API 交互，实现告警管理、监控和可观测性任务。

## 兼容性

- **Nightingale**：v8.0.0+

## 主要用途

- **告警管理**：查询活跃告警和历史告警，查看告警规则和订阅
- **目标监控**：浏览和搜索被监控的主机，分析目标状态
- **事件响应**：创建和管理告警屏蔽规则、通知规则和事件流水线
- **团队协作**：查询用户、团队和业务组

## 快速开始

### 1.获取 API Token
1. 确保在 config.toml 中，启用了 HTTP.TokenAuth
  ```toml
    [HTTP.TokenAuth]
    Enable = true
  ```
2. 登录夜莺 Web 界面
3. 进入 **个人设置** > **个人信息** > **Token 管理**
4. 创建一个具有适当权限的新 Token

![image-20260205172354525](./doc/img/image-20260205172354525.png)

> **安全提示**：请妥善保管 API Token。切勿将 Token 提交到版本控制系统。请使用环境变量或安全的密钥管理系统。

### 2.与 MCP 客户端配合使用

#### Cursor（stdio 模式，默认）

在 `~/.cursor/mcp.json` 中添加：

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

#### HTTP 模式（可选）

以 HTTP 方式运行服务端（MCP streamable 传输，仅 JSON，无 SSE）时，使用 `http` 子命令启动。

**共享模式 vs 非共享（仅 HTTP）：**

- **`--shared=false`**（默认）：启动时可不提供 token 和 base URL。每个客户端在 mcp.json 里通过 `X-User-Token`、`X-N9e-Base-Url` 提供自己的夜莺身份或实例；若启动时设置了 `N9E_TOKEN` 和 `N9E_BASE_URL`，则作为默认值，客户端仍可通过 header 覆盖。
- **`--shared=true`**：启动时**必须**设置 `N9E_TOKEN` 和 `N9E_BASE_URL`。服务端仅使用该配置，**忽略**客户端请求头中的 `X-User-Token` 和 `X-N9e-Base-Url`。适用于组织统一提供的 MCP 服务、不允许用户覆盖凭证的场景。

```bash
# 非共享：由用户在 mcp.json 提供 token/URL（或启动时设默认值）
n9e-mcp-server http --listen :8080

# 共享：统一凭证，启动时必填，忽略客户端 header
N9E_TOKEN=xxx N9E_BASE_URL=https://n9e.example.com n9e-mcp-server http --listen :8080 --shared
```

**Cursor 连接 HTTP 服务端**

若服务端已以 HTTP 模式运行（例如在 `http://localhost:8080`），在 `~/.cursor/mcp.json` 中添加以 URL 方式配置的条目（无需 `command`/`args`，Cursor 会使用 streamable HTTP 传输）。

**Token 传递**：二选一即可，不必在 mcp.json 里传 token（只要服务端启动时配了 `N9E_TOKEN`）。

1. **仅服务端启动时**：启动时设置 `N9E_TOKEN`（如 `N9E_TOKEN=xxx ./n9e-mcp-server http`），所有连接该服务的客户端都会用这个 token，Cursor 里**无需**配置任何 header。
2. **客户端请求头（可选）**：可携带：
   - `X-User-Token`：用该 token 调夜莺 API，替代启动时的 `N9E_TOKEN`；
   - `X-N9e-Base-Url`：用该 URL 作为夜莺 API 地址（如 `https://n9e.other-env.com`），替代服务端启动时的 `N9E_BASE_URL`。
   这样每人可使用自己的 token 或指向不同夜莺环境（或同时覆盖两者）。

若服务端**已用** `N9E_TOKEN` 启动（Cursor 里不必写 header）：

```json
{
  "mcpServers": {
    "nightingale": {
      "url": "http://localhost:8080"
    }
  }
}
```

若由 Cursor 通过请求头传 token 和/或夜莺地址（例如服务未设 `N9E_TOKEN`、或连到不同夜莺环境时）：

```json
{
  "mcpServers": {
    "nightingale": {
      "url": "http://localhost:8080",
      "headers": {
        "X-User-Token": "你的夜莺-api-token",
        "X-N9e-Base-Url": "http://your-n9e-server:17000"
      }
    }
  }
}
```

可只写其中一个 header；未写的项会使用服务端启动时的 `N9E_TOKEN` / `N9E_BASE_URL`。**若服务端以 `--shared` 启动，则这些 header 会被忽略，请勿依赖。**

若 MCP 服务前还有网关等需要认证，可同时配置对应 headers（如 `Authorization: Bearer your-gateway-token`）。服务端仅用 `X-User-Token` 作为调用夜莺 API 的凭证。
#### Docker

官方代码默认采用 `stdio` 协议进行进程间通信。若需要对接 Dify、FastGPT 等仅支持网络调用 (HTTP/SSE) 的大模型编排平台，推荐使用我们提供的 Docker Compose 方案。底层将自动引入 `mcp-proxy` 桥接器实现网络协议支持。

**部署步骤：**

1. 克隆本仓库到本地：

   ```bash
   git clone -b main https://github.com/n9e/n9e-mcp-server.git
   cd n9e-mcp-server/docker
   ```

2. 修改 `docker-compose.yml` 中的配置：

   - `MCP_VERSION`: (可选) 默认 Dockerfile 会在构建镜像时通过 NPM 安装 `@n9e/n9e-mcp-server`。请显式指定版本号，如 `0.1.1`；如需安装最新版，请填写 `latest`，不要留空。
   - `N9E_BASE_URL`: 替换为夜莺的实际 API 地址。
   - `N9E_TOKEN`: 替换为您生成的 API Token。

3. 启动服务：

   ```bash
   # 默认将通过 NPM 安装指定版本的 MCP Server 并启动
   docker compose up -d --build
   ```

4. 在 Dify 的 MCP 插件中配置：

   - **连接方式**：基于 HTTP 的流式连接 (SSE)
   - **URL**：`http://<您的服务器IP>:8082/sse`

> **开发者说明**：若您修改了源代码并希望构建本地测试镜像，请在 `docker-compose.yml` 中将 `dockerfile` 字段修改为 `docker/Dockerfile.source`。该 Dockerfile 会拷贝仓库源码并在镜像中编译 Go 服务端，同时仍通过 `mcp-proxy` 桥接器启动服务。

### 3.重启 Cursor 等进程，即可使用

## 可用工具

工具按两档安全等级组织:

- **read** — 始终注册
- **write** — `--read-only` 关闭时注册(create/update/import/clone/toggle 等)

完整 toolset 列表: `alerts、targets、datasource、mutes、busi_groups、notify_rules、alert_subscribes、event_pipelines、users、metrics、logs、dashboards、roles`,可通过 `N9E_TOOLSETS` 缩减。

下面只列出新增/扩展的部分;原有工具(targets、busi_groups、event_pipelines、alert_subscribes、mutes)保持不变。

### `alerts`(扩展)

| 工具 | 等级 | 说明 |
|---|---|---|
| `list_active_alerts` / `get_active_alert` / `list_history_alerts` / `get_history_alert` / `list_alert_rules` / `get_alert_rule` | read | 已有读工具 |
| `create_alert_rule` / `update_alert_rule` | write | 单条 CRUD |
| `import_alert_rules` / `import_prom_rules` | write | 批量导入(n9e JSON 或 Prometheus YAML) |
| `clone_alert_rules_to_bgs` / `toggle_alert_rules` | write | 跨业务组克隆 / 批量启停(走 `PUT .../fields`) |

### `dashboards`(新)

| 工具 | 等级 | 说明 |
|---|---|---|
| `list_dashboards` / `get_dashboard` / `get_dashboard_pure` | read | 列表 + 元数据 + 完整面板 JSON |
| `create_dashboard` / `update_dashboard_meta` / `update_dashboard_panels` | write | 元数据与面板分开更新 |
| `set_dashboard_public` / `clone_dashboard` | write | 可见性切换 + 克隆 |

### `notify_rules`(扩展,含 channels + templates)

| 工具 | 等级 | 说明 |
|---|---|---|
| `list_notify_rules` / `get_notify_rule` / `list_notify_channels` / `get_notify_channel` / `list_notify_templates` | read | |
| `create_notify_rules` / `update_notify_rule` / `test_notify_rule` | write | 规则 CRUD + 干跑测试 |
| `create_notify_channel` / `update_notify_channel` | write | 通道 CRUD |
| `create_notify_template` / `update_notify_template` / `update_notify_template_content` | write | 模板 CRUD |

### `datasource`(扩展)

| 工具 | 等级 | 说明 |
|---|---|---|
| `list_datasources` / `list_datasources_full` / `get_datasource` / `list_datasource_plugins` | read | brief / 完整 / 插件类型 |
| `upsert_datasource` | write | 创建或更新一体化(n9e 在 upsert 内部做连通性校验,无独立 test 端点) |
| `set_datasource_status` | write | 批量启停 |

### `users`(扩展)

| 工具 | 等级 | 说明 |
|---|---|---|
| `list_users` / `get_user` / `list_user_groups` / `get_user_group` | read | |
| `create_user` / `update_user_profile` / `reset_user_password` | write | profile.roles 字段即角色分配(n9e 没有独立分配端点) |
| `create_user_group` / `update_user_group` / `add_user_group_members` | write | 团队管理 |

### `roles`(新)

| 工具 | 等级 | 说明 |
|---|---|---|
| `list_roles` / `list_operations` / `list_role_operations` | read | |
| `create_role` / `update_role` / `bind_role_operations` | write | bind 是替换语义,不是增量 |

### `metrics`(新,Prom 查询代理)

| 工具 | 等级 | 说明 |
|---|---|---|
| `query_instant` | read | `/api/v1/query` |
| `query_range` | read | 自动调整 `step` 让结果点数 ≤ `max_points`(默认 1000),被降采样时返回 `truncated:true` |
| `query_label_values` / `query_series` | read | |

### `logs`(新)

| 工具 | 等级 | 说明 |
|---|---|---|
| `query_logs` | read | n9e 插件分发的 `/logs-query`(Loki/ES/OS 通吃);默认 `limit=200`;时间跨度 > 7 天直接拒绝 |
| `list_log_indices` / `list_log_fields` | read | ES(默认)或 OpenSearch(`engine=os`) |

## 示例提示词

配置完成后，您可以使用自然语言与夜莺交互：

- "显示过去 24 小时内所有紧急告警"
- "当前有哪些告警正在触发？"
- "列出所有离线超过 5 分钟的监控目标"
- "业务组 1 配置了哪些告警规则？"
- "由于维护原因，为 service=api 的告警创建一个 2 小时的屏蔽规则"
- "查看事件流水线的执行历史"
- "运维团队有哪些成员？"

## 配置

### 运行模式

- **stdio**（默认）：通过 stdin/stdout 进行 MCP 通信。适用于 Cursor 等会拉起服务进程的客户端。
- **http**：通过 HTTP 使用 MCP streamable 传输（仅 JSON 请求/响应，无 SSE）。使用 `n9e-mcp-server http` 启动，客户端需支持 streamable HTTP（如 `StreamableClientTransport`）。

### 环境变量

| 变量 | 命令行参数 | 说明 | 默认值 |
|-----|-----------|------|-------|
| `N9E_TOKEN` | `--token` | 夜莺 API Token（必需） | - |
| `N9E_BASE_URL` | `--base-url` | 夜莺 API 地址 | `http://localhost:17000` |
| `N9E_READ_ONLY` | `--read-only` | 仅注册 read 工具,屏蔽全部 write 工具 | `false` |
| `N9E_TOOLSETS` | `--toolsets` | 启用的工具集（逗号分隔） | `all` |
| `N9E_LISTEN` | `--listen` | HTTP 模式：监听地址 | `:8080` |
| `N9E_SESSION_TIMEOUT` | `--session-timeout` | HTTP 模式：空闲会话超时（0 表示不超时） | `0` |
| `N9E_SHARED` | `--shared` | HTTP 模式：为 true 时启动必须提供 N9E_TOKEN 和 N9E_BASE_URL，并忽略客户端 header | `false` |

### 工具集选择

默认启用所有工具集。可以通过 `--toolsets` 参数或 `N9E_TOOLSETS` 环境变量只启用需要的工具集，减少暴露给 AI 助手的工具数量，节省上下文窗口的 token 消耗。

可用工具集：`alerts`、`targets`、`datasource`、`mutes`、`busi_groups`、`notify_rules`、`alert_subscribes`、`event_pipelines`、`users`、`metrics`、`logs`、`dashboards`、`roles`

例如，只启用告警和监控目标相关工具：

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

## 开源协议

Apache License 2.0

## 相关项目

- [Nightingale](https://github.com/ccfos/nightingale) - 企业级云原生监控系统
- [MCP Go SDK](https://github.com/modelcontextprotocol/go-sdk) - 官方 MCP Go SDK
- [MCP 规范](https://modelcontextprotocol.io/) - Model Context Protocol 规范
