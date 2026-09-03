# MCP

Gateway 常驻运行，使用 Streamable HTTP `/mcp`；Codex 不负责生命周期。MCP Handler 只做参数解析/校验、调用 Application Service、封装 MCP Result。

V0.1 使用官方 `modelcontextprotocol/go-sdk`，当前注册 28 个 DB/Linux/Kubernetes/确认工具。容器文件以 MCP EmbeddedResource blob 返回；`ops_confirm` 只能执行 Gateway 已冻结的 Linux/Kubernetes restart 操作。

正确链路：MCP -> Application Service -> Policy -> Resource Manager/Adapter。不得直接连接 DB、SSH 或 client-go；REST 复用相同 Service。测试覆盖 schema、参数校验、ALLOW/CONFIRM/DENY 和绕过检查。
