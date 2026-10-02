import { describe, expect, it } from 'vitest';

import {
  answersFor,
  bandSegments,
  bandTone,
  filledThisMonth,
  groupQuestions,
  hrtNote,
  initialAnswers,
  isComplete,
  markerPercent,
} from './score';

const BANDS = [
  { code: 'severe', title: 'شدید', min: 17, max: 44 },
  { code: 'none', title: 'بدون علامت', min: 0, max: 4 },
  { code: 'mild', title: 'خفیف', min: 5, max: 8 },
  { code: 'moderate', title: 'متوسط', min: 9, max: 16 },
];

const Q = (code: string, domain: string) => ({ code, title: code, body: null, domain, max: 4 });

describe('menopause score view model (CB-MENO-08)', () => {
  it('sizes the band bar by range, low → high, with the board tints', () => {
    const segs = bandSegments(BANDS);
    expect(segs.map((s) => s.code)).toEqual(['none', 'mild', 'moderate', 'severe']);
    expect(segs.map((s) => s.tone)).toEqual(['data', 'brand', 'bloom', 'danger']);
    expect(segs.map((s) => s.share)).toEqual([11.1, 8.9, 17.8, 62.2]);
    expect(bandTone('moderate', BANDS)).toBe('bloom');
    expect(bandTone('nope', BANDS)).toBe('neutral');
  });

  it('places the marker inside the bar', () => {
    expect(markerPercent(0, BANDS)).toBe(1.1);
    expect(markerPercent(14, BANDS)).toBe(32.2);
    expect(markerPercent(99, BANDS)).toBe(98.9);
    expect(markerPercent(3, [])).toBe(0);
  });

  it('words the HRT change', () => {
    expect(hrtNote(-8)).toEqual({ kind: 'less', points: 8 });
    expect(hrtNote(3)).toEqual({ kind: 'more', points: 3 });
    expect(hrtNote(0)).toEqual({ kind: 'same' });
    expect(hrtNote(null)).toBeNull();
  });

  it('knows whether this month is filled and prefills only then', () => {
    const latest = { month: '2026-09-23', total: 3, max: 44, band: null, domains: [], answers: { hot_flashes: 3 }, delta: null };
    const filled = { trend: [{ month: '2026-09-23', total: 3, band: 'none' }], latest };
    const open = { trend: [{ month: '2026-08-23', total: 3, band: 'none' }, { month: '2026-09-23', total: null, band: null }], latest: { ...latest, month: '2026-08-23' } };
    expect(filledThisMonth(filled)).toBe(true);
    expect(filledThisMonth(open)).toBe(false);
    expect(initialAnswers(filled)).toEqual({ hot_flashes: 3 });
    expect(initialAnswers(open)).toEqual({});
    expect(initialAnswers(undefined)).toEqual({});
  });

  it('groups questions by domain and checks completeness', () => {
    const qs = [Q('a', 'somatic'), Q('b', 'psychological'), Q('c', 'somatic')];
    expect(groupQuestions(qs).map((g) => [g.domain, g.questions.map((q) => q.code)])).toEqual([
      ['somatic', ['a', 'c']],
      ['psychological', ['b']],
    ]);
    expect(isComplete(qs, { a: 0, b: 4, c: 2 })).toBe(true);
    expect(isComplete(qs, { a: 0, b: 5, c: 2 })).toBe(false);
    expect(isComplete(qs, { a: 0, b: 1 })).toBe(false);
    expect(isComplete([], {})).toBe(false);
    expect(answersFor(qs, { a: 1, b: 2, c: 3, stale: 4 })).toEqual({ a: 1, b: 2, c: 3 });
  });
});
