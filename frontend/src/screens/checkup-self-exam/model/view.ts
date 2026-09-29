import type { CheckupItem, CheckupRecord } from '@/entities/checkup';

export const SELF_EXAM_KEY = 'breast_self_exam';

/**
 * «سه مرحله، حدود ۵ دقیقه» — the artboard and the seeded prep step («حدود ۵
 * دقیقه وقت آرام») both say five; never derived from the step count (audit E1).
 */
export const SELF_EXAM_MINUTES = 5;

export function findSelfExam(items: readonly CheckupItem[] | undefined): CheckupItem | null {
  return items?.find((i) => i.key === SELF_EXAM_KEY) ?? null;
}

const monthIndex = (ymd: string) => {
  const [y, m] = ymd.split('-').map(Number);
  return y * 12 + (m - 1);
};

/** Distinct months with a record among the last `window` months (incl. this one). */
export function adherence(
  records: readonly Pick<CheckupRecord, 'doneOn'>[],
  todayYmd: string,
  window = 12,
): number {
  const now = monthIndex(todayYmd);
  const months = new Set(
    records.map((r) => monthIndex(r.doneOn)).filter((m) => m <= now && m > now - window),
  );
  return months.size;
}

export function doneThisMonth(records: readonly Pick<CheckupRecord, 'doneOn'>[], todayYmd: string): boolean {
  return adherence(records, todayYmd, 1) > 0;
}

/**
 * When the best self-exam window (cycle days `from`…`to`) is, relative to
 * today's cycle day: `0` = inside it now, otherwise days until it next opens.
 */
export function daysUntilWindow(day: number, cycleLength: number, from: number, to: number): number {
  if (day >= from && day <= to) return 0;
  if (day < from) return from - day;
  return Math.max(1, cycleLength - day + from);
}
