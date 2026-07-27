'use client';

import clsx from 'clsx';

import { useLocale, useTranslations } from 'next-intl';

import {
  useWeeklyWellbeing,
  weeklyWellbeingTone,
  wellbeingTrend,
  type WellbeingMetricKey,
} from '@/entities/wellbeing';
import { Link, type Locale } from '@/shared/i18n';
import { formatJalaliDayMonth, fromApiDate } from '@/shared/lib/date';
import { Icon, type IconName } from '@/shared/ui';

/** Tile presentation per metric — a UI concern, so it stays out of the entity. */
const TILE: Record<WellbeingMetricKey, { icon: IconName; color: string }> = {
  mood: { icon: 'smile', color: 'var(--blush)' },
  sleep: { icon: 'moon', color: 'var(--indigo)' },
  energy: { icon: 'zap', color: 'var(--amber)' },
};

const TREND_COLOR = { up: 'var(--green-deep)', down: 'var(--danger-deep)' } as const;

/**
 * «خلاصه هفته» — the real seven-day averages of mood, sleep and energy, scored
 * by the backend from the user's daily logs, each with its change against the
 * week before.
 *
 * A metric with nothing scoreable shows a dash rather than 0%, and a week with
 * no logs at all turns the caption into an invitation to log — the card never
 * implies the user "scored zero" on a week it knows nothing about.
 */
export function WeekSummaryCard({ date }: { date?: string }) {
  const t = useTranslations('home.weekSummary');
  const locale = useLocale() as Locale;
  // `isLoading` (not `isPending`) — the query is disabled until the session is
  // ready, and a disabled query stays "pending" forever.
  const { data: week, isLoading, isError } = useWeeklyWellbeing(date);

  const dash = t('unavailable');
  const metrics = week?.metrics ?? (['mood', 'sleep', 'energy'] as WellbeingMetricKey[]).map((key) => ({
    key,
    percent: null,
    previousPercent: null,
    delta: null,
  }));
  const tone = weeklyWellbeingTone(week?.overallPercent ?? null);
  // Three distinct states, because "no logs at all" and "logs that carried no
  // mood/sleep/energy" need different invitations — telling someone who logged
  // their period that they've "logged nothing" is simply wrong.
  const nothingLogged = !week || week.loggedDays === 0;
  const nothingScored = !nothingLogged && week.overallPercent === null;

  const subtitle = week
    ? t('range', {
        from: formatJalaliDayMonth(fromApiDate(week.from), locale),
        to: formatJalaliDayMonth(fromApiDate(week.to), locale),
      })
    : t('subtitle');

  return (
    <div className="sec">
      <div aria-busy={isLoading} className="ws-card">
        <div className="ws-head">
          <span className="ws-title">
            {t('title')}
          </span>
          <Link
            href="/log"
            className="ws-link"
          >
            {t('viewAll')}
          </Link>
        </div>

        <div className="ws-sub">
          {subtitle}
          {week && week.loggedDays > 0 && <> · {t('loggedDays', { n: week.loggedDays })}</>}
        </div>

        <div className="ws-tiles">
          {metrics.map((metric) => {
            const { icon, color } = TILE[metric.key];
            const trend = wellbeingTrend(metric.delta);

            return (
              <div
                key={metric.key}
                className="ws-tile"
              >
                <span className="ws-tile-ic" style={{ color }}>
                  <Icon name={icon} size={20} stroke="currentColor" />
                </span>
                <span className="ws-tile-l">{t(metric.key)}</span>
                <span className="ws-tile-v">
                  {metric.percent === null ? dash : t('percent', { n: metric.percent })}
                </span>
                {trend && metric.delta !== null && (
                  <span
                    // The arrow repeats the number, so the chip carries the full
                    // sentence for screen readers instead of a bare glyph.
                    aria-label={t(`trend.${trend}`, { n: Math.abs(metric.delta) })}
                    style={{
                      display: 'flex',
                      alignItems: 'center',
                      gap: 2,
                      fontSize: 11,
                      fontWeight: 800,
                      color: TREND_COLOR[trend],
                    }}
                  >
                    <span
                      aria-hidden="true"
                      className={clsx('ws-trend', trend === 'up' && 'is-up')}
                    >
                      <Icon name="chevronDown" size={12} stroke="currentColor" />
                    </span>
                    {t('percent', { n: Math.abs(metric.delta) })}
                  </span>
                )}
              </div>
            );
          })}
        </div>

        <div className="ws-note">
          {isError ? (
            t('error')
          ) : isLoading && !week ? (
            t('loading')
          ) : nothingLogged || nothingScored ? (
            <>
              ✨ {nothingScored ? t('tone.unscored') : t('tone.none')}{' '}
              <Link href="/log" className="ws-note-cta">
                {nothingScored ? t('unscoredCta') : t('emptyCta')}
              </Link>
            </>
          ) : (
            <>✨ {t(`tone.${tone}`)}</>
          )}
        </div>
      </div>
    </div>
  );
}
