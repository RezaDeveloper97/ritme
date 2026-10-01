import { existsSync, readFileSync } from 'node:fs';
import { resolve } from 'node:path';

import { describe, expect, it } from 'vitest';

import { barShares, lowestIndex, shareOfMax, stripCells } from '../model/layout';
import { analysisKeys } from './keys';
import { analysisSummarySchema } from './schema';

/*
 * Boundary contract for GET /analysis/summary, against the Go engine's own
 * goldens (backend-go/internal/analysis/testdata/golden, B-N3-07) so a backend
 * shape change fails here too.
 */
const GOLDEN_DIR = resolve(process.cwd(), '../backend-go/internal/analysis/testdata/golden');
const golden = (name: string): unknown => JSON.parse(readFileSync(resolve(GOLDEN_DIR, `${name}.json`), 'utf8'));

// Skipped when the frontend is checked out alone (no backend-go next to it).
describe.skipIf(!existsSync(GOLDEN_DIR))('analysisSummarySchema', () => {
  it('parses a full entitled hub', () => {
    const s = analysisSummarySchema.parse(golden('regular_summary_6m'));
    expect(s.range.key).toBe('6m');
    expect(s.topFinding.kind).toBe('cycle_regular');
    expect(s.topFinding.parts).toHaveLength(2);
    expect(s.sections.cycle.data?.medianCycle).toBe(29);
    expect(s.sections.cycle.data?.bars[0]).toEqual({ start: '2026-04-13', length: 38, inFigoRange: true });
    expect(s.sections.recentCycles.data?.[0].isCurrent).toBe(true);
    expect(s.sections.symptoms.data?.highlight?.relation).toBe('before_period');
    expect(s.sections.moodByPhase.data?.phases.map((p) => p.phase)).toEqual(['period', 'follicular', 'fertile', 'luteal']);
    expect(s.sections.sleepMood.data?.groups[1]).toMatchObject({ key: 'sleep_under_6', pct: 69 });
    expect(s.sections.weight.data?.delta30d).toBe(-0.3);
    expect(s.sections.vitals.data?.bloodPressure).toEqual({ systolic: 120, diastolic: 77, readings: 26 });
    expect(s.sections.labs).toEqual({ plus: true, locked: false, ready: false, data: null });
  });

  it('parses an empty, free hub (Plus sections locked, no data)', () => {
    const s = analysisSummarySchema.parse(golden('empty_summary_6m'));
    expect(s.topFinding.kind).toBe('no_data');
    expect(s.sections.moodByPhase).toEqual({ plus: true, locked: true, ready: false, data: null });
    expect(s.sections.weight.data?.current).toBeNull();
    expect(s.sections.vitals.data).toEqual({ bloodPressure: null, bloodSugar: null });
  });

  it('parses the teen and irregular fixtures', () => {
    expect(analysisSummarySchema.parse(golden('teen_summary_6m')).topFinding.kind).toBe('not_enough_data');
    const irr = analysisSummarySchema.parse(golden('irregular_summary_6m'));
    expect(irr.sections.cycle.data?.regularity).toBe('irregular');
  });

  it('degrades unknown enums instead of failing', () => {
    const raw = golden('regular_summary_6m') as { top_finding: { kind: string } };
    raw.top_finding.kind = 'something_new';
    expect(analysisSummarySchema.parse(raw).topFinding.kind).toBeNull();
  });
});

describe('analysis layout helpers', () => {
  it('lays out a cycle strip', () => {
    const cells = stripCells({
      start: '2026-09-14',
      length: 10,
      isCurrent: true,
      daysSoFar: 6,
      periodDays: 3,
      fertileStartDay: 5,
      fertileEndDay: 8,
      ovulationDay: 8,
    });
    expect(cells.map((c) => c.kind)).toEqual([
      'period', 'period', 'period', 'other', 'fertile', 'fertile', 'fertile', 'ovulation', 'other', 'other',
    ]);
    expect(cells.filter((c) => c.future).map((c) => c.day)).toEqual([7, 8, 9, 10]);
  });

  it('scales bars and shares', () => {
    expect(barShares([28, 28])).toEqual([1, 1]);
    const s = barShares([26, 41, 29]);
    expect(s[1]).toBe(1);
    expect(s[0]).toBeCloseTo(0.35);
    expect(shareOfMax([18, 9, 0])).toEqual([100, 50, 0]);
    expect(lowestIndex([50, null, 30, 80])).toBe(2);
    expect(lowestIndex([null])).toBe(-1);
  });

  it('keys reports by range', () => {
    expect(analysisKeys.summary('3m')).toEqual(['analysis', 'summary', '3m']);
  });
});
