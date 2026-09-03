# 配置

环境变量：`AI_OPS_GATEWAY_CONF`、`AI_OPS_GATEWAY_LOGS`、`AI_OPS_GATEWAY_DATA`；默认分别为 `~/.ai-ops-gateway/conf`、`logs`、`data`，启动自动创建并校验目录。

开发期可选使用 `AI_OPS_GATEWAY_PG_DSN` 与 `AI_OPS_GATEWAY_MYSQL_DSN` 注册名为 `postgres`、`mysql` 的本地测试资源；DSN 不得写入日志或 API。正式配置只保存 `dsn_env` 环境变量名，不保存 DSN 原文。

配置拆分为 `gateway.yaml`、`database/{env}.yaml`、`linux/{env}.yaml`、`kubernetes/{env}.yaml`。示例见 `configs/examples/`，分别复制到配置根目录及对应资源子目录。数据库配置使用 `driver` 与 `dsn_env`；Linux 使用 address/user、`password_env` 或 `private_key_path`，并强制提供 `known_hosts_path`；Kubernetes 使用 kubeconfig/context。`yaml.v3` 只解析；ConfigManager 负责 Load、Validate、Current、Reload、Snapshot。业务层只能依赖 typed snapshot。

在线 Save、热更新、版本历史与回滚留到 V0.2；V0.1 启动时加载并校验，失败拒绝启动。
