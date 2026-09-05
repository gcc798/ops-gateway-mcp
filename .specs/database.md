# 数据库

支持 PostgreSQL（pgx v5）和 MySQL（go-sql-driver/mysql），通过 `database/sql` 访问。Agent 只引用 `environment/connection`，不传 host、密码或 DSN Secret。

Gateway 自身 SQLite 使用 sqlx 的结构体映射与命名参数、goose 版本化迁移，升级流程见 `config.md`。远端业务数据库不引入 goose，不改变 SQL AST、Policy、行数和输出大小限制。

Tool：db_list_connections、db_ping、db_list_tables、db_describe_table、db_query、db_explain、db_prepare_execute。Prepare 生成待确认 operation；Web/REST Confirm 校验 operation_id、过期、真实环境、资源指纹、SQL hash 和当前 Policy，禁止替换 SQL。环境只取 SQLite 配置。资源列表支持分页、名称/环境/driver 筛选，并允许内部开发者明文查看 DSN。

Parser 测试逐步覆盖 DML/DDL、CTE、多 statement、注释、大小写、换行、子查询、JOIN、RETURNING；`WITH ... DELETE` 必须按 AST 最终类型处理。
