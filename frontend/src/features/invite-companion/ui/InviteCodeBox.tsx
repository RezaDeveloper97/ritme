'use client';

import { useLocale, useTranslations } from 'next-intl';
import { useEffect, useRef, useState } from 'react';

import { appInviteUrl, type CompanionInvite, hoursUntil } from '@/entities/companion';
import type { Locale } from '@/shared/i18n';
import { formatNumber } from '@/shared/lib/date';
import { Icon, PrimaryButton, SecondaryButton } from '@/shared/ui';

import { copyText, shareText } from '../lib/share';

interface InviteCodeBoxProps {
  invite: CompanionInvite;
  className?: string;
}

/**
 * The ready code (companion-v2 `code`): the big dictatable code, «تا N ساعت
 * دیگر معتبر است», «اشتراک‌گذاری کد» (Web Share — the code plus the app's
 * public address — else the clipboard) and «کپی کد». The code is only ever in
 * this component's props — never logged, stored or put in a URL.
 */
export function InviteCodeBox({ invite, className }: InviteCodeBoxProps) {
  const t = useTranslations('companions.code');
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

  const { code } = invite;
  const hours = hoursUntil(invite.expiresAt);

  const onCopy = async () => flash((await copyText(code)) ? 'copied' : 'failed');
  const onShare = async () => {
    const outcome = await shareText(t('shareTitle'), t('shareText', { code, url: appInviteUrl() }));
    if (outcome !== 'cancelled') flash(outcome);
  };

  let live = '';
  if (feedback === 'failed') live = t('failed');
  else if (feedback === 'copied') live = t('copied');
  else if (feedback === 'shared') live = t('shared');

  return (
    <section className={className ? `cmp2-code ${className}` : 'cmp2-code'} aria-label={t('title')}>
      <div className="nb-card cmp2-codebox">
        <p className="cmp2-code-value" dir="ltr" role="img" aria-label={t('label', { code: code.split('').join(' ') })}>
          {code}
        </p>
        <p className={hours > 0 ? 'cmp2-code-exp' : 'cmp2-code-exp is-expired'}>
          <Icon name="clock" size={14} />
          {hours > 0 ? t('validity', { hours: formatNumber(hours, locale) }) : t('expired')}
        </p>
      </div>
      <div className="cmp2-code-btns">
        <PrimaryButton icon={feedback === 'shared' ? 'check' : 'share'} onClick={() => void onShare()}>
          {t('share')}
        </PrimaryButton>
        <SecondaryButton icon={feedback === 'copied' ? 'check' : 'copy'} onClick={() => void onCopy()}>
          {feedback === 'copied' ? t('copied') : t('copy')}
        </SecondaryButton>
      </div>
      <p className="cmp2-privacy">
        <Icon name="shield" size={16} className="cmp2-privacy-icon" />
        <span>{t('accessNote')}</span>
      </p>
      <p className={feedback === 'failed' ? 'cmp2-code-live is-failed' : 'cmp2-code-live'} role="status" aria-live="polite">
        {live}
      </p>
    </section>
  );
}
