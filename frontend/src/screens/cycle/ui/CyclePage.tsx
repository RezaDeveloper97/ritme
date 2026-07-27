'use client';

import { useFormatter, useLocale, useTranslations } from 'next-intl';
import { useEffect, useState } from 'react';

import {
  CycleValuesCard,
  deriveCyclePredictions,
  useCycleStatus,
  useCycleToday,
  useRecalculateCycle,
  type CycleCalculation,
  type CyclePhase,
  type CyclePredictions,
} from '@/entities/cycle';
import { useDailyMessage, type DailyMessage } from '@/entities/message';
import { Link, useRouter } from '@/shared/i18n';
import type { Locale } from '@/shared/i18n';
import { addDays, formatJalaliDayMonth, today } from '@/shared/lib/date';
import { DropSolid, Icon } from '@/shared/ui';
import { BottomNav } from '@/widgets/bottom-nav';

type T = ReturnType<typeof useTranslations>;

// Per-phase accent colors, shared with the calendar/home palette so a phase
// reads the same everywhere. Follicular/luteal are the calm "between events"
// violet; the active phases keep their signal colors.
const PHASE_COLOR: Record<CyclePhase, { c: string; bg: string }> = {
  period: { c: 'var(--brand)', bg: 'var(--pink-bg)' },
  follicular: { c: 'var(--indigo)', bg: 'var(--indigo-soft)' },
  fertile: { c: 'var(--amber)', bg: 'var(--amber-soft)' },
  ovulation: { c: 'var(--green-dot)', bg: 'var(--green-tint)' },
  luteal: { c: 'var(--indigo)', bg: 'var(--indigo-soft)' },
};

const clampPct = (v: number) => Math.min(100, Math.max(0, v));

// Cycle regularity comes back as a free-form string (or null). Narrow it to the
// three labels we actually translate; anything unknown reads as "being assessed".
function variabilityKey(raw: string | null): 'regular' | 'irregular' | 'unknown' {
  if (raw === 'regular' || raw === 'irregular') return raw;
  return 'unknown';
}

// ── Header (title + recalculate) ───────────────────────────────
function CycleHeader({
  t, tagline, onRecalculate, recalculating,
}: {
  t: T;
  tagline: string;
  onRecalculate: () => void;
  recalculating: boolean;
}) {
  return (
    <div className="cyc-hdr">
      <button className="iconbtn cyc-hdr-btn">
        <Icon name="bell" size={20} />
      </button>
      <div className="cyc-brand">
        <div className="cyc-title">{t('title')}</div>
        <div className="cyc-tagline">{tagline}</div>
      </div>
      {/* Dimming while recalculating comes from `.iconbtn:disabled`. */}
      <button
        className="iconbtn cyc-hdr-btn is-brand"
        onClick={onRecalculate}
        disabled={recalculating}
        aria-label={t('recalculate')}
      >
        <Icon name={recalculating ? 'loader' : 'refresh'} size={20} />
      </button>
    </div>
  );
}

