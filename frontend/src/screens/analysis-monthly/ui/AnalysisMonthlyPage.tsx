'use client';

import { clsx } from 'clsx';
import { useLocale, useTranslations } from 'next-intl';
import { useEffect, useMemo, type ReactNode } from 'react';

import {
  isMonthlyCalendar,
  useMonthlyReport,
  type BloodPressureValue,
  type MonthlyCalendar,
  type MonthlyMetric,
  type MonthlyReport,
} from '@/entities/analysis';
import { useDirection, useRouter, type Locale } from '@/shared/i18n';
import {
  calendarSystem,
  formatDayMonth,
  formatDecimal,
  formatMonthLabel,
  formatNumber,
  fromApiDate,
} from '@/shared/lib/date';
import { useMounted } from '@/shared/lib/use-mounted';
import { openSheet } from '@/shared/sheet';
import {
  EmptyState,
  Icon,
  PrimaryButton,
  ScreenHeader,
  SecondaryButton,
  Skeleton,
  SkeletonGroup,
  SkyLayer,
  StatusPill,
} from '@/shared/ui';
import { PlusBadge } from '@/shared/ui/plus-gate';
import { BottomNav } from '@/widgets/bottom-nav';
import { ChartCard } from '@/widgets/charts';

import {
  deltaTone,
  formatYm,
  monthBounds,
  neighbour,
  parseYm,
  placeMonth,
  resolveMonth,
  signed,
  type YearMonth,
} from '../model/monthly';

interface Props {
  /** `YYYY-MM` from the route. */
  ym: string;
  /** `?calendar=` from the route (the hub sends the reader's calendar). */
  calendar?: string;
}

const href = (m: YearMonth, calendar: MonthlyCalendar) => `/analysis/monthly/${formatYm(m)}?calendar=${calendar}`;

/**
 * `/analysis/monthly/[ym]` (An_Monthly, B-N3-10): month stepper (no future
 * months), headline + summary sentence, «شاخص / این ماه / تغییر» table against
 * the month before, top 3 symptoms, the next-month suggestion and «ساخت PDF
 * برای پزشک» — Plus-gated, and «به‌زودی» until the doctor PDF (B-N6-04) exists.
 * Every sentence comes rendered from the server in the request language.
 */
export function AnalysisMonthlyPage({ ym, calendar }: Props) {
  const t = useTranslations('analysis.reports');
  const tm = useTranslations('analysis.reports.monthly');
  const tDetail = useTranslations('analysis.detail');
  const loc = useLocale() as Locale;
  const router = useRouter();
  const mounted = useMounted();

  const own = calendarSystem(loc) as MonthlyCalendar;
  const asked = isMonthlyCalendar(calendar) ? calendar : own;
  const parsed = parseYm(ym);
  const shown = parsed ? resolveMonth(parsed, asked, loc) : null;
  // Today is read on the client only (the server's clock and zone may differ).
  const bounds = useMemo(() => (mounted ? monthBounds(loc) : null), [mounted, loc]);
  const placement = !shown ? 'invalid' : bounds ? placeMonth(shown, bounds) : 'pending';
  const shownYm = shown ? formatYm(shown) : ym;

  // A link in the other calendar (or without one) is rewritten to the reader's month.
  const canonical = shown && (asked !== own || shownYm !== ym || calendar !== own) ? href(shown, own) : null;
  useEffect(() => {
    if (canonical) router.replace(canonical);
  }, [canonical, router]);

  const query = useMonthlyReport(shownYm, own, { enabled: placement === 'ok' });
  const r = query.data;

  const label = shown ? formatMonthLabel(shown.year, shown.month, loc) : '';
  const title = shown ? tm('title', { month: label }) : tDetail('titles.monthly');
  const prev = shown && bounds ? neighbour(shown, -1, bounds) : null;
  const next = shown && bounds ? neighbour(shown, 1, bounds) : null;
  const go = (m: YearMonth) => router.replace(href(m, own));

  let body: ReactNode;
  if (placement === 'invalid' || placement === 'future' || placement === 'too_old') {
    body = (
      <EmptyState
        icon="calendar"
        title={tm('invalid.title')}
        body={tm('invalid.body')}
        action={
          bounds ? (
            <SecondaryButton block={false} onClick={() => go(bounds.current)}>
              {tm('invalid.cta')}
            </SecondaryButton>
          ) : null
        }
      />
    );
  } else if (placement === 'pending' || (query.isPending && query.fetchStatus !== 'idle')) {
    body = (
      <SkeletonGroup label={t('loading')} className="axc-skel">
        <Skeleton shape="block" className="amr-skel-hero" />
        <Skeleton shape="card" className="amr-skel-table" />
        <Skeleton shape="card" />
      </SkeletonGroup>
    );
  } else if (query.isError || !r) {
    body = (
      <EmptyState
        icon="warning"
        title={t('error.title')}
        body={t('error.body')}
        action={
          <SecondaryButton icon="refresh" onClick={() => void query.refetch()} loading={query.isFetching}>
            {t('error.retry')}
          </SecondaryButton>
        }
      />
    );
  } else {
    body = <MonthlyBody report={r} monthLabel={label} />;
  }

  const subtitle =
    r && !r.month.complete && placement === 'ok'
      ? tm('running', { date: formatDayMonth(fromApiDate(r.month.to), loc) })
      : undefined;

  return (
    <div className="view axc-page">
      <SkyLayer />
      <div className="scroll axc-scroll">
        <ScreenHeader
          title={title}
          subtitle={subtitle}
          onBack={() => router.push('/analysis')}
          backLabel={t('back')}
          className="axc-hdr"
        />
        <div className="axc-content">
          {shown ? <MonthStepper prev={prev} next={next} loc={loc} onGo={go} /> : null}
          {body}
        </div>
      </div>
      <BottomNav />
    </div>
  );
}

