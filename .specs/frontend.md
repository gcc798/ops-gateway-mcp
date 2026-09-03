# React Web

页面：Overview、Database、Linux、Kubernetes、Policy、Audit、Settings。采用简洁工程化的 HashiCorp 风格，支持 Light/Dark；dev/test/staging/prod 清晰区分，prod 有明显风险提示。

当前实际是 React 19 + TypeScript + Vite + Lucide，并使用 `i18next` + `react-i18next` 提供中英文切换；默认 `zh-CN`，用户选择保存在 localStorage，专业资源名词保持英文。shadcn/ui、Tailwind、React Router、TanStack Query/Table、React Hook Form、Zod、Monaco、ECharts 是后续目标能力，只有真正引入后才能依赖。API 数据使用 TanStack Query 时再接入，避免默认 Zustand。连接列表不得返回密码、Token、私钥或 raw kubeconfig；未实现能力显示 Empty State，不伪造真实状态。
