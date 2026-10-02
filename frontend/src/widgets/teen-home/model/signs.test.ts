import { describe, expect, it } from 'vitest';

import type { TeenCatalogItem } from '@/entities/teen';

import { estimatePercent, signsCard } from './signs';

const item = (code: string, severity: string | null = null): TeenCatalogItem => ({
  code,
  title: `${code}-title`,
  body: `${code}-body`,
  severity,
});

describe('teen signs card', () => {
  it('leads with the first sign and closes with the estimate line', () => {
    const card = signsCard([item('approaching_signs'), item('growth_spurt')], item('estimate_coming_months'));
    expect(card).toMatchObject({
      title: 'approaching_signs-title',
      estimate: 'estimate_coming_months-title',
      percent: 65,
      caution: false,
    });
    expect(card?.extra.map((s) => s.code)).toEqual(['growth_spurt']);
  });

  it('draws no bar for the «talk to someone» estimate', () => {
    const card = signsCard([item('approaching_signs')], item('estimate_talk', 'caution'));
    expect(card?.percent).toBeNull();
    expect(card?.caution).toBe(true);
  });

  it('uses the estimate as the whole card after the first period', () => {
    const card = signsCard([], item('estimate_settling'));
    expect(card).toMatchObject({ title: 'estimate_settling-title', body: 'estimate_settling-body', estimate: null });
  });

  it('has no card without signs or estimate, and no bar for unknown codes', () => {
    expect(signsCard([], null)).toBeNull();
    expect(estimatePercent(item('admin_added'))).toBeNull();
    expect(estimatePercent(null)).toBeNull();
  });
});
