import { useEffect, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { Link } from 'react-router-dom';
import { ArrowLeft } from 'lucide-react';
import { get, request, message } from '../../lib/api';
import type { Operation } from '../../types/api';
import Details from '../Details';
import { ErrorBox, Loading } from '../Feedback';

export default function OperationDetail({
  id,
  token,
  back,
}: {
  id: string;
  token: string;
  back: () => void;
}) {
  const { t } = useTranslation();
  const [op, setOp] = useState<Operation>();
  const [error, setError] = useState('');
  const [busy, setBusy] = useState(false);
  const [revision, setRevision] = useState(0);
  const [now, setNow] = useState(Date.now());
  useEffect(() => {
    const timer = window.setInterval(() => setNow(Date.now()), 1000);
    return () => clearInterval(timer);
  }, []);
  useEffect(() => {
    const controller = new AbortController();
    get<Operation>('/api/v1/audit/operations/' + encodeURIComponent(id), token, controller.signal)
      .then(setOp)
      .catch((error) => {
        if (!controller.signal.aborted) setError(message(error));
      });
    return () => controller.abort();
  }, [id, token, revision]);
  const confirm = async () => {
    setBusy(true);
    setError('');
    try {
      await request('/api/v1/operations/' + encodeURIComponent(id) + '/confirm', token, {
        method: 'POST',
      });
      setOp(await get<Operation>('/api/v1/audit/operations/' + encodeURIComponent(id), token));
    } catch (error) {
      setError(message(error));
      setOp(undefined);
      setRevision((value) => value + 1);
    } finally {
      setBusy(false);
    }
  };
  const expired = !!op?.expires_at && new Date(op.expires_at).getTime() <= now;
  return (
    <section className="panel">
      <div className="panel-head">
        <button className="back-button" onClick={back}>
          <ArrowLeft size={15} />
          Audit
        </button>
        <button
          className="button"
          onClick={() => {
            setError('');
            setRevision((value) => value + 1);
          }}
        >
          {t('refresh')}
        </button>
      </div>
      <div className="detail-body">
        <h2>{t('operationDetail')}</h2>
        <ErrorBox error={error} />
        {op ? (
          <>
            {['database', 'linux', 'kubernetes'].includes(op.resource_type) && op.resource && (
              <Link
                className="link"
                to={'/' + op.resource_type + '/' + encodeURIComponent(op.resource)}
              >
                {t('resourceDetail')}
              </Link>
            )}
            {[
              [
                'basicInfo',
                ['operation_id', 'timestamp', 'resource_type', 'resource', 'environment', 'target'],
              ],
              [
                'requestInfo',
                ['request_id', 'client', 'tool', 'action', 'statement', 'expires_at'],
              ],
              ['policyResult', ['risk', 'decision', 'reason']],
              [
                'executionResult',
                ['status', 'confirmed_by', 'duration_ms', 'affected_rows', 'error'],
              ],
            ].map(([title, keys]) => (
              <section key={String(title)}>
                <h3 className="section-title">{t(String(title))}</h3>
                <Details
                  value={Object.fromEntries(
                    Object.entries(op).filter(([key]) => (keys as string[]).includes(key)),
                  )}
                />
              </section>
            ))}
            {op.status === 'pending' && (
              <div className="approval">
                <p>{expired ? t('operationExpired') : t('confirmHint')}</p>
                <button
                  className="button primary"
                  disabled={busy || expired}
                  onClick={() => void confirm()}
                >
                  {busy ? t('confirming') : t('confirmAction')}
                </button>
              </div>
            )}
          </>
        ) : (
          !error && <Loading />
        )}
      </div>
    </section>
  );
}