// ── Status hero: cycle-day ring + phase ────────────────────────
function CycleStatusCard({
  t, pred, calc, phaseDesc, subphase,
}: {
  t: T;
  pred: CyclePredictions;
  calc: CycleCalculation;
  phaseDesc: string;
  subphase: string | null;
}) {
  const format = useFormatter();
  const { c, bg } = PHASE_COLOR[pred.phase];
  const ringPct = clampPct((pred.cycleDay / pred.cycleLength) * 100);

  return (
    <div className="cyc-hero-wrap">
      <div className="card cyc-hero">
        <div className="cyc-basedon">
          {t('basedOn')}
        </div>

        <div className="cyc-hero-row">
          {/* Cycle-day donut. Fill proportion = today's day within the cycle,
              so the sweep and the phase colour are the only inline values. */}
          <div
            className="cyc-donut"
            style={{ background: `conic-gradient(${c} ${ringPct * 3.6}deg, var(--line) 0deg)` }}
          >
            <div className="cyc-donut-core">
              <span className="cyc-donut-day">
                {format.number(pred.cycleDay)}
              </span>
              <span className="cyc-donut-of">
                {t('status.ofN', { n: pred.cycleLength })}
              </span>
            </div>
          </div>

          <div className="cyc-hero-side">
            <div className="cyc-phase-lbl">
              {t('status.phaseLabel')}
            </div>
            <div className="cyc-phase-chip" style={{ background: bg, color: c }}>
              <DropSolid size={14} color={c} />
              <span className="cyc-phase-name">{t(`phaseLabel.${pred.phase}`)}</span>
            </div>

            <div className="cyc-stats">
              <div className="cyc-stat">
                <div className="cyc-stat-lbl">{t('status.fertility')}</div>
                <div className="cyc-stat-val">
                  {t('percent', { n: pred.fertilityPercent })}
                </div>
              </div>
            </div>
          </div>
        </div>

        {/* Fertile-window / PMS status line, only when relevant. */}
        {(pred.isFertileWindow || calc.isPmsWindow) && (
          <div className="cyc-note">
            <span style={{ color: pred.isFertileWindow ? 'var(--amber)' : 'var(--indigo)' }}>
              <Icon name="info" size={16} />
            </span>
            <span className="cyc-note-text">
              {pred.isFertileWindow ? t('status.fertileWindow') : t('status.pms')}
            </span>
          </div>
        )}

        <p className="sub cyc-desc">
          {phaseDesc}
        </p>

        {/* Deep-link into the full educational content for the current phase.
            Only shown once the engine has a confident sub-phase; the target
            reads that phase from live cycle data, never from the URL (§11). */}
        {subphase && (
          <Link href="/cycle/phase" className="btn btn-ghost cyc-cta">
            <Icon name="info" size={16} /> {t('phaseDetailsCta')}
          </Link>
        )}
      </div>
    </div>
  );
}

// ── Timeline: linear cycle bar + upcoming events ───────────────
function CycleTimeline({
  t, pred, calc, windowDate, ovulationDate, nextPeriodDate,
}: {
  t: T;
  pred: CyclePredictions;
  calc: CycleCalculation;
  windowDate: string;
  ovulationDate: string;
  nextPeriodDate: string;
}) {
  const len = pred.cycleLength;
  const at = (day: number) => clampPct((day / len) * 100);
  const ovulation = calc.estimatedOvulationDay;
  const fertileStart = ovulation - 5; // matches the backend fertile window (ovulation − 5)
  const todayPos = at(pred.cycleDay);

  const rows = [
    { l: t('timeline.window'), d: windowDate, c: 'var(--amber)', bg: 'var(--amber-soft)' },
    { l: t('timeline.ovulation'), d: ovulationDate, c: 'var(--green-dot)', bg: 'var(--green-tint)' },
    { l: t('timeline.nextPeriod'), d: nextPeriodDate, c: 'var(--brand)', bg: 'var(--pink-bg)' },
  ];

  return (
    <div className="sec-tight">
      <div className="card pad-card">
        <div className="card-titr">
          {t('timeline.title')}
        </div>

        {/* Left→right timeline regardless of locale — a cycle always runs
            day 1 → day N, so pin the bar to LTR (§12-safe: not layout chrome).
            Shares `.cyclebar*` with the home screen's copy of this chart. */}
        <div dir="ltr" className="cyclebar is-roomy">
          {/* Fertile band */}
          <span
            className="cyclebar-band"
            style={{ left: `${at(fertileStart)}%`, width: `${at(ovulation + 1) - at(fertileStart)}%` }}
          />
          {/* Progress up to today */}
          <span className="cyclebar-fill" style={{ width: `${todayPos}%` }} />
          {/* Ovulation tick */}
          <span className="cyclebar-tick" style={{ left: `${at(ovulation)}%` }} />
          {/* Today marker */}
          <span className="cyclebar-now" style={{ left: `${todayPos}%` }} />
        </div>

        {rows.map(r => (
          <div key={r.l} className="cyc-event">
            <div className="cyc-event-left">
              <span className="dot" style={{ background: r.bg, color: r.c }}>
                <DropSolid size={16} color={r.c} />
              </span>
              <span className="cyc-event-name">{r.l}</span>
            </div>
            <span className="cyc-event-date">{r.d}</span>
          </div>
        ))}
      </div>
    </div>
  );
}

