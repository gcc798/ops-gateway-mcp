import { useEffect, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { get, message } from '../lib/api';
import Details from '../components/Details';
import { ErrorBox, Loading } from '../components/Feedback';

export default function SettingsView({ token }: { token: string }) {
  const [paths, setPaths] = useState<Record<string, string>>();
  const [error, setError] = useState('');
  const { t } = useTranslation();
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
      <div className="panel-head">
        <h3>{t('runtimePaths')}</h3>
      </div>
      <ErrorBox error={error} />
      {paths ? <Details value={paths} /> : !error && <Loading />}
    </section>
  );
}
