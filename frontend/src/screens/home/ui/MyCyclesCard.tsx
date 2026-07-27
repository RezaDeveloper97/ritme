'use client';

import { useLocale, useTranslations } from 'next-intl';
import { useState } from 'react';

import { useMyCyclesSection, type CycleRecord } from '@/entities/cycle';
import { PeriodDateEditor } from '@/features/log-period';
import type { Locale } from '@/shared/i18n';
import { formatJalali, formatJalaliDayMonth, fromApiDate, toJalali } from '@/shared/lib/date';
import { DropSolid, Icon } from '@/shared/ui';

import { SectionHead } from './SectionHead';

/** How many past cycles are visible before the user expands the list. */
const COLLAPSED_COUNT = 3;

/**
 * «سیکل‌های من» — where the user is in the current cycle, and the cycles behind
 * it with the length each one actually ran.
 *
 * Every number is server-derived (`/home/sections/my_cycles`): a past cycle's
 * length is the gap to the period that followed it, which only the backend can
 * know. Nothing here is a placeholder — a cycle with no measured length shows a
 * dash rather than a guess, and the list stays empty until real periods exist.
 */
export function MyCyclesCard() {
  const t = useTranslations('home');
  const loc = useLocale() as Locale;
  const { data: section, isLoading } = useMyCyclesSection();
  const [expanded, setExpanded] = useState(false);
  const [editorOpen, setEditorOpen] = useState(false);

  if (isLoading) return <MyCyclesSkeleton title={t('cycles.title')} />;
  if (!section) return null;

  const { current, previous, previous_count: previousCount, averages } = section.data;
  const start = fromApiDate(current.started_at);
  const visible = expanded ? previous : previous.slice(0, COLLAPSED_COUNT);
  const hasMore = previousCount > COLLAPSED_COUNT;

  return (
    <div style={{ padding: '16px 16px 0' }}>
      <div className="card" style={{ padding: '16px 14px' }}>
        <SectionHead title={t('cycles.title')} />

        {/* Current cycle — day number, when it began, and how the bleed went. */}
        <div style={{ display: 'flex', alignItems: 'center', gap: 10, padding: '4px 2px 12px' }}>
          <span className="dot" style={{ width: 40, height: 40, background: 'var(--pink-bg)', color: 'var(--brand)', flex: '0 0 auto' }}>
            <DropSolid size={18} color="var(--brand)" />
          </span>
          <div style={{ textAlign: 'start', flex: 1, minWidth: 0 }}>
            <div style={{ fontSize: 14, fontWeight: 700, color: 'var(--ink)' }}>
              {t('cycles.current')}: {t('cycles.dayN', { n: current.cycle_day })}
            </div>
            <div style={{ fontSize: 11, color: 'var(--muted)', marginTop: 3 }}>
              {t('cycles.startedOn')} {formatJalali(start, loc)}
            </div>
          </div>
          {current.is_ongoing ? (
            <Chip tone="brand">{t('cycles.ongoing')}</Chip>
          ) : current.period_length !== null ? (
            <Chip>{t('cycles.periodDays', { n: current.period_length })}</Chip>
          ) : null}
        </div>

        {/* Averages only appear once they summarize more than one cycle. */}
        {averages.based_on_cycles > 1 && (
          <div style={{ display: 'flex', flexWrap: 'wrap', gap: 8, padding: '0 2px 12px' }}>
            {averages.cycle_length !== null && (
              <Stat label={t('cycles.avgCycle')} value={t('days', { n: averages.cycle_length })} />
            )}
            {averages.period_length !== null && (
              <Stat label={t('cycles.avgPeriod')} value={t('days', { n: averages.period_length })} />
            )}
          </div>
        )}

        <div style={{ height: 1, background: 'var(--line)', margin: '2px 0 12px' }} />

        {/* Previous cycles */}
        {previousCount === 0 ? (
          <p style={{ margin: '0 2px 12px', fontSize: 12, color: 'var(--muted)', textAlign: 'start', lineHeight: 1.7 }}>
            {t('cycles.none')}
          </p>
        ) : (
          <>
            <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', margin: '0 2px 8px' }}>
              <span style={{ fontSize: 12.5, fontWeight: 700, color: 'var(--ink)' }}>
                {t('cycles.previousTitle')}
              </span>
              <span style={{ fontSize: 11, color: 'var(--muted)', fontVariantNumeric: 'tabular-nums' }}>
                {t('cycles.recorded', { n: previousCount })}
              </span>
            </div>
            <ul style={{ listStyle: 'none', margin: '0 0 10px', padding: 0, display: 'grid', gap: 6 }}>
              {visible.map(cycle => (
                <PreviousCycleRow key={cycle.id} cycle={cycle} loc={loc} t={t} />
              ))}
            </ul>
            {hasMore && (
              <button
                type="button"
                onClick={() => setExpanded(v => !v)}
                style={{ background: 'none', border: 0, padding: '0 2px 10px', fontSize: 12.5, fontWeight: 700, color: 'var(--brand)', cursor: 'pointer' }}
              >
                {expanded ? t('cycles.showLess') : t('cycles.showAll', { n: previousCount })}
              </button>
            )}
          </>
        )}

        {/* Figma Frame 21: divider + pink inline action with add-circle icon */}
        <div style={{ height: 1, background: 'var(--line)', margin: '2px 0 12px' }} />
        <button
          type="button"
          onClick={() => setEditorOpen(true)}
          style={{ display: 'flex', width: '100%', alignItems: 'center', justifyContent: 'center', gap: 8, background: 'none', border: 0, cursor: 'pointer', color: 'var(--pink-vivid)', fontSize: 14, fontWeight: 700, padding: '2px 0' }}
        >
          <Icon name="plus" size={18} stroke="currentColor" />
          {section.action?.label ?? t('cycles.addPrevious')}
        </button>
      </div>

      {/* The same editor the calendar uses — it opens on the current Jalali month
          and scrolls back through the past year, so older periods can be added. */}
      <PeriodDateEditor
        open={editorOpen}
        onClose={() => setEditorOpen(false)}
        initialView={{ year: toJalali(start).year, month: toJalali(start).month }}
      />
    </div>
  );
}

