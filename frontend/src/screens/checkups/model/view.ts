import {
  CHECKUP_LIST_FILTERS,
  type CheckupItem,
  type CheckupListFilter,
  type CheckupSection,
  type CheckupSummary,
} from '@/entities/checkup';

/*
 * Pure view-state for the Checkups screen (v14_Checkups) — no React, no locale.
 */

/** `?filter=` → a tab; anything unknown (or absent) is «همه». */
export function parseFilter(value: string | null | undefined): CheckupListFilter {
  return CHECKUP_LIST_FILTERS.includes(value as CheckupListFilter)
    ? (value as CheckupListFilter)
    : 'all';
}

/** The URL for a tab — «همه» is the bare route. */
export function filterHref(filter: CheckupListFilter): string {
  return filter === 'all' ? '/checkups' : `/checkups?filter=${filter}`;
}

/** Arrow-key movement across the tablist (wraps; RTL flips the arrows). */
export function nextFilter(
  current: CheckupListFilter,
  key: string,
  dir: 'rtl' | 'ltr',
): CheckupListFilter | null {
  const tabs = CHECKUP_LIST_FILTERS;
  const i = tabs.indexOf(current);
  const last = tabs.length - 1;
  const forward = dir === 'rtl' ? 'ArrowLeft' : 'ArrowRight';
  const backward = dir === 'rtl' ? 'ArrowRight' : 'ArrowLeft';
  if (key === forward) return tabs[i === last ? 0 : i + 1];
  if (key === backward) return tabs[i === 0 ? last : i - 1];
  if (key === 'Home') return tabs[0];
  if (key === 'End') return tabs[last];
  return null;
}

export interface SectionGroup {
  section: CheckupSection;
  items: CheckupItem[];
}

/** Sections pinned to the top of the list, in this order (v14_Checkups: این ماه → عقب‌افتاده). */
const PINNED_SECTIONS: readonly CheckupSection[] = ['this_month', 'overdue'];

/**
 * Items grouped by `section`. «این ماه» and «عقب‌افتاده» always come first (in
 * that order), whatever order the server sends; the category sections follow in
 * the order they first appear (the admin's `sort_order`). Items keep their order.
 */
export function groupBySection(items: readonly CheckupItem[]): SectionGroup[] {
  const groups = new Map<CheckupSection, CheckupItem[]>();
  for (const item of items) {
    const list = groups.get(item.section);
    if (list) list.push(item);
    else groups.set(item.section, [item]);
  }
  const rank = (section: CheckupSection) => {
    const i = PINNED_SECTIONS.indexOf(section);
    return i === -1 ? PINNED_SECTIONS.length : i;
  };
  // Array.prototype.sort is stable, so unpinned sections keep first-seen order.
  return [...groups]
    .map(([section, list]) => ({ section, items: list }))
    .sort((a, b) => rank(a.section) - rank(b.section));
}

export type SummaryStatus = 'overdue' | 'due' | 'up_to_date';

/** The status pill of the summary card: the worst state present. */
export function worstStatus(summary: CheckupSummary): SummaryStatus {
  if (summary.overdue > 0) return 'overdue';
  if (summary.due > 0) return 'due';
  return 'up_to_date';
}

export interface BarSegment {
  status: SummaryStatus;
  count: number;
  /** Width share in percent (0–100). */
  percent: number;
}

/** Stacked-bar segments (green → amber → rose, always LTR); empty ones dropped. */
export function barSegments(summary: CheckupSummary): BarSegment[] {
  const parts: [SummaryStatus, number][] = [
    ['up_to_date', summary.upToDate],
    ['due', summary.due],
    ['overdue', summary.overdue],
  ];
  const total = parts.reduce((sum, [, n]) => sum + Math.max(0, n), 0);
  if (total === 0) return [];
  return parts
    .filter(([, n]) => n > 0)
    .map(([status, count]) => ({ status, count, percent: (count / total) * 100 }));
}
