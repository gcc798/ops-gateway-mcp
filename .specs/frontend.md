# React Web

界面品牌文案为单行 `Ops Gateway`。侧栏、登录页和 favicon 共用洋红方底的门框、通路与中央节点标识，32px 图标搭配中等偏粗字重文字。仓库、二进制和 API 产品标识不变。

页面：Overview、Database、Linux、Kubernetes、Policy、Audit、Settings。采用简洁工程化的 HashiCorp 风格，品牌采用 Consul 启发的洋红色、中性灰底与独立 Gateway 标识，状态色与品牌色分离；默认 Light，支持 Light/Dark 并持久化主题偏好；dev/test/staging/prod 清晰区分，prod 有明显风险提示。

当前实际是 React 19 + 严格 TypeScript + Vite + Lucide，并使用 i18next/react-i18next 提供中英文切换；默认 zh-CN，语言偏好保存在 localStorage。使用 React hooks、原生表单和 react-datepicker 日期时间组件（date-fns 本地化），不引入表格和状态框架；使用 React Router HashRouter，保证嵌入静态服务直接刷新详情链接。pnpm build 先执行 tsc --noEmit，再执行 Vite。

Web Token 保存在 sessionStorage，刷新时重新校验，登出或 API 401 时清除。开发代理包含 /auth 和 /api。

目录按职责组织：`main.tsx` 仅负责启动；`app/` 负责应用外壳；`pages/` 按页面拆分；`components/` 放共享和领域组件；`hooks/` 管理认证与分页请求；`lib/` 封装 HTTP；`types/` 定义 API 类型；`config/` 定义资源页面配置。页面复用筛选、分页和详情组件，不在入口文件堆叠业务逻辑。

使用 Prettier 统一缩进、换行与引号：`pnpm format` 格式化，`pnpm format:check` 检查。

Audit 日期支持日历、月份/年份选择、秒级时间编辑和清除；显示本地时间，提交带时区的 RFC3339。工具/REST 路由、Token 身份下拉从全库审计记录去重读取，不局限当前页，不展示 Token 原文。刷新按钮重新获取选项；加载失败显示错误，已有 URL 筛选值仍保留。

`make build-web` 构建成功后先清理旧的嵌入目录，再复制当前产物，避免历史哈希资源进入单二进制。

Database/Linux/Kubernetes 配置列表和 Audit 都由后端分页筛选；提供每页 10/20/50/100 条、上一页/下一页、总条数、查询和重置。切换筛选或每页数量回到第一页；请求切换时取消旧请求，显示 loading/error/empty，不展示旧结果。查看详情后返回保留列表条件和页码。

所有分页复用 Radix Select 的每页数量选择器，数字使用等宽数值和统一字号；弹层跟随深浅主题，支持方向键、Enter、Escape 与焦点恢复，靠近视口边缘自动调整位置。后端分页契约不变。

内部开发者可直接明文查看完整资源配置，包括密码、DSN、私钥和 Token。详情不自动测试连接；用户可单独测试并查看数据库表或 Linux 系统信息。K8s 展示 context、kubeconfig 路径和 Token。

Audit 详情展示真实字段、原因、request_id、Token 身份、确认身份、耗时、行数；待确认 SQL 展示冻结原文、到期时间和确认按钮。成功后重新读取服务端状态，失败有提示。Overview 使用全库统计，15 秒刷新，不把某一页当总量，不显示伪造在线状态或审计覆盖率。

页面路径使用 `#/overview`、`#/database/:id?`、`#/linux/:id?`、`#/kubernetes/:id?`、`#/audit/:id?`、`#/policy`、`#/settings`。筛选、分页和资源排序保存在 URL 查询参数，刷新与浏览器前进后退可恢复；从详情返回保留列表条件。资源名称通过 URL 编码处理。

资源列表点击列头进行后端排序，不在单页本地排序。资源详情分为基本信息、明文配置与最近连接测试，字段可复制并有结果反馈；测试只代表本次检查，不显示持续在线状态。资源详情可跳转精确筛选的相关审计，审计详情可返回对应资源。

Overview 移除宣传区，展示配置资源数、待确认记录数、全历史决策统计和最近操作，统计可跳转筛选列表。待确认数遵循后端 pending 状态，可能包含尚未被服务端转为 expired 的记录；详情根据到期时间禁用执行按钮。Audit 提供全部/待确认/失败快捷视图，详情分组呈现目标、请求、策略和结果。确认期间禁用按钮，完成后读取真实状态，失败显示错误并重取详情。

Policy 从现有 `/api/v1/policies` 读取后端默认规则摘要（不是可编辑规则引擎）；Settings 分开呈现界面偏好与运行时路径。移动端导航横向滚动，宽表仅在表格容器内滚动。
