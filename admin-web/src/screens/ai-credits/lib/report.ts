/** Pure helpers of the «مصرف هوش مصنوعی» report. */

export interface DayPoint {
  day: string; // YYYY-MM-DD (Tehran)
  calls: number;
  costUsd: number;
  credits: number;
}

/** Every day of [from, to] (inclusive, YYYY-MM-DD), with the provider cost and the credits spent (0 when absent). */
export function mergeDays(
  from: string,
  to: string,
  cost: readonly { day: string; calls: number; cost_usd: number }[],
  credits: readonly { day: string; credits: number }[],
): DayPoint[] {
  const costBy = new Map(cost.map((d) => [d.day, d]));
  const creditBy = new Map(credits.map((d) => [d.day, d.credits]));
  const out: DayPoint[] = [];
  const end = Date.parse(`${to}T00:00:00Z`);
  for (let t = Date.parse(`${from}T00:00:00Z`); Number.isFinite(t) && t <= end && out.length < 400; t += 86_400_000) {
    const day = new Date(t).toISOString().slice(0, 10);
    const c = costBy.get(day);
    out.push({ day, calls: c?.calls ?? 0, costUsd: c?.cost_usd ?? 0, credits: creditBy.get(day) ?? 0 });
  }
  return out;
}

/** USD → toman with the admin's rate (IRR per USD; 1 toman = 10 rials); null without a rate. */
export function usdToToman(usd: number, irrPerUsd: number | null): number | null {
  if (!irrPerUsd || irrPerUsd <= 0) return null;
  return Math.round((usd * irrPerUsd) / 10);
}

/** A model price in milli-USD per 1M tokens → USD per 1M tokens. */
export function milliToUsd(milli: number): number {
  return milli / 1000;
}

/** USD per 1M tokens (input text, ASCII digits, up to 3 decimals) → milli-USD, or null when invalid. */
export function usdToMilli(text: string): number | null {
  const trimmed = text.trim();
  if (trimmed === '') return 0;
  if (!/^\d+(\.\d{1,3})?$/.test(trimmed)) return null;
  return Math.round(Number(trimmed) * 1000);
}

/** The last `days` days up to `today` (YYYY-MM-DD) as {from, to}. */
export function lastDays(today: string, days: number): { from: string; to: string } {
  const t = Date.parse(`${today}T00:00:00Z`) - (days - 1) * 86_400_000;
  return { from: new Date(t).toISOString().slice(0, 10), to: today };
}
