/**
 * One cell of the week grid: a week has content when it has a v1 text row (`textId`) or
 * structured v2 details (`hasDetails`, weeks 1–42). QA 2026-09-29-c L7: the grid used to read
 * only the v1 rows, so weeks with details alone looked empty and opened the create page.
 */
export interface WeekCell {
  week: number;
  textId: number | null;
  hasDetails: boolean;
}

export function weekCells(
  texts: readonly { week: number; id: number | null }[],
  details: readonly { week_number: number; exists: boolean }[] | undefined,
): WeekCell[] {
  const cells = new Map<number, WeekCell>();
  for (const t of texts) cells.set(t.week, { week: t.week, textId: t.id, hasDetails: false });
  for (const d of details ?? []) {
    if (!d.exists) continue;
    const cell = cells.get(d.week_number) ?? { week: d.week_number, textId: null, hasDetails: false };
    cells.set(d.week_number, { ...cell, hasDetails: true });
  }
  return [...cells.values()].sort((a, b) => a.week - b.week);
}

export const cellFilled = (c: WeekCell) => c.textId !== null || c.hasDetails;

/** The text row when there is one; else the week's details tab (weeks are keyed by number there). */
export function cellHref(c: WeekCell): string {
  if (c.textId !== null) return `/pregnancy-weeks/${c.textId}`;
  return c.hasDetails ? detailsHref(c.week) : `/pregnancy-weeks/new?week=${c.week}`;
}

export const detailsHref = (week: number) => `/pregnancy-weeks/new?week=${week}&tab=details`;

/**
 * `/pregnancy-weeks/:id` takes a text row id. When no such row exists but the number is a week
 * with structured details, the page opens those details instead of «not found».
 */
export function detailsFallback(
  id: number,
  notFound: boolean,
  details: readonly { week_number: number; exists: boolean }[] | undefined,
): string | null {
  if (!notFound) return null;
  return details?.some((d) => d.week_number === id && d.exists) ? detailsHref(id) : null;
}
