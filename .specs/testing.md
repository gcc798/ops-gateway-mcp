# 测试规范

Policy 最高优先级，table-driven 覆盖 SELECT=ALLOW、UPDATE/DELETE + WHERE=CONFIRM、无 WHERE=DENY、DROP/TRUNCATE=DENY，并覆盖空白、大小写、注释、CTE、多 statement、子查询等绕过场景。

Config 覆盖 SQLite 初始化与版本升级、事务失败回滚、明文字段映射、配置缓存失效。资源和 Audit 测试覆盖超过 50 条分页、组合筛选、日期边界、空结果、非法分页和 SQL 注入式筛选值。HTTP 使用 httptest 覆盖认证、已授权明文读取、错误输入；MCP 验证非空唯一名称、分页契约、确认 Tool 不暴露。前端 build 必须先通过严格 TypeScript 检查。

检查命令：`gofmt -w .`、`go test ./...`、`go vet ./...`、`cd web && pnpm format:check && pnpm build`、`make build`。

`internal/storage` 覆盖空库初始化、已版本化数据库升级、备份内容和权限、重复启动、较新数据库拒绝启动、SQL 迁移事务回滚和取消启动。Audit 覆盖全部字段的 sqlx 往返映射。迁移测试只使用临时 SQLite，不接触开发者实际资源库。

浏览器回归使用 `GATEWAY_BROWSER_TEST_ADDR=127.0.0.1:19095 go test ./internal/server -run TestBrowserFixture -v -timeout 20m`，服务只创建临时 SQLite 与测试 Token，不操作实际资源库。先执行前端构建，在浏览器登录 `browser-test-token` 并进入 Database 页，再通过 Playwright CLI `run-code --filename web/tests/browser-smoke.js` 检查排序、分页、筛选、详情刷新、浏览器历史、关联审计、确认失败/过期、明文复制、主题与移动端。确认失败连接固定本机不可连接地址，不访问真实数据库；成功执行仍由后端相关测试和实际资源环境验证。
