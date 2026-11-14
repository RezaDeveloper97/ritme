'use client';

import { useLocale, useTranslations } from 'next-intl';
import { useId, useMemo, useState } from 'react';

import { useLocalized } from '@/shared/i18n';
import { cn, formatDate, formatNumber } from '@/shared/lib';
import { Badge, DataTable, ErrorState, PageHeader, Panel, Skeleton, TextInput, type Column } from '@/shared/ui';

import { useFeatures, useUsage, type UsageReport } from '../api/ai-credits';
import { lastDays, mergeDays, usdToToman, type DayPoint } from '../lib/report';

const PRESETS = [7, 30, 90] as const;

/**
 * /ai/usage — what AI cost us (provider calls, tokens, USD) next to what users spent in «اعتبار» over the same days
 * (GET /ai/usage, admin-api.md §17). The toman figures use the admin's ai_cost_usd_irr_rate setting when it is set.
 */
export function UsageScreen() {
  const t = useTranslations('aiCredits.usage');
  const locale = useLocale();
  const [range, setRange] = useState<{ from: string; to: string } | null>(null);
  const [preset, setPreset] = useState<number | null>(30);
  const [today, setToday] = useState<string | null>(null);
  const query = useUsage(range);
  const data = query.data;
  if (data && today === null && range === null) setToday(data.to);

  const pick = (days: number) => {
    if (!today) return;
    setPreset(days);
    setRange(lastDays(today, days));
  };
  const setBound = (key: 'from' | 'to', value: string) => {
    if (!value || !data) return;
    setPreset(null);
    setRange({ from: data.from, to: data.to, [key]: value });
  };

  return (
    <div className="flex flex-col gap-5">
      <PageHeader
        title={t('title')}
        meta={data ? <span>{t('window', { from: formatDate(data.from, locale), to: formatDate(data.to, locale) })}</span> : null}
        actions={
          <div role="group" aria-label={t('range')} className="flex flex-wrap items-end gap-2">
            {PRESETS.map((d) => (
              <button
                key={d}
                type="button"
                className={cn('btn btn-sm', preset === d ? 'btn-primary' : 'btn-ghost')}
                aria-pressed={preset === d}
                disabled={!today}
                onClick={() => pick(d)}
              >
                {t('days', { n: formatNumber(d, locale) })}
              </button>
            ))}
          </div>
        }
      />
      <Panel bodyClassName="">
        <div className="form-grid p-4 sm:p-5">
          <TextInput label={t('from')} type="date" dir="ltr" value={data?.from ?? ''} max={data?.to} onChange={(e) => setBound('from', e.target.value)} />
          <TextInput label={t('to')} type="date" dir="ltr" value={data?.to ?? ''} min={data?.from} onChange={(e) => setBound('to', e.target.value)} />
        </div>
        <p className="panel-note">{t('rangeHint')}</p>
      </Panel>
      {query.error && !data ? (
        <Panel>
          <ErrorState error={query.error} onRetry={() => query.refetch()} />
        </Panel>
      ) : !data ? (
        <Skeleton className="h-72 w-full" />
      ) : (
        <Report data={data} />
      )}
    </div>
  );
}

