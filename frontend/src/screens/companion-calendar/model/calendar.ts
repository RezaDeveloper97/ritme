import type { CompanionCalendarPartner } from '@/entities/companion';
import { type Locale } from '@/shared/i18n';
import { monthMatrix, toApiDate } from '@/shared/lib/date';

/** One of his own appointments as the calendar needs it. */
export interface DayAppointment {
  id: number;
  title: string;
  /** Tehran wall-clock `Y-m-d H:i:s`; null = undated (never on the calendar). */
  scheduledAt: string | null;
  status: string;
}

/** First and last day of a locale-calendar month as `Y-m-d` (the API window, ≤ 31 days). */
export function monthWindow(year: number, month: number, locale: Locale): { from: string; to: string } {
  const cells = monthMatrix(year, month, locale).flat().filter((c) => c !== null);
  return { from: toApiDate(cells[0]!.date), to: toApiDate(cells[cells.length - 1]!.date) };
}

/** Scheduled (not cancelled) appointments grouped by their `Y-m-d`, each day sorted by time. */
export function appointmentsByDay<T extends DayAppointment>(items: readonly T[]): Map<string, T[]> {
  const out = new Map<string, T[]>();
  for (const a of items) {
    if (a.status === 'cancelled' || !a.scheduledAt || !/^\d{4}-\d{2}-\d{2}/.test(a.scheduledAt)) continue;
    const day = a.scheduledAt.slice(0, 10);
    const list = out.get(day) ?? [];
    list.push(a);
    out.set(day, list);
  }
  for (const list of out.values()) list.sort((x, y) => (x.scheduledAt ?? '').localeCompare(y.scheduledAt ?? ''));
  return out;
}

/** The partner shown: the picked link, else the first one (null without partners). */
export function pickPartner(
  partners: readonly CompanionCalendarPartner[],
  linkId: number | null,
): CompanionCalendarPartner | null {
  return partners.find((p) => p.linkId === linkId) ?? partners[0] ?? null;
}

/** Her likely period days as a set (empty when she does not share them). */
export function periodDaySet(partner: CompanionCalendarPartner | null): ReadonlySet<string> {
  return new Set(partner?.periodDays ?? []);
}
