'use client';

import { useLocale, useTranslations } from 'next-intl';

import type { Locale } from '@/shared/i18n';
import { formatNumber } from '@/shared/lib/date';
import { IconCircle } from '@/shared/ui';

import { bannerActions, type BannerAction, type LatenessTier, type TodayBannerKind } from '../model/today-ui';

export interface TodayBannerProps {
  banner: TodayBannerKind;
  daysLate: number;
  tier: LatenessTier;
  showPregnancyTest: boolean;
  /** `end_missing` → the period editor on the open period. */
  onEditEnd: () => void;
  /** `ask_last_period` (and the long overdue tier) → the period date editor. */
  onAskLastPeriod: () => void;
  /** `overdue` → the log sheet for today. */
  onLogPregnancyTest: () => void;
  /** «پریود نداشتم» on the overdue banner; omitted when the skip is not offered. */
  onSkip?: () => void;
}

/** The single Today banner (§2.43): one of end missing, overdue (text by tier) or ask for the last period. */
export function TodayBanner({
  banner,
  daysLate,
  tier,
  showPregnancyTest,
  onEditEnd,
  onAskLastPeriod,
  onLogPregnancyTest,
  onSkip,
}: TodayBannerProps) {
  const t = useTranslations('home.nb');
  const tSkip = useTranslations('home.skip');
  const loc = useLocale() as Locale;

  const handlers: Record<BannerAction, { label: string; onClick: () => void }> = {
    edit_end: { label: t('endMissing.cta'), onClick: onEditEnd },
    log_pregnancy_test: {
      label: t('banner.overdue.test'),
      onClick: onLogPregnancyTest,
    },
    edit_last_period: {
      label: t('banner.askLast.cta'),
      onClick: onAskLastPeriod,
    },
  };
  const actions = bannerActions(banner, tier, showPregnancyTest);

  const title =
    banner === 'end_missing'
      ? t('endMissing.title')
      : banner === 'overdue'
        ? t('banner.overdue.title', { n: formatNumber(daysLate, loc) })
        : t('banner.askLast.title');
  const text =
    banner === 'end_missing'
      ? t('endMissing.text')
      : banner === 'overdue'
        ? // Without the test button (teen, menopause, suppression) the copy never mentions pregnancy.
          showPregnancyTest
          ? t(tier === 'ask' ? 'banner.overdue.textLong' : 'banner.overdue.text')
          : t(tier === 'ask' ? 'banner.overdue.textLongNoTest' : 'banner.overdue.textNoTest')
        : t('banner.askLast.text');

  return (
    <div role="status" className="ch-alert" data-banner={banner}>
      <IconCircle icon="info" tone="warm" size="sm" />
      <span className="ch-alert-body">
        <b className="ch-alert-title">{title}</b>
        <span className="ch-alert-text">{text}</span>
        {actions.map((a) => (
          <button key={a} type="button" className="ch-link" data-action={a} onClick={handlers[a].onClick}>
            {handlers[a].label}
          </button>
        ))}
        {banner === 'overdue' && onSkip && (
          <button type="button" className="ch-link" data-action="skip_period" onClick={onSkip}>
            {tSkip('action')}
          </button>
        )}
      </span>
    </div>
  );
}
