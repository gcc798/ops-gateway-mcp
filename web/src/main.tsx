import { StrictMode, useEffect, useMemo, useState } from 'react';
import { createRoot } from 'react-dom/client';
import { useTranslation } from 'react-i18next';
import { Activity, Database, FileCheck2, Gauge, Hexagon, Moon, Network, Server, Settings, ShieldAlert, Sun } from 'lucide-react';
import './style.css';
import './i18n';

const nav = [['Overview', Gauge], ['Database', Database], ['Linux', Server], ['Kubernetes', Hexagon], ['Policy', ShieldAlert], ['Audit', FileCheck2], ['Settings', Settings]] as const;
type Page = typeof nav[number][0];
type Operation = { operation_id: string; timestamp: string; client: string; environment: string; resource: string; action: string; risk: string; decision: string; status: string };
type Data = { operations: Operation[]; databases: string[]; linux: string[]; kubernetes: string[]; paths: Record<string, string> };
const empty: Data = { operations: [], databases: [], linux: [], kubernetes: [], paths: {} };

async function get<T>(path: string): Promise<T> {
  const response = await fetch(path);
  if (!response.ok) throw new Error(`${response.status} ${response.statusText}`);
  return response.json() as Promise<T>;
}

function App() {
  const { t, i18n } = useTranslation();
  const [page, setPage] = useState<Page>('Overview');
  const [dark, setDark] = useState(true);
  const [data, setData] = useState<Data>(empty);
  const [error, setError] = useState('');
  useEffect(() => {
    Promise.all([
      get<Operation[]>('/api/v1/audit/operations'), get<string[]>('/api/v1/databases'), get<string[]>('/api/v1/linux/hosts'),
      get<string[]>('/api/v1/kubernetes/clusters'), get<Record<string, string>>('/api/v1/config/paths'),
    ]).then(([operations, databases, linux, kubernetes, paths]) => setData({ operations, databases, linux, kubernetes, paths })).catch(e => setError(String(e)));
  }, []);
  const counts = useMemo(() => {
    const count = (decision: string) => data.operations.filter(x => x.decision.toLowerCase() === decision).length;
    return [[t('stats.calls'), data.operations.length], [t('stats.allowed'), count('allow')], [t('stats.confirm'), count('confirm')], [t('stats.denied'), count('deny')], [t('stats.failed'), data.operations.filter(x => x.status === 'failed').length]] as const;
  }, [data.operations, t]);
  const toggleLanguage = () => i18n.changeLanguage(i18n.language === 'zh-CN' ? 'en-US' : 'zh-CN');

  return <div className={dark ? 'app dark' : 'app'}>
    <aside><div className="brand"><span className="mark">A</span><span>AI OPS<br /><b>GATEWAY</b></span></div><div className="env"><span className="dot" />{t('local')}</div>
      <nav>{nav.map(([label, Icon]) => <button className={page === label ? 'active' : ''} onClick={() => setPage(label)} key={label}><Icon size={16} />{t(`nav.${label}`)}</button>)}</nav>
      <div className="side-foot">v0.1.0 · local<br /><span>{t('active')}</span></div>
    </aside>
    <main><header><div><span className="eyebrow">{t('control')} / {t(`nav.${page}`).toUpperCase()}</span><h1>{t(`nav.${page}`)}</h1></div><div className="header-actions"><span className="status"><Activity size={14} /> {t('online')}</span><button className="icon" onClick={toggleLanguage} aria-label="切换语言" title="中文 / English">{i18n.language === 'zh-CN' ? 'EN' : '中'}</button><button className="icon" onClick={() => setDark(!dark)} aria-label={t('toggleTheme')}>{dark ? <Sun size={17} /> : <Moon size={17} />}</button></div></header>
      {error && <div className="error">{t('apiUnavailable')}: {error}</div>}
      {page === 'Overview' && <Overview counts={counts} operations={data.operations} openAudit={() => setPage('Audit')} />}
      {page === 'Database' && <Resources icon={Database} title={t('databaseConnections')} items={data.databases} />}
      {page === 'Linux' && <Resources icon={Server} title={t('linuxHosts')} items={data.linux} />}
      {page === 'Kubernetes' && <Resources icon={Hexagon} title={t('kubernetesClusters')} items={data.kubernetes} />}
      {page === 'Policy' && <Policy />}
      {page === 'Audit' && <Operations operations={data.operations} />}
      {page === 'Settings' && <SettingsView paths={data.paths} />}
    </main>
  </div>;
}

