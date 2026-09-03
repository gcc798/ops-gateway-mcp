import i18n from 'i18next';
import { initReactI18next } from 'react-i18next';

const zh = {
  nav: { Overview: 'Overview', Database: 'Database', Linux: 'Linux', Kubernetes: 'Kubernetes', Policy: 'Policy', Audit: 'Audit', Settings: 'Settings' },
  control: '控制平面', online: 'Gateway 在线', local: '本地控制平面', active: 'Policy enforcement active', toggleTheme: '切换主题',
  security: '安全态势', every: '每一次操作', decided: '都有决策。', intent: 'Agent 意图从这里进入，Policy 决定哪些请求可以触达资源。', successfulAudit: '成功审计',
  stats: { calls: 'Tool calls', allowed: 'Allowed', confirm: 'Confirm', denied: 'Denied', failed: 'Failed' },
  recent: '最近操作', viewAudit: '查看 Audit →', policySignal: 'Policy signal', live: '实时', records: '条记录',
  table: { time: '时间', client: '客户端', env: '环境', resource: '资源', action: '操作', risk: '风险', decision: '决策', status: '状态' },
  emptyAudit: '暂无审计操作', emptyResources: '暂无配置资源', configure: '请在 ~/.ai-ops-gateway/conf 配置资源并重启 Gateway。',
  databaseConnections: 'Database connections', linuxHosts: 'Linux hosts', kubernetesClusters: 'Kubernetes clusters', policyDefaults: 'V1 Policy 默认规则', readOnly: '只读', runtimePaths: '运行时路径',
  allowRule: 'AST 校验通过的只读 SQL 与固定 Linux/Kubernetes 读取工具。', confirmRule: 'INSERT、带 WHERE 的 UPDATE/DELETE、CREATE 与 ALTER。', denyRule: 'DROP、TRUNCATE、无 WHERE 的 UPDATE/DELETE 及多语句 SQL。',
  success: '成功', pending: '待确认', confirmAction: '确认执行', confirming: '确认中…', confirmationFailed: '确认失败', apiUnavailable: 'API 不可用',
};
const en = {
  nav: { Overview: 'Overview', Database: 'Database', Linux: 'Linux', Kubernetes: 'Kubernetes', Policy: 'Policy', Audit: 'Audit', Settings: 'Settings' },
  control: 'CONTROL PLANE', online: 'Gateway online', local: 'LOCAL CONTROL PLANE', active: 'Policy enforcement active', toggleTheme: 'Toggle theme',
  security: 'SECURITY POSTURE', every: 'Every operation', decided: 'decided.', intent: 'Agent intent enters here. Policy decides what reaches your resources.', successfulAudit: 'SUCCESSFUL AUDIT',
  stats: { calls: 'Tool calls', allowed: 'Allowed', confirm: 'Confirm', denied: 'Denied', failed: 'Failed' },
  recent: 'Recent operations', viewAudit: 'View audit →', policySignal: 'Policy signal', live: 'LIVE', records: 'RECORDS',
  table: { time: 'TIME', client: 'CLIENT', env: 'ENV', resource: 'RESOURCE', action: 'ACTION', risk: 'RISK', decision: 'DECISION', status: 'STATUS' },
  emptyAudit: 'No audited operations yet', emptyResources: 'No resources configured', configure: 'Configure resources under ~/.ai-ops-gateway/conf and restart the gateway.',
  databaseConnections: 'Database connections', linuxHosts: 'Linux hosts', kubernetesClusters: 'Kubernetes clusters', policyDefaults: 'V1 enforced defaults', readOnly: 'READ ONLY', runtimePaths: 'Runtime paths',
  allowRule: 'AST-verified database reads and fixed Linux/Kubernetes read tools.', confirmRule: 'INSERT, scoped UPDATE/DELETE, CREATE and ALTER.', denyRule: 'DROP, TRUNCATE, unscoped UPDATE/DELETE and multiple SQL statements.',
  success: 'succeeded', pending: 'pending', confirmAction: 'Confirm', confirming: '…', confirmationFailed: 'confirmation failed', apiUnavailable: 'API unavailable',
};

const saved = localStorage.getItem('ai-ops-gateway-language');
i18n.use(initReactI18next).init({ resources: { 'zh-CN': { translation: zh }, 'en-US': { translation: en } }, lng: saved === 'en-US' ? 'en-US' : 'zh-CN', fallbackLng: 'en-US', interpolation: { escapeValue: false } });
i18n.on('languageChanged', language => localStorage.setItem('ai-ops-gateway-language', language));
export default i18n;
