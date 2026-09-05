import { useState } from 'react';
import { useTranslation } from 'react-i18next';
import type { Field, Params } from '../types/api';
import DateTimePicker from './DateTimePicker';

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
        <div
          className={'filter-field' + (field.type === 'datetime-local' ? ' filter-date' : '')}
          key={field.name}
        >
          <label htmlFor={'filter-' + field.name}>{t('fields.' + field.name)}</label>
          {field.type === 'datetime-local' ? (
            <DateTimePicker
              name={field.name}
              value={draft[field.name] || ''}
              onChange={(value) => setDraft({ ...draft, [field.name]: value })}
            />
          ) : field.options ? (
            <select
              id={'filter-' + field.name}
              name={field.name}
              value={draft[field.name] || ''}
              onChange={(event) => setDraft({ ...draft, [field.name]: event.target.value })}
            >
              <option value="">{t('all')}</option>
              {[
                ...new Set([...field.options, ...(draft[field.name] ? [draft[field.name]] : [])]),
              ].map((value) => (
                <option key={value}>{value}</option>
              ))}
            </select>
          ) : (
            <input
              id={'filter-' + field.name}
              name={field.name}
              value={draft[field.name] || ''}
              onChange={(event) => setDraft({ ...draft, [field.name]: event.target.value })}
              type={field.type || 'search'}
              step={field.type === 'datetime-local' ? '1' : undefined}
              placeholder={field.type ? undefined : t('fields.' + field.name)}
            />
          )}
        </div>
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