function MonthStepper({
  prev,
  next,
  loc,
  onGo,
}: {
  prev: YearMonth | null;
  next: YearMonth | null;
  loc: Locale;
  onGo: (m: YearMonth) => void;
}) {
  const tm = useTranslations('analysis.reports.monthly');
  const rtl = useDirection() === 'rtl';
  const name = (m: YearMonth) => formatMonthLabel(m.year, m.month, loc);
  // Each side names the month it opens; a side with nowhere to go (no future months) stays empty.
  return (
    <nav className="amr-stepper" aria-label={tm('stepper')}>
      {prev ? (
        <button type="button" className="amr-step" aria-label={tm('prev', { month: name(prev) })} onClick={() => onGo(prev)}>
          <Icon name={rtl ? 'chevronRight' : 'chevronLeft'} size={16} strokeWidth={2.4} />
          <span aria-hidden>{name(prev)}</span>
        </button>
      ) : (
        <span />
      )}
      {next ? (
        <button type="button" className="amr-step" aria-label={tm('next', { month: name(next) })} onClick={() => onGo(next)}>
          <span aria-hidden>{name(next)}</span>
          <Icon name={rtl ? 'chevronLeft' : 'chevronRight'} size={16} strokeWidth={2.4} />
        </button>
      ) : (
        <span />
      )}
    </nav>
  );
}

