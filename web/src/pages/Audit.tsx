import { RefreshCw } from 'lucide-react';
import { useEffect, useState } from 'react';
import { get, message } from '../lib/api';
import { useListLocation } from '../hooks/useListLocation';
import { useTranslation } from 'react-i18next';
import { useList } from '../hooks/usePaginatedQuery';
import type { Field, Operation } from '../types/api';
import Filters from '../components/Filters';
import Pagination from '../components/Pagination';
import OperationDetail from '../components/audit/OperationDetail';
import Operations from '../components/audit/OperationsTable';
import { ErrorBox, Loading } from '../components/Feedback';

const auditFields: Field[] = [
  { name: 'from', type: 'datetime-local' },
  { name: 'to', type: 'datetime-local' },
  { name: 'tool', options: [] },
  { name: 'resource_type', options: ['database', 'linux', 'kubernetes', 'gateway'] },
  { name: 'resource' },
  { name: 'environment' },
  { name: 'client', options: [] },
  { name: 'decision', options: ['allow', 'confirm', 'deny'] },
  {
    name: 'status',
    options: ['evaluated', 'pending', 'executing', 'succeeded', 'failed', 'expired'],
  },
];

export default function AuditView({ token }: { token: string }) {
  const { t } = useTranslation();
  const { params, setParams, selected, setSelected } = useListLocation('/audit');
  const [options, setOptions] = useState<{ tools: string[]; clients: string[] }>({
    tools: [],
    clients: [],
  });
  const [optionsError, setOptionsError] = useState('');
  const [optionsRevision, setOptionsRevision] = useState(0);
  useEffect(() => {
    const controller = new AbortController();
    setOptionsError('');
    get<typeof options>('/api/v1/audit/filter-options', token, controller.signal)
      .then(setOptions)
      .catch((error) => {
        if (!controller.signal.aborted) setOptionsError(message(error));
      });
    return () => controller.abort();
  }, [token, optionsRevision]);
  const { data, loading, error, refresh } = useList<Operation>(
    '/api/v1/audit/operations',
    token,
    params,
  );
  if (selected)
    return (
      <OperationDetail
        key={selected}
        id={selected}
        token={token}
        back={() => {
          setSelected('');
          refresh();
        }}
      />
    );
  return (
    <section className="panel">
      <div className="panel-head">
        <h3>Audit</h3>
        <button
          className="icon"
          title={t('refresh')}
          aria-label={t('refresh')}
          onClick={() => {
            refresh();
            setOptionsRevision((value) => value + 1);
          }}
        >
          <RefreshCw size={16} />
        </button>
      </div>
      <div className="tabs" aria-label={t('fields.status')}>
        {['', 'pending', 'failed'].map((status) => (
          <button
            key={status}
            aria-pressed={(params.status || '') === status}
            onClick={() => setParams({ ...params, status, page: '1' })}
          >
            {t(status || 'all')}
          </button>
        ))}
      </div>
      <Filters
        key={JSON.stringify(params)}
        fields={auditFields.map((field) =>
          field.name === 'tool'
            ? { ...field, options: options.tools }
            : field.name === 'client'
              ? { ...field, options: options.clients }
              : field,
        )}
        values={params}
        onApply={(filters) => setParams({ ...filters, page: '1', page_size: params.page_size })}
      />
      <ErrorBox error={error} />
      <ErrorBox error={optionsError} />
      {loading ? <Loading /> : data && <Operations items={data.items} select={setSelected} />}
      <Pagination data={data} params={params} onChange={setParams} loading={loading} />
    </section>
  );
}
