import { Check, ChevronDown, ChevronLeft, ChevronRight } from 'lucide-react';
import * as Select from '@radix-ui/react-select';
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
      <div className="pagination-size">
        <span>{t('pageSize')}</span>
        <Select.Root
          value={String(size)}
          onValueChange={(value) => onChange({ ...params, page_size: value, page: '1' })}
        >
          <Select.Trigger className="pagination-select" aria-label={t('pageSize')}>
            <Select.Value />
            <Select.Icon asChild>
              <ChevronDown size={14} />
            </Select.Icon>
          </Select.Trigger>
          <Select.Portal>
            <Select.Content
              className="pagination-menu"
              position="popper"
              side="top"
              align="start"
              sideOffset={6}
              collisionPadding={12}
            >
              <Select.Viewport>
                {[10, 20, 50, 100].map((size) => (
                  <Select.Item className="pagination-option" key={size} value={String(size)}>
                    <Select.ItemText>{size}</Select.ItemText>
                    <Select.ItemIndicator>
                      <Check size={14} />
                    </Select.ItemIndicator>
                  </Select.Item>
                ))}
              </Select.Viewport>
            </Select.Content>
          </Select.Portal>
        </Select.Root>
      </div>
      <button
        className="icon"
        title={t('previous')}
        aria-label={t('previous')}
        disabled={loading || page <= 1}
        onClick={() => onChange({ ...params, page: String(page - 1) })}
      >
        <ChevronLeft size={16} />
      </button>
      <span className="pagination-position">
        {page} / {pages}
      </span>
      <button
        className="icon"
        title={t('next')}
        aria-label={t('next')}
        disabled={loading || !data || page >= pages}
        onClick={() => onChange({ ...params, page: String(page + 1) })}
      >
        <ChevronRight size={16} />
      </button>
    </div>
  );
}
