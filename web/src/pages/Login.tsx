import { useState } from 'react';
import { useTranslation } from 'react-i18next';
import { request, message } from '../lib/api';
import { ErrorBox } from '../components/Feedback';

export default function Login({ onLogin }: { onLogin: (token: string) => void }) {
  const { t } = useTranslation();
  const [value, setValue] = useState('');
  const [error, setError] = useState('');
  const [loading, setLoading] = useState(false);
  const submit = async () => {
    const token = value.trim();
    if (!token) return;
    setLoading(true);
    setError('');
    try {
      await request('/auth/login', token, { method: 'POST' });
      onLogin(token);
    } catch (error) {
      setError(message(error));
    } finally {
      setLoading(false);
    }
  };
  return (
    <div className="login-shell">
      <form
        className="login-panel"
        onSubmit={(event) => {
          event.preventDefault();
          void submit();
        }}
      >
        <div className="login-mark">A</div>
        <p className="eyebrow">AI OPS GATEWAY</p>
        <h1>{t('signIn')}</h1>
        <label>
          {t('tokenLabel')}
          <input
            type="password"
            value={value}
            onChange={(event) => setValue(event.target.value)}
            autoFocus
            autoComplete="off"
            placeholder={t('tokenPlaceholder')}
          />
        </label>
        <ErrorBox error={error} />
        <button className="login-button" disabled={!value.trim() || loading}>
          {loading ? t('loading') : t('signIn')}
        </button>
      </form>
    </div>
  );
}
