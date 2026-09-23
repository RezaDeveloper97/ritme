'use client';

import { useTranslations } from 'next-intl';

import { Icon } from './Icon';
import { useToastStore } from './toast';

/** Mounted once in app/providers. Polite live region; errors are assertive. */
export function Toaster() {
  const items = useToastStore((s) => s.items);
  const dismiss = useToastStore((s) => s.dismiss);
  const t = useTranslations('common');
  return (
    <div className="toast-stack" aria-live="polite">
      {items.map((item) => (
        <div key={item.id} className="toast" data-tone={item.tone} role={item.tone === 'error' ? 'alert' : 'status'}>
          <span className="flex-1">{item.text}</span>
          <button
            type="button"
            className="btn btn-ghost btn-sm btn-icon"
            onClick={() => dismiss(item.id)}
            aria-label={t('close')}
          >
            <Icon name="close" size={14} />
          </button>
        </div>
      ))}
    </div>
  );
}
