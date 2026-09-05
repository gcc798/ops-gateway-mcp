# 总体架构

`ops-gateway-mcp` 面向 Codex、Claude Code、Cursor 等 MCP Agent，第一阶段管理 PostgreSQL、MySQL、Linux、Kubernetes。

```text
Agent --MCP--> ops-gateway-mcp <--REST/Web-- Admin
                         |
                 Application Service
                         |
                    Policy Engine
                 /          |          \
            Database      Linux      Kubernetes
```

当前保持单体、单二进制，不拆微服务、不引入 MQ/Redis；MCP 与 REST 复用 Application Service，真实执行前必须经过 Policy 和 Resource Adapter。Config Manager、Secret、Audit、Logging、Metrics、Tracing 为横切模块。

内部 SQLite 由 `internal/storage` 统一初始化、备份和 goose 迁移，资源、认证和审计 Store 使用共享 sqlx 连接池。sqlx 负责字段映射和命名参数，不增加 ORM 或通用 Repository 层。远端 PostgreSQL/MySQL 执行适配器继续使用 `database/sql`，不参与内部库迁移。
