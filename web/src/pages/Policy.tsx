import { useEffect, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { get, message } from '../lib/api';
import { ErrorBox, Loading } from '../components/Feedback';

type Rule = { domain: string; action: string; decision: string };
export default function Policy({ token }: { token: string }) {
  const { t } = useTranslation();
  const [rules, setRules] = useState<Rule[]>();
  const [error, setError] = useState('');
  useEffect(() => {
    const controller = new AbortController();
    get<Rule[]>('/api/v1/policies', token, controller.signal)
      .then(setRules)
      .catch((error) => {
        if (!controller.signal.aborted) setError(message(error));
      });
    return () => controller.abort();
  }, [token]);
  return (
    <section className="panel">
      <div className="panel-head">
        <h3>{t('policyDefaults')}</h3>
        <span className="muted">{t('readOnly')}</span>
      </div>
      <ErrorBox error={error} />
      {rules ? (
        <div className="table-scroll">
          <table>
            <thead>
              <tr>
                <th>{t('fields.resource_type')}</th>
                <th>{t('fields.action')}</th>
                <th>{t('fields.decision')}</th>
              </tr>
            </thead>
            <tbody>
              {rules.map((rule) => (
                <tr key={rule.domain + rule.action}>
                  <td>{rule.domain}</td>
                  <td>{t('policyActions.' + rule.action, { defaultValue: rule.action })}</td>
                  <td className={'decision-' + rule.decision}>{rule.decision.toUpperCase()}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      ) : (
        !error && <Loading />
      )}
    </section>
  );
}
