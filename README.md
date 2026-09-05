# Ops Gateway MCP

面向企业开发者的运维安全 MCP Server。

Ops Gateway MCP 为数据库、Linux 和 Kubernetes 操作提供统一入口，在真实执行前强制经过 Policy、确认和审计。

> 仅面向受信任的内部开发人员部署，不作为公共服务。

## 特性

- MCP、REST 和 Web 使用同一套执行链
- PostgreSQL / MySQL SQL AST Policy
- Linux、Kubernetes 固定只读运维工具
- 危险写操作默认确认，高危操作拒绝
- SQLite 存储资源、认证和 Audit
- Web 明文查看内部资源配置
- Goose 迁移、sqlx 字段映射
- 单二进制部署

## 快速开始

构建需要 Go 1.26、支持当前 Vite 的 Node.js、pnpm，以及 C 编译器（SQL Parser 使用 CGO）。运行时只需一个二进制。

```bash
export OPS_GATEWAY_MCP_REST_TOKEN="$(openssl rand -hex 32)"
export OPS_GATEWAY_MCP_MCP_TOKEN="$(openssl rand -hex 32)"

make build
./ops-gateway-mcp serve
```

使用 `OPS_GATEWAY_MCP_REST_TOKEN` 的值登录 Web。上述 Token 生成命令仅用于首次配置；后续启动不设置 Token 环境变量即可保留已有认证，显式设置为空会撤销对应 Token。

| 地址 | 用途 |
| --- | --- |
| http://127.0.0.1:9095/ | Web 控制台 |
| http://127.0.0.1:9095/mcp | MCP Streamable HTTP |
| http://127.0.0.1:9464/metrics | Prometheus Metrics |
| http://127.0.0.1:9095/healthz | 健康检查 |

## MCP 配置

```toml
[mcp_servers.ops_gateway]
url = "http://127.0.0.1:9095/mcp"
bearer_token_env_var = "OPS_GATEWAY_MCP_MCP_TOKEN"
```

## 配置

| 环境变量 | 默认值 | 说明 |
| --- | --- | --- |
| `OPS_GATEWAY_MCP_DATA` | `~/.ops-gateway-mcp/data` | SQLite 数据目录 |
| `OPS_GATEWAY_MCP_LOGS` | `~/.ops-gateway-mcp/logs` | 日志目录 |
| `OPS_GATEWAY_MCP_MCP_TOKEN` | 无 | MCP Token |
| `OPS_GATEWAY_MCP_REST_TOKEN` | 无 | REST/Web Token |
| `OPS_GATEWAY_MCP_ADMIN_TOKEN` | 无 | REST + MCP Token |

资源配置示例见 [`configs/examples/resources.sql`](configs/examples/resources.sql)。内部 SQLite 数据库为 `ops-gateway-mcp.db`，资源凭据按当前内部部署约定明文保存，认证 Token 只保存哈希。启动时 goose 自动执行版本化迁移。

已有部署需停服备份后迁移运行目录、数据库文件及资源中的本地路径，详见 [配置与升级](.specs/config.md)。环境变量仅支持上述新名称，Web 需重新登录，已有 Token 不变。

首次启动后，将示例中的连接配置替换为自己的值再写入资源表；Web 目前用于查看配置、连接测试和操作确认，不提供资源编辑表单。

## 安全边界

| 操作 | 默认处理 |
| --- | --- |
| AST 校验通过的只读 SQL、固定读取工具 | 允许 |
| 带 WHERE 的 UPDATE/DELETE、受支持的写入与重启 | 人工确认 |
| DROP、TRUNCATE、无 WHERE 的 UPDATE/DELETE | 拒绝 |

生产环境策略更严格。MCP 不能自行确认写操作，开发者须在 Web Audit 详情核对后执行。资源凭据和有权限读取的文件可明文查看，但日志与审计不主动记录凭据。

当前仅监听本机，未实现远程部署、TLS、RBAC 或 HA，不应直接暴露到公网。升级前停止旧实例；goose 在存在待执行迁移时备份已有数据库，失败拒绝启动。详见 [配置与升级](.specs/config.md)、[安全规则](.specs/safety.md)。

## 开发

```bash
make dev       # 构建前端并启动
make test      # Go 测试
make lint      # go vet
make fmt       # gofmt
cd web && pnpm build
```

架构、安全边界、API 和数据库说明见 [`.specs/`](.specs/)；贡献前请阅读 [`AGENTS.md`](AGENTS.md)。
