import { useEffect, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { get, message } from '../lib/api';
import Details from '../components/Details';
import { ErrorBox, Loading } from '../components/Feedback';

export default function SettingsView({
  token,
  dark,
  setDark,
}: {
  token: string;
  dark: boolean;
  setDark: (value: boolean) => void;
}) {
  const [paths, setPaths] = useState<Record<string, string>>();
  const [error, setError] = useState('');
  const { t, i18n } = useTranslation();
  useEffect(() => {
    const controller = new AbortController();
    get<Record<string, string>>('/api/v1/config/paths', token, controller.signal)
      .then(setPaths)
      .catch((error) => {
        if (!controller.signal.aborted) setError(message(error));
      });
    return () => controller.abort();
  }, [token]);
  return (
    <section className="panel">
      <h3 className="section-title">{t('preferences')}</h3>
      <div className="preferences">
        <label>
          <input
            type="checkbox"
            checked={dark}
            onChange={(event) => setDark(event.target.checked)}
          />
          {t('darkTheme')}
        </label>
        <label>
          {t('language')}
          <select
            value={i18n.language}
            onChange={(event) => void i18n.changeLanguage(event.target.value)}
          >
            <option value="zh-CN">中文</option>
            <option value="en-US">English</option>
          </select>
        </label>
      </div>
      <div className="panel-head">
        <h3>{t('runtimePaths')}</h3>
      </div>
      <ErrorBox error={error} />
      {paths ? <Details value={paths} /> : !error && <Loading />}
    </section>
  );
}
