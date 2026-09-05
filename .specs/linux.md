# Linux / SSH

使用 golang.org/x/crypto/ssh，不依赖本机 ssh 命令。Agent 引用逻辑资源，Gateway 解析 host、port、user 和认证；已认证内部开发者允许在资源详情明文查看密码和私钥。资源列表支持分页、name/environment/address/user 筛选。

Tool：list_hosts、system_info、disk_usage、memory_usage、processes、service_status、service_logs、restart_service、list_directory、read_file。restart_service 必须经 Policy，prod 推荐 CONFIRM。禁止任意 shell、rm、mkfs、dd、fdisk、userdel、任意 chmod/chown。

当前已实现 SSH Client/Manager、system-info、disk-usage、memory-usage、processes、service status/logs、目录/文件读取；restart service 生成冻结确认单，确认后执行固定 `sudo -n systemctl restart -- service`。SSH 强制校验 known_hosts，主机由 typed config 注册。

文件读取不限制目录和敏感内容；只要求绝对路径、不含 .. 路径分段，并正确引用参数。输出限制不变。SSH 握手和固定命令执行设置超时。确认在 Web/REST 完成，不提供 MCP 自确认。
