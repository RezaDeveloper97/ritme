'use client';

import { useLocale, useTranslations } from 'next-intl';

import { cn, formatNumber } from '@/shared/lib';
import { Panel, Skeleton } from '@/shared/ui';

import type { Distribution, OverviewMetrics } from '../api/overview';
import { bucketRows, engineReadable } from '../lib/engine-health';

const CONFIDENCE = ['high', 'medium', 'low', 'unknown'] as const;
const QUALITY = ['good', 'partial', 'poor', 'insufficient'] as const;
const BAR: Record<string, string> = { high: 'bg-green', good: 'bg-green', medium: 'bg-data-deep', partial: 'bg-data-deep', low: 'bg-amber', poor: 'bg-amber', unknown: 'bg-track', insufficient: 'bg-track' };

/** Engine health: MAE and ±2-day accuracy from real predictions, plus the confidence and data-quality distributions. */
export function EngineHealthCard({ m }: { m: OverviewMetrics | null }) {
  const t = useTranslations('overview');
  const locale = useLocale();
  const e = m?.engine;
  return (
    <Panel title={t('engineTitle')}>
      {!e ? (
        <Skeleton className="h-32 w-full" />
      ) : (
        <div className="flex flex-col gap-4">
          {!engineReadable(e) ? (
            <p className="m-0 text-ink-3">{t('engineSuppressed')}</p>
          ) : (
            <>
              <p className="m-0 text-xs text-muted">{t('engineReal')}</p>
              <dl className="m-0 flex flex-col gap-2 text-sm">
                <div className="flex justify-between">
                  <dt className="text-ink-3">{t('engineStatus')}</dt>
                  <dd className={cn('m-0 font-semibold', e.status === 'healthy' ? 'text-green-deep' : 'text-amber-deep')}>
                    {e.status === 'healthy' ? t('engineHealthy') : t('engineWatch')}
                  </dd>
                </div>
                <div className="flex justify-between">
                  <dt className="text-ink-3">{t('engineMeanError')}</dt>
                  <dd className="m-0">{t('daysValue', { n: formatNumber(e.mean_error_days ?? 0, locale) })}</dd>
                </div>
                <div className="flex justify-between">
                  <dt className="text-ink-3">{t('engineWithin2')}</dt>
                  <dd className="m-0">{`${formatNumber(e.within_2_days_pct ?? 0, locale)}٪`}</dd>
                </div>
                <div className="flex justify-between">
                  <dt className="text-ink-3">{t('engineSamples')}</dt>
                  <dd className="m-0">{formatNumber(e.samples, locale)}</dd>
                </div>
              </dl>
            </>
          )}
          <Dist title={t('engineConfidence')} d={e.confidence} order={CONFIDENCE} kind="confidence" />
          <Dist title={t('engineDataQuality')} d={m?.kpi?.distributions?.data_quality} order={QUALITY} kind="quality" />
        </div>
      )}
    </Panel>
  );
}

function Dist({ title, d, order, kind }: { title: string; d: Distribution | null | undefined; order: readonly string[]; kind: 'confidence' | 'quality' }) {
  const t = useTranslations('overview');
  const locale = useLocale();
  const rows = bucketRows(d, order);
  return (
    <div className="flex flex-col gap-1.5">
      <p className="m-0 text-[13px] font-semibold text-ink-3">{title}</p>
      {rows.length === 0 ? (
        <p className="m-0 text-xs text-muted">{t('notAvailable')}</p>
      ) : (
        <ul className="m-0 flex list-none flex-col gap-1.5 p-0 text-xs">
          {rows.map((r) => (
            <li key={r.key} className="grid grid-cols-[5.5rem_1fr_4rem] items-center gap-2">
              <span className="text-ink">{t(`${kind}.${r.key}` as 'confidence.high')}</span>
              <div className="h-2 overflow-hidden rounded-full bg-track" aria-hidden="true">
                {r.pct != null ? <div className={cn('h-2 w-(--w) rounded-full', BAR[r.key] ?? 'bg-data-deep')} style={{ '--w': `${Math.max(2, r.pct)}%` } as React.CSSProperties} /> : null}
              </div>
              <span className="text-end text-ink-3" dir="ltr">{r.hidden ? t('suppressedShort') : `${formatNumber(r.pct ?? 0, locale)}٪`}</span>
            </li>
          ))}
        </ul>
      )}
    </div>
  );
}
