import { useState } from 'react';
import { Check, Copy } from 'lucide-react';
import { useTranslation } from 'react-i18next';

export default function CopyButton({ value }: { value: string }) {
  const { t } = useTranslation();
  const [state, setState] = useState('copy');
  return (
    <span className="copy-control">
      <button
        className="icon"
        title={t(state)}
        aria-label={t(state)}
        onClick={async () => {
          try {
            await navigator.clipboard.writeText(value);
            setState('copied');
          } catch {
            setState('copyFailed');
          }
        }}
      >
        {state === 'copied' ? <Check size={15} /> : <Copy size={15} />}
      </button>
      <span className="sr-only" role="status">
        {state !== 'copy' && t(state)}
      </span>
    </span>
  );
}
