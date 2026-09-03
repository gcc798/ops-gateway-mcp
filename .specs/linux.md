# Linux / SSH

使用 `golang.org/x/crypto/ssh`，不依赖本机 ssh 命令。Agent 只引用逻辑资源如 `prod/api-01`；Gateway 解析 host、port、user、认证和能力，私钥与密码不出 Agent 或浏览器。

Tool：list_hosts、system_info、disk_usage、memory_usage、processes、service_status、service_logs、restart_service、list_directory、read_file。restart_service 必须经 Policy，prod 推荐 CONFIRM。禁止任意 shell、rm、mkfs、dd、fdisk、userdel、任意 chmod/chown。

当前已实现 SSH Client/Manager、system-info、disk-usage、memory-usage、processes、service status/logs、目录/文件读取；restart service 生成冻结确认单，确认后执行固定 `sudo -n systemctl restart -- service`。SSH 强制校验 known_hosts，主机由 typed config 注册。