function Overview({ counts, operations, openAudit }: { counts: readonly (readonly [string, number])[]; operations: Operation[]; openAudit: () => void }) {
  const { t } = useTranslation();
  const total = counts[0][1]; const allowed = counts[1][1]; const coverage = total ? Math.round((total - counts[4][1]) / total * 100) : 100;
  return <><section className="hero"><div><p className="eyebrow">{t('security')}</p><h2>{t('every')}<br /><em>{t('decided')}</em></h2><p className="muted">{t('intent')}</p></div><div className="rings"><div className="ring r1"><span>{coverage}%</span><small>{t('successfulAudit')}</small></div></div></section>
    <section className="stats">{counts.map(([label, value]) => <div className="stat" key={label}><span>{label}</span><strong>{value}</strong><small>{total ? `${Math.round(value / total * 100)}%` : '—'}</small></div>)}</section>
    <section className="content-grid"><div className="panel"><div className="panel-head"><h3>{t('recent')}</h3><button className="link" onClick={openAudit}>{t('viewAudit')}</button></div><Operations operations={operations.slice(0, 8)} embedded /></div><div className="panel signal"><div className="panel-head"><h3>{t('policySignal')}</h3><span className="live">{t('live')}</span></div><div className="signal-chart"><div className="bar allow" style={{ height: `${total ? allowed / total * 100 : 0}%` }} /><div className="bar confirm" style={{ height: `${total ? counts[2][1] / total * 100 : 0}%` }} /><div className="bar deny" style={{ height: `${total ? counts[3][1] / total * 100 : 0}%` }} /><div className="bar fail" style={{ height: `${total ? counts[4][1] / total * 100 : 0}%` }} /></div><div className="legend"><span><i className="c-allow" />{t('stats.allowed')}</span><span><i className="c-confirm" />{t('stats.confirm')}</span><span><i className="c-deny" />{t('stats.denied')}</span></div></div></section></>;
}

function Operations({ operations, embedded = false }: { operations: Operation[]; embedded?: boolean }) {
  const { t } = useTranslation();
  const [confirming, setConfirming] = useState('');
  const [confirmed, setConfirmed] = useState<Record<string, string>>({});
  const confirm = async (id: string) => { setConfirming(id); try { const response = await fetch(`/api/v1/operations/${id}/confirm`, { method: 'POST' }); if (!response.ok) throw new Error('confirmation failed'); setConfirmed({ ...confirmed, [id]: 'succeeded' }); } finally { setConfirming(''); } };
  const headers = ['time', 'client', 'env', 'resource', 'action', 'risk', 'decision', 'status'];
  const table = operations.length ? <table><thead><tr>{headers.map(x => <th key={x}>{t(`table.${x}`)}</th>)}</tr></thead><tbody>{operations.map(op => <tr key={op.operation_id}><td>{new Date(op.timestamp).toLocaleString()}</td><td>{op.client || '—'}</td><td>{op.environment || '—'}</td><td>{op.resource}</td><td>{op.action}</td><td className={`risk-${op.risk.toLowerCase()}`}>{op.risk}</td><td className={`decision-${op.decision.toLowerCase()}`}>{op.decision}</td><td>{confirmed[op.operation_id] || op.status}{op.status === 'pending' && <button className="confirm" disabled={confirming === op.operation_id} onClick={() => confirm(op.operation_id)}>{confirming === op.operation_id ? t('confirming') : t('confirmAction')}</button>}</td></tr>)}</tbody></table> : <Empty text={t('emptyAudit')} />;
  return embedded ? table : <section className="panel"><div className="panel-head"><h3>{t('recent')}</h3><span className="live">{operations.length} {t('records')}</span></div>{table}</section>;
}

function Resources({ icon: Icon, title, items }: { icon: typeof Database; title: string; items: string[] }) {
  const { t } = useTranslation(); return <section className="panel resource-panel"><div className="panel-head"><h3>{title}</h3><span className="live">{items.length} {t('configured')}</span></div>{items.length ? <div className="resource-list">{items.map(name => <div className="resource" key={name}><Icon size={18} /><div><strong>{name}</strong><small>{t('logical')}</small></div><span className="dot" /></div>)}</div> : <Empty text={t('emptyResources')} />}</section>;
}
function Policy() { const { t } = useTranslation(); return <section className="panel policy"><div className="panel-head"><h3>{t('policyDefaults')}</h3><span className="live">{t('live')}</span></div><div className="rules"><div><b>ALLOW</b><p>{t('allowRule')}</p></div><div><b>CONFIRM</b><p>{t('confirmRule')}</p></div><div><b>DENY</b><p>{t('denyRule')}</p></div></div></section>; }
function SettingsView({ paths }: { paths: Record<string, string> }) { const { t } = useTranslation(); return <section className="panel"><div className="panel-head"><h3>{t('runtimePaths')}</h3><span className="live">{t('readOnly')}</span></div><dl>{Object.entries(paths).map(([key, value]) => <div key={key}><dt>{key}</dt><dd>{value}</dd></div>)}</dl></section>; }
function Empty({ text }: { text: string }) { const { t } = useTranslation(); return <div className="placeholder"><Network size={30} /><h2>{text}</h2><p>{t('configure')}</p></div>; }

createRoot(document.getElementById('root')!).render(<StrictMode><App /></StrictMode>);
