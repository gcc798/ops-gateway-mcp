import { useEffect, useState } from 'react';
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
import { NavLink, Navigate, Route, Routes, useLocation } from 'react-router-dom';
import Brand from '../components/Brand';
import { useAuth } from '../hooks/useAuth';
import Login from '../pages/Login';
import Overview from '../pages/Overview';
import Resources from '../pages/Resources';
import AuditView from '../pages/Audit';
import Policy from '../pages/Policy';
import SettingsView from '../pages/Settings';
import { ErrorBox } from '../components/Feedback';

// App 负责全局布局、路由和登录状态，页面业务放在 pages 目录。
const nav = [
  ['Overview', Gauge],
  ['Database', Database],
  ['Linux', Server],
  ['Kubernetes', Hexagon],
  ['Policy', ShieldAlert],
  ['Audit', FileCheck2],
  ['Settings', Settings],
] as const;

export default function App() {
  const { t, i18n } = useTranslation();
  const location = useLocation();
  const page =
    nav.find(([label]) => location.pathname.split('/')[1] === label.toLowerCase())?.[0] ||
    'Overview';
  const [dark, setDark] = useState(() => localStorage.getItem('ops-gateway-mcp-theme') === 'dark');
  useEffect(() => {
    localStorage.setItem('ops-gateway-mcp-theme', dark ? 'dark' : 'light');
    document.documentElement.dataset.theme = dark ? 'dark' : 'light';
  }, [dark]);
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
        <Brand />
        <div className="env">{t('internalConsole')}</div>
        <nav>
          {nav.map(([label, Icon]) => (
            <NavLink to={'/' + label.toLowerCase()} key={label}>
              <Icon size={16} />
              {t('nav.' + label)}
            </NavLink>
          ))}
        </nav>
        <div className="side-foot">{t('local')}</div>
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
            <button
              className="icon"
              onClick={() => setDark(!dark)}
              aria-label={t('toggleTheme')}
              title={t('toggleTheme')}
            >
              {dark ? <Sun size={17} /> : <Moon size={17} />}
            </button>
          </div>
        </header>
        <Routes>
          <Route path="/overview" element={<Overview token={token} />} />
          <Route
            path="/database/:id?"
            element={<Resources key="database" kind="database" token={token} />}
          />
          <Route
            path="/linux/:id?"
            element={<Resources key="linux" kind="linux" token={token} />}
          />
          <Route
            path="/kubernetes/:id?"
            element={<Resources key="kubernetes" kind="kubernetes" token={token} />}
          />
          <Route path="/audit/:id?" element={<AuditView token={token} />} />
          <Route path="/policy" element={<Policy token={token} />} />
          <Route
            path="/settings"
            element={<SettingsView token={token} dark={dark} setDark={setDark} />}
          />
          <Route path="*" element={<Navigate to="/overview" replace />} />
        </Routes>
      </main>
    </div>
  );
}
