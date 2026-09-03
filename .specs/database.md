# 数据库

支持 PostgreSQL（pgx v5）和 MySQL（go-sql-driver/mysql），通过 `database/sql` 访问。Agent 只引用 `environment/connection`，不传 host、密码或 DSN Secret。

Tool：`db_list_connections`、`db_ping`、`db_list_tables`、`db_describe_table`、`db_query`、`db_explain`、`db_prepare_execute`、`db_confirm_execute`。Prepare 生成待确认 operation；Confirm 校验 operation_id、过期、环境、资源、target、SQL hash 和当前 Policy，禁止替换 SQL。

Parser 测试逐步覆盖 DML/DDL、CTE、多 statement、注释、大小写、换行、子查询、JOIN、RETURNING；`WITH ... DELETE` 必须按 AST 最终类型处理。
