# ai-ops-gateway Agent 指南

## 项目定位
`ai-ops-gateway` 是面向 Codex、Claude Code、Cursor 等 MCP Agent 的运维安全网关，第一阶段覆盖 PostgreSQL、MySQL、Linux 和 Kubernetes。

核心原则：**Agent thinks. Gateway decides and executes.** Agent 负责理解、推理、选 Tool 和分析结果；Gateway 负责资源定位、Policy、确认、执行、Secret、Audit、日志和可观测。安全不能依赖 Prompt 或 Agent 自觉。

## 架构与安全红线
当前保持单体、单二进制：MCP / REST / Web → Application Service → Policy Engine → Resource Adapter。入口不得各自实现业务逻辑，真实资源执行前必须经过 Policy，适配器不能被绕过。详见 `.specs/architecture.md`。

- 不新增万能 shell、`kubectl` 或 unrestricted SQL Tool。
- SQL 安全边界必须是 Dialect Parser + AST + Policy，不能用正则或字符串扫描作最终判断。
- UPDATE/DELETE 无 WHERE、DROP、TRUNCATE 默认 DENY；写操作默认 CONFIRM，读操作默认 ALLOW。
- 危险操作由 Gateway 强制控制，不能由 Agent 绕过；prod 比 dev/test 更严格。
- 密码、DSN 密码、SSH 私钥、K8s Token、Authorization Header 和 Secret 原文不得进入日志、Audit、HTTP/MCP 响应或错误文本。
- 未经明确安全设计，不增加任意删除、exec、apply、patch 等能力。

## 技术栈与发布
当前实际依赖：Go 1.26、Echo v5.3.1、官方 MCP Go SDK、Cobra、database/sql、pgx v5、go-sql-driver/mysql、x/crypto/ssh、client-go、yaml.v3、modernc SQLite、Prometheus client_golang，以及 React 19、TypeScript、Vite、Lucide。

`pnpm build` 生成 `web/dist`；`go build` 将前端嵌入单个 `ai-ops-gateway`。运行时只需 `./ai-ops-gateway serve`，地址为 `127.0.0.1:9095`，端点包括 `/`、`/api/v1/...`、`/mcp` 和 `/healthz`。

## 按需阅读 `.specs/`
不要每次读取全部文档：架构读 `architecture.md`；安全、Policy、确认、Secret 读 `safety.md`；配置读 `config.md`；数据库读 `database.md` + `safety.md`；Linux/Kubernetes 读各自文档 + `safety.md`；MCP 读 `mcp.md` + `safety.md`；日志/Audit/Metrics 读 `audit-observability.md`；React 读 `frontend.md`；REST 读 `api.md`；测试读 `testing.md`；版本边界读 `roadmap.md`。涉及 SQL、Linux/K8s 修改、Policy、确认或 Secret 时必须读 `safety.md`。

## 开发约束
Go 使用 idiomatic Go、Echo v5 当前 API、`context.Context`、可取消的外部 I/O、`database/sql` 和 `slog`；不使用 GORM/Viper，不为单一实现提前造接口或大量分层。前端使用严格 TypeScript、React function component/hooks，避免滥用全局状态。

如果架构、配置、Policy、安全边界、MCP/REST 合约、Audit schema、环境变量、构建方式或关键前端交互改变，必须同步更新对应 `.specs/*.md`；普通内部重构无需机械改文档。

## 完成检查
```bash
gofmt -w .
go test ./...
go vet ./...
cd web && pnpm build
make build
```