function Report({ data }: { data: UsageReport }) {
  const t = useTranslations('aiCredits.usage');
  const locale = useLocale();
  const n = (v: number) => formatNumber(v, locale);
  const usd = (v: number) => `$${formatNumber(Math.round(v * 100) / 100, locale)}`;
  const toman = (v: number) => {
    const tm = usdToToman(v, data.usd_irr_rate);
    return tm === null ? null : t('toman', { amount: n(tm) });
  };
  const days = useMemo(
    () => mergeDays(data.from, data.to, data.by_day, data.credits.by_day),
    [data.from, data.to, data.by_day, data.credits.by_day],
  );
  const costToman = toman(data.totals.cost_usd);

  const tiles: { key: string; value: string; sub?: string | null }[] = [
    { key: 'calls', value: n(data.totals.calls), sub: t('okCalls', { n: n(data.totals.ok_calls) }) },
    {
      key: 'credits',
      value: n(data.credits.total),
      sub: t('creditsSplit', { uses: n(data.credits.uses), plan: n(data.credits.plan_credits), extra: n(data.credits.extra_credits) }),
    },
    { key: 'costUsd', value: usd(data.totals.cost_usd), sub: t('tokens', { input: n(data.totals.input_tokens), output: n(data.totals.output_tokens) }) },
    { key: 'costToman', value: costToman ?? '—', sub: costToman ? t('rate', { rate: n(data.usd_irr_rate ?? 0) }) : t('noRate') },
  ];

  return (
    <>
      <Panel bodyClassName="">
        <dl className="ledger m-0">
          {tiles.map((tile) => (
            <div key={tile.key} className="ledger-cell">
              <dt className="text-[13px] font-semibold text-ink-3">{t(`tiles.${tile.key}` as 'tiles.calls')}</dt>
              <dd className="m-0 mt-1 flex flex-col">
                <span className="ledger-value" dir="auto">
                  {tile.value}
                </span>
                {tile.sub ? <span className="text-xs text-ink-3">{tile.sub}</span> : null}
              </dd>
            </div>
          ))}
        </dl>
        {data.today.cap_usd !== null ? (
          <p className="panel-note flex flex-wrap items-center gap-2">
            {t('today', { spent: usd(data.today.spent_usd ?? 0), cap: usd(data.today.cap_usd) })}
            {data.today.exhausted ? <Badge tone="red">{t('capExhausted')}</Badge> : null}
          </p>
        ) : null}
      </Panel>
      <DayChart days={days} />
      <FeatureTable data={data} />
      <ModelTable data={data} />
    </>
  );
}

const W = 640;
const H = 220;
const PAD = { l: 8, r: 8, t: 12, b: 22 };

