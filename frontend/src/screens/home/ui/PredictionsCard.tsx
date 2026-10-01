'use client';

import clsx from 'clsx';
import { useLocale, useTranslations } from 'next-intl';
import type { ReactNode } from 'react';

import { Link, type Locale } from '@/shared/i18n';
import { formatNumber } from '@/shared/lib/date';
import { Card, Icon, IconCircle, StatusPill, type IconName, type Tone } from '@/shared/ui';

/** One predicted event: its display dates and the days from today to its start / end. */
export interface PredictionSlot {
  text: string;
  startIn: number;
  endIn: number;
}

export interface CycleLengthSummary {
  median: number;
  basedOnCycles: number | null;
  spreadDays: number | null;
  regularity: string | null;
}

const REGULARITY: Record<string, { tone: Tone; icon?: IconName }> = {
  relatively_regular: { tone: 'data', icon: 'check' },
  irregular_possible: { tone: 'warm' },
  not_enough_data: { tone: 'neutral' },
};

/**
 * «پیش‌بینی‌ها» (nbl_Cycle_Home): next period, PMS, fertile window and
 * ovulation as dot rows, then the cycle length (median ± spread) with its
 * regularity pill. The dates are the engine's (`cycle_view` + `/home/cycle-overview`);
 * this card only words the countdown.
 */
export function PredictionsCard({
  nextPeriod,
  pms,
  window,
  ovulation,
  length,
  footer,
  hideFertility = false,
}: {
  nextPeriod: PredictionSlot | null;
  pms: PredictionSlot | null;
  window: PredictionSlot | null;
  ovulation: string | null;
  length: CycleLengthSummary | null;
  /** Optional extra (the profile-vs-recent-cycles sync nudge). */
  footer?: ReactNode;
  /** Teen mode (B-N2-03): no fertile-window / ovulation rows. */
  hideFertility?: boolean;
}) {
  const t = useTranslations('home.nb.predictions');
  const tHome = useTranslations('home');
  const loc = useLocale() as Locale;
  const num = (n: number) => formatNumber(n, loc);
  const dash = tHome('unavailable');

  const when = (slot: PredictionSlot | null, range: boolean): string | null => {
    if (!slot) return null;
    if (slot.startIn > 1) return t('inDays', { n: num(slot.startIn) });
    if (slot.startIn === 1) return range ? t('fromTomorrow') : t('tomorrow');
    if (slot.startIn === 0) return t('today');
    return slot.endIn >= 0 ? t('ongoing') : null;
  };

  const allRows = [
    { key: 'period', label: t('nextPeriod'), value: nextPeriod?.text ?? dash, rel: when(nextPeriod, false) },
    { key: 'pms', label: t('pms'), value: pms?.text ?? dash, rel: when(pms, true) },
    { key: 'fertile', label: t('window'), value: window?.text ?? dash, rel: when(window, true) },
    {
      key: 'ovulation',
      label: t('ovulation'),
      value: ovulation ? t('about', { date: ovulation }) : dash,
      rel: ovulation ? t('estimated') : null,
    },
  ];
  const rows = hideFertility ? allRows.filter((r) => r.key !== 'fertile' && r.key !== 'ovulation') : allRows;

  const reg = length?.regularity ? REGULARITY[length.regularity] : undefined;

  return (
    <Card as="section" className="ch-pred" aria-labelledby="ch-pred-title">
      <div className="ch-card-head">
        <h2 id="ch-pred-title" className="ch-overline">{t('title')}</h2>
        {/* B-N3-08: «جزئیات» opens the analysis hub; the cycle-length row below keeps /cycle (history). */}
        <Link href="/analysis" className="ch-link">{t('details')}</Link>
      </div>
      <ul className="ch-pred-list">
        {rows.map((row) => (
          <li key={row.key} className="ch-pred-row">
            <span className={clsx('ch-pred-dot', `is-${row.key}`)} aria-hidden />
            <span className="ch-pred-label">{row.label}</span>
            <b className="ch-pred-value">{row.value}</b>
            {row.rel && <span className="ch-pred-rel">{row.rel}</span>}
          </li>
        ))}
      </ul>
      {length && (
        <Link href="/cycle" className="ch-len">
          <span className="ch-len-body">
            <span className="ch-len-title">
              {length.basedOnCycles ? t('lengthTitle', { n: num(length.basedOnCycles) }) : t('lengthTitleBasic')}
            </span>
            <span className="ch-len-value">
              <span className="ch-len-num">{num(length.median)}</span>
              <span className="ch-len-unit">
                {length.spreadDays ? t('lengthSpread', { n: num(length.spreadDays) }) : t('lengthUnit')}
              </span>
            </span>
          </span>
          {reg && length.regularity && (
            <StatusPill tone={reg.tone} icon={reg.icon} className="ch-len-pill">
              {t(`regularity.${length.regularity}` as 'regularity.relatively_regular')}
            </StatusPill>
          )}
        </Link>
      )}
      {footer}
    </Card>
  );
}

/**
 * The insight row under the challenges (nbl_Cycle_Home «PMS احتمالاً از فردا»):
 * shown while the PMS window is at most a week away or running. Its «در تحلیل ببین»
 * opens the analysis hub (B-N3-08).
 */
export function PmsInsightCard({ pms, basedOnCycles }: { pms: PredictionSlot | null; basedOnCycles: number | null }) {
  const t = useTranslations('home.nb.insight');
  const loc = useLocale() as Locale;
  if (!pms || pms.startIn > 7 || pms.endIn < 0) return null;
  const title =
    pms.startIn <= 0
      ? t('pmsNow')
      : pms.startIn === 1
        ? t('pmsTomorrow')
        : t('pmsIn', { n: formatNumber(pms.startIn, loc) });
  return (
    <Link href="/analysis" className="ch-insight">
      <IconCircle icon="sparkle" tone="brand" />
      <span className="ch-insight-body">
        <b className="ch-insight-title">{title}</b>
        <span className="ch-insight-sub">
          {basedOnCycles ? t('basis', { n: formatNumber(basedOnCycles, loc) }) : t('basisBasic')}
        </span>
      </span>
      <Icon name="chevronLeft" size={18} strokeWidth={2} className="ch-chev" />
    </Link>
  );
}
