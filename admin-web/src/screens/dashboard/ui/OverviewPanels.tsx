'use client';

import { useId, useState } from 'react';
import Link from 'next/link';
import { useLocale, useTranslations } from 'next-intl';

import { cn, formatDateTime, formatNumber } from '@/shared/lib';
import { ErrorState, Panel, Skeleton } from '@/shared/ui';

import { useOverview, type OverviewMetrics, type SeriesPoint } from '../api/overview';

const RANGES = [7, 30, 90] as const;

/** The overview dashboard (B-N9-03): KPI tiles, DAU chart, modes, engine health, alerts, OKR. Aggregates only. */
export function Overview() {
  const t = useTranslations('overview');
  const locale = useLocale();
  const q = useOverview();

  if (q.error && !q.data) {
    return (
      <Panel>
        <ErrorState error={q.error} onRetry={() => q.refetch()} />
      </Panel>
    );
  }
  const m = q.data?.metrics ?? null;
  return (
    <div className="flex flex-col gap-4">
      <p className="m-0 flex items-center gap-2 text-ink-3">
        <span className="data-tick" aria-hidden="true" />
        {q.data
          ? q.data.computed_at
            ? t('updated', { at: formatDateTime(q.data.computed_at, locale) })
            : t('noSnapshot')
          : <Skeleton className="w-56" />}
      </p>
      {q.data && !m ? (
        <Panel>
          <p className="m-0 text-ink-3">{t('noSnapshotBody')}</p>
        </Panel>
      ) : (
        <>
          <Tiles m={m} />
          <div className="grid gap-4 lg:grid-cols-[2fr_1fr]">
            <DauChart series={m?.series} />
            <Modes m={m} />
          </div>
          <div className="grid gap-4 lg:grid-cols-3">
            <Engine m={m} />
            <Alerts m={m} />
            <Panel title={t('okrTitle')}>
              {m?.okr ? (
                <p className="m-0 flex flex-col gap-1 text-ink-3">
                  <span className="text-ink">
                    {t('okrSummary', { n: formatNumber(m.okr.on_track + m.okr.done, locale), of: formatNumber(m.okr.key_results, locale) })}
                  </span>
                  <Link href="/okr" className="text-brand-deep">{t('okrOpen')}</Link>
                </p>
              ) : (
                <p className="m-0 text-ink-3">{t('okrNone')}</p>
              )}
            </Panel>
          </div>
        </>
      )}
    </div>
  );
}

function delta(v: { value: number | null; prev: number | null }): number | null {
  if (v.value == null || v.prev == null || v.prev === 0) return null;
  return Math.round(((v.value - v.prev) / v.prev) * 100);
}

function Tiles({ m }: { m: OverviewMetrics | null }) {
  const t = useTranslations('overview');
  const locale = useLocale();
  const n = (v: number | null | undefined) => (v == null ? '—' : formatNumber(v, locale));
  const tiles: { key: string; label: string; value: string; d: number | null; note?: string }[] = m
    ? [
        { key: 'dau', label: t('dau'), value: n(m.dau.value), d: delta(m.dau) },
        { key: 'mau', label: t('mau'), value: n(m.mau.value), d: delta(m.mau) },
        {
          key: 'stick',
          label: t('stickiness'),
          value: m.stickiness_pct.value == null ? '—' : `${formatNumber(m.stickiness_pct.value, locale)}٪`,
          d: null,
        },
        { key: 'signups', label: t('signupsToday'), value: n(m.signups_today.value), d: delta(m.signups_today) },
        { key: 'plus', label: t('plusActive'), value: n(m.plus_active.value), d: null },
        {
          key: 'trial',
          label: t('trialToPaid'),
          value: m.trial_to_paid.pct == null ? '—' : `${formatNumber(m.trial_to_paid.pct, locale)}٪`,
          d: null,
          note: m.trial_to_paid.suppressed ? t('suppressed') : undefined,
        },
        { key: 'mrr', label: t('mrr'), value: n(m.mrr_toman), d: null, note: t('toman') },
      ]
    : [];
  return (
    <Panel bodyClassName="">
      <dl className="ledger m-0">
        {(m ? tiles : Array.from({ length: 7 }, (_, i) => ({ key: String(i) }) as (typeof tiles)[number])).map((c) => (
          <div key={c.key} className="ledger-cell">
            <dt className="text-[13px] font-semibold text-ink-3">{c.label ?? <Skeleton className="w-24" />}</dt>
            <dd className="m-0 mt-1">
              {m ? <span className="ledger-value">{c.value}</span> : <Skeleton className="h-8 w-20" />}
              <span
                className={cn(
                  'mt-0.5 block text-xs',
                  c.d == null ? 'text-muted' : c.d >= 0 ? 'font-semibold text-green-deep' : 'font-semibold text-danger-deep',
                )}
              >
                {c.d != null ? (
                  <span dir="ltr">{`${c.d >= 0 ? '+' : '−'}${formatNumber(Math.abs(c.d), locale)}٪`}</span>
                ) : (
                  (c.note ?? ' ')
                )}
              </span>
            </dd>
          </div>
        ))}
      </dl>
    </Panel>
  );
}

