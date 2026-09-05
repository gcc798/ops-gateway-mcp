# REST API

命名空间 `/api/v1`。端点：`GET /healthz`、`GET /api/v1/info`、`GET /api/v1/config/paths`、数据库/Linux/Kubernetes 列表与 test connection、`GET /policies`、Audit 列表与详情。

除 `/healthz` 外，`/api/v1/*` 需要 `Authorization: Bearer <token>`，并校验 Token 的 `rest` scope。

资源连接测试使用 `POST /api/v1/databases/test`、`POST /api/v1/linux/hosts/test`、`POST /api/v1/kubernetes/clusters/test`，请求为 `{name}`。

Kubernetes 只读端点：Pod/Deployment/Service 列表与详情、Pod logs，以及 `POST /api/v1/kubernetes/:cluster/namespaces/:namespace/pods/:pod/download`。下载请求只接收 `container` 与非空 `path`，V1 不限制读取路径，响应为二进制且有大小上限。

Linux 另有 system-info、disk/memory usage、processes、service status/logs、目录与文件读取。Linux service restart 与 Kubernetes rollout restart 的 POST 端点只创建 pending operation；`POST /api/v1/operations/:id/confirm` 才执行冻结动作。

Handler 只负责参数、Validate、调用 Application Service、返回响应；不得直接写 SQL、SSH、client-go 或 Policy 判断。错误逐步统一为 `{code,message,requestId}`，不回传堆栈和 Secret；列表接口应支持分页。

配置资源列表（REST 和 MCP）统一后端分页：page 默认 1，page_size 默认 20、范围 1–100，page 最大 1000000；非法值返回 400。响应为 `{items:[],total,page,page_size}`，默认按名称升序排序。REST 资源列表可指定 sort（name/environment，数据库另有 driver，Linux 另有 address/user，K8s 另有 context）与 order（asc/desc）；只接受白名单，非法值返回 400，同值以 name 升序稳定排序。空结果的 items 始终为数组，不返回 null。

- GET /api/v1/databases：name 包含匹配、environment 精确匹配、driver 精确匹配。
- GET /api/v1/linux/hosts：name、address、user 包含匹配，environment 精确匹配。
- GET /api/v1/kubernetes/clusters：name、context 包含匹配，environment 精确匹配。
- 各列表路径追加 /:name 可获取完整明文配置。列表 items 同样是资源对象；已认证内部用户可查看敏感值，Cache-Control: no-store。连接测试单独触发，不作为查看配置的前置条件。

GET /api/v1/audit/operations 使用相同分页结构，支持 from/to（RFC3339、必须带时区，开始含、结束不含）、tool、resource_type、resource、environment、client、decision、status 精确匹配。按时间降序、operation_id 降序稳定排序。GET /api/v1/audit/summary 返回全库 total/allow/confirm/deny/failed 统计。

GET /api/v1/audit/operations/:id 返回审计详情；pending 操作附 expires_at，SQL pending 附冻结 statement 明文。POST /api/v1/operations/:id/confirm 是 Web 使用的统一执行端点；沿用 REST scope。MCP 不提供确认入口。

此处分页针对配置资源和 Audit；远端 tables、pods、services 等现有只读接口契约保持不变。

GET /api/v1/audit/filter-options 返回 `{tools:[],clients:[]}`，从全部审计记录中去重、排除空字符串并按字典序排序。clients 为历史 Token 身份名称（包括已停用身份），不返回 Token 原文。端点要求 REST 认证，响应 Cache-Control: no-store；不受当前筛选或分页限制。
