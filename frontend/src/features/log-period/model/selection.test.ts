import { describe, expect, it } from 'vitest';

import { focusRun, runStarts, seedSelection, toggleDay, toSegments, type PeriodSelection } from './selection';

const sorted = (s: Set<string>) => [...s].sort();
const empty = (): PeriodSelection => ({ selected: new Set(), suggested: new Set() });
const period = (start: string, end: string | null) =>
  ({ id: 1, period_start_date: start, period_end_date: end }) as unknown as Parameters<typeof seedSelection>[0][number];

describe('seedSelection', () => {
  it('fills closed periods solid and open ones solid up to today, suggested after', () => {
    const s = seedSelection([period('2026-09-01', '2026-09-03'), period('2026-10-04', null)], 5, '2026-10-05');
    expect(sorted(s.selected)).toEqual(['2026-09-01', '2026-09-02', '2026-09-03', '2026-10-04', '2026-10-05']);
    expect(sorted(s.suggested)).toEqual(['2026-10-06', '2026-10-07', '2026-10-08']);
  });
});

describe('toggleDay', () => {
  it('suggests the usual length after a new start', () => {
    const s = toggleDay(empty(), '2026-10-12', 5);
    expect(sorted(s.selected)).toEqual(['2026-10-12']);
    expect(sorted(s.suggested)).toEqual(['2026-10-13', '2026-10-14', '2026-10-15', '2026-10-16']);
  });

  it('trims a run from a tapped suggested day', () => {
    const s = toggleDay(toggleDay(empty(), '2026-10-12', 5), '2026-10-14', 5);
    expect(sorted(s.suggested)).toEqual(['2026-10-13']);
  });

  it('extends a run by one day without new suggestions', () => {
    const base = toggleDay(empty(), '2026-10-12', 1);
    const s = toggleDay(base, '2026-10-13', 5);
    expect(sorted(s.selected)).toEqual(['2026-10-12', '2026-10-13']);
    expect(s.suggested.size).toBe(0);
  });

  it('clears a selected day', () => {
    const s = toggleDay(toggleDay(empty(), '2026-10-12', 1), '2026-10-12', 1);
    expect(s.selected.size).toBe(0);
  });

  it('stops suggestions at another run', () => {
    const base: PeriodSelection = { selected: new Set(['2026-10-15']), suggested: new Set() };
    const s = toggleDay(base, '2026-10-12', 5);
    expect(sorted(s.suggested)).toEqual(['2026-10-13', '2026-10-14']);
  });
});

describe('runs', () => {
  const days = ['2026-10-12', '2026-10-13', '2026-10-20', '2026-09-01'];
  it('groups contiguous days', () => {
    expect(toSegments(days)).toEqual([
      { start: '2026-09-01', end: '2026-09-01' },
      { start: '2026-10-12', end: '2026-10-13' },
      { start: '2026-10-20', end: '2026-10-20' },
    ]);
    expect(sorted(runStarts(days))).toEqual(['2026-09-01', '2026-10-12', '2026-10-20']);
  });
  it('focuses the run containing today, else the latest started one', () => {
    expect(focusRun(days, '2026-10-13')).toEqual({ start: '2026-10-12', end: '2026-10-13' });
    expect(focusRun(days, '2026-10-16')).toEqual({ start: '2026-10-12', end: '2026-10-13' });
    expect(focusRun([], '2026-10-16')).toBeNull();
  });
});