const W = 640;
const H = 220;
const PAD = { l: 8, r: 8, t: 12, b: 22 };

function DauChart({ series }: { series: SeriesPoint[] | undefined }) {
  const t = useTranslations('overview');
  const locale = useLocale();
  const [range, setRange] = useState<(typeof RANGES)[number]>(30);
  const [hover, setHover] = useState<number | null>(null);
  const gid = useId();

  const pts = (series ?? []).slice(-range);
  const max = Math.max(1, ...pts.map((p) => Math.max(p.dau, p.loggers)));
  const x = (i: number) => PAD.l + (pts.length <= 1 ? 0 : (i / (pts.length - 1)) * (W - PAD.l - PAD.r));
  const y = (v: number) => PAD.t + (1 - v / max) * (H - PAD.t - PAD.b);
  const line = (k: 'dau' | 'loggers') => pts.map((p, i) => `${i ? 'L' : 'M'}${x(i).toFixed(1)} ${y(p[k]).toFixed(1)}`).join(' ');
  const hp = hover == null ? null : pts[hover];

  return (
    <Panel
      title={t('chartTitle')}
      actions={
        <div role="group" aria-label={t('range')} className="flex gap-1">
          {RANGES.map((r) => (
            <button
              key={r}
              type="button"
              className={cn('btn btn-sm', r === range ? 'btn-primary' : 'btn-ghost')}
              aria-pressed={r === range}
              onClick={() => setRange(r)}
            >
              {t('days', { n: formatNumber(r, locale) })}
            </button>
          ))}
        </div>
      }
    >
      {!series ? (
        <Skeleton className="h-56 w-full" />
      ) : (
        <div className="relative">
          <svg
            viewBox={`0 0 ${W} ${H}`}
            className="h-auto w-full"
            role="img"
            aria-label={t('chartTitle')}
            onPointerLeave={() => setHover(null)}
            onPointerMove={(e) => {
              const r = e.currentTarget.getBoundingClientRect();
              const px = ((e.clientX - r.left) / r.width) * W;
              setHover(Math.max(0, Math.min(pts.length - 1, Math.round(((px - PAD.l) / (W - PAD.l - PAD.r)) * (pts.length - 1)))));
            }}
          >
            <defs>
              <linearGradient id={gid} x1="0" y1="0" x2="0" y2="1">
                <stop offset="0" stopColor="var(--data)" stopOpacity="0.28" />
                <stop offset="1" stopColor="var(--data)" stopOpacity="0" />
              </linearGradient>
            </defs>
            {[0, 0.5, 1].map((f) => (
              <line key={f} x1={PAD.l} x2={W - PAD.r} y1={y(max * f)} y2={y(max * f)} stroke="var(--line)" strokeWidth="1" />
            ))}
            {pts.length > 1 ? (
              <>
                <path d={`${line('dau')} L${x(pts.length - 1)} ${y(0)} L${x(0)} ${y(0)} Z`} fill={`url(#${gid})`} />
                <path d={line('dau')} fill="none" stroke="var(--data-deep)" strokeWidth="2" strokeLinejoin="round" strokeLinecap="round" />
                <path d={line('loggers')} fill="none" stroke="var(--brand)" strokeWidth="2" strokeDasharray="5 4" strokeLinejoin="round" strokeLinecap="round" />
              </>
            ) : null}
            {hp ? (
              <g>
                <line x1={x(hover!)} x2={x(hover!)} y1={PAD.t} y2={H - PAD.b} stroke="var(--muted-3)" strokeWidth="1" />
                <circle cx={x(hover!)} cy={y(hp.dau)} r="4" fill="var(--data-deep)" stroke="var(--surface)" strokeWidth="2" />
                <circle cx={x(hover!)} cy={y(hp.loggers)} r="4" fill="var(--brand)" stroke="var(--surface)" strokeWidth="2" />
              </g>
            ) : null}
          </svg>
          {hp ? (
            <div
              className="pointer-events-none absolute start-(--at) top-1 rounded-md border border-line bg-surface px-3 py-2 text-xs shadow-sm"
              style={{ '--at': `${Math.min(70, Math.max(2, (x(hover!) / W) * 100 - 12))}%` } as React.CSSProperties}
              role="status"
            >
              <div className="font-semibold text-ink" dir="ltr">{hp.date}</div>
              <div className="text-ink-3">{t('dau')}: {formatNumber(hp.dau, locale)}</div>
              <div className="text-ink-3">{t('loggers')}: {formatNumber(hp.loggers, locale)}</div>
            </div>
          ) : null}
          <ul className="m-0 mt-2 flex list-none gap-4 p-0 text-xs text-ink-3">
            <li className="flex items-center gap-1.5">
              <span className="inline-block h-0.5 w-4 rounded bg-data-deep" />
              {t('legendAll')}
            </li>
            <li className="flex items-center gap-1.5">
              <span className="inline-block h-0.5 w-4 rounded border-t-2 border-dashed border-brand" />
              {t('legendLoggers')}
            </li>
          </ul>
          <table className="sr-only">
            <caption>{t('chartTitle')}</caption>
            <thead>
              <tr><th>{t('date')}</th><th>{t('dau')}</th><th>{t('loggers')}</th></tr>
            </thead>
            <tbody>
              {pts.map((p) => (
                <tr key={p.date}><td>{p.date}</td><td>{p.dau}</td><td>{p.loggers}</td></tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </Panel>
  );
}

const MODE_BAR: Record<string, string> = {
  cycle: 'bg-danger',
  ttc: 'bg-amber',
  pregnancy: 'bg-brand',
  postpartum: 'bg-data-deep',
  companion: 'bg-brand-deep',
  menopause: 'bg-green',
  teen: 'bg-amber',
};

function Modes({ m }: { m: OverviewMetrics | null }) {
  const t = useTranslations('overview');
  const locale = useLocale();
  const max = Math.max(1, ...(m?.modes.map((x) => x.count ?? 0) ?? [1]));
  return (
    <Panel title={t('modesTitle')}>
      {!m ? (
        <Skeleton className="h-40 w-full" />
      ) : (
        <ul className="m-0 flex list-none flex-col gap-3 p-0">
          {m.modes.map((x) => (
            <li key={x.mode} className="flex flex-col gap-1">
              <div className="flex justify-between text-sm">
                <span className="text-ink">{t(`mode.${x.mode}` as never, { default: x.mode } as never)}</span>
                <span className="text-ink-3">{x.count == null ? t('suppressed') : formatNumber(x.count, locale)}</span>
              </div>
              <div className="h-2 rounded-full bg-track" aria-hidden="true">
                <div className={`h-2 w-(--w) rounded-full ${MODE_BAR[x.mode] ?? 'bg-data-deep'}`} style={{ '--w': `${((x.count ?? 0) / max) * 100}%` } as React.CSSProperties} />
              </div>
            </li>
          ))}
        </ul>
      )}
    </Panel>
  );
}

function Engine({ m }: { m: OverviewMetrics | null }) {
  const t = useTranslations('overview');
  const locale = useLocale();
  const e = m?.engine;
  return (
    <Panel title={t('engineTitle')}>
      {!e ? (
        <Skeleton className="h-32 w-full" />
      ) : e.suppressed || e.mean_error_days == null ? (
        <p className="m-0 text-ink-3">{t('engineSuppressed')}</p>
      ) : (
        <dl className="m-0 flex flex-col gap-2 text-sm">
          <div className="flex justify-between">
            <dt className="text-ink-3">{t('engineStatus')}</dt>
            <dd className={cn('m-0 font-semibold', e.status === 'healthy' ? 'text-green-deep' : 'text-amber-deep')}>
              {e.status === 'healthy' ? t('engineHealthy') : t('engineWatch')}
            </dd>
          </div>
          <div className="flex justify-between">
            <dt className="text-ink-3">{t('engineMeanError')}</dt>
            <dd className="m-0">{t('daysValue', { n: formatNumber(e.mean_error_days, locale) })}</dd>
          </div>
          <div className="flex justify-between">
            <dt className="text-ink-3">{t('engineWithin2')}</dt>
            <dd className="m-0">{`${formatNumber(e.within_2_days_pct ?? 0, locale)}٪`}</dd>
          </div>
          <div className="flex justify-between">
            <dt className="text-ink-3">{t('engineConfidence')}</dt>
            <dd className="m-0 text-muted">{t('notAvailable')}</dd>
          </div>
        </dl>
      )}
    </Panel>
  );
}

function Alerts({ m }: { m: OverviewMetrics | null }) {
  const t = useTranslations('overview');
  return (
    <Panel title={t('alertsTitle')}>
      {!m ? (
        <Skeleton className="h-24 w-full" />
      ) : m.alerts.length === 0 ? (
        <p className="m-0 text-ink-3">{t('alertsNone')}</p>
      ) : (
        <ul className="m-0 flex list-none flex-col gap-2 p-0 text-sm">
          {m.alerts.map((a) => (
            <li key={a.key} className="text-ink">{a.text}</li>
          ))}
        </ul>
      )}
    </Panel>
  );
}
