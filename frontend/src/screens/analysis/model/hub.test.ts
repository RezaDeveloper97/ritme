import { describe, expect, it } from 'vitest';

import type { AnalysisSection, AnalysisSummary } from '@/entities/analysis';

import { asksForCycles, availableCategories, groupCards, hasNoData, hubVariant, signedDecimal, visibleCards } from './hub';

const open = <T,>(data: T, ready = true): AnalysisSection<T> => ({ plus: false, locked: false, ready, data });
const locked: AnalysisSection<never> = { plus: true, locked: true, ready: false, data: null };

const summary = (overrides: Partial<AnalysisSummary['sections']> = {}): AnalysisSummary => ({
  range: { key: '6m', from: '2026-04-01', to: '2026-09-30', days: 182 },
  topFinding: { kind: 'not_enough_data', parts: [], text: '', strength: null, score: null, evidenceDate: null, daysNeeded: 56 },
  sections: {
    cycle: open(null as never, false),
    recentCycles: open([], false),
    symptoms: open(null as never, false),
    moodByPhase: locked,
    sleepMood: locked,
    weight: open(null as never, false),
    vitals: open(null as never, false),
    labs: locked,
    ...overrides,
  },
});

describe('analysis hub model', () => {
  it('picks the hub by life-stage mode', () => {
    expect(hubVariant('teen')).toBe('teen');
    expect(hubVariant('menopause')).toBe('menopause');
    expect(hubVariant('pregnancy')).toBe('pregnancy');
    expect(hubVariant('ttc')).toBe('cycle');
    expect(hubVariant('postpartum')).toBe('cycle');
    expect(hubVariant(null)).toBe('cycle');
  });

  it('drops the empty period strip and, for teens, the locked Plus cards', () => {
    expect(visibleCards(summary(), 'cycle')).toEqual([
      'cycle', 'symptoms', 'moodByPhase', 'sleepMood', 'weight', 'vitals', 'labs',
    ]);
    expect(visibleCards(summary(), 'teen')).toEqual(['cycle', 'symptoms', 'weight', 'vitals']);
    expect(availableCategories(visibleCards(summary(), 'teen'))).toEqual(['all', 'cycle', 'symptoms', 'body']);
  });

  it('never asks a menopause user for cycles (B-N3-14b)', () => {
    expect(visibleCards(summary(), 'menopause')).toEqual(['symptoms', 'sleepMood', 'weight', 'vitals', 'labs']);
    expect(visibleCards(summary({ cycle: open(null as never, true) }), 'menopause')).toContain('cycle');
    expect(asksForCycles('menopause')).toBe(false);
    expect(asksForCycles('teen')).toBe(true);
  });

  it('groups cards under their headings for a category', () => {
    const cards = visibleCards(summary(), 'cycle');
    expect(groupCards(cards, 'all').map((g) => g.group)).toEqual(['cycle', 'symptomsMood', 'bodyLabs']);
    expect(groupCards(cards, 'moodSleep')).toEqual([{ group: 'symptomsMood', cards: ['moodByPhase', 'sleepMood'] }]);
    expect(groupCards(cards, 'labs')).toEqual([{ group: 'bodyLabs', cards: ['labs'] }]);
  });

  it('shows the empty hub only when the days nudge has nothing behind it (§4.1)', () => {
    expect(hasNoData(summary())).toBe(true);
    const finding = summary();
    finding.topFinding = { ...finding.topFinding, kind: 'correlation', daysNeeded: null, strength: 0.7, score: 0.7 };
    expect(hasNoData(finding)).toBe(false);
    expect(hasNoData(summary({ recentCycles: open([{ start: '2026-09-01' } as never], false) }))).toBe(false);
  });

  it('formats signed deltas', () => {
    expect(signedDecimal(-0.63)).toEqual({ sign: '−', abs: '0.6' });
    expect(signedDecimal(1)).toEqual({ sign: '+', abs: '1' });
    expect(signedDecimal(0.02)).toEqual({ sign: '', abs: '0' });
  });
});
