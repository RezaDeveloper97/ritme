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

const isSelfExam = (item: Pick<CheckupItem, 'key'>) => item.key?.includes('self_exam') ?? false;

/**
 * Overdue → amber «ثبت نوبت» (a visit to book), whatever its cycle timing;
 * the self-exam and anything merely due → rose «راهنما».
 */
export function highlightKind(item: CheckupItem): HighlightKind {
  return item.status === 'overdue' && !isSelfExam(item) ? 'book' : 'guide';
}

/**
 * The row's sub-line: the *when* first — «عقب‌افتاده از فروردین» alone for an
 * overdue row, else «۳ روز دیگر، روز ۷ تا ۱۰ سیکل».
 */
export function highlightMeta(item: CheckupItem, separator: string): string | null {
  const parts =
    item.status === 'overdue' ? [item.nextDueLabel] : [item.nextDueLabel, item.timingLabel];
  const line = parts.filter(Boolean).join(separator);
  return line || item.timingLabel || item.subtitle;
}

/** Self-exam types open the step-by-step guide; everything else the detail page. */
export function guideHref(item: CheckupItem): string {
  return isSelfExam(item) ? '/checkups/self-exam' : `/checkups/${item.id}`;
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
