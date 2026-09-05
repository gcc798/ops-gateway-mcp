import { useTranslation } from 'react-i18next';

export default function Details({ value }: { value: object }) {
  const { t } = useTranslation();
  return (
    <dl className="details">
      {Object.entries(value).map(([key, value]) => (
        <div key={key}>
          <dt>{t('fields.' + key, { defaultValue: key })}</dt>
          <dd>
            <pre>{String(value ?? '') || '—'}</pre>
          </dd>
        </div>
      ))}
    </dl>
  );
}