// ── Cycle summary ──────────────────────────────────────────────
function CycleSummaryCard({
  t, pred, calc,
}: {
  t: T;
  pred: CyclePredictions;
  calc: CycleCalculation;
}) {
  const rows: [string, string][] = [
    [t('summary.length'), t('days', { n: pred.cycleLength })],
    [t('summary.ovulationDay'), t('dayN', { n: calc.estimatedOvulationDay })],
    [t('summary.phase'), t(`phaseLabel.${pred.phase}`)],
    [t('summary.variability'), t(`variability.${variabilityKey(calc.cycleVariability)}`)],
  ];

  return (
    <div className="sec-tight">
      <div className="card pad-card">
        <div className="card-titr is-tight">
          {t('summary.title')}
        </div>
        {rows.map(([label, val]) => (
          <div key={label} className="data-row">
            <span className="data-row-label">{label}</span>
            <span className="data-row-value">{val}</span>
          </div>
        ))}
        <div className="cyc-more-wrap">
          <Link href="/calendar" className="btn btn-primary cyc-more-cta">
            {t('summary.viewMore')}
          </Link>
        </div>
      </div>
    </div>
  );
}

// ── My cycles ──────────────────────────────────────────────────
function MyCycles({
  t, pred, cycleStartDate,
}: {
  t: T;
  pred: CyclePredictions;
  cycleStartDate: string;
}) {
  return (
    <div className="sec-tight">
      <div className="card pad-card">
        <div className="card-titr is-tight">
          {t('cycles.title')}
        </div>
        <div className="cyc-cycles-row">
          <span className="dot cyc-cycles-dot">
            <DropSolid size={18} color="var(--brand)" />
          </span>
          <div className="text-start">
            <div className="cyc-cycles-name">
              {t('cycles.current')}: {t('dayN', { n: pred.cycleDay })}
            </div>
            <div className="cyc-cycles-sub">
              {t('cycles.startedOn')} {cycleStartDate}
            </div>
          </div>
        </div>
        <Link href="/log" className="btn btn-ghost cyc-add-cta">
          <Icon name="plus" size={16} /> {t('cycles.addPrevious')}
        </Link>
      </div>
    </div>
  );
}

// ── Smart tip (educational, non-diagnostic — §11) ──────────────
function SmartTip({ t, body, quote }: { t: T; body: string; quote: string }) {
  return (
    <div className="sec-tight">
      <div className="card pad-card-sm">
        <div className="cyc-tip-title">
          {t('smartTip.title')}
        </div>
        <p className="sub cyc-tip-body">{body}</p>
        <div className="tip-action">
          <span className="tip-action-icon">
            <Icon name="sparkle" size={20} fill="currentColor" strokeWidth={0} />
          </span>
          <span className="tip-action-text">
            {quote}
          </span>
        </div>
      </div>
    </div>
  );
}

// ── Empty state (no cycle logged yet) ──────────────────────────
function EmptyState({ t }: { t: T }) {
  return (
    <div className="cyc-empty">
      <span className="cyc-empty-icon">
        <Icon name="drop" size={30} fill="currentColor" strokeWidth={0} />
      </span>
      <div className="cyc-empty-title">{t('empty.title')}</div>
      <p className="sub cyc-empty-body">{t('empty.body')}</p>
      <Link href="/log" className="btn btn-primary cyc-empty-cta">
        {t('empty.cta')}
      </Link>
    </div>
  );
}

