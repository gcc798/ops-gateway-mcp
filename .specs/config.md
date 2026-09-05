# 配置

工程名、Go module、MCP/REST 产品标识、命令和运行目录统一为 `ops-gateway-mcp`。仅支持 `OPS_GATEWAY_MCP_*` 环境变量，不提供旧名称回退。Web 存储键使用 `ops-gateway-mcp-` 前缀，改名后需重新登录，已有认证 Token 不变。Metrics 使用 `ops_gateway_mcp_` 前缀，客户端环境变量和监控查询需同步更新。

已有部署改名时必须先停服并备份，将原运行目录整体迁移至 `~/.ops-gateway-mcp`，数据库改名为 `data/ops-gateway-mcp.db`，同步修正资源中的 kubeconfig、known_hosts 等本地路径。保留凭据、认证、审计和历史备份，不重写历史日志内容；历史日志改名遇到重名时使用额外后缀，禁止覆盖。确认迁移完整后再启动，避免新建空库。

环境变量：`OPS_GATEWAY_MCP_LOGS`、`OPS_GATEWAY_MCP_DATA`；默认分别为 `~/.ops-gateway-mcp/logs`、`~/.ops-gateway-mcp/data`，启动自动创建两个目录。

数据库、Linux 和 Kubernetes 资源统一存储在 `OPS_GATEWAY_MCP_DATA/ops-gateway-mcp.db`。数据库支持完整明文 dsn；Linux 支持 password、private_key；Kubernetes 支持 kubeconfig 路径、context 和明文 token，token 非空时覆盖 kubeconfig 的身份认证。已认证内部开发者可查看这些字段。

资源凭据直接使用 dsn、password、private_key、token 字段，不支持环境变量引用或私钥路径回退。项目不接管无 goose 版本记录的旧库，不执行字段探测或兼容迁移；使用空数据库初始化。

## SQLite 升级与存储

`internal/storage.Open` 统一打开数据库并通过 goose 执行迁移；启动时有 5 分钟超时，迁移成功后才启动后台任务和 HTTP/MCP/Metrics。资源、Audit、认证共享一个 sqlx 连接池（最大 1 个连接），由启动入口统一关闭；各 Store 的 `New(db)` 不执行迁移，`Open(path)` 是独立打开数据库的便捷入口。查询结果在审计回调前关闭，不在持有事务时通过另一个 Store 重入连接池。

版本 1 的 `internal/storage/migrations/00001_initial.sql` 创建全部表结构，版本 2 的 SQL 文件创建列表索引。所有表结构完全由 goose SQL 迁移管理，不使用 Go 基线迁移。版本记录位于 `goose_db_version`，数据库版本高于程序支持版本时拒绝启动。

只有存在待执行迁移且已有业务表时才生成备份：使用 SQLite `VACUUM INTO` 创建同目录 `ops-gateway-mcp.db.pre-migrate-*.db` 一致性快照，权限 0600；备份失败不执行升级。正常重复启动不重复备份或执行迁移。备份含明文资源凭据，应按数据库同等保护并由运维人员管理保留周期。全新空库无需备份。

新增变更放在 `internal/storage/migrations/00003_<名称>.sql` 等递增版本文件中，包含 `-- +goose Up` 指令；迁移文件嵌入单二进制，无需额外安装 goose CLI。已发布的 SQL 迁移不可修改，只追加版本。保留机器指令原语法，其余说明性注释使用中文。

当前只支持单实例停机升级，不提供多进程迁移锁；升级前停止旧进程及其他写入者，不能同时启动两个版本。每个迁移独立事务，失败回滚当前迁移，之前成功版本保留，修复问题后重启继续。服务不提供自动 Down 或自动恢复；需要恢复时先停止所有数据库访问，核验备份，完整保留失败数据库及 WAL/SHM，再使用备份替换数据库并启动匹配版本，不能只覆盖主文件而保留旧 WAL。

不使用配置文件。Gateway 地址和 Metrics 地址使用固定默认值；资源表是 SQLite 中的唯一来源，资源每次按名称查询，客户端按需创建并复用。

资源客户端再次访问时检查 10 分钟闲置期限；每次执行前检查最新配置，更新/删除后失效旧缓存。没有后台闲置连接回收。资源变更每 5 秒检查一次并写 Audit，启动时建立基线；短于轮询间隔的外部修改可能无法观察到。数据库外部文件的内容更改不会改变配置字段指纹。

认证 Token 通过 `OPS_GATEWAY_MCP_MCP_TOKEN`、`OPS_GATEWAY_MCP_REST_TOKEN`、`OPS_GATEWAY_MCP_ADMIN_TOKEN` 注入，启动时仅将哈希写入 SQLite `api_tokens` 表。Token scope 为 `mcp`、`rest` 或 `rest+mcp`。

启动时未设置 Token 环境变量则保留数据库中的认证配置；显式设置为空字符串才撤销相应内置 Token，非空值更新 Token。Web 使用 sessionStorage 保存 Token，刷新时向 /auth/login 校验；退出删除保存值，API 401 清除会话。保留现有 Token scope，不引入 SSO 或 RBAC。
