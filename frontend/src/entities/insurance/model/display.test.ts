import { describe, expect, it } from 'vitest';

import { compactAmount, homeCoverages, remainingOf } from './display';
import type { InsurancePolicy, PolicyCoverage } from './types';

const cov = (category: string, cap: number | null, used = 0, status: PolicyCoverage['status'] = 'covered'): PolicyCoverage => ({
  category,
  status,
  capAmount: cap,
  deductiblePct: null,
  waitingDays: null,
  note: null,
  used,
  remaining: null,
  waitingEndsOn: null,
  waitingOver: false,
});

describe('insurance display', () => {
  it('compacts toman like the boards', () => {
    expect(compactAmount(1_800_000)).toEqual({ unit: 'million', value: 1.8 });
    expect(compactAmount(950_000)).toEqual({ unit: 'thousand', value: 950 });
    expect(compactAmount(420)).toEqual({ unit: 'toman', value: 420 });
  });

  it('computes what is left and never goes negative', () => {
    expect(remainingOf(cov('delivery', 40_000_000, 6_000_000))).toBe(34_000_000);
    expect(remainingOf(cov('delivery', 1_000_000, 3_000_000))).toBe(0);
    expect(remainingOf({ ...cov('delivery', 5), remaining: 2 })).toBe(2);
  });

  it('shows at most three capped coverages in board order', () => {
    const policy = {
      coverages: [cov('dental', 10), cov('inpatient', 300), cov('optical', 4, 0, 'not_covered'), cov('paraclinic', 20), cov('delivery', 40), cov('visit_drug', 8)],
    } as InsurancePolicy;
    expect(homeCoverages(policy).map((c) => c.category)).toEqual(['delivery', 'paraclinic', 'inpatient']);
  });
});
