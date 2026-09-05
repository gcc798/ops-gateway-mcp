import { useState } from 'react';
import { useTranslation } from 'react-i18next';
import type { Field, Params } from '../types/api';

export default function Filters({
  fields,
  values,
  onApply,
}: {
  fields: Field[];
  values: Params;
  onApply: (params: Params) => void;
}) {
  const { t } = useTranslation();
  const [draft, setDraft] = useState<Params>(() =>
    Object.fromEntries(
      fields.map((field) => {
        let value = values[field.name] || '';
        if (value && field.type === 'datetime-local') {
          const date = new Date(value);
          value = new Date(date.getTime() - date.getTimezoneOffset() * 60000)
            .toISOString()
            .slice(0, 19);
        }
        return [field.name, value];
      }),
    ),
  );
  return (
    <form
      className="filters"
      onSubmit={(event) => {
        event.preventDefault();
        const values: Params = {};
        for (const [key, raw] of new FormData(event.currentTarget)) {
          const value = String(raw).trim();
          if (value)
            values[key] = key === 'from' || key === 'to' ? new Date(value).toISOString() : value;
        }
        onApply(values);
      }}
    >
      {fields.map((field) => (
        <label key={field.name}>
          {t('fields.' + field.name)}
          {field.options ? (
            <select
              name={field.name}
              value={draft[field.name] || ''}
              onChange={(event) => setDraft({ ...draft, [field.name]: event.target.value })}
            >
              <option value="">{t('all')}</option>
              {field.options.map((value) => (
                <option key={value}>{value}</option>
              ))}
            </select>
          ) : (
            <input
              name={field.name}
              value={draft[field.name] || ''}
              onChange={(event) => setDraft({ ...draft, [field.name]: event.target.value })}
              type={field.type || 'search'}
              step={field.type === 'datetime-local' ? '1' : undefined}
              placeholder={field.type ? undefined : t('fields.' + field.name)}
            />
          )}
        </label>
      ))}
      <div className="filter-actions">
        <button className="button primary" type="submit">
          {t('search')}
        </button>
        <button
          className="button"
          type="button"
          onClick={() => {
            setDraft({});
            onApply({});
          }}
        >
          {t('reset')}
        </button>
      </div>
    </form>
  );
}
