# 安全与 Policy

Gateway 是真正安全边界，不能依赖 Prompt、Agent 推理或文档。禁止万能 shell、`linux_exec(command)`、`kubectl(...)` 和 unrestricted SQL。执行链必须是 `Parser/Validator -> Policy -> Resource Adapter`。

SQL 必须 `Dialect Parser -> AST -> Policy`，正则、`strings.Contains`、`HasPrefix` 不能作最终边界。SELECT/SHOW/DESC/EXPLAIN=ALLOW；INSERT、UPDATE/DELETE + WHERE、CREATE、ALTER=CONFIRM；UPDATE/DELETE 无 WHERE、DROP、TRUNCATE=DENY。高危 DENY 不能靠普通配置改为 allow。

确认时 Gateway 保存冻结的 operation_id、SQL hash、target、resource、environment 和过期时间；confirm 只能引用原操作并再次校验 Policy。prod 比 dev/test 更严格。

密码、DSN 密码、SSH 私钥/密码、K8s Token、Authorization Header、API/Vault Token 和 Secret 原文禁止进入日志、Audit、响应、错误或 debug dump。
