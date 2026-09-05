import { useEffect, useState } from 'react';
import { useTranslation } from 'react-i18next';
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
      setOp(undefined);
      setRevision((value) => value + 1);
    } catch (error) {
      setError(message(error));
      setRevision((value) => value + 1);
    } finally {
      setBusy(false);
    }
  };
  const expired = !!op?.expires_at && new Date(op.expires_at).getTime() <= Date.now();
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
            <Details value={op} />
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
