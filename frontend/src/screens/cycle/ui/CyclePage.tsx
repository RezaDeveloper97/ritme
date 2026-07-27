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
    <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', padding: '6px 18px 10px' }}>
      <button className="iconbtn" style={{ background: 'rgba(255,255,255,.6)' }}>
        <Icon name="bell" size={20} />
      </button>
      <div style={{ textAlign: 'center' }}>
        <div style={{ fontWeight: 900, fontSize: 20, color: 'var(--brand-deep)' }}>{t('title')}</div>
        <div style={{ fontSize: 10, color: 'var(--muted)', marginTop: 1 }}>{tagline}</div>
      </div>
      <button
        className="iconbtn"
        style={{ background: 'rgba(255,255,255,.6)', color: 'var(--brand)', opacity: recalculating ? 0.5 : 1 }}
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
    <div style={{ padding: '2px 16px 0' }}>
      <div
        className="card"
        style={{ padding: '18px 16px', background: 'linear-gradient(135deg,var(--surface) 0%,var(--surface-2) 100%)' }}
      >
        <div style={{ fontSize: 12, fontWeight: 700, color: 'var(--brand)', textAlign: 'start', marginBottom: 14 }}>
          {t('basedOn')}
        </div>

        <div style={{ display: 'flex', alignItems: 'center', gap: 16 }}>
          {/* Cycle-day donut. Fill proportion = today's day within the cycle. */}
          <div
            style={{
              position: 'relative',
              width: 104,
              height: 104,
              flex: '0 0 104px',
              borderRadius: '50%',
              background: `conic-gradient(${c} ${ringPct * 3.6}deg, var(--line) 0deg)`,
            }}
          >
            <div
              style={{
                position: 'absolute',
                inset: 9,
                borderRadius: '50%',
                background: 'var(--surface)',
                display: 'flex',
                flexDirection: 'column',
                alignItems: 'center',
                justifyContent: 'center',
                gap: 1,
              }}
            >
              <span style={{ fontSize: 30, fontWeight: 900, color: 'var(--ink)', lineHeight: 1, fontVariantNumeric: 'tabular-nums' }}>
                {format.number(pred.cycleDay)}
              </span>
              <span style={{ fontSize: 10, color: 'var(--muted)', fontWeight: 600 }}>
                {t('status.ofN', { n: pred.cycleLength })}
              </span>
            </div>
          </div>

          <div style={{ flex: 1, textAlign: 'start' }}>
            <div style={{ fontSize: 11, color: 'var(--muted)', fontWeight: 600, marginBottom: 4 }}>
              {t('status.phaseLabel')}
            </div>
            <div style={{ display: 'inline-flex', alignItems: 'center', gap: 6, background: bg, color: c, borderRadius: 20, padding: '5px 12px' }}>
              <DropSolid size={14} color={c} />
              <span style={{ fontSize: 13, fontWeight: 800 }}>{t(`phaseLabel.${pred.phase}`)}</span>
            </div>

            <div style={{ display: 'flex', gap: 8, marginTop: 12 }}>
              <div style={{ flex: 1, background: 'var(--surface-2)', borderRadius: 12, padding: '8px 10px' }}>
                <div style={{ fontSize: 10, color: 'var(--muted)', fontWeight: 600 }}>{t('status.fertility')}</div>
                <div style={{ fontSize: 15, fontWeight: 800, color: 'var(--brand)', marginTop: 2 }}>
                  {t('percent', { n: pred.fertilityPercent })}
                </div>
              </div>
            </div>
          </div>
        </div>

        {/* Fertile-window / PMS status line, only when relevant. */}
        {(pred.isFertileWindow || calc.isPmsWindow) && (
          <div style={{ display: 'flex', alignItems: 'center', gap: 8, marginTop: 14, background: 'var(--line)', borderRadius: 12, padding: '9px 12px' }}>
            <span style={{ color: pred.isFertileWindow ? 'var(--amber)' : 'var(--indigo)' }}>
              <Icon name="info" size={16} />
            </span>
            <span style={{ flex: 1, textAlign: 'start', fontSize: 12, fontWeight: 700, color: 'var(--ink)' }}>
              {pred.isFertileWindow ? t('status.fertileWindow') : t('status.pms')}
            </span>
          </div>
        )}

        <p className="sub" style={{ textAlign: 'start', margin: '14px 2px 0', lineHeight: 1.9 }}>
          {phaseDesc}
        </p>

        {/* Deep-link into the full educational content for the current phase.
            Only shown once the engine has a confident sub-phase; the target
            reads that phase from live cycle data, never from the URL (§11). */}
        {subphase && (
          <Link
            href="/cycle/phase"
            className="btn btn-ghost"
            style={{ height: 44, borderRadius: 14, marginTop: 14, gap: 8, fontSize: 13, textDecoration: 'none' }}
          >
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
    <div style={{ padding: '14px 16px 0' }}>
      <div className="card" style={{ padding: '16px 14px' }}>
        <div style={{ fontSize: 15, fontWeight: 800, color: 'var(--ink)', textAlign: 'start', marginBottom: 16 }}>
          {t('timeline.title')}
        </div>

        {/* Left→right timeline regardless of locale — a cycle always runs
            day 1 → day N, so pin the bar to LTR (§12-safe: not layout chrome). */}
        <div dir="ltr" style={{ position: 'relative', height: 10, borderRadius: 99, background: 'var(--line)', overflow: 'visible', marginBottom: 26 }}>
          {/* Fertile band */}
          <span
            style={{
              position: 'absolute', top: 0, bottom: 0,
              left: `${at(fertileStart)}%`, width: `${at(ovulation + 1) - at(fertileStart)}%`,
              background: 'var(--amber-soft)', borderRadius: 99,
            }}
          />
          {/* Progress up to today */}
          <span
            style={{
              position: 'absolute', top: 0, bottom: 0, left: 0, width: `${todayPos}%`,
              background: 'linear-gradient(90deg,var(--blush),var(--brand))', borderRadius: 99, opacity: 0.85,
            }}
          />
          {/* Ovulation tick */}
          <span
            style={{
              position: 'absolute', top: '50%', left: `${at(ovulation)}%`,
              width: 10, height: 10, marginTop: -5, marginLeft: -5,
              borderRadius: '50%', background: 'var(--green-dot)', border: '2px solid var(--surface)',
            }}
          />
          {/* Today marker */}
          <span
            style={{
              position: 'absolute', top: -5, left: `${todayPos}%`,
              width: 4, height: 20, marginLeft: -2, borderRadius: 2, background: 'var(--brand)',
            }}
          />
        </div>

        {rows.map((r, i) => (
          <div
            key={r.l}
            style={{
              display: 'flex', alignItems: 'center', justifyContent: 'space-between',
              padding: '11px 2px',
              ...(i < rows.length - 1 ? { borderBottom: '1px solid var(--line)' } : {}),
            }}
          >
            <div style={{ display: 'flex', alignItems: 'center', gap: 10 }}>
              <span className="dot" style={{ background: r.bg, color: r.c }}>
                <DropSolid size={16} color={r.c} />
              </span>
              <span style={{ fontSize: 14, fontWeight: 700, color: 'var(--ink)' }}>{r.l}</span>
            </div>
            <span style={{ fontSize: 13, fontWeight: 700, color: 'var(--muted)', fontVariantNumeric: 'tabular-nums' }}>{r.d}</span>
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
    <div style={{ padding: '14px 16px 0' }}>
      <div className="card" style={{ padding: '16px 14px' }}>
        <div style={{ fontSize: 15, fontWeight: 800, color: 'var(--ink)', textAlign: 'start', marginBottom: 4 }}>
          {t('summary.title')}
        </div>
        {rows.map(([label, val], i) => (
          <div
            key={label}
            style={{
              display: 'flex', alignItems: 'center', justifyContent: 'space-between',
              padding: '12px 2px',
              ...(i < rows.length - 1 ? { borderBottom: '1px solid var(--line)' } : {}),
            }}
          >
            <span style={{ fontSize: 13, color: 'var(--muted)', fontWeight: 600 }}>{label}</span>
            <span style={{ fontSize: 13, fontWeight: 800, color: 'var(--ink)', fontVariantNumeric: 'tabular-nums' }}>{val}</span>
          </div>
        ))}
        <div style={{ paddingTop: 14 }}>
          <Link href="/calendar" className="btn btn-primary" style={{ borderRadius: 14, textDecoration: 'none' }}>
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
    <div style={{ padding: '14px 16px 0' }}>
      <div className="card" style={{ padding: '16px 14px' }}>
        <div style={{ fontSize: 15, fontWeight: 800, color: 'var(--ink)', textAlign: 'start', marginBottom: 4 }}>
          {t('cycles.title')}
        </div>
        <div style={{ display: 'flex', alignItems: 'center', gap: 10, padding: '10px 2px 14px' }}>
          <span className="dot" style={{ width: 40, height: 40, background: 'var(--pink-bg)', color: 'var(--brand)' }}>
            <DropSolid size={18} color="var(--brand)" />
          </span>
          <div style={{ textAlign: 'start' }}>
            <div style={{ fontSize: 14, fontWeight: 700, color: 'var(--ink)' }}>
              {t('cycles.current')}: {t('dayN', { n: pred.cycleDay })}
            </div>
            <div style={{ fontSize: 11, color: 'var(--muted)', marginTop: 3 }}>
              {t('cycles.startedOn')} {cycleStartDate}
            </div>
          </div>
        </div>
        <Link
          href="/log"
          className="btn btn-ghost"
          style={{ height: 44, borderRadius: 14, gap: 8, fontSize: 13, textDecoration: 'none' }}
        >
          <Icon name="plus" size={16} /> {t('cycles.addPrevious')}
        </Link>
      </div>
    </div>
  );
}

// ── Smart tip (educational, non-diagnostic — §11) ──────────────
function SmartTip({ t, body, quote }: { t: T; body: string; quote: string }) {
  return (
    <div style={{ padding: '14px 16px 0' }}>
      <div className="card" style={{ padding: '14px 12px' }}>
        <div style={{ fontSize: 14, fontWeight: 800, color: 'var(--ink)', textAlign: 'start', marginBottom: 10 }}>
          {t('smartTip.title')}
        </div>
        <p className="sub" style={{ textAlign: 'start', margin: '0 2px 14px' }}>{body}</p>
        <div style={{ display: 'flex', alignItems: 'center', gap: 10, background: 'linear-gradient(90deg,var(--surface-2),var(--indigo-soft))', borderRadius: 12, padding: 12 }}>
          <span style={{ color: 'var(--pink)' }}>
            <Icon name="sparkle" size={20} fill="currentColor" strokeWidth={0} />
          </span>
          <span style={{ flex: 1, textAlign: 'start', fontSize: 12.5, fontWeight: 700, color: 'var(--ink)' }}>
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
    <div style={{ flex: 1, display: 'flex', flexDirection: 'column', alignItems: 'center', justifyContent: 'center', gap: 14, padding: 24 }}>
      <span
        style={{
          width: 72, height: 72, borderRadius: '50%', background: 'var(--surface)', color: 'var(--brand)',
          display: 'flex', alignItems: 'center', justifyContent: 'center',
          boxShadow: '0 10px 30px rgba(17,32,47,.08)',
        }}
      >
        <Icon name="drop" size={30} fill="currentColor" strokeWidth={0} />
      </span>
      <div style={{ fontSize: 15, fontWeight: 800, color: 'var(--ink)' }}>{t('empty.title')}</div>
      <p className="sub" style={{ textAlign: 'center', margin: 0, maxWidth: 260 }}>{t('empty.body')}</p>
      <Link href="/log" className="btn btn-primary" style={{ width: 'auto', padding: '0 26px', borderRadius: 14, textDecoration: 'none' }}>
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
    <div className="view" style={{ background: 'var(--page)' }}>
      <div className="home-grad" style={{ position: 'absolute', top: 0, insetInlineStart: 0, insetInlineEnd: 0, height: 300 }} />

      <div style={{ position: 'relative', zIndex: 1 }}>
      </div>

      <div className="scroll" style={{ position: 'relative', zIndex: 1, display: 'flex', flexDirection: 'column' }}>
        <CycleHeader
          t={t}
          tagline={t('tagline')}
          onRecalculate={() => recalc.mutate()}
          recalculating={recalculating}
        />

        {recalculating && (
          <div style={{ textAlign: 'center', fontSize: 11, color: 'var(--muted)', padding: '0 16px 4px' }}>
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
            <div style={{ height: 26 }} />
          </>
        ) : (
          mounted && !todayQuery.isLoading && <EmptyState t={t} />
        )}
      </div>

      <BottomNav />
    </div>
  );
}