function MonthlyBody({ report: r, monthLabel }: { report: MonthlyReport; monthLabel: string }) {
  const t = useTranslations('analysis.reports');
  const tm = useTranslations('analysis.reports.monthly');
  const tPlus = useTranslations('plus.gate');
  const loc = useLocale() as Locale;
  const router = useRouter();
  const num = (n: number) => formatNumber(n, loc);
  const daysLogged = r.metrics.find((m) => m.key === 'days_logged')?.value;

  return (
    <>
      <section className="amr-hero" aria-labelledby="amr-headline">
        <h2 id="amr-headline" className="amr-headline">
          {r.headline.text}
        </h2>
        {r.summary.text ? <p className="amr-summary">{r.summary.text}</p> : null}
      </section>

      {daysLogged === 0 ? (
        <EmptyState
          icon="note"
          title={t('empty.title')}
          body={t('empty.body')}
          action={
            <PrimaryButton icon="plus" block={false} onClick={() => openSheet('log')}>
              {t('empty.cta')}
            </PrimaryButton>
          }
        />
      ) : (
        <>
          <section className="nb-card amr-table-card">
            <table className="amr-table">
              <caption className="amr-sr">{tm('table.caption', { month: monthLabel })}</caption>
              <thead>
                <tr>
                  <th scope="col" className="amr-th-metric">
                    {tm('table.metric')}
                  </th>
                  <th scope="col" className="amr-th-num">
                    {tm('table.value')}
                  </th>
                  <th scope="col" className="amr-th-num">
                    {tm('table.delta')}
                  </th>
                </tr>
              </thead>
              <tbody>
                {r.metrics.map((m) => (
                  <MetricRow key={m.key} metric={m} />
                ))}
              </tbody>
            </table>
          </section>

          <ChartCard title={tm('top.title')}>
            {r.topSymptoms.length ? (
              <ul className="amr-top">
                {r.topSymptoms.map((s) => (
                  <li key={s.key} className="amr-top-row">
                    <span className="amr-top-name">{s.label}</span>
                    <b className="amr-top-days">{tm('top.days', { n: num(s.days) })}</b>
                  </li>
                ))}
              </ul>
            ) : (
              <p className="anr-note">{tm('top.none')}</p>
            )}
          </ChartCard>
        </>
      )}

      <ChartCard title={tm('suggestion')}>
        <p className="amr-suggestion">{r.suggestion.text}</p>
      </ChartCard>

      <div className="amr-pdf-wrap">
        <button
          type="button"
          className={clsx('amr-pdf', !r.pdf.locked && 'is-soon')}
          aria-disabled={!r.pdf.locked || undefined}
          aria-describedby="amr-pdf-note"
          onClick={r.pdf.locked ? () => router.push('/plus') : undefined}
        >
          <Icon name="note" size={18} strokeWidth={2} />
          <span className="amr-pdf-label">{tm('pdf.cta')}</span>
          {r.pdf.locked ? (
            <PlusBadge label={tPlus('label')} />
          ) : (
            <StatusPill tone="brand">{tm('pdf.soon')}</StatusPill>
          )}
        </button>
        <p id="amr-pdf-note" className="amr-pdf-note">
          {r.pdf.locked ? tm('pdf.locked') : tm('pdf.soonNote')}
        </p>
      </div>
    </>
  );
}

function MetricRow({ metric: m }: { metric: MonthlyMetric }) {
  const t = useTranslations('analysis.reports');
  const tm = useTranslations('analysis.reports.monthly');
  const loc = useLocale() as Locale;
  const num = (n: number | string) => formatNumber(n, loc);
  const dec = (n: number | string) => formatDecimal(n, loc);
  const none = t('none');

  const bp = (v: BloodPressureValue) => `${num(v.systolic)}/${num(v.diastolic)}`;
  let value: ReactNode = none;
  if (m.value != null) {
    if (typeof m.value !== 'number') value = <bdi dir="ltr">{bp(m.value)}</bdi>;
    else if (m.key === 'cycle_length') value = tm('units.days', { n: num(m.value) });
    else if (m.key === 'sleep') value = tm('units.hours', { n: dec(m.value) });
    else if (m.key === 'weight') value = tm('units.kg', { n: dec(m.value) });
    else if (m.key === 'good_mood') value = tm('units.percent', { n: num(m.value) });
    else value = num(m.value);
  }

  let delta: ReactNode = none;
  let tone = 'flat';
  if (m.delta != null && m.value != null) {
    tone = deltaTone(m.key, m.delta);
    if (typeof m.delta !== 'number') {
      const s = signed(m.delta.systolic, 0);
      const d = signed(m.delta.diastolic, 0);
      delta = `${s.sign}${num(s.abs)}/${d.sign}${num(d.abs)}`;
    } else {
      const decimals = m.key === 'sleep' || m.key === 'weight' ? 1 : 0;
      const s = signed(m.delta, decimals);
      const text = `${s.sign}${decimals ? dec(s.abs) : num(s.abs)}`;
      delta = m.key === 'good_mood' ? tm('units.deltaPercent', { n: text }) : text;
    }
    delta = <bdi dir="ltr">{delta}</bdi>;
  }

  return (
    <tr>
      <th scope="row" className="amr-td-metric">
        {tm(`metrics.${m.key}`)}
      </th>
      <td className="amr-td-value">{value}</td>
      <td className={clsx('amr-td-delta', `is-${tone}`)}>{delta}</td>
    </tr>
  );
}
