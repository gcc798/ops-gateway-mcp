import { useState } from 'react';
import { useTranslation } from 'react-i18next';
import { useList } from '../hooks/usePaginatedQuery';
import type { Field, Params, Operation } from '../types/api';
import Filters from '../components/Filters';
import Pagination from '../components/Pagination';
import OperationDetail from '../components/audit/OperationDetail';
import Operations from '../components/audit/OperationsTable';
import { ErrorBox, Loading } from '../components/Feedback';

const auditFields: Field[] = [
  { name: 'from', type: 'datetime-local' },
  { name: 'to', type: 'datetime-local' },
  { name: 'tool' },
  { name: 'resource_type', options: ['database', 'linux', 'kubernetes', 'gateway'] },
  { name: 'resource' },
  { name: 'environment' },
  { name: 'client' },
  { name: 'decision', options: ['allow', 'confirm', 'deny'] },
  {
    name: 'status',
    options: ['evaluated', 'pending', 'executing', 'succeeded', 'failed', 'expired'],
  },
];

export default function AuditView({ token }: { token: string }) {
  const { t } = useTranslation();
  const [params, setParams] = useState<Params>({ page: '1', page_size: '20' });
  const [selected, setSelected] = useState('');
  const { data, loading, error, refresh } = useList<Operation>(
    '/api/v1/audit/operations',
    token,
    params,
  );
  if (selected)
    return (
      <OperationDetail
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
        <button className="button" onClick={refresh}>
          {t('refresh')}
        </button>
      </div>
      <Filters
        fields={auditFields}
        values={params}
        onApply={(filters) => setParams({ ...filters, page: '1', page_size: params.page_size })}
      />
      <ErrorBox error={error} />
      {loading ? <Loading /> : data && <Operations items={data.items} select={setSelected} />}
      <Pagination data={data} params={params} onChange={setParams} loading={loading} />
    </section>
  );
}
