'use client';

import { useId, useState } from 'react';
import { useLocale, useTranslations } from 'next-intl';

import { cn, formatDateTime, formatNumber } from '@/shared/lib';
import { ErrorState, PageHeader, Panel, Skeleton } from '@/shared/ui';

import { useKpi, type Kpi, type KpiCohort } from '../api/kpi';

const TABS = ['growth', 'retention', 'engine', 'ai'] as const;
type Tab = (typeof TABS)[number];
const DAYS = [30, 60, 90] as const;

/** «KPI و مانیتورها» (B-N9-08): activation funnel, weekly cohort retention, engine accuracy, AI indicators. */
export function KpiScreen() {
  const t = useTranslations('kpi');
  const locale = useLocale();
  const [tab, setTab] = useState<Tab>('growth');
  const [days, setDays] = useState<(typeof DAYS)[number]>(90);
  const [event, setEvent] = useState<'any' | 'log_saved'>('any');
  const q = useKpi(days, event);
  const k = q.data;

  return (
    <div className="flex flex-col gap-4">
      <PageHeader title={t('title')} meta={k ? t('updated', { at: formatDateTime(k.generated_at, locale) }) : undefined} />
      <div role="tablist" aria-label={t('title')} className="tabs">
        {TABS.map((id) => (
          <button
            key={id}
            role="tab"
            type="button"
            className="tab"
            aria-selected={tab === id}
            aria-current={tab === id ? 'page' : undefined}
            onClick={() => setTab(id)}
          >
            {t(`tab.${id}`)}
          </button>
        ))}
      </div>
      {q.error && !k ? (
        <Panel>
          <ErrorState error={q.error} onRetry={() => q.refetch()} />
        </Panel>
      ) : (
        <>
          {k && !k.events_since && (tab === 'growth' || tab === 'retention') ? (
            <Panel>
              <p className="m-0 text-ink-3">{t('noEvents')}</p>
            </Panel>
          ) : null}
          {tab === 'growth' ? <Funnel k={k} days={days} setDays={setDays} /> : null}
          {tab === 'retention' ? <Retention k={k} event={event} setEvent={setEvent} /> : null}
          {tab === 'engine' ? <Engine k={k} /> : null}
          {tab === 'ai' ? <Ai k={k} /> : null}
          {k ? <p className="m-0 text-xs text-muted">{k.notes.events} {k.notes.suppression}</p> : null}
        </>
      )}
    </div>
  );
}

function Funnel({ k, days, setDays }: { k: Kpi | undefined; days: number; setDays: (d: (typeof DAYS)[number]) => void }) {
  const t = useTranslations('kpi');
  const locale = useLocale();
  return (
    <Panel
      title={t('funnelTitle')}
      actions={
        <div role="group" aria-label={t('window')} className="flex gap-1">
          {DAYS.map((d) => (
            <button key={d} type="button" className={cn('btn btn-sm', d === days ? 'btn-primary' : 'btn-ghost')} aria-pressed={d === days} onClick={() => setDays(d)}>
              {t('days', { n: formatNumber(d, locale) })}
            </button>
          ))}
        </div>
      }
    >
      {!k ? (
        <Skeleton className="h-48 w-full" />
      ) : (
        <ol className="m-0 flex list-none flex-col gap-3 p-0">
          {k.funnel.steps.map((s) => (
            <li key={s.key} className="grid items-center gap-3 sm:grid-cols-[11rem_1fr_5rem]">
              <span className="text-[13px] font-semibold text-ink-3">{t(`step.${s.key}` as 'step.signup')}</span>
              <div className="h-3 overflow-hidden rounded bg-track" role="img" aria-label={`${t(`step.${s.key}` as 'step.signup')}: ${s.pct ?? '—'}`}>
                {s.pct != null ? <div className="h-full w-(--w) rounded bg-data-deep" style={{ '--w': `${Math.max(2, s.pct)}%` } as React.CSSProperties} /> : null}
              </div>
              <span className="text-end font-semibold text-ink" dir="ltr">
                {s.pct != null ? `${formatNumber(s.pct, locale)}٪` : '—'}
                {s.suppressed ? <span className="ms-1 text-xs font-normal text-muted">{t('suppressed')}</span> : null}
              </span>
            </li>
          ))}
        </ol>
      )}
    </Panel>
  );
}

function cellStyle(v: number | null): React.CSSProperties {
  // One hue, light → dark: the brand mixed into the surface, never above 45 % so ink text stays readable.
  return v == null ? {} : { background: `color-mix(in srgb, var(--brand) ${Math.round(Math.min(100, v) * 0.45)}%, var(--surface))` };
}

