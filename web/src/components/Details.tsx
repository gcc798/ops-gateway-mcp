import CopyButton from './CopyButton';
import { useTranslation } from 'react-i18next';

export default function Details({ value }: { value: object }) {
  const { t } = useTranslation();
  return (
    <dl className="details">
      {Object.entries(value).map(([key, value]) => (
        <div key={key}>
          <dt>{t('fields.' + key, { defaultValue: key })}</dt>
          <dd>
            <pre>
              {typeof value === 'object' && value !== null
                ? JSON.stringify(value, null, 2)
                : String(value ?? '') || '—'}
            </pre>
            <CopyButton
              value={
                typeof value === 'object' && value !== null
                  ? JSON.stringify(value, null, 2)
                  : String(value ?? '')
              }
            />
          </dd>
        </div>
      ))}
    </dl>
  );
}
