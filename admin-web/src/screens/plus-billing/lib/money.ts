/**
 * Money and rates of the Plus admin. The API stores integer rials (admin-api.md §15) and VAT in basis points;
 * the panel shows and takes toman (= rials / 10, like the app's price cards) and VAT as a percent.
 */

/** Rials → toman (whole; rials are always multiples of 10 for admin-entered prices). */
export function rialsToToman(rials: number): number {
  return Math.floor(rials / 10);
}

/** Toman input text (ASCII digits) → rials, or null when empty / not a whole number ≥ 0. */
export function tomanToRials(text: string): number | null {
  const trimmed = text.trim();
  if (!/^\d+$/.test(trimmed)) return null;
  return Number(trimmed) * 10;
}

/** Basis points → percent text (`1000` → `"10"`, `950` → `"9.5"`). */
export function bpsToPercentText(bps: number): string {
  return String(bps / 100);
}

/** Percent input text → basis points (rounded), or null when empty / invalid / out of 0–100. */
export function percentToBps(text: string): number | null {
  const trimmed = text.trim();
  if (trimmed === '' || !/^\d+(\.\d+)?$/.test(trimmed)) return null;
  const bps = Math.round(Number(trimmed) * 100);
  return bps >= 0 && bps <= 10000 ? bps : null;
}
