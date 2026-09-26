import type { CheckupItem, CheckupSummary } from '@/entities/checkup';

export type CountPart = 'countUpToDate' | 'countDue' | 'countOverdue';

/** Non-zero parts of «N مورد به‌روز · N موعدش رسیده · N عقب‌افتاده», in order. */
export function countParts(summary: CheckupSummary): { key: CountPart; count: number }[] {
  const parts: { key: CountPart; count: number }[] = [
    { key: 'countUpToDate', count: summary.upToDate },
    { key: 'countDue', count: summary.due },
    { key: 'countOverdue', count: summary.overdue },
  ];
  return parts.filter((p) => p.count > 0);
}

/** Joins the translated parts with the separator. */
export function countsLine(
  summary: CheckupSummary,
  label: (key: CountPart, count: number) => string,
  separator: string,
): string {
  return countParts(summary)
    .map((p) => label(p.key, p.count))
    .join(separator);
}

export type HighlightKind = 'guide' | 'book';

/** Overdue → amber «ثبت نوبت»; cycle-timed / due → rose «راهنما». */
export function highlightKind(item: CheckupItem): HighlightKind {
  return item.status === 'overdue' && item.timingLabel == null ? 'book' : 'guide';
}

/** Self-exam types open the step-by-step guide; everything else the detail page. */
export function guideHref(item: CheckupItem): string {
  return item.key?.includes('self_exam') ? '/checkups/self-exam' : `/checkups/${item.id}`;
}

/** AddAppointment with the checkup as the prefilled title. */
export function bookHref(item: CheckupItem): string {
  const q = new URLSearchParams({ kind: 'in_person', title: item.title });
  return `/reminders/appointment/new?${q.toString()}`;
}

/** Ring arc length fraction, clamped to [0, 1]. */
export function ringFraction(summary: CheckupSummary): number {
  if (summary.total <= 0) return 0;
  return Math.min(1, Math.max(0, summary.upToDate / summary.total));
}
