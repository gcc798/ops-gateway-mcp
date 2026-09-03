# 测试规范

Policy 最高优先级，table-driven 覆盖 SELECT=ALLOW、UPDATE/DELETE + WHERE=CONFIRM、无 WHERE=DENY、DROP/TRUNCATE=DENY，并覆盖空白、大小写、注释、CTE、多 statement、子查询等绕过场景。

Config 覆盖默认路径、环境变量、目录创建、Gateway YAML、SQLite 资源、invalid config、reload/snapshot；HTTP 使用 `httptest` 覆盖正常、坏输入、应用错误和 Secret 不泄露；MCP 覆盖 schema、参数校验、Policy 和 Service 边界。

检查命令：`gofmt -w .`、`go test ./...`、`go vet ./...`、`cd web && pnpm build`、`make build`。
