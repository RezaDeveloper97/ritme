'use client';

import { useTranslations } from 'next-intl';

import { Link } from '@/shared/i18n';
import { IconCircle } from '@/shared/ui';
import '@/shared/ui/styles/ch.css';

/** After a positive test the engine offers pregnancy mode (`limits.prompt_pregnancy_mode`); never forced. */
export function PregnancyPromptCard({ onDismiss }: { onDismiss: () => void }) {
  const t = useTranslations('home.nb.limits.pregnancyPrompt');
  return (
    <div role="status" className="ch-alert" data-banner="pregnancy_prompt">
      <IconCircle icon="info" tone="warm" size="sm" />
      <span className="ch-alert-body">
        <b className="ch-alert-title">{t('title')}</b>
        <span className="ch-alert-text">{t('body')}</span>
        <Link href="/profile/mode" className="ch-link">
          {t('cta')}
        </Link>
        <button type="button" className="ch-link" onClick={onDismiss}>
          {t('dismiss')}
        </button>
      </span>
    </div>
  );
}
