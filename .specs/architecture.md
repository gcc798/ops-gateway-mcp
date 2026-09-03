# 总体架构

`ai-ops-gateway` 面向 Codex、Claude Code、Cursor 等 MCP Agent，第一阶段管理 PostgreSQL、MySQL、Linux、Kubernetes。

```text
Agent --MCP--> ai-ops-gateway <--REST/Web-- Admin
                         |
                 Application Service
                         |
                    Policy Engine
                 /          |          \
            Database      Linux      Kubernetes
```

当前保持单体、单二进制，不拆微服务、不引入 MQ/Redis；MCP 与 REST 复用 Application Service，真实执行前必须经过 Policy 和 Resource Adapter。Config Manager、Secret、Audit、Logging、Metrics、Tracing 为横切模块。