/** One past cycle: when it ran, how many days of bleeding, and its total length. */
function PreviousCycleRow({
  cycle,
  loc,
  t,
}: {
  cycle: CycleRecord;
  loc: Locale;
  t: ReturnType<typeof useTranslations>;
}) {
  const start = fromApiDate(cycle.period_start_date);
  const end = cycle.period_end_date ? fromApiDate(cycle.period_end_date) : null;
  const dates = end
    ? t('dateRange', { from: formatJalaliDayMonth(start, loc), to: formatJalali(end, loc) })
    : formatJalali(start, loc);

  return (
    <li
      style={{
        display: 'flex',
        alignItems: 'center',
        gap: 10,
        background: 'var(--surface-3)',
        border: '1px solid var(--line)',
        borderRadius: 10,
        padding: '9px 10px',
      }}
    >
      <span style={{ width: 6, height: 6, borderRadius: '50%', background: 'var(--brand)', flex: '0 0 auto' }} />
      <div style={{ flex: 1, minWidth: 0, textAlign: 'start' }}>
        <div style={{ fontSize: 12.5, fontWeight: 700, color: 'var(--ink)' }}>{dates}</div>
        <div style={{ fontSize: 11, color: 'var(--muted)', marginTop: 2 }}>
          {cycle.period_length !== null
            ? t('cycles.periodDays', { n: cycle.period_length })
            : t('cycles.periodUnknown')}
        </div>
      </div>
      <Chip>
        {cycle.cycle_length !== null ? t('days', { n: cycle.cycle_length }) : t('unavailable')}
      </Chip>
    </li>
  );
}

function Chip({ children, tone }: { children: React.ReactNode; tone?: 'brand' }) {
  const brand = tone === 'brand';
  return (
    <span
      style={{
        flex: '0 0 auto',
        fontSize: 11,
        fontWeight: 700,
        borderRadius: 6,
        padding: '3px 8px',
        whiteSpace: 'nowrap',
        fontVariantNumeric: 'tabular-nums',
        color: brand ? 'var(--brand-strong)' : 'var(--teal-deep)',
        background: brand ? 'var(--pink-bg)' : 'var(--teal-tint)',
      }}
    >
      {children}
    </span>
  );
}

function Stat({ label, value }: { label: string; value: string }) {
  return (
    <span
      style={{
        display: 'inline-flex',
        alignItems: 'baseline',
        gap: 6,
        background: 'var(--surface-3)',
        borderRadius: 8,
        padding: '6px 10px',
        fontSize: 11.5,
      }}
    >
      <span style={{ color: 'var(--muted)', fontWeight: 600 }}>{label}</span>
      <span style={{ color: 'var(--ink)', fontWeight: 800, fontVariantNumeric: 'tabular-nums' }}>{value}</span>
    </span>
  );
}

/** Placeholder with the same rhythm as the real card, so the page doesn't jump. */
function MyCyclesSkeleton({ title }: { title: string }) {
  return (
    <div style={{ padding: '16px 16px 0' }}>
      <div className="card" style={{ padding: '16px 14px' }}>
        <SectionHead title={title} />
        <div aria-hidden style={{ display: 'grid', gap: 10 }}>
          <span className="skeleton-line" style={{ height: 40, borderRadius: 12 }} />
          <span className="skeleton-line" style={{ height: 44, borderRadius: 10 }} />
          <span className="skeleton-line" style={{ height: 44, borderRadius: 10, width: '92%' }} />
        </div>
      </div>
    </div>
  );
}
