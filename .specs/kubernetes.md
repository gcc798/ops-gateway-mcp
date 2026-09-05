# Kubernetes

使用官方 `client-go`，不调用 `kubectl`。Gateway 管理 kubeconfig、context、API server、凭据和 namespace 约束，Agent 只引用逻辑 cluster。

集群配置支持后端分页与名称、环境、context 筛选，已认证内部用户可查看完整配置及明文 Token。配置 Token 非空时覆盖 kubeconfig 认证（移除原有用户密码、证书和 exec 认证），客户端请求超时为 10 秒。配置变更或删除后，下次访问会失效旧客户端缓存。

Tool：list_clusters、get_pods/pod、logs、get_deployments/deployment、get_services、rollout_restart。读操作 ALLOW，rollout restart CONFIRM，prod 更严格；不提供 exec、apply、delete、任意 patch。

当前已实现 client-go 的集群注册、Pod/Deployment 列表与详情、Service 列表、Pod 日志读取；rollout restart 生成冻结确认单，确认后只 patch Deployment Pod template 的 restart annotation。任意 exec/apply/delete/patch 仍未开放。

`k8s_download_file` 是开发者使用的只读能力：通过 Kubernetes exec/stream 读取容器内任意非空路径，V1 不限制目录或文件类型。Gateway 只生成固定 `cat` 命令，不接收任意 shell command，也不提供修改或删除能力；输出默认限制 16 MiB 以保护 Gateway 内存。

由于 Kubernetes 没有通用的容器文件 API，目标容器必须包含固定读取程序（当前直接执行 `cat -- path`，不经过 shell）；distroless/scratch 镜像需要提供专用 debug/sidecar 方案后才能下载。
