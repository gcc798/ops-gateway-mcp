import { StrictMode, useEffect, useMemo, useState } from 'react';
import { createRoot } from 'react-dom/client';
import { Activity, Database, FileCheck2, Gauge, Hexagon, Moon, Network, Server, Settings, ShieldAlert, Sun } from 'lucide-react';
import './style.css';

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
    return [['Tool calls', data.operations.length], ['Allowed', count('allow')], ['Confirm', count('confirm')], ['Denied', count('deny')], ['Failed', data.operations.filter(x => x.status === 'failed').length]] as const;
  }, [data.operations]);

  return <div className={dark ? 'app dark' : 'app'}>
    <aside><div className="brand"><span className="mark">A</span><span>AI OPS<br /><b>GATEWAY</b></span></div><div className="env"><span className="dot" />LOCAL CONTROL PLANE</div>
      <nav>{nav.map(([label, Icon]) => <button className={page === label ? 'active' : ''} onClick={() => setPage(label)} key={label}><Icon size={16} />{label}</button>)}</nav>
      <div className="side-foot">v0.1.0 · local<br /><span>Policy enforcement active</span></div>
    </aside>
    <main><header><div><span className="eyebrow">CONTROL PLANE / {page.toUpperCase()}</span><h1>{page}</h1></div><div className="header-actions"><span className="status"><Activity size={14} /> Gateway online</span><button className="icon" onClick={() => setDark(!dark)} aria-label="Toggle theme">{dark ? <Sun size={17} /> : <Moon size={17} />}</button></div></header>
      {error && <div className="error">API unavailable: {error}</div>}
      {page === 'Overview' && <Overview counts={counts} operations={data.operations} openAudit={() => setPage('Audit')} />}
      {page === 'Database' && <Resources icon={Database} title="Database connections" items={data.databases} />}
      {page === 'Linux' && <Resources icon={Server} title="Linux hosts" items={data.linux} />}
      {page === 'Kubernetes' && <Resources icon={Hexagon} title="Kubernetes clusters" items={data.kubernetes} />}
      {page === 'Policy' && <Policy />}
      {page === 'Audit' && <Operations operations={data.operations} />}
      {page === 'Settings' && <SettingsView paths={data.paths} />}
    </main>
  </div>;
}

function Overview({ counts, operations, openAudit }: { counts: readonly (readonly [string, number])[]; operations: Operation[]; openAudit: () => void }) {
  const total = counts[0][1]; const allowed = counts[1][1]; const coverage = total ? Math.round((total - counts[4][1]) / total * 100) : 100;
  return <><section className="hero"><div><p className="eyebrow">SECURITY POSTURE</p><h2>Every operation<br /><em>decided.</em></h2><p className="muted">Agent intent enters here. Policy decides what reaches your resources.</p></div><div className="rings"><div className="ring r1"><span>{coverage}%</span><small>SUCCESSFUL AUDIT</small></div></div></section>
    <section className="stats">{counts.map(([label, value]) => <div className="stat" key={label}><span>{label}</span><strong>{value}</strong><small>{total ? `${Math.round(value / total * 100)}%` : '—'}</small></div>)}</section>
    <section className="content-grid"><div className="panel"><div className="panel-head"><h3>Recent operations</h3><button className="link" onClick={openAudit}>View audit log →</button></div><Operations operations={operations.slice(0, 8)} embedded /></div><div className="panel signal"><div className="panel-head"><h3>Policy signal</h3><span className="live">LIVE</span></div><div className="signal-chart"><div className="bar allow" style={{ height: `${total ? allowed / total * 100 : 0}%` }} /><div className="bar confirm" style={{ height: `${total ? counts[2][1] / total * 100 : 0}%` }} /><div className="bar deny" style={{ height: `${total ? counts[3][1] / total * 100 : 0}%` }} /><div className="bar fail" style={{ height: `${total ? counts[4][1] / total * 100 : 0}%` }} /></div><div className="legend"><span><i className="c-allow" />Allowed</span><span><i className="c-confirm" />Confirm</span><span><i className="c-deny" />Denied</span></div></div></section></>;
}

function Operations({ operations, embedded = false }: { operations: Operation[]; embedded?: boolean }) {
  const [confirming, setConfirming] = useState('');
  const [confirmed, setConfirmed] = useState<Record<string, string>>({});
  const confirm = async (id: string) => { setConfirming(id); try { const response = await fetch(`/api/v1/operations/${id}/confirm`, { method: 'POST' }); if (!response.ok) throw new Error('confirmation failed'); setConfirmed({ ...confirmed, [id]: 'succeeded' }); } finally { setConfirming(''); } };
  const table = operations.length ? <table><thead><tr>{['TIME', 'CLIENT', 'ENV', 'RESOURCE', 'ACTION', 'RISK', 'DECISION', 'STATUS'].map(x => <th key={x}>{x}</th>)}</tr></thead><tbody>{operations.map(op => <tr key={op.operation_id}><td>{new Date(op.timestamp).toLocaleString()}</td><td>{op.client || '—'}</td><td>{op.environment || '—'}</td><td>{op.resource}</td><td>{op.action}</td><td className={`risk-${op.risk.toLowerCase()}`}>{op.risk}</td><td className={`decision-${op.decision.toLowerCase()}`}>{op.decision}</td><td>{confirmed[op.operation_id] || op.status}{op.status === 'pending' && <button className="confirm" disabled={confirming === op.operation_id} onClick={() => confirm(op.operation_id)}>{confirming === op.operation_id ? '...' : 'Confirm'}</button>}</td></tr>)}</tbody></table> : <Empty text="No audited operations yet" />;
  return embedded ? table : <section className="panel"><div className="panel-head"><h3>Immutable operation history</h3><span className="live">{operations.length} RECORDS</span></div>{table}</section>;
}

function Resources({ icon: Icon, title, items }: { icon: typeof Database; title: string; items: string[] }) {
  return <section className="panel resource-panel"><div className="panel-head"><h3>{title}</h3><span className="live">{items.length} CONFIGURED</span></div>{items.length ? <div className="resource-list">{items.map(name => <div className="resource" key={name}><Icon size={18} /><div><strong>{name}</strong><small>Gateway-managed logical resource</small></div><span className="dot" /></div>)}</div> : <Empty text="No resources configured" />}</section>;
}
function Policy() { return <section className="panel policy"><div className="panel-head"><h3>V1 enforced defaults</h3><span className="live">ACTIVE</span></div><div className="rules"><div><b>ALLOW</b><p>AST-verified database reads and fixed Linux/Kubernetes read tools.</p></div><div><b>CONFIRM</b><p>INSERT, scoped UPDATE/DELETE, CREATE and ALTER.</p></div><div><b>DENY</b><p>DROP, TRUNCATE, unscoped UPDATE/DELETE and multiple SQL statements.</p></div></div></section>; }
function SettingsView({ paths }: { paths: Record<string, string> }) { return <section className="panel"><div className="panel-head"><h3>Runtime paths</h3><span className="live">READ ONLY</span></div><dl>{Object.entries(paths).map(([key, value]) => <div key={key}><dt>{key}</dt><dd>{value}</dd></div>)}</dl></section>; }
function Empty({ text }: { text: string }) { return <div className="placeholder"><Network size={30} /><h2>{text}</h2><p>Configure resources under ~/.ai-ops-gateway/conf and restart the gateway.</p></div>; }

createRoot(document.getElementById('root')!).render(<StrictMode><App /></StrictMode>);
