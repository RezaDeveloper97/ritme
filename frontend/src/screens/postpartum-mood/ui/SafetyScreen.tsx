'use client';

import { useLocale, useTranslations } from 'next-intl';

import type { SafetyMessage } from '@/entities/postpartum';
import { Link, type Locale } from '@/shared/i18n';
import { formatNumber } from '@/shared/lib/date';
import { Card, Icon, SecondaryButton } from '@/shared/ui';

import { safetyCalls } from '../model/flow';

type T = ReturnType<typeof useTranslations<'postpartum'>>;

/** The urgent path: call buttons come first and render whatever the network did. */
export function SafetyScreen({
  safety,
  saved,
  retrying,
  onRetry,
  t,
}: {
  safety: SafetyMessage | null;
  saved: boolean;
  retrying: boolean;
  onRetry?: () => void;
  t: T;
}) {
  const locale = useLocale() as Locale;
  const calls = safetyCalls(safety);
  return (
    <div className="pp-form pp-safety" role="alert" aria-labelledby="pp-safety-title">
      <ul className="pp-safety-calls" aria-label={t('mood.safety.hotlines')}>
        {calls.map((c, i) => (
          <li key={c.number}>
            <a href={`tel:${c.number}`} className={i === 0 ? 'nb-btn is-primary is-block pp-call' : 'nb-btn is-outline is-block pp-call'}>
              <Icon name="phone" size={20} />
              <span>{c.label ?? (c.labelKey ? t(`mood.safety.${c.labelKey}`) : formatNumber(c.number, locale))}</span>
            </a>
          </li>
        ))}
      </ul>
      <Card as="section" className="pp-safety-card">
        <h2 id="pp-safety-title" className="pp-safety-title">
          {safety?.title ?? t('mood.safety.title')}
        </h2>
        <p className="pp-safety-body">{safety?.body ?? t('mood.safety.body')}</p>
      </Card>
      {!saved && (
        <>
          <p className="pp-result-note">{t('mood.safety.notSaved')}</p>
          {onRetry && (
            <SecondaryButton icon="refresh" loading={retrying} onClick={onRetry}>
              {t('mood.safety.retrySave')}
            </SecondaryButton>
          )}
        </>
      )}
      <Link href="/postpartum" className="nb-btn is-text is-block">
        {t('mood.safety.home')}
      </Link>
    </div>
  );
}
