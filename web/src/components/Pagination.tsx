import { useTranslation } from 'react-i18next';
import type { Params } from '../types/api';

export default function Pagination({
  data,
  params,
  onChange,
  loading,
}: {
  data?: { total: number };
  params: Params;
  onChange: (params: Params) => void;
  loading: boolean;
}) {
  const { t } = useTranslation();
  const page = Number(params.page || 1);
  const size = Number(params.page_size || 20);
  const pages = Math.max(1, Math.ceil((data?.total || 0) / size));
  return (
    <div className="pagination">
      <span>{data ? t('totalRecords', { count: data.total }) : '—'}</span>
      <label>
        {t('pageSize')}
        <select
          value={size}
          onChange={(event) => onChange({ ...params, page_size: event.target.value, page: '1' })}
        >
          {[10, 20, 50, 100].map((size) => (
            <option key={size}>{size}</option>
          ))}
        </select>
      </label>
      <button
        className="button"
        disabled={loading || page <= 1}
        onClick={() => onChange({ ...params, page: String(page - 1) })}
      >
        {t('previous')}
      </button>
      <span>
        {page} / {pages}
      </span>
      <button
        className="button"
        disabled={loading || !data || page >= pages}
        onClick={() => onChange({ ...params, page: String(page + 1) })}
      >
        {t('next')}
      </button>
    </div>
  );
}
