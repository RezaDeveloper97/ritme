'use client';

import clsx from 'clsx';

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
    <div className="sec">
      <div className="card pad-card">
        <SectionHead title={t('cycles.title')} />

        {/* Current cycle — day number, when it began, and how the bleed went. */}
        <div className="myc-current">
          <span className="dot myc-dot">
            <DropSolid size={18} color="var(--brand)" />
          </span>
          <div className="myc-current-b">
            <div className="myc-current-t">
              {t('cycles.current')}: {t('cycles.dayN', { n: current.cycle_day })}
            </div>
            <div className="myc-current-s">
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
          <div className="myc-stats">
            {averages.cycle_length !== null && (
              <Stat label={t('cycles.avgCycle')} value={t('days', { n: averages.cycle_length })} />
            )}
            {averages.period_length !== null && (
              <Stat label={t('cycles.avgPeriod')} value={t('days', { n: averages.period_length })} />
            )}
          </div>
        )}

        <div className="myc-rule" />

        {/* Previous cycles */}
        {previousCount === 0 ? (
          <p className="myc-none">
            {t('cycles.none')}
          </p>
        ) : (
          <>
            <div className="myc-prev-head">
              <span className="myc-prev-t">
                {t('cycles.previousTitle')}
              </span>
              <span className="myc-prev-count">
                {t('cycles.recorded', { n: previousCount })}
              </span>
            </div>
            <ul className="myc-list">
              {visible.map(cycle => (
                <PreviousCycleRow key={cycle.id} cycle={cycle} loc={loc} t={t} />
              ))}
            </ul>
            {hasMore && (
              <button
                type="button"
                className="myc-more"
                onClick={() => setExpanded(v => !v)}
              >
                {expanded ? t('cycles.showLess') : t('cycles.showAll', { n: previousCount })}
              </button>
            )}
          </>
        )}

        {/* Figma Frame 21: divider + pink inline action with add-circle icon */}
        <div className="myc-rule" />
        <button
          type="button"
          className="myc-add"
          onClick={() => setEditorOpen(true)}
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
    <li className="myc-row">
      <span className="myc-row-dot" />
      <div className="myc-row-b">
        <div className="myc-row-t">{dates}</div>
        <div className="myc-row-s">
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
    <span className={clsx('myc-chip', brand && 'is-brand')}>
      {children}
    </span>
  );
}

function Stat({ label, value }: { label: string; value: string }) {
  return (
    <span className="myc-stat">
      <span className="myc-stat-l">{label}</span>
      <span className="myc-stat-v">{value}</span>
    </span>
  );
}

/** Placeholder with the same rhythm as the real card, so the page doesn't jump. */
function MyCyclesSkeleton({ title }: { title: string }) {
  return (
    <div className="sec">
      <div className="card pad-card">
        <SectionHead title={title} />
        <div aria-hidden className="myc-skel">
          <span className="skeleton-line myc-skel-a" />
          <span className="skeleton-line myc-skel-b" />
          <span className="skeleton-line myc-skel-c" />
        </div>
      </div>
    </div>
  );
}
