import type { ClaimStatus, InsuranceClaimRow } from '@/entities/insurance';
import type { Locale } from '@/shared/i18n';
import { fromApiDate, toParts } from '@/shared/lib/date';

/** Status chips of nbl_Ins_Claims; `open` = anything still moving (not paid, not rejected). */
export const CLAIM_FILTERS = ['all', 'open', 'paid', 'rejected'] as const;
export type ClaimFilter = (typeof CLAIM_FILTERS)[number];

const OPEN: readonly ClaimStatus[] = ['submitted', 'reviewing', 'docs_requested', 'appealed'];

export function matchesFilter(status: ClaimStatus, filter: ClaimFilter): boolean {
  if (filter === 'all') return true;
  if (filter === 'open') return OPEN.includes(status);
  return status === filter;
}

/** Client-side search over the title, the person and the cost type's label (the list is one policy period). */
export function filterClaims(
  claims: readonly InsuranceClaimRow[],
  filter: ClaimFilter,
  query: string,
  titleOf: (c: InsuranceClaimRow) => string,
): InsuranceClaimRow[] {
  const q = query.trim().toLowerCase();
  return claims.filter(
    (c) => matchesFilter(c.status, filter) && (q === '' || `${titleOf(c)} ${c.memberName ?? ''}`.toLowerCase().includes(q)),
  );
}

export interface ClaimMonth {
  key: string;
  year: number;
  month: number;
  claims: InsuranceClaimRow[];
}

/** Newest month first, each month newest day first; months follow the locale's calendar. */
export function groupByMonth(claims: readonly InsuranceClaimRow[], locale: Locale): ClaimMonth[] {
  const sorted = [...claims].sort((a, b) => (a.serviceDate < b.serviceDate ? 1 : a.serviceDate > b.serviceDate ? -1 : b.id - a.id));
  const months: ClaimMonth[] = [];
  for (const c of sorted) {
    const { year, month } = toParts(fromApiDate(c.serviceDate), locale);
    const key = `${year}-${month}`;
    const last = months[months.length - 1];
    if (last && last.key === key) last.claims.push(c);
    else months.push({ key, year, month, claims: [c] });
  }
  return months;
}
