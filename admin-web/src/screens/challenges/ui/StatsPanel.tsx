'use client';

import { useLocale, useTranslations } from 'next-intl';
import type { CSSProperties } from 'react';

import { formatNumber } from '@/shared/lib';
import { ErrorState, Panel, Skeleton } from '@/shared/ui';

import { useChallengeStats } from '../api/challenges';

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
        </div>
      ) : null}
    </Panel>
  );
}