/** Credits spent per day (bars, data colour) and the provider cost (line, brand), each on its own scale. */
function DayChart({ days }: { days: DayPoint[] }) {
  const t = useTranslations('aiCredits.usage');
  const locale = useLocale();
  const [hover, setHover] = useState<number | null>(null);
  const gid = useId();
  const maxCredits = Math.max(1, ...days.map((d) => d.credits));
  const maxCost = Math.max(0.000001, ...days.map((d) => d.costUsd));
  const step = days.length > 0 ? (W - PAD.l - PAD.r) / days.length : 0;
  const cx = (i: number) => PAD.l + step * (i + 0.5);
  const yC = (v: number) => PAD.t + (1 - v / maxCredits) * (H - PAD.t - PAD.b);
  const yU = (v: number) => PAD.t + (1 - v / maxCost) * (H - PAD.t - PAD.b);
  const line = days.map((d, i) => `${i ? 'L' : 'M'}${cx(i).toFixed(1)} ${yU(d.costUsd).toFixed(1)}`).join(' ');
  const hp = hover === null ? null : days[hover];
  const usd = (v: number) => `$${formatNumber(Math.round(v * 100) / 100, locale)}`;

  return (
    <Panel title={t('chartTitle')}>
      <div className="relative">
        <svg
          viewBox={`0 0 ${W} ${H}`}
          className="h-auto w-full"
          role="img"
          aria-label={t('chartTitle')}
          onPointerLeave={() => setHover(null)}
          onPointerMove={(e) => {
            if (days.length === 0) return;
            const r = e.currentTarget.getBoundingClientRect();
            const px = ((e.clientX - r.left) / r.width) * W;
            setHover(Math.max(0, Math.min(days.length - 1, Math.floor((px - PAD.l) / step))));
          }}
        >
          <defs>
            <linearGradient id={gid} x1="0" y1="0" x2="0" y2="1">
              <stop offset="0" stopColor="var(--data)" stopOpacity="0.7" />
              <stop offset="1" stopColor="var(--data)" stopOpacity="0.25" />
            </linearGradient>
          </defs>
          {[0, 0.5, 1].map((f) => (
            <line key={f} x1={PAD.l} x2={W - PAD.r} y1={yC(maxCredits * f)} y2={yC(maxCredits * f)} stroke="var(--line)" strokeWidth="1" />
          ))}
          {days.map((d, i) => (
            <rect
              key={d.day}
              x={cx(i) - Math.max(1, step * 0.32)}
              width={Math.max(2, step * 0.64)}
              y={yC(d.credits)}
              height={Math.max(0, yC(0) - yC(d.credits))}
              rx="2"
              fill={`url(#${gid})`}
            />
          ))}
          {days.length > 1 ? (
            <path d={line} fill="none" stroke="var(--brand)" strokeWidth="2" strokeLinejoin="round" strokeLinecap="round" />
          ) : null}
          {hp ? (
            <g>
              <line x1={cx(hover!)} x2={cx(hover!)} y1={PAD.t} y2={H - PAD.b} stroke="var(--muted-3)" strokeWidth="1" />
              <circle cx={cx(hover!)} cy={yU(hp.costUsd)} r="4" fill="var(--brand)" stroke="var(--surface)" strokeWidth="2" />
            </g>
          ) : null}
        </svg>
        {hp ? (
          <div
            className="pointer-events-none absolute start-(--at) top-1 rounded-md border border-line bg-surface px-3 py-2 text-xs shadow-sm"
            style={{ '--at': `${Math.min(70, Math.max(2, (cx(hover!) / W) * 100 - 12))}%` } as React.CSSProperties}
            role="status"
          >
            <div className="font-semibold text-ink">{formatDate(hp.day, locale)}</div>
            <div className="text-ink-3">
              {t('creditsSpent')}: {formatNumber(hp.credits, locale)}
            </div>
            <div className="text-ink-3">
              {t('cost')}: <span dir="ltr">{usd(hp.costUsd)}</span>
            </div>
            <div className="text-ink-3">
              {t('calls')}: {formatNumber(hp.calls, locale)}
            </div>
          </div>
        ) : null}
        <ul className="m-0 mt-2 flex list-none gap-4 p-0 text-xs text-ink-3">
          <li className="flex items-center gap-1.5">
            <span className="inline-block h-2.5 w-2.5 rounded-sm bg-data" />
            {t('creditsSpent')}
          </li>
          <li className="flex items-center gap-1.5">
            <span className="inline-block h-0.5 w-4 rounded bg-brand" />
            {t('costLegend')}
          </li>
        </ul>
        <table className="sr-only">
          <caption>{t('chartTitle')}</caption>
          <thead>
            <tr>
              <th>{t('date')}</th>
              <th>{t('creditsSpent')}</th>
              <th>{t('cost')}</th>
            </tr>
          </thead>
          <tbody>
            {days.map((d) => (
              <tr key={d.day}>
                <td>{d.day}</td>
                <td>{d.credits}</td>
                <td>{d.costUsd}</td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </Panel>
  );
}

interface FeatureRow {
  feature: string;
  uses: number;
  credits: number;
  calls: number;
  costUsd: number;
}

/** Per feature: credits users spent vs what its provider calls cost. */
function FeatureTable({ data }: { data: UsageReport }) {
  const t = useTranslations('aiCredits.usage');
  const locale = useLocale();
  const localize = useLocalized();
  const features = useFeatures();
  const titles = new Map((features.data?.items ?? []).map((f) => [f.key, localize(f.title)]));
  const n = (v: number) => formatNumber(v, locale);
  const rows = useMemo(() => {
    const by = new Map<string, FeatureRow>();
    const row = (feature: string) => {
      let r = by.get(feature);
      if (!r) {
        r = { feature, uses: 0, credits: 0, calls: 0, costUsd: 0 };
        by.set(feature, r);
      }
      return r;
    };
    for (const c of data.credits.by_feature) Object.assign(row(c.feature), { uses: c.uses, credits: c.credits });
    for (const f of data.by_feature) {
      const r = row(f.feature);
      r.calls += f.calls;
      r.costUsd += f.cost_usd;
    }
    return [...by.values()].sort((a, b) => b.credits - a.credits || b.costUsd - a.costUsd);
  }, [data]);

  const columns: Column<FeatureRow>[] = [
    {
      key: 'feature',
      header: t('feature'),
      cell: (r) => (
        <span className="flex flex-col items-start">
          <span className="font-semibold">{titles.get(r.feature) || r.feature}</span>
          <code dir="ltr" className="text-xs text-muted">
            {r.feature}
          </code>
        </span>
      ),
    },
    { key: 'uses', header: t('uses'), cell: (r) => n(r.uses), className: 'cell-num' },
    { key: 'credits', header: t('creditsSpent'), cell: (r) => n(r.credits), className: 'cell-num' },
    { key: 'calls', header: t('calls'), cell: (r) => n(r.calls), className: 'cell-num' },
    {
      key: 'cost',
      header: t('cost'),
      cell: (r) => (
        <span className="flex flex-col items-end">
          <span dir="ltr">${formatNumber(Math.round(r.costUsd * 100) / 100, locale)}</span>
          {usdToToman(r.costUsd, data.usd_irr_rate) !== null ? (
            <span className="text-xs text-ink-3">{t('toman', { amount: n(usdToToman(r.costUsd, data.usd_irr_rate)!) })}</span>
          ) : null}
        </span>
      ),
      className: 'cell-num',
    },
    {
      key: 'perCredit',
      header: t('costPerCredit'),
      cell: (r) =>
        r.credits > 0 ? <span dir="ltr">${formatNumber(Math.round((r.costUsd / r.credits) * 10000) / 10000, locale)}</span> : '—',
      className: 'cell-num',
    },
  ];

  return (
    <Panel title={t('byFeature')} bodyClassName="">
      <DataTable columns={columns} items={rows} rowKey={(r) => r.feature} emptyText={t('empty')} caption={t('byFeature')} />
    </Panel>
  );
}

type ModelRow = UsageReport['by_feature'][number];

/** The provider side: per feature × provider × model, highest cost first (as the API orders it). */
function ModelTable({ data }: { data: UsageReport }) {
  const t = useTranslations('aiCredits.usage');
  const locale = useLocale();
  const n = (v: number) => formatNumber(v, locale);
  const columns: Column<ModelRow>[] = [
    { key: 'feature', header: t('feature'), cell: (r) => <code dir="ltr">{r.feature}</code> },
    {
      key: 'model',
      header: t('model'),
      cell: (r) => (
        <span className="flex flex-col items-start">
          <code dir="ltr">{r.model}</code>
          <span className="text-xs text-muted">{r.provider}</span>
        </span>
      ),
    },
    { key: 'calls', header: t('calls'), cell: (r) => `${n(r.ok_calls)} / ${n(r.calls)}`, className: 'cell-num' },
    { key: 'users', header: t('users'), cell: (r) => n(r.users), className: 'cell-num' },
    { key: 'in', header: t('inputTokens'), cell: (r) => n(r.input_tokens), className: 'cell-num' },
    { key: 'out', header: t('outputTokens'), cell: (r) => n(r.output_tokens), className: 'cell-num' },
    { key: 'latency', header: t('latency'), cell: (r) => t('ms', { n: n(r.avg_latency_ms) }), className: 'cell-num' },
    {
      key: 'cost',
      header: t('cost'),
      cell: (r) => <span dir="ltr">${formatNumber(Math.round(r.cost_usd * 100) / 100, locale)}</span>,
      className: 'cell-num',
    },
  ];
  return (
    <Panel title={t('byModel')} bodyClassName="">
      <DataTable
        columns={columns}
        items={data.by_feature}
        rowKey={(r) => `${r.feature}|${r.provider}|${r.model}`}
        emptyText={t('empty')}
        caption={t('byModel')}
      />
    </Panel>
  );
}
