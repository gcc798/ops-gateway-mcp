import { useEffect, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { useList } from '../hooks/usePaginatedQuery';
import { get, message } from '../lib/api';
import type { Operation, Summary } from '../types/api';
import OperationDetail from '../components/audit/OperationDetail';
import Operations from '../components/audit/OperationsTable';
import { ErrorBox, Loading } from '../components/Feedback';

export default function Overview({ token, openAudit }: { token: string; openAudit: () => void }) {
  const { t } = useTranslation();
  const [summary, setSummary] = useState<Summary>();
  const [error, setError] = useState('');
  const [selected, setSelected] = useState('');
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
    }, 15000);
    return () => window.clearInterval(timer);
  }, []);
  if (selected)
    return (
      <OperationDetail
        id={selected}
        token={token}
        back={() => {
          setSelected('');
          refresh();
          setRevision((value) => value + 1);
        }}
      />
    );
  const cards = [
    ['calls', summary?.total],
    ['allowed', summary?.allow],
    ['confirm', summary?.confirm],
    ['denied', summary?.deny],
    ['failed', summary?.failed],
  ] as const;
  return (
    <>
      <section className="hero">
        <div>
          <p className="eyebrow">{t('security')}</p>
          <h2>
            {t('every')}
            <br />
            <em>{t('decided')}</em>
          </h2>
          <p className="muted">{t('intent')}</p>
        </div>
      </section>
      <ErrorBox error={error || listError} />
      <p className="muted">{t('allTimeStats')}</p>
      <section className="stats">
        {cards.map(([label, value]) => (
          <div className="stat" key={label}>
            <span>{t('stats.' + label)}</span>
            <strong>{value ?? '—'}</strong>
          </div>
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
