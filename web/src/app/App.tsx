import { useState } from 'react';
import { useTranslation } from 'react-i18next';
import {
  Database,
  FileCheck2,
  Gauge,
  Hexagon,
  Moon,
  Server,
  Settings,
  ShieldAlert,
  Sun,
} from 'lucide-react';
import { useAuth } from '../hooks/useAuth';
import Login from '../pages/Login';
import Overview from '../pages/Overview';
import Resources from '../pages/Resources';
import AuditView from '../pages/Audit';
import Policy from '../pages/Policy';
import SettingsView from '../pages/Settings';
import { ErrorBox } from '../components/Feedback';

const nav = [
  ['Overview', Gauge],
  ['Database', Database],
  ['Linux', Server],
  ['Kubernetes', Hexagon],
  ['Policy', ShieldAlert],
  ['Audit', FileCheck2],
  ['Settings', Settings],
] as const;
type Page = (typeof nav)[number][0];

export default function App() {
  const { t, i18n } = useTranslation();
  const [page, setPage] = useState<Page>('Overview');
  const [dark, setDark] = useState(true);
  const { token, verified, error, retry, login, logout } = useAuth();
  if (!token) return <Login onLogin={login} />;
  if (!verified)
    return (
      <div className="login-shell">
        <div className="login-panel">
          {error ? (
            <>
              <ErrorBox error={error} />
              <button className="button" onClick={retry}>
                {t('refresh')}
              </button>
              <button className="button" onClick={logout}>
                {t('signOut')}
              </button>
            </>
          ) : (
            t('loading')
          )}
        </div>
      </div>
    );
  return (
    <div className={dark ? 'app dark' : 'app'}>
      <aside>
        <div className="brand">
          <span className="mark">A</span>
          <span>
            AI OPS
            <br />
            <b>GATEWAY</b>
          </span>
        </div>
        <div className="env">{t('internalConsole')}</div>
        <nav>
          {nav.map(([label, Icon]) => (
            <button
              className={page === label ? 'active' : ''}
              onClick={() => setPage(label)}
              key={label}
            >
              <Icon size={16} />
              {t('nav.' + label)}
            </button>
          ))}
        </nav>
        <div className="side-foot">
          v0.1.0 · local
          <br />
          <span>{t('active')}</span>
        </div>
      </aside>
      <main>
        <header>
          <div>
            <span className="eyebrow">
              {t('control')} / {t('nav.' + page)}
            </span>
            <h1>{t('nav.' + page)}</h1>
          </div>
          <div className="header-actions">
            <button className="icon" onClick={logout}>
              {t('signOut')}
            </button>
            <button
              className="icon"
              onClick={() =>
                void i18n.changeLanguage(i18n.language === 'zh-CN' ? 'en-US' : 'zh-CN')
              }
              aria-label="中文 / English"
            >
              {i18n.language === 'zh-CN' ? 'EN' : '中'}
            </button>
            <button className="icon" onClick={() => setDark(!dark)} aria-label={t('toggleTheme')}>
              {dark ? <Sun size={17} /> : <Moon size={17} />}
            </button>
          </div>
        </header>
        {page === 'Overview' && <Overview token={token} openAudit={() => setPage('Audit')} />}
        {page === 'Database' && <Resources key="database" kind="database" token={token} />}
        {page === 'Linux' && <Resources key="linux" kind="linux" token={token} />}
        {page === 'Kubernetes' && <Resources key="kubernetes" kind="kubernetes" token={token} />}
        {page === 'Audit' && <AuditView token={token} />}
        {page === 'Policy' && <Policy />}
        {page === 'Settings' && <SettingsView token={token} />}
      </main>
    </div>
  );
}
