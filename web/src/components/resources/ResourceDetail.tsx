import { useEffect, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { ArrowLeft } from 'lucide-react';
import { resourcePaths } from '../../config/resources';
import { get, request, message } from '../../lib/api';
import type { Resource, ResourceKind } from '../../types/api';
import Details from '../Details';
import { ErrorBox, Loading } from '../Feedback';

export default function ResourceDetail({
  kind,
  name,
  token,
  back,
}: {
  kind: ResourceKind;
  name: string;
  token: string;
  back: () => void;
}) {
  const { t } = useTranslation();
  const [resource, setResource] = useState<Resource>();
  const [error, setError] = useState('');
  const [testing, setTesting] = useState(false);
  const [status, setStatus] = useState('');
  const [output, setOutput] = useState('');
  useEffect(() => {
    const controller = new AbortController();
    get<Resource>(resourcePaths[kind] + '/' + encodeURIComponent(name), token, controller.signal)
      .then(setResource)
      .catch((error) => {
        if (!controller.signal.aborted) setError(message(error));
      });
    return () => controller.abort();
  }, [kind, name, token]);
  const test = async () => {
    setTesting(true);
    setStatus('');
    setOutput('');
    try {
      await request(resourcePaths[kind] + '/test', token, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ name }),
      });
      setStatus(t('connected'));
      if (kind === 'database')
        setOutput(
          (
            (await get<string[]>(
              resourcePaths[kind] + '/' + encodeURIComponent(name) + '/tables',
              token,
            )) || []
          ).join('\n') || t('noTables'),
        );
      if (kind === 'linux')
        setOutput(
          await (
            await request(
              resourcePaths[kind] + '/' + encodeURIComponent(name) + '/system-info',
              token,
            )
          ).text(),
        );
    } catch (error) {
      setStatus(message(error));
    } finally {
      setTesting(false);
    }
  };
  return (
    <section className="panel resource-detail">
      <div className="panel-head">
        <button className="back-button" onClick={back}>
          <ArrowLeft size={15} />
          {t('backToResources')}
        </button>
        <button className="button" disabled={testing} onClick={() => void test()}>
          {testing ? t('checking') : t('testConnection')}
        </button>
      </div>
      <div className="detail-body">
        <h2>{name}</h2>
        <p className="muted">{t('plaintextConfig')}</p>
        <ErrorBox error={error} />
        {resource ? <Details value={resource} /> : !error && <Loading />}
        {status && <p role="status">{status}</p>}
        {output && <pre className="detail-output">{output}</pre>}
      </div>
    </section>
  );
}
