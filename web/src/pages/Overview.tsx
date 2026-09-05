import { useEffect, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { useList } from '../hooks/usePaginatedQuery';
import { get, message } from '../lib/api';
import type { Operation, Summary } from '../types/api';
import { useNavigate } from 'react-router-dom';
import { resourcePaths } from '../config/resources';
import type { ResourceKind } from '../types/api';
import Operations from '../components/audit/OperationsTable';
import { ErrorBox, Loading } from '../components/Feedback';

export default function Overview({ token }: { token: string }) {
  const navigate = useNavigate();
  const openAudit = () => navigate('/audit');
  const { t } = useTranslation();
  const [summary, setSummary] = useState<Summary>();
  const [error, setError] = useState('');
  const setSelected = (id: string) => navigate('/audit/' + encodeURIComponent(id));
  const pending = useList<Operation>('/api/v1/audit/operations', token, {
    status: 'pending',
    page_size: '1',
  });
  const [revision, setRevision] = useState(0);
  const {
    data,
    error: listError,
    refresh,
  } = useList<Operation>('/api/v1/audit/operations', token, { page: '1', page_size: '8' });
  useEffect(() => {
    const controller = new AbortController();
    get<Summary>('/api/v1/audit/summary', token, controller.signal)
      .then((value) => {
        setSummary(value);
        setError('');
      })
      .catch((error) => {
        if (!controller.signal.aborted) setError(message(error));
      });
    return () => controller.abort();
  }, [token, revision]);
  useEffect(() => {
    const timer = window.setInterval(() => {
      setRevision((value) => value + 1);
      refresh();
      pending.refresh();
    }, 15000);
    return () => window.clearInterval(timer);
  }, []);
  const cards = [
    ['calls', summary?.total, ''],
    ['allowed', summary?.allow, '?decision=allow'],
    ['confirm', summary?.confirm, '?decision=confirm'],
    ['denied', summary?.deny, '?decision=deny'],
    ['failed', summary?.failed, '?status=failed'],
  ] as const;
  return (
    <>
      <section className="resource-totals">
        {(Object.keys(resourcePaths) as ResourceKind[]).map((kind) => (
          <ResourceCount key={kind} kind={kind} token={token} />
        ))}
        <button className="stat" onClick={() => navigate('/audit?status=pending')}>
          <span>{t('pending')}</span>
          <strong>{pending.data?.total ?? '—'}</strong>
        </button>
      </section>
      <ErrorBox error={pending.error} />
      <ErrorBox error={error || listError} />
      <p className="muted">{t('allTimeStats')}</p>
      <section className="stats">
        {cards.map(([label, value, query]) => (
          <button className="stat" key={label} onClick={() => navigate('/audit' + query)}>
            <span>{t('stats.' + label)}</span>
            <strong>{value ?? '—'}</strong>
          </button>
        ))}
      </section>
      <section className="panel overview-recent">
        <div className="panel-head">
          <h3>{t('recent')}</h3>
          <button className="link" onClick={openAudit}>
            {t('viewAudit')}
          </button>
        </div>
        {data ? <Operations items={data.items} select={setSelected} /> : !listError && <Loading />}
      </section>
    </>
  );
}

function ResourceCount({ kind, token }: { kind: ResourceKind; token: string }) {
  const navigate = useNavigate();
  const { data, error } = useList(resourcePaths[kind], token, { page_size: '1' });
  return (
    <div>
      <button className="stat" onClick={() => navigate('/' + kind)}>
        <span>{kind === 'database' ? 'Database' : kind === 'linux' ? 'Linux' : 'Kubernetes'}</span>
        <strong>{data?.total ?? '—'}</strong>
      </button>
      <ErrorBox error={error} />
    </div>
  );
}
