import type { InsurancePolicy, PolicyCoverage } from './types';

export const TOMAN_THOUSAND = 1_000;
export const TOMAN_MILLION = 1_000_000;

/** Whole toman → amount in millions rounded to one decimal (`1_800_000` → 1.8). */
export function inMillions(toman: number): number {
  return Math.round((toman / TOMAN_MILLION) * 10) / 10;
}

export type CompactAmount =
  | { unit: 'million'; value: number }
  | { unit: 'thousand'; value: number }
  | { unit: 'toman'; value: number };

/** Board style: «۱٫۸ میلیون», «۹۵۰ هزار»; below a thousand the plain amount. */
export function compactAmount(toman: number): CompactAmount {
  if (toman >= TOMAN_MILLION) return { unit: 'million', value: inMillions(toman) };
  if (toman >= TOMAN_THOUSAND) return { unit: 'thousand', value: Math.round(toman / TOMAN_THOUSAND) };
  return { unit: 'toman', value: toman };
}

/** Coverages that carry a cap (the usage bars); the rest are notes. */
export function capped(cov: PolicyCoverage): boolean {
  return cov.status !== 'not_covered' && cov.capAmount !== null && cov.capAmount > 0;
}

/** Remaining toman for a cap, never negative. */
export function remainingOf(cov: PolicyCoverage): number {
  if (cov.remaining !== null) return Math.max(0, cov.remaining);
  return Math.max(0, (cov.capAmount ?? 0) - cov.used);
}

const HOME_ORDER = ['delivery', 'paraclinic', 'inpatient', 'visit_drug'];

/** The up-to-three bars the home card shows, board order first. */
export function homeCoverages(policy: InsurancePolicy): PolicyCoverage[] {
  const rank = (c: string) => {
    const i = HOME_ORDER.indexOf(c);
    return i === -1 ? HOME_ORDER.length : i;
  };
  return policy.coverages
    .filter(capped)
    .sort((a, b) => rank(a.category) - rank(b.category))
    .slice(0, 3);
}

/** Coverages with a deductible, for the tiles on the coverage screen. */
export function deductibleTiles(policy: InsurancePolicy): { category: string; pct: number }[] {
  return policy.coverages
    .filter((c) => c.status !== 'not_covered' && c.deductiblePct !== null && c.deductiblePct > 0)
    .map((c) => ({ category: c.category, pct: c.deductiblePct as number }));
}

export type CategoryCode = 'inpatient' | 'delivery' | 'paraclinic' | 'visit_drug' | 'infertility' | 'dental' | 'optical' | 'other';
const KNOWN: readonly string[] = ['inpatient', 'delivery', 'paraclinic', 'visit_drug', 'infertility', 'dental', 'optical'];

/** A category code the copy knows, else `other`. */
export function categoryKey(category: string): CategoryCode {
  return KNOWN.includes(category) ? (category as CategoryCode) : 'other';
}

export type CostTypeCode = 'paraclinic' | 'visit' | 'drug' | 'inpatient' | 'delivery' | 'dental' | 'optical' | 'infertility' | 'other';
const COSTS: readonly string[] = ['paraclinic', 'visit', 'drug', 'inpatient', 'delivery', 'dental', 'optical', 'infertility'];

/** A claim cost type the copy knows, else `other`. */
export function costTypeKey(costType: string): CostTypeCode {
  return COSTS.includes(costType) ? (costType as CostTypeCode) : 'other';
}

/** Last four characters of a number as a masked tail («•••• 4821»). */
export function maskedTail(last4: string | null): string | null {
  return last4 ? `•••• ${last4}` : null;
}
