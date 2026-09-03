# ai-ops-gateway

面向 AI Agent 的运维安全网关。Agent 负责思考，Gateway 负责最终 Policy 决策和实际执行。

当前 V0.1 提供 PostgreSQL/MySQL AST 安全策略、Linux/Kubernetes 固定运维能力、不可替换的确认流程、官方 MCP Streamable HTTP、SQLite Audit、JSON 日志、Prometheus Metrics 与真实数据管理端。

## 运行
```bash
make build
./ai-ops-gateway serve
```
打开 http://127.0.0.1:9095/；MCP 地址为 `http://127.0.0.1:9095/mcp`。运行时不需要 Node、pnpm 或 nginx。

路径环境变量为 `AI_OPS_GATEWAY_CONF`、`AI_OPS_GATEWAY_LOGS`、`AI_OPS_GATEWAY_DATA`，默认分别为 `~/.ai-ops-gateway/conf`、`logs`、`data`。

资源配置示例见 `configs/examples/`。配置只引用 DSN/密码环境变量或本地凭据文件，API、日志和 Audit 不返回 Secret。Metrics 默认为 `http://127.0.0.1:9464/metrics`。

本地数据库开发可选设置 `AI_OPS_GATEWAY_PG_DSN`、`AI_OPS_GATEWAY_MYSQL_DSN`；Gateway 只登记逻辑资源名，不向 API 返回 DSN。

Codex 配置：

```toml
[mcp_servers.ai_ops_gateway]
url = "http://127.0.0.1:9095/mcp"
```

## 开发
`make dev-backend`、`make dev-web`、`make build-web`、`make build`、`make test`、`make lint`。安全红线和按需设计文档见 `AGENTS.md` 与 `.specs/`。
