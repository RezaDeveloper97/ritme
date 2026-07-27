'use client';

import { useTranslations } from 'next-intl';

import { useCycleSummarySection, type CycleSummaryItem } from '@/entities/cycle';
import { Link } from '@/shared/i18n';

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
  const t = useTranslations('home');
  const { data: section, isLoading } = useCycleSummarySection();

  if (isLoading) return <CycleSummarySkeleton title={t('cycleSummary.title')} />;
  if (!section) return null;

  const { items, has_history: hasHistory } = section.data;

  return (
    <div style={{ padding: '16px 16px 0' }}>
      <div style={{ background: 'var(--surface)', borderRadius: 12, padding: '16px 14px' }}>
        <SectionHead title={section.title ?? t('cycleSummary.title')} />

        {/* Figma Frame 41: rows live inside a bordered box */}
        <div style={{ border: '1px solid var(--line)', borderRadius: 12, padding: '2px 14px' }}>
          {items.map((item, i) => (
            <SummaryRow key={item.key} item={item} t={t} last={i === items.length - 1} />
          ))}
        </div>

        {!hasHistory && (
          <p style={{ margin: '12px 2px 0', fontSize: 11.5, color: 'var(--muted)', textAlign: 'start', lineHeight: 1.7 }}>
            {t('cycleSummary.empty')}
          </p>
        )}

        <div style={{ paddingTop: 14 }}>
          <Link href="/cycle" className="btn btn-primary" style={{ height: 40, textDecoration: 'none' }}>
            {section.action?.label ?? t('cycleSummary.viewMore')}
          </Link>
        </div>
      </div>
    </div>
  );
}

function SummaryRow({
  item,
  t,
  last,
}: {
  item: CycleSummaryItem;
  t: ReturnType<typeof useTranslations>;
  last: boolean;
}) {
  const tone = STATUS_TONE[item.status];

  return (
    <div
      style={{
        display: 'flex',
        alignItems: 'flex-start',
        justifyContent: 'space-between',
        gap: 12,
        padding: '12px 2px',
        ...(last ? {} : { borderBottom: '1px solid var(--line)' }),
      }}
    >
      <div style={{ textAlign: 'start', minWidth: 0 }}>
        <div style={{ fontSize: 14, color: 'var(--muted-2)', fontWeight: 700 }}>{item.label}</div>
        {item.hint && (
          <div style={{ fontSize: 11, color: 'var(--muted)', marginTop: 4, lineHeight: 1.6 }}>{item.hint}</div>
        )}
      </div>
      <div style={{ display: 'flex', flexDirection: 'column', alignItems: 'flex-end', gap: 4, flex: '0 0 auto' }}>
        <span style={{ fontSize: 13.5, fontWeight: 700, color: 'var(--ink)', fontVariantNumeric: 'tabular-nums', whiteSpace: 'nowrap' }}>
          {formatValue(item, t)}
        </span>
        <span style={{ fontSize: 10.5, fontWeight: 700, color: tone.fg, background: tone.bg, borderRadius: 5, padding: '2px 7px', whiteSpace: 'nowrap' }}>
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
    <div style={{ padding: '16px 16px 0' }}>
      <div style={{ background: 'var(--surface)', borderRadius: 12, padding: '16px 14px' }}>
        <SectionHead title={title} />
        <div aria-hidden style={{ display: 'grid', gap: 10, border: '1px solid var(--line)', borderRadius: 12, padding: 14 }}>
          {['100%', '96%', '88%'].map(width => (
            <span key={width} className="skeleton-line" style={{ width, height: 20 }} />
          ))}
        </div>
      </div>
    </div>
  );
}
