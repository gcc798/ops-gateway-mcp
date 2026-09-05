# React Web

页面：Overview、Database、Linux、Kubernetes、Policy、Audit、Settings。采用简洁工程化的 HashiCorp 风格，支持 Light/Dark；dev/test/staging/prod 清晰区分，prod 有明显风险提示。

当前实际是 React 19 + 严格 TypeScript + Vite + Lucide，并使用 i18next/react-i18next 提供中英文切换；默认 zh-CN，语言偏好保存在 localStorage。使用 React hooks、原生表单和 datetime-local，不引入表格、状态或路由框架。pnpm build 先执行 tsc --noEmit，再执行 Vite。

Web Token 保存在 sessionStorage，刷新时重新校验，登出或 API 401 时清除。开发代理包含 /auth 和 /api。

目录按职责组织：`main.tsx` 仅负责启动；`app/` 负责应用外壳；`pages/` 按页面拆分；`components/` 放共享和领域组件；`hooks/` 管理认证与分页请求；`lib/` 封装 HTTP；`types/` 定义 API 类型；`config/` 定义资源页面配置。页面复用筛选、分页和详情组件，不在入口文件堆叠业务逻辑。

使用 Prettier 统一缩进、换行与引号：`pnpm format` 格式化，`pnpm format:check` 检查；不增加运行时依赖。

`make build-web` 构建成功后先清理旧的嵌入目录，再复制当前产物，避免历史哈希资源进入单二进制。

Database/Linux/Kubernetes 配置列表和 Audit 都由后端分页筛选；提供每页 10/20/50/100 条、上一页/下一页、总条数、查询和重置。切换筛选或每页数量回到第一页；请求切换时取消旧请求，显示 loading/error/empty，不展示旧结果。查看详情后返回保留列表条件和页码。

内部开发者可直接明文查看完整资源配置，包括密码、DSN、私钥和 Token。详情不自动测试连接；用户可单独测试并查看数据库表或 Linux 系统信息。K8s 展示 context、kubeconfig 路径和 Token。

Audit 详情展示真实字段、原因、request_id、Token 身份、确认身份、耗时、行数；待确认 SQL 展示冻结原文、到期时间和确认按钮。成功后重新读取服务端状态，失败有提示。Overview 使用全库统计，15 秒刷新，不把某一页当总量，不显示伪造在线状态或审计覆盖率。
