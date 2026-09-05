import DatePicker from 'react-datepicker';
import { zhCN, enUS } from 'date-fns/locale';
import { CalendarDays } from 'lucide-react';
import { useTranslation } from 'react-i18next';
import 'react-datepicker/dist/react-datepicker.css';

export default function DateTimePicker({
  name,
  value,
  onChange,
}: {
  name: string;
  value: string;
  onChange: (value: string) => void;
}) {
  const { t, i18n } = useTranslation();
  const date = value ? new Date(value) : null;
  return (
    <div className="date-time-picker">
      <input type="hidden" name={name} value={value} />
      <DatePicker
        id={'filter-' + name}
        selected={date && !Number.isNaN(date.getTime()) ? date : null}
        onChange={(date: Date | null) => onChange(date?.toISOString() || '')}
        locale={i18n.language === 'zh-CN' ? zhCN : enUS}
        dateFormat="yyyy-MM-dd HH:mm:ss"
        placeholderText={t('chooseDateTime')}
        showMonthDropdown
        showYearDropdown
        dropdownMode="select"
        isClearable
        clearButtonTitle={t('clearDate')}
        ariaLabelClose={t('clearDate')}
        previousMonthAriaLabel={t('previousMonth')}
        nextMonthAriaLabel={t('nextMonth')}
        showIcon
        icon={<CalendarDays size={16} />}
        toggleCalendarOnIconClick
        popperPlacement="bottom-start"
        autoComplete="off"
        strictParsing
        shouldCloseOnSelect={false}
      >
        <label className="calendar-time">
          {t('fields.timestamp')}
          <input
            type="time"
            step="1"
            value={
              date && !Number.isNaN(date.getTime())
                ? [date.getHours(), date.getMinutes(), date.getSeconds()]
                    .map((value) => String(value).padStart(2, '0'))
                    .join(':')
                : '00:00:00'
            }
            onChange={(event) => {
              if (!event.target.value) return;
              const next = date && !Number.isNaN(date.getTime()) ? new Date(date) : new Date();
              const [hours, minutes, seconds = 0] = event.target.value.split(':').map(Number);
              next.setHours(hours, minutes, seconds, 0);
              onChange(next.toISOString());
            }}
          />
        </label>
      </DatePicker>
    </div>
  );
}
