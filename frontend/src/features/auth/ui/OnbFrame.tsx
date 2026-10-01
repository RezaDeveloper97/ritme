'use client';

import { useTranslations } from 'next-intl';
import type { ReactNode } from 'react';
import { clsx } from 'clsx';

import { useDirection } from '@/shared/i18n';
import { HeaderButton, Icon, PrimaryButton, ProgressSteps, SecondaryButton, SkyLayer } from '@/shared/ui';

interface OnbFrameProps {
  /** Filled header segments; omit for a screen without the bar (Ready). */
  progress?: { current: number; total: number };
  /** Back arrow target handler — routes forward to a known screen (CLAUDE.md §4.2). */
  onBack?: () => void;
  /** «رد کردن» at the header end (optional steps). */
  onSkip?: () => void;
  title?: ReactNode;
  subtitle?: ReactNode;
  /** The primary CTA (54px pill) at the bottom. */
  primary?: { label: ReactNode; onClick: () => void; disabled?: boolean; loading?: boolean };
  /** A text button under the CTA («الان نه», «بعداً وصل می‌شوم»). */
  secondary?: { label: ReactNode; onClick: () => void; disabled?: boolean };
  /** The «اطلاعات تو نزد ریتمی محرمانه است…» line above the CTA (default on). */
  privacyNote?: boolean;
  className?: string;
  children?: ReactNode;
}

/**
 * The frame every sign-up and onboarding screen shares (nbl_Onb_*): sky
 * layer, 44px back button + progress bar + «رد کردن» header, a display title
 * and subtitle, the scrolling body, and the footer with the privacy line and
 * the CTA. Lives with the auth feature because the phone and OTP screens use
 * it as well as the onboarding steps (sibling screens can't share a slice).
 */
export function OnbFrame({
  progress,
  onBack,
  onSkip,
  title,
  subtitle,
  primary,
  secondary,
  privacyNote = true,
  className,
  children,
}: OnbFrameProps) {
  const t = useTranslations('common.onbFrame');
  const rtl = useDirection() === 'rtl';
  const hasHeader = Boolean(onBack || progress || onSkip);
  return (
    <div className={clsx('view onb2', className)}>
      <SkyLayer />
      {hasHeader ? (
        <header className="onb2-hdr">
          {onBack ? (
            <HeaderButton label={t('back')} icon={rtl ? 'arrowR' : 'arrowL'} onClick={onBack} />
          ) : (
            <span className="nb-hdr-spacer" aria-hidden />
          )}
          {progress ? (
            <ProgressSteps
              className="onb2-steps"
              total={progress.total}
              current={progress.current}
              label={t('progress', { step: progress.current, total: progress.total })}
            />
          ) : (
            <span className="onb2-steps" />
          )}
          {onSkip ? (
            <button type="button" className="onb2-skip" onClick={onSkip}>
              {t('skip')}
            </button>
          ) : null}
        </header>
      ) : null}

      <div className="scroll onb2-body">
        {title ? <h1 className="onb2-title">{title}</h1> : null}
        {subtitle ? <p className="onb2-sub">{subtitle}</p> : null}
        {children}
      </div>

      {primary || secondary ? (
        <div className="onb2-foot">
          {privacyNote ? (
            <p className="onb2-privacy">
              <Icon name="shield" size={16} className="onb2-privacy-icon" />
              <span>{t('privacy')}</span>
            </p>
          ) : null}
          {primary ? (
            <PrimaryButton onClick={primary.onClick} disabled={primary.disabled} loading={primary.loading}>
              {primary.label}
            </PrimaryButton>
          ) : null}
          {secondary ? (
            <SecondaryButton variant="text" onClick={secondary.onClick} disabled={secondary.disabled}>
              {secondary.label}
            </SecondaryButton>
          ) : null}
        </div>
      ) : null}
    </div>
  );
}