// ── Main export ────────────────────────────────────────────────
export function CyclePage() {
  const t = useTranslations('cycle');
  const loc = useLocale() as Locale;
  const router = useRouter();

  // Server render has no query data yet, so the server always emits the "loading"
  // shell. Gate the data/empty branches on mount so the first client render
  // matches the server HTML and hydration doesn't mismatch (§ React hydration).
  const [mounted, setMounted] = useState(false);
  useEffect(() => setMounted(true), []);

  // Server state (§8) — today's cycle calculation + personalized message.
  const todayQuery = useCycleToday();
  const todayData = todayQuery.data;
  const { data: daily } = useDailyMessage();
  const recalc = useRecalculateCycle();

  const calc = todayData?.calculation ?? null;
  const pred = calc ? deriveCyclePredictions(calc) : null;

  // While the backend recalculates, poll status and refetch today's calc once
  // it settles so the screen reflects the fresh result without a manual reload.
  const recalculating = Boolean(todayData?.isRecalculating) || recalc.isPending;
  const { data: status } = useCycleStatus({ poll: recalculating });
  const settled = status ? !status.is_processing : false;
  useEffect(() => {
    if (recalculating && settled) {
      void todayQuery.refetch();
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [recalculating, settled]);

  // Turn day-offsets into Jalali dates only here, at the display boundary (§7).
  const base = today();
  const fmt = (offset: number) => formatJalaliDayMonth(addDays(base, offset), loc);

  const message: DailyMessage | undefined = daily;
  // Status card uses the short line, the smart tip uses the long one — otherwise the
  // same paragraph shows twice on the page (§ no duplicate copy).
  const phaseDesc = message?.primary.shortMessage
    || (pred ? t(`phaseDesc.${pred.phase}`) : '');
  const smartTipBody = message?.primary.longMessage || t('smartTip.body');
  const smartTipQuote = message?.primary.actionSuggestion || t('smartTip.quote');

  return (
    <div className="view cyc-page">
      <div className="home-grad cyc-grad" />

      <div className="scroll page-scroll cyc-scroll">
        <CycleHeader
          t={t}
          tagline={t('tagline')}
          onRecalculate={() => recalc.mutate()}
          recalculating={recalculating}
        />

        {recalculating && (
          <div className="page-updating">
            {t('updating')}
          </div>
        )}

        {mounted && calc && pred ? (
          <>
            <CycleStatusCard t={t} pred={pred} calc={calc} phaseDesc={phaseDesc} subphase={todayData?.cycleView?.subphase ?? null} />
            <CycleTimeline
              t={t}
              pred={pred}
              calc={calc}
              windowDate={fmt(Math.max(0, pred.daysUntilFertileWindow))}
              ovulationDate={fmt(Math.max(0, pred.daysUntilOvulation))}
              nextPeriodDate={fmt(pred.daysUntilNextPeriod)}
            />
            <CycleSummaryCard t={t} pred={pred} calc={calc} />
            {todayData?.cycleView && (
              <CycleValuesCard
                title={t('values.title')}
                loggedLabel={t('values.logged')}
                loggedValue={
                  todayData.cycleView.profileValues.cycleLength != null
                    ? t('days', { n: todayData.cycleView.profileValues.cycleLength })
                    : t('unavailable')
                }
                suggestion={
                  todayData.cycleView.calculatedValues.cycleLength != null &&
                  todayData.cycleView.profileValues.cycleLength != null &&
                  todayData.cycleView.calculatedValues.cycleLength !==
                    todayData.cycleView.profileValues.cycleLength
                    ? {
                        text: t('values.suggestion', {
                          n: todayData.cycleView.calculatedValues.cycleLength,
                        }),
                        ctaLabel: t('values.syncCta', {
                          n: todayData.cycleView.calculatedValues.cycleLength,
                        }),
                        onSync: () => router.push('/profile/health'),
                      }
                    : null
                }
                basedOnText={t('values.basedOn', {
                  source: t(
                    todayData.cycleView.effectiveValues.source === 'recent_valid_cycles'
                      ? 'values.source.recent_valid_cycles'
                      : todayData.cycleView.effectiveValues.source === 'profile'
                        ? 'values.source.profile'
                        : 'values.source.default',
                  ),
                })}
              />
            )}
            <MyCycles t={t} pred={pred} cycleStartDate={fmt(-(pred.cycleDay - 1))} />
            <SmartTip t={t} body={smartTipBody} quote={smartTipQuote} />
            <div className="page-tail" />
          </>
        ) : (
          mounted && !todayQuery.isLoading && <EmptyState t={t} />
        )}
      </div>

      <BottomNav />
    </div>
  );
}
