# Audit 与可观测

Application Log 使用 JSON `slog`，写入 `AI_OPS_GATEWAY_LOGS`，按自然日生成 `ai-ops-gateway-YYYY-MM-DD.log`。字段包括 time、level、request_id、operation_id、client、tool、environment、resource、action、policy_decision、status、duration_ms、error，禁止 Secret。

Audit 写入 `AI_OPS_GATEWAY_DATA/audit.db`，规划 operations、confirmations、config_changes；operations 包含 operation_id、request_id、时间、client/tool、环境、资源、action/target、risk、decision/reason、status、影响行数、耗时、statement_hash、error。SQL 原文由 `store_statement`/`redact_literals` 控制，默认脱敏。

Metrics 使用 Prometheus 官方 Go client，默认监听 `127.0.0.1:9464/metrics`，当前提供按 method/route/status 的请求数与耗时；禁止 SQL、request_id、operation_id、完整 path、pod/table/user arbitrary value 作高基数 label。Trace 仅预留到后续版本。
