import { useTranslation } from 'react-i18next';
import { Network } from 'lucide-react';

export function ErrorBox({ error }: { error: string }) {
  return error ? (
    <div role="alert" className="error">
      {error}
    </div>
  ) : null;
}
export function Loading() {
  const { t } = useTranslation();
  return (
    <p role="status" className="loading">
      {t('loading')}
    </p>
  );
}
export function Empty({ text }: { text: string }) {
  return (
    <div className="placeholder">
      <Network size={30} />
      <h2>{text}</h2>
    </div>
  );
}
