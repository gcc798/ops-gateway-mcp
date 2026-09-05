import { ArrowDown, ArrowUp, ArrowRight, RefreshCw } from 'lucide-react';
import { useListLocation } from '../hooks/useListLocation';
import { useTranslation } from 'react-i18next';
import { resourcePaths, resourceFields } from '../config/resources';
import { useList } from '../hooks/usePaginatedQuery';
import type { Resource, ResourceKind } from '../types/api';
import Filters from '../components/Filters';
import Pagination from '../components/Pagination';
import ResourceDetail from '../components/resources/ResourceDetail';
import { ErrorBox, Loading, Empty } from '../components/Feedback';

export default function Resources({ kind, token }: { kind: ResourceKind; token: string }) {
  const { t } = useTranslation();
  const { params, setParams, selected, setSelected } = useListLocation('/' + kind);
  const { data, loading, error, refresh } = useList<Resource>(resourcePaths[kind], token, params);
  const columns =
    kind === 'database'
      ? ['name', 'environment', 'driver']
      : kind === 'linux'
        ? ['name', 'environment', 'address', 'user']
        : ['name', 'environment', 'context'];
  if (selected)
    return (
      <ResourceDetail
        key={selected}
        kind={kind}
        name={selected}
        token={token}
        back={() => setSelected('')}
      />
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
        <button className="icon" title={t('refresh')} aria-label={t('refresh')} onClick={refresh}>
          <RefreshCw size={16} />
        </button>
      </div>
      <Filters
        key={JSON.stringify(params)}
        fields={resourceFields[kind]}
        values={params}
        onApply={(filters) =>
          setParams({
            ...filters,
            sort: params.sort || 'name',
            order: params.order || 'asc',
            page: '1',
            page_size: params.page_size,
          })
        }
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
                  <th
                    key={col}
                    aria-sort={
                      params.sort === col || (!params.sort && col === 'name')
                        ? params.order === 'desc'
                          ? 'descending'
                          : 'ascending'
                        : 'none'
                    }
                  >
                    <button
                      className="sort-button"
                      onClick={() =>
                        setParams({
                          ...params,
                          sort: col,
                          order:
                            (params.sort || 'name') === col && params.order !== 'desc'
                              ? 'desc'
                              : 'asc',
                          page: '1',
                        })
                      }
                    >
                      {t('fields.' + col)}
                      {(params.sort || 'name') === col &&
                        (params.order === 'desc' ? <ArrowDown size={13} /> : <ArrowUp size={13} />)}
                    </button>
                  </th>
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
                        {col === 'name' ? (
                          <button className="link" onClick={() => setSelected(item.name)}>
                            {item.name}
                          </button>
                        ) : (
                          item[col] || '—'
                        )}
                      </span>
                    </td>
                  ))}
                  <td>
                    <button
                      className="icon"
                      title={t('details')}
                      aria-label={t('details')}
                      onClick={() => setSelected(item.name)}
                    >
                      <ArrowRight size={16} />
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
