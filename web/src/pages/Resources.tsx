import { useState } from 'react';
import { useTranslation } from 'react-i18next';
import { resourcePaths, resourceFields } from '../config/resources';
import { useList } from '../hooks/usePaginatedQuery';
import type { Params, Resource, ResourceKind } from '../types/api';
import Filters from '../components/Filters';
import Pagination from '../components/Pagination';
import ResourceDetail from '../components/resources/ResourceDetail';
import { ErrorBox, Loading, Empty } from '../components/Feedback';

export default function Resources({ kind, token }: { kind: ResourceKind; token: string }) {
  const { t } = useTranslation();
  const [params, setParams] = useState<Params>({ page: '1', page_size: '20' });
  const [selected, setSelected] = useState('');
  const { data, loading, error, refresh } = useList<Resource>(resourcePaths[kind], token, params);
  const columns =
    kind === 'database'
      ? ['name', 'environment', 'driver']
      : kind === 'linux'
        ? ['name', 'environment', 'address', 'user']
        : ['name', 'environment', 'context'];
  if (selected)
    return (
      <ResourceDetail kind={kind} name={selected} token={token} back={() => setSelected('')} />
    );
  return (
    <section className="panel resource-panel">
      <div className="panel-head">
        <h3>
          {t(
            kind === 'database'
              ? 'databaseConnections'
              : kind === 'linux'
                ? 'linuxHosts'
                : 'kubernetesClusters',
          )}
        </h3>
        <button className="button" onClick={refresh}>
          {t('refresh')}
        </button>
      </div>
      <Filters
        fields={resourceFields[kind]}
        values={params}
        onApply={(filters) => setParams({ ...filters, page: '1', page_size: params.page_size })}
      />
      <ErrorBox error={error} />
      {loading ? (
        <Loading />
      ) : data?.items.length ? (
        <div className="table-scroll">
          <table>
            <thead>
              <tr>
                {columns.map((col) => (
                  <th key={col}>{t('fields.' + col)}</th>
                ))}
                <th>{t('details')}</th>
              </tr>
            </thead>
            <tbody>
              {data.items.map((item) => (
                <tr key={item.name}>
                  {columns.map((col) => (
                    <td key={col}>
                      <span
                        className={col === 'environment' && item[col] === 'prod' ? 'risk-high' : ''}
                      >
                        {item[col] || '—'}
                      </span>
                    </td>
                  ))}
                  <td>
                    <button className="link" onClick={() => setSelected(item.name)}>
                      {t('details')} →
                    </button>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      ) : (
        !error && <Empty text={t('noResults')} />
      )}
      <Pagination data={data} params={params} onChange={setParams} loading={loading} />
    </section>
  );
}
