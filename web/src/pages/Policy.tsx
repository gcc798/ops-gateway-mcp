import { useTranslation } from 'react-i18next';

export default function Policy() {
  const { t } = useTranslation();
  return (
    <section className="panel policy">
      <div className="panel-head">
        <h3>{t('policyDefaults')}</h3>
      </div>
      <div className="rules">
        <div>
          <b>ALLOW</b>
          <p>{t('allowRule')}</p>
        </div>
        <div>
          <b>CONFIRM</b>
          <p>{t('confirmRule')}</p>
        </div>
        <div>
          <b>DENY</b>
          <p>{t('denyRule')}</p>
        </div>
      </div>
    </section>
  );
}
