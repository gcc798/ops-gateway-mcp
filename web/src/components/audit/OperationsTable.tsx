import { useTranslation } from 'react-i18next';
import type { Operation } from '../../types/api';
import { Empty } from '../Feedback';

export default function Operations({
  items,
  select,
}: {
  items: Operation[];
  select: (id: string) => void;
}) {
  const { t } = useTranslation();
  if (!items.length) return <Empty text={t('noResults')} />;
  return (
    <div className="table-scroll">
      <table>
        <thead>
          <tr>
            {[
              'timestamp',
              'tool',
              'resource_type',
              'resource',
              'environment',
              'decision',
              'status',
            ].map((key) => (
              <th key={key}>{t('fields.' + key)}</th>
            ))}
            <th>{t('details')}</th>
          </tr>
        </thead>
        <tbody>
          {items.map((op) => (
            <tr key={op.operation_id}>
              <td>{new Date(op.timestamp).toLocaleString()}</td>
              <td>{op.tool}</td>
              <td>{op.resource_type}</td>
              <td>{op.resource || '—'}</td>
              <td>{op.environment || '—'}</td>
              <td className={'decision-' + op.decision}>{op.decision}</td>
              <td>{op.status}</td>
              <td>
                <button className="link" onClick={() => select(op.operation_id)}>
                  {t('details')} →
                </button>
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}
