import { describe, expect, it } from 'vitest';

import type { InsuranceClaimRow } from '@/entities/insurance';

import { filterClaims, groupByMonth, matchesFilter } from './claims';

const row = (id: number, serviceDate: string, status: InsuranceClaimRow['status'], title = 'x', memberName: string | null = null): InsuranceClaimRow => ({
  id,
  title,
  costType: 'visit',
  memberName,
  serviceDate,
  amount: 1000,
  paidAmount: null,
  status,
  needsAction: status === 'docs_requested',
});

describe('claims list model', () => {
  it('open covers everything that is still moving', () => {
    expect(matchesFilter('docs_requested', 'open')).toBe(true);
    expect(matchesFilter('appealed', 'open')).toBe(true);
    expect(matchesFilter('paid', 'open')).toBe(false);
    expect(matchesFilter('rejected', 'rejected')).toBe(true);
    expect(matchesFilter('paid', 'all')).toBe(true);
  });

  it('searches the title and the person', () => {
    const rows = [row(1, '2026-10-01', 'paid', 'Ultrasound', 'Sara'), row(2, '2026-10-02', 'paid', 'Glasses', 'Ali')];
    const titleOf = (c: InsuranceClaimRow) => c.title ?? '';
    expect(filterClaims(rows, 'all', 'ultra', titleOf).map((c) => c.id)).toEqual([1]);
    expect(filterClaims(rows, 'all', 'ali', titleOf).map((c) => c.id)).toEqual([2]);
    expect(filterClaims(rows, 'paid', '', titleOf)).toHaveLength(2);
  });

  it('groups by month, newest first', () => {
    const rows = [row(1, '2026-08-05', 'paid'), row(2, '2026-10-01', 'paid'), row(3, '2026-10-20', 'paid')];
    const months = groupByMonth(rows, 'en');
    expect(months.map((m) => m.month)).toEqual([10, 8]);
    expect(months[0].claims.map((c) => c.id)).toEqual([3, 2]);
  });
});
