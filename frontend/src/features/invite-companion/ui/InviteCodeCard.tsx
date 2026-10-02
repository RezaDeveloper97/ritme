'use client';

import { useLocale, useTranslations } from 'next-intl';
import { useEffect, useRef, useState, type ReactNode } from 'react';

import { type CompanionInvite, hoursUntil } from '@/entities/companion';
import type { Locale } from '@/shared/i18n';
import { formatNumber } from '@/shared/lib/date';
import { Icon } from '@/shared/ui';

import { copyText, shareText } from '../lib/share';

const CODE_SLOTS = 6;

interface InviteCodeCardProps {
  /** The one-time invite; `null` before it exists (placeholder slots + `action`). */
  invite: CompanionInvite | null;
  title?: ReactNode;
  lead?: ReactNode;
  /** Shown in place of copy/share while there is no code («فقط کد بساز»). */
  action?: ReactNode;
  className?: string;
}

/**
 * «کد همدم» (Hamdam_Invite): the six code slots, «کپی کد» / «فرستادن» (Web
 * Share, else the clipboard) and «یک‌بارمصرف · تا ۲۴ ساعت معتبر». The code is
 * only ever in this component's props — never logged, stored or put in a URL.
 */
export function InviteCodeCard({ invite, title, lead, action, className }: InviteCodeCardProps) {
  const t = useTranslations('companions');
  const locale = useLocale() as Locale;
  const [feedback, setFeedback] = useState<'copied' | 'shared' | 'failed' | null>(null);
  const timer = useRef<ReturnType<typeof setTimeout> | null>(null);

  useEffect(() => () => {
    if (timer.current) clearTimeout(timer.current);
  }, []);

  const flash = (value: 'copied' | 'shared' | 'failed') => {
    setFeedback(value);
    if (timer.current) clearTimeout(timer.current);
    timer.current = setTimeout(() => setFeedback(null), 2500);
  };

  const code = invite?.code ?? '';
  const slots = Array.from({ length: Math.max(CODE_SLOTS, code.length) }, (_, i) => code[i] ?? '');
  const hours = invite ? hoursUntil(invite.expiresAt) : 24;

  const onCopy = async () => flash((await copyText(code)) ? 'copied' : 'failed');
  const onShare = async () => {
    const outcome = await shareText(t('code.shareTitle'), t('code.shareText', { code }));
    if (outcome !== 'cancelled') flash(outcome);
  };

  return (
    <section className={className ? `nb-card cmp-code ${className}` : 'nb-card cmp-code'} aria-labelledby="cmp-code-title">
      <div className="cmp-code-text">
        <h3 id="cmp-code-title" className="cmp-code-title">
          {title ?? t('flow.invite.codeTitle')}
        </h3>
        {lead ? <p className="cmp-code-lead">{lead}</p> : null}
      </div>
      <div
        className={invite ? 'cmp-code-slots' : 'cmp-code-slots is-empty'}
        role={invite ? 'img' : undefined}
        aria-label={invite ? t('code.label', { code: code.split('').join(' ') }) : undefined}
        dir="ltr"
      >
        {slots.map((ch, i) => (
          <span key={i} className="cmp-code-slot" aria-hidden>
            {ch}
          </span>
        ))}
      </div>
      {invite ? (
        <>
          <div className="cmp-code-btns">
            <button type="button" className="cmp-code-btn" onClick={() => void onCopy()}>
              <Icon name={feedback === 'copied' ? 'check' : 'copy'} size={16} />
              {feedback === 'copied' ? t('code.copied') : t('code.copy')}
            </button>
            <button type="button" className="cmp-code-btn" onClick={() => void onShare()}>
              <Icon name={feedback === 'shared' ? 'check' : 'share'} size={16} />
              {feedback === 'shared' ? t('code.shared') : t('code.share')}
            </button>
          </div>
          <p className="cmp-code-meta">{t('code.validity', { hours: formatNumber(hours, locale) })}</p>
          <p className="cmp-code-once">{t('code.once')}</p>
        </>
      ) : (
        <>
          <p className="cmp-code-meta">{t('flow.invite.codeHidden')}</p>
          {action}
        </>
      )}
      <p className="cmp-code-live" role="status" aria-live="polite">
        {feedback === 'failed' ? t('code.failed') : feedback === 'copied' ? t('code.copied') : ''}
      </p>
    </section>
  );
}
