import { describe, expect, it } from 'vitest';

import { notADiagnosis, programCards } from './cards';

describe('programCards', () => {
  const overview = {
    programs: [
      { code: 'endo' as const, enrolled: true, enrolledOn: '2026-09-01' },
      { code: 'pcos' as const, enrolled: false, enrolledOn: null },
    ],
  };

  it('keeps the API order and joins the catalog copy by code', () => {
    const cards = programCards(overview, [
      { code: 'pcos', title: 'PCOS', body: 'Cycle regularity', logs: 'acne' },
      { code: 'endo', title: 'Endo', body: null, logs: null },
    ]);
    expect(cards.map((c) => c.code)).toEqual(['endo', 'pcos']);
    expect(cards[0]).toMatchObject({ enrolled: true, enrolledOn: '2026-09-01', title: 'Endo', sub: null });
    expect(cards[1]).toMatchObject({ title: 'PCOS', sub: 'Cycle regularity', logs: 'acne' });
  });

  it('falls back to bundled copy (nulls) without a catalog', () => {
    expect(programCards(overview, undefined)[1]).toMatchObject({ title: null, sub: null, logs: null });
  });
});

describe('notADiagnosis', () => {
  it('reads the alert row, null without one', () => {
    expect(notADiagnosis([{ code: 'not_a_diagnosis', title: 'T', body: 'B', logs: null }])).toEqual({ title: 'T', body: 'B' });
    expect(notADiagnosis([{ code: 'pain_scale', title: 'T', body: null, logs: null }])).toBeNull();
    expect(notADiagnosis(undefined)).toBeNull();
  });
});
