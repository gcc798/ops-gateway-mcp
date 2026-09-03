# 配置

环境变量：`AI_OPS_GATEWAY_CONF`、`AI_OPS_GATEWAY_LOGS`、`AI_OPS_GATEWAY_DATA`；默认分别为 `~/.ai-ops-gateway/conf`、`logs`、`data`，启动自动创建并校验目录。

数据库、Linux 和 Kubernetes 资源描述统一存储在 `AI_OPS_GATEWAY_DATA/ai-ops-gateway.db`。数据库资源只保存 `dsn_env` 环境变量名；Linux 资源只保存 `password_env` 或私钥路径；Kubernetes 资源只保存 kubeconfig 路径和 context，不保存 DSN、密码、私钥、Token 或 kubeconfig 原文。

`gateway.yaml` 仅承载 Gateway 应用配置，目前包括监听地址和 Metrics 地址；不从 YAML 导入资源。`yaml.v3` 只解析应用配置；ConfigManager 负责从 YAML 和 SQLite Load、Validate、Current、Reload、Snapshot。业务层只能依赖 typed snapshot。

在线 Save、连接热替换、版本历史与回滚留到后续版本；当前启动时从 SQLite 加载并校验资源，失败拒绝启动。
