# 安全与 Policy

Gateway 是真正安全边界，不能依赖 Prompt、Agent 推理或文档。禁止万能 shell、`linux_exec(command)`、`kubectl(...)` 和 unrestricted SQL。执行链必须是 `Parser/Validator -> Policy -> Resource Adapter`。

SQL 必须 `Dialect Parser -> AST -> Policy`，正则、`strings.Contains`、`HasPrefix` 不能作最终边界。SELECT/SHOW/DESC/EXPLAIN=ALLOW；INSERT、UPDATE/DELETE + WHERE、CREATE、ALTER=CONFIRM；UPDATE/DELETE 无 WHERE、DROP、TRUNCATE=DENY。高危 DENY 不能靠普通配置改为 allow。

确认时 Gateway 保存冻结的 operation_id、SQL hash、target、resource、environment、资源配置指纹和过期时间；confirm 只能引用原操作并再次校验 Policy。环境从 SQLite 资源读取，忽略调用方填写的 environment。配置变化或资源删除后必须重新提交操作；旧确认单没有指纹时也必须重新提交。prod 比 dev/test 更严格。

本项目只面向受信任的内部开发人员，不作为公共服务。允许在 SQLite 明文存储 DSN、数据库/Linux 密码、SSH 私钥和 K8s Token；允许已认证用户通过 Web、REST、MCP 明文查看资源配置及读取结果中的任何敏感数据，不做字段掩码。资源响应使用 Cache-Control: no-store。资源凭据不提供旧版引用回退，查看配置不要求资源连接成功。

Linux/Kubernetes 支持读取远端身份有权限访问的任意文件，包括凭据文件；不增加路径目录白名单。Linux 仍校验绝对路径和参数引用，K8s 下载只执行固定 cat 命令；保留输出上限，禁止通过读取接口注入 shell 或修改、删除文件。

明文查看的授权不要求复制凭据到应用日志或 Audit。日志、审计、错误不主动记录 DSN、密码、私钥、Token 或 Authorization Header 原文；Gateway 访问 Token 继续只保存哈希。

MCP 仅提交待确认写操作，不提供 db_confirm_execute 或 ops_confirm。内部用户在 Web 操作详情核对 SQL/目标后，通过已有 REST scope 确认；不引入审批角色或 RBAC。审计记录认证 Token 名称和确认 Token 名称，共享 Token 不等同于个人身份。

HTTP 认证由 Gateway 中间件强制执行：`/mcp` 需要 `mcp` 或 `rest+mcp` scope，`/api/v1/*` 需要 `rest` 或 `rest+mcp` scope，`/healthz` 公开。授权表只保存 Token 哈希；Skill/Prompt 不作为安全边界。

不实现 RBAC、不可篡改审计、TLS、远程部署或 HA。默认单进程、单机、127.0.0.1。