function Retention({ k, event, setEvent }: { k: Kpi | undefined; event: 'any' | 'log_saved'; setEvent: (e: 'any' | 'log_saved') => void }) {
  const t = useTranslations('kpi');
  const locale = useLocale();
  const cap = useId();
  return (
    <Panel
      title={t('retentionTitle')}
      actions={
        <div role="group" aria-label={t('activity')} className="flex gap-1">
          {(['any', 'log_saved'] as const).map((e) => (
            <button key={e} type="button" className={cn('btn btn-sm', e === event ? 'btn-primary' : 'btn-ghost')} aria-pressed={e === event} onClick={() => setEvent(e)}>
              {t(`activityOpt.${e}`)}
            </button>
          ))}
        </div>
      }
    >
      {!k ? (
        <Skeleton className="h-48 w-full" />
      ) : k.retention.cohorts.length === 0 ? (
        <p className="m-0 text-ink-3">{t('noCohorts')}</p>
      ) : (
        <div className="overflow-x-auto">
          <table className="w-full border-separate border-spacing-0.5 text-sm" aria-describedby={cap}>
            <caption id={cap} className="sr-only">{t('retentionTitle')}</caption>
            <thead>
              <tr className="text-xs text-muted">
                <th scope="col" className="text-start font-semibold">{t('cohort')}</th>
                <th scope="col" className="text-end font-semibold">{t('size')}</th>
                {k.retention.offsets.map((o) => (
                  <th key={o} scope="col" className="font-semibold" dir="ltr">D{o}</th>
                ))}
              </tr>
            </thead>
            <tbody>
              {k.retention.cohorts.map((c) => (
                <Row key={c.week_start} c={c} locale={locale} suppressedText={t('suppressed')} />
              ))}
            </tbody>
          </table>
        </div>
      )}
    </Panel>
  );
}

function Row({ c, locale, suppressedText }: { c: KpiCohort; locale: string; suppressedText: string }) {
  return (
    <tr>
      <th scope="row" className="whitespace-nowrap text-start font-medium text-ink-3" dir="ltr">{c.week_start}</th>
      <td className="text-end text-ink-3">{c.size != null ? formatNumber(c.size, locale) : suppressedText}</td>
      {c.cells.map((v, i) => (
        <td key={i} className="rounded text-center text-ink" style={cellStyle(v)} dir="ltr">
          {v != null ? `${formatNumber(v, locale)}٪` : '—'}
        </td>
      ))}
    </tr>
  );
}

function Stat({ label, value, note }: { label: string; value: string; note?: string }) {
  return (
    <div className="ledger-cell">
      <dt className="text-[13px] font-semibold text-ink-3">{label}</dt>
      <dd className="m-0 mt-1">
        <span className="ledger-value">{value}</span>
        {note ? <span className="mt-0.5 block text-xs text-muted">{note}</span> : null}
      </dd>
    </div>
  );
}

function Engine({ k }: { k: Kpi | undefined }) {
  const t = useTranslations('kpi');
  const locale = useLocale();
  if (!k) return <Skeleton className="h-32 w-full" />;
  const e = k.engine;
  const n = (v: number | null | undefined, suffix = '') => (v == null ? '—' : `${formatNumber(v, locale)}${suffix}`);
  return (
    <>
      <Panel title={t('engineTitle')} bodyClassName="">
        <dl className="ledger m-0">
          <Stat label={t('meanError')} value={n(e?.mean_error_days)} note={t('days', { n: '' }).trim()} />
          <Stat label={t('within2')} value={n(e?.within_2_days_pct, '٪')} note={e?.suppressed ? t('suppressed') : undefined} />
          <Stat label={t('engineUsers')} value={n(e?.users)} />
          <Stat label={t('engineStatus')} value={e?.status ? t(`status.${e.status}` as 'status.watch') : '—'} />
        </dl>
      </Panel>
      <Panel title={t('dataQualityTitle')}>
        <p className="m-0 text-ink-3">{k.notes.data_quality}</p>
      </Panel>
    </>
  );
}

function Ai({ k }: { k: Kpi | undefined }) {
  const t = useTranslations('kpi');
  const locale = useLocale();
  if (!k) return <Skeleton className="h-32 w-full" />;
  const a = k.ai;
  const n = (v: number | null, suffix = '') => (v == null ? '—' : `${formatNumber(v, locale)}${suffix}`);
  return (
    <Panel title={t('aiTitle', { n: formatNumber(a.days, locale) })} bodyClassName="">
      <dl className="ledger m-0">
        <Stat label={t('labsPerDay')} value={n(a.labs_per_day)} />
        <Stat label={t('ocrCorrection')} value={n(a.ocr_correction_pct, '٪')} />
        <Stat label={t('costPerAnalysis')} value={a.cost_per_analysis_usd == null ? '—' : `$${a.cost_per_analysis_usd.toFixed(4)}`} />
        <Stat label={t('assistantMsgs')} value={n(a.assistant_msgs_per_day)} />
        <Stat label={t('referral')} value={n(a.referral_pct, '٪')} />
      </dl>
    </Panel>
  );
}
