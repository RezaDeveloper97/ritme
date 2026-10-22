import { describe, expect, it } from 'vitest';

import { toTargetsBody, type GlucoseTargets } from '@/entities/vital';

import { parseTarget, targetIssues, targetRows } from './targets';

const targets: GlucoseTargets = {
  diabetic: true,
  bounds: { min: 50, max: 300 },
  defaults: {
    fasting: { lowMgDl: 80, highMgDl: 130, highInclusive: true },
    pre_meal: { lowMgDl: 80, highMgDl: 130, highInclusive: true },
    post_meal: { lowMgDl: null, highMgDl: 180, highInclusive: false },
    bedtime: null,
  },
  items: [{ context: 'fasting', lowMgDl: 90, highMgDl: 140, highInclusive: true, setBy: 'doctor', note: null }],
};

describe('glucose targets form', () => {
  it('prefills one row per context from the saved targets', () => {
    const rows = targetRows(targets);
    expect(rows.map((r) => r.context)).toEqual(['fasting', 'pre_meal', 'post_meal', 'bedtime']);
    expect(rows[0]).toMatchObject({ lowMgDl: 90, highMgDl: 140, setBy: 'doctor' });
    expect(rows[2]).toMatchObject({ lowMgDl: null, highMgDl: null });
  });

  it('parses Persian and Latin digits; empty is open', () => {
    expect(parseTarget('۱۳۰')).toBe(130);
    expect(parseTarget(' 95 ')).toBe(95);
    expect(parseTarget('')).toBeNull();
  });

  it('flags a low ≥ high and values outside the bounds', () => {
    const rows = targetRows(targets);
    expect(targetIssues(rows, targets.bounds)).toEqual({});
    rows[0] = { ...rows[0], lowMgDl: 150 };
    rows[2] = { ...rows[2], highMgDl: 400 };
    expect(targetIssues(rows, targets.bounds)).toEqual({ fasting: 'lowHigh', post_meal: 'bounds' });
  });

  it('sends only rows with a number', () => {
    expect(toTargetsBody(targetRows(targets))).toEqual({
      items: [{ context: 'fasting', low_mgdl: 90, high_mgdl: 140, set_by: 'doctor' }],
    });
  });
});
