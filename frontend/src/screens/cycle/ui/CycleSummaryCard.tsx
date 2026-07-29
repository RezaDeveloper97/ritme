'use client';

import { useTranslations } from 'next-intl';

import { useCycleSummarySection, type CycleSummaryItem } from '@/entities/cycle';

import { SectionHead } from './SectionHead';

/** Row accent per how the value reads against the range typical for most people. */
const STATUS_TONE: Record<CycleSummaryItem['status'], { fg: string; bg: string }> = {
  normal: { fg: 'var(--green-deep)', bg: 'var(--green-soft)' },
  outside_range: { fg: 'var(--amber-deep)', bg: 'var(--amber-soft)' },
  unknown: { fg: 'var(--muted-2)', bg: 'var(--surface-3)' },
};

/**
 * «خلاصه سیکل» — the user's own cycle numbers, each read against the usual
 * range.
 *
 * The rows are entirely server-driven (`/home/sections/cycle_summary`): the
 * backend decides which ones it has the data to show and words each verdict, so
 * the card grows from "not known yet" to averages and spread as periods are
 * logged. Statuses are descriptive, never diagnostic (§11).
 */
export function CycleSummaryCard() {
  const t = useTranslations('cycle');
  const { data: section, isLoading } = useCycleSummarySection();

  if (isLoading) return <CycleSummarySkeleton title={t('cycleSummary.title')} />;
  if (!section) return null;

  const { items, has_history: hasHistory } = section.data;

  return (
    <div className="sec">
      <div className="csum-card">
        <SectionHead title={section.title ?? t('cycleSummary.title')} />

        {/* Figma Frame 41: rows live inside a bordered box */}
        <div className="csum-list">
          {items.map((item) => (
            <SummaryRow key={item.key} item={item} t={t} />
          ))}
        </div>

        {!hasHistory && (
          <p className="csum-note">
            {t('cycleSummary.empty')}
          </p>
        )}
      </div>
    </div>
  );
}

function SummaryRow({
  item,
  t,
}: {
  item: CycleSummaryItem;
  t: ReturnType<typeof useTranslations>;
}) {
  const tone = STATUS_TONE[item.status];

  return (
    <div className="csum-row">
      <div className="csum-row-l">
        <div className="csum-row-lbl">{item.label}</div>
        {item.hint && (
          <div className="csum-row-sub">{item.hint}</div>
        )}
      </div>
      <div className="csum-row-r">
        <span className="csum-row-val">
          {formatValue(item, t)}
        </span>
        <span className="csum-row-tag" style={{ color: tone.fg, background: tone.bg }}>
          {item.status_label}
        </span>
      </div>
    </div>
  );
}

/**
 * Render a row's value. Numbers cross the API as raw integers so digits are
 * localized here (§7); rows that aren't numeric carry pre-worded `text`.
 */
function formatValue(item: CycleSummaryItem, t: ReturnType<typeof useTranslations>): string {
  if (item.value_min !== null && item.value_max !== null) {
    return t('cycleSummary.dayRange', { from: item.value_min, to: item.value_max });
  }
  if (item.text !== null) return item.text;
  if (item.value === null) return t('unavailable');
  return item.unit === 'days' ? t('days', { n: item.value }) : String(item.value);
}

/** Placeholder rows while the summary loads — same height as the real ones. */
function CycleSummarySkeleton({ title }: { title: string }) {
  return (
    <div className="sec">
      <div className="csum-card">
        <SectionHead title={title} />
        <div aria-hidden className="csum-skel">
          <span className="skeleton-line csum-skel-a" />
          <span className="skeleton-line csum-skel-b" />
          <span className="skeleton-line csum-skel-c" />
        </div>
      </div>
    </div>
  );
}
