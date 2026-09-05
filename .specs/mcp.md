# MCP

Gateway 常驻运行，使用 Streamable HTTP `/mcp`；Codex 不负责生命周期。MCP Handler 只做参数解析/校验、调用 Application Service、封装 MCP Result。

`/mcp` 需要 `Authorization: Bearer <token>`，并校验 `mcp` scope；Skill 仅用于引导 Agent 使用 MCP，不能替代认证中间件。

V0.1 使用官方 modelcontextprotocol/go-sdk，当前注册 26 个 DB/Linux/Kubernetes 工具。db_confirm_execute 和 ops_confirm 不再注册，用户在 Web/REST 确认冻结操作。db_list_tables 必须有正确非空名称，测试校验工具名称唯一及确认工具未暴露。

db_list_connections、linux_list_hosts、k8s_list_clusters 接收 page/page_size、name、environment 和各自的 driver 或 address/user 或 context 条件。返回 value 为 {items,total,page,page_size}，items 是包含明文配置的资源对象。分页和筛选复用 REST 的 Service/Store。容器文件以 EmbeddedResource blob 返回，内部用户允许查看任意文件内容。

正确链路：MCP -> Application Service -> Policy -> Resource Manager/Adapter。不得直接连接 DB、SSH 或 client-go；REST 复用相同 Service。测试覆盖 schema、参数校验、ALLOW/CONFIRM/DENY 和绕过检查。
