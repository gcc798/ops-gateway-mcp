# ai-ops-gateway

面向内部开发人员与 AI Agent 的运维安全网关。Agent 负责思考，Gateway 负责最终 Policy 决策和实际执行。允许内部用户明文查看敏感资源配置和任意文件；不作为公共服务。

当前 V0.1 提供 PostgreSQL/MySQL AST 安全策略、Linux/Kubernetes 固定运维能力、不可替换的确认流程、官方 MCP Streamable HTTP、SQLite Audit、JSON 日志、Prometheus Metrics 与真实数据管理端。

## 运行
```bash
make build
./ai-ops-gateway serve
```
打开 http://127.0.0.1:9095/；MCP 地址为 `http://127.0.0.1:9095/mcp`。运行时不需要 Node、pnpm 或 nginx。

路径环境变量为 `AI_OPS_GATEWAY_LOGS`、`AI_OPS_GATEWAY_DATA`，默认分别为 `~/.ai-ops-gateway/logs`、`~/.ai-ops-gateway/data`。

接口认证使用 Bearer Token。启动前设置 `AI_OPS_GATEWAY_MCP_TOKEN`、`AI_OPS_GATEWAY_REST_TOKEN` 或 `AI_OPS_GATEWAY_ADMIN_TOKEN`；Token 哈希存入 SQLite，原文不落盘。`mcp` Token 只能访问 `/mcp`，`rest` Token 只能访问 `/api/v1/*`，`admin` Token 可访问两者；`/healthz` 保持公开。示例：`export AI_OPS_GATEWAY_MCP_TOKEN=$(openssl rand -hex 32)`。

Database、Linux 和 Kubernetes 配置存储在 AI_OPS_GATEWAY_DATA/ai-ops-gateway.db，支持完整明文 DSN、密码、SSH 私钥和 K8s Token，不提供旧环境变量或私钥路径回退。Web 详情可直接查看明文；日志、Audit 不复制凭据。客户端按需创建，执行前检查配置变化，下一次访问时淘汰闲置超过 10 分钟的连接。Metrics 默认为 http://127.0.0.1:9464/metrics。

三类配置资源与 Audit 均支持后端分页筛选。资源按名称、环境及类型相关字段检索；Audit 按日期、工具、资源类型、名称、身份和状态检索。REST 列表返回 {items,total,page,page_size}，默认每页 20、最多 100。MCP 三个资源列表 Tool 的 value 使用相同结构。

Web 登录在同一标签页刷新后保留，退出清除。MCP 写操作只创建待确认记录，用户在 Web 的 Audit 详情核对并执行；不提供 MCP 自确认 Tool。沿用 REST/MCP Token scope，不做 RBAC、不可篡改审计、TLS、远程部署或 HA。

首次启动会自动创建资源表，示例 SQL 见 `configs/examples/resources.sql`。

内部 SQLite 已接入 goose 版本化升级和 sqlx，初始表结构完全由 SQL 迁移创建；不兼容无版本旧库，请使用新的空数据目录。升级时先停止旧进程，再启动新二进制；启动会先备份已有待升级数据库、执行迁移，成功后才监听服务。备份为数据目录内的 `ai-ops-gateway.db.pre-migrate-*.db`，含明文凭据，权限 0600；请妥善保管。升级失败或数据库版本过新时拒绝启动，不自动降级。新增迁移与恢复注意事项见 [.specs/config.md](.specs/config.md)。

Codex 配置：

```toml
[mcp_servers.ai_ops_gateway]
url = "http://127.0.0.1:9095/mcp"
bearer_token_env_var = "AI_OPS_GATEWAY_MCP_TOKEN"
```

## 开发
执行 `make dev` 会先构建前端并启动网关，浏览器访问 `http://127.0.0.1:9095/` 即可；不需要单独启动前端。执行 `make stop` 停止网关。`make dev-backend` 和 `make dev-web` 仅用于分别调试后端或前端。其他命令：`make build-web`、`make build`、`make test`、`make lint`。安全红线和按需设计文档见 `AGENTS.md` 与 `.specs/`。
