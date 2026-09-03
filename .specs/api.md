# REST API

命名空间 `/api/v1`。端点：`GET /healthz`、`GET /api/v1/info`、`GET /api/v1/config/paths`、数据库/Linux/Kubernetes 列表与 test connection、`GET /policies`、Audit 列表与详情。

资源连接测试使用 `POST /api/v1/databases/test`、`POST /api/v1/linux/hosts/test`、`POST /api/v1/kubernetes/clusters/test`，请求为 `{name}`。

Kubernetes 只读端点：Pod/Deployment/Service 列表与详情、Pod logs，以及 `POST /api/v1/kubernetes/:cluster/namespaces/:namespace/pods/:pod/download`。下载请求只接收 `container` 与非空 `path`，V1 不限制读取路径，响应为二进制且有大小上限。

Linux 另有 system-info、disk/memory usage、processes、service status/logs、目录与文件读取。Linux service restart 与 Kubernetes rollout restart 的 POST 端点只创建 pending operation；`POST /api/v1/operations/:id/confirm` 才执行冻结动作。

Handler 只负责参数、Validate、调用 Application Service、返回响应；不得直接写 SQL、SSH、client-go 或 Policy 判断。错误逐步统一为 `{code,message,requestId}`，不回传堆栈和 Secret；列表接口应支持分页。
