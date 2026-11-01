'use client';

import { useLocale, useTranslations } from 'next-intl';
import type { CSSProperties } from 'react';

import { formatNumber } from '@/shared/lib';
import { ErrorState, Panel, Skeleton } from '@/shared/ui';

import { useChallengeStats, type ChallengeStats } from '../api/challenges';

/** Participants, completion and rewards of one v2 challenge, plus the users active on each day of the run. */
export function StatsPanel({ id }: { id: number }) {
  const t = useTranslations('challenges.program');
  const locale = useLocale();
  const query = useChallengeStats(id);
  const n = (v: number) => formatNumber(v, locale);
  const s = query.data?.stats;
  const max = Math.max(1, ...(s?.per_day.map((d) => d.users) ?? [1]));
  return (
    <Panel title={t('stats')}>
      {query.isPending ? <Skeleton className="h-16 w-full" /> : null}
      {query.error ? <ErrorState error={query.error} onRetry={() => query.refetch()} /> : null}
      {s ? (
        <div className="flex flex-col gap-4">
          <dl className="m-0 grid grid-cols-2 gap-3 sm:grid-cols-5">
            {[
              [t('participants'), n(s.participants)],
              [t('activeRuns'), n(s.active)],
              [t('completedRuns'), n(s.completed)],
              [t('completionRate'), `${n(s.completion_rate)}٪`],
              [t('rewardsGranted'), `${n(s.rewards_granted)} · ${t('plusDaysShort', { count: n(s.plus_days_granted) })}`],
            ].map(([label, value]) => (
              <div key={label}>
                <dt className="field-label m-0">{label}</dt>
                <dd className="m-0 text-lg font-semibold">{value}</dd>
              </div>
            ))}
          </dl>
          {s.per_day.length > 0 ? (
            <div>
              <span className="field-label">{t('usersPerDay')}</span>
              <ol className="m-0 flex h-[72px] list-none items-end gap-1 p-0">
                {s.per_day.map((d) => (
                  <li key={d.day} className="flex flex-1 flex-col items-center justify-end gap-1" title={`${t('day', { day: d.day })}: ${n(d.users)}`}>
                    <span
                      className="h-(--h) w-full rounded-sm bg-brand-fill"
                      style={{ '--h': `${Math.max(6, (d.users / max) * 52)}px` } as CSSProperties}
                    />
                    <span className="text-xs text-muted">{n(d.day)}</span>
                  </li>
                ))}
              </ol>
            </div>
          ) : null}
          {s.drop_off.length > 0 ? <DropOffChart rows={s.drop_off} /> : null}
        </div>
      ) : null}
    </Panel>
  );
}

/**
 * Day-by-day drop-off (T-M10-10): for every day of the run the share of the runs that reached it and then stopped.
 * Bars use the data accent (not the brand rose); the numbers are also in the table-free title / label of each bar.
 */
function DropOffChart({ rows }: { rows: ChallengeStats['drop_off'] }) {
  const t = useTranslations('challenges.program');
  const locale = useLocale();
  const n = (v: number) => formatNumber(v, locale);
  const pct = (v: number | null) => (v === null ? '—' : `${n(v)}٪`);
  return (
    <div data-testid="drop-off">
      <span className="field-label">{t('dropOff')}</span>
      <p className="field-hint m-0 mb-2">{t('dropOffHint')}</p>
      <ol className="m-0 flex h-[96px] list-none items-end gap-1 p-0">
        {rows.map((d) => (
          <li
            key={d.day}
            className="flex flex-1 flex-col items-center justify-end gap-1"
            title={t('dropOffTitle', { day: d.day, reached: n(d.reached), dropped: n(d.dropped), rate: pct(d.drop_off_rate) })}
          >
            <span className="text-[11px] text-muted">{pct(d.drop_off_rate)}</span>
            <span
              className="h-(--h) w-full rounded-sm bg-data"
              style={{ '--h': `${d.drop_off_rate === null ? 2 : Math.max(2, (d.drop_off_rate / 100) * 52)}px` } as CSSProperties}
            />
            <span className="text-xs text-muted">{n(d.day)}</span>
          </li>
        ))}
      </ol>
    </div>
  );
}
