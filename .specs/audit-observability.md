# Audit 与可观测

Application Log 使用 JSON `slog`，写入 `OPS_GATEWAY_MCP_LOGS`，按自然日生成 `ops-gateway-mcp-YYYY-MM-DD.log`。字段包括 time、level、request_id、operation_id、client、tool、environment、resource、action、policy_decision、status、duration_ms、error，禁止 Secret。

Audit 与资源描述统一写入 OPS_GATEWAY_MCP_DATA/ops-gateway-mcp.db；资源表与 Audit 表逻辑隔离。operations 包含 operation_id、request_id、时间、client/tool、环境、资源、action/target、risk、decision/reason、status、影响行数、耗时、statement_hash、resource_revision、confirmed_by、error。client/confirmed_by 来自认证 Token 名称，不采信调用方自报身份；共享 Token 无法区分开发者个人。SQL 原文保存在 pending_operations，确认完成后删除；pending 详情可明文查看。

Audit 列表按日期、工具、资源类型等条件在 SQLite 分页，索引覆盖时间以及 tool/resource_type + 时间；页内稳定排序，总数与记录在同一读事务中获取。Overview 使用独立全库聚合，不从分页结果计算总量。

表结构由统一 goose 迁移管理，Audit Store 不再自行创建或补列。sqlx 按 `db` 标签映射字段并使用命名参数写入；时间通过内部行类型转换为 RFC3339Nano 文本，保持 API 时间格式。提交操作与冻结参数仍在同一事务中写入。

MCP 工具调用及 REST 资源读取/测试均记录 Audit。SQL 查询由 Service 记录 Policy 决策，写操作由 Service 记录准备、执行和完成生命周期。完成记录使用独立短超时，即使客户端断开也尝试落盘；MCP Audit 写入失败返回错误，REST 已返回响应后的完成记录失败写应用日志。外部资源执行与 SQLite 不是分布式事务，崩溃时 executing 可能需要人工核对，不能自动重试。

资源表新增、更新、删除在运行期间被发现时写入 operations 审计，记录资源类型、名称和状态变化；不记录 DSN、密码、Token 或凭据原文。

资源变更以 5 秒轮询观察，指纹包含全部配置但只保存哈希用于比较；不提供不可篡改审计或外部数据库直接修改者的身份追踪。

Metrics 使用 Prometheus 官方 Go client，默认监听 `127.0.0.1:9464/metrics`，当前提供按 method/route/status 的请求数与耗时；禁止 SQL、request_id、operation_id、完整 path、pod/table/user arbitrary value 作高基数 label。Trace 仅预留到后续版本。
