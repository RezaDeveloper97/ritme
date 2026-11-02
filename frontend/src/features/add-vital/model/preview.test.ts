import { describe, expect, it } from 'vitest';

import { DEFAULT_THRESHOLDS, type GlucoseTargets, type VitalThresholds } from '@/entities/vital';

import { previewGlucoseClass, targetBand } from './preview';

// Mirrors backend-go/internal/vitals (Reading.Class): the add-form preview equals the saved class (T-M10-19b).

const registry: GlucoseTargets['defaults'] = {
  fasting: { lowMgDl: 80, highMgDl: 130, highInclusive: true },
  pre_meal: { lowMgDl: 80, highMgDl: 130, highInclusive: true },
  post_meal: { lowMgDl: null, highMgDl: 180, highInclusive: false },
  bedtime: null,
};
const diabetic: GlucoseTargets = { diabetic: true, bounds: { min: 50, max: 300 }, defaults: registry, items: [] };
const spec: VitalThresholds = { ...DEFAULT_THRESHOLDS, glucose: { ...DEFAULT_THRESHOLDS.glucose, contextsSpec: true } };
const code = (...a: Parameters<typeof previewGlucoseClass>) => previewGlucoseClass(...a)?.code ?? null;

describe('add-vital glucose preview', () => {
  it('diabetic: registry targets, hypoglycaemia wins, post-meal ceiling exclusive', () => {
    expect(code(79, 'fasting', 'glucometer', DEFAULT_THRESHOLDS, diabetic)).toBe('below_target');
    expect(code(130, 'before_meal', 'glucometer', DEFAULT_THRESHOLDS, diabetic)).toBe('in_target');
    expect(code(131, 'fasting', 'glucometer', DEFAULT_THRESHOLDS, diabetic)).toBe('above_target');
    expect(code(179, 'after_meal', 'glucometer', DEFAULT_THRESHOLDS, diabetic)).toBe('in_target');
    expect(code(180, 'after_meal', 'lab', DEFAULT_THRESHOLDS, diabetic)).toBe('above_target');
    expect(code(65, 'fasting', 'glucometer', DEFAULT_THRESHOLDS, diabetic)).toBe('low');
    expect(code(50, 'after_meal', 'glucometer', DEFAULT_THRESHOLDS, diabetic)).toBe('urgent_low');
  });

  it("diabetic: her own row replaces the registry; bedtime only with a row; random never has a target", () => {
    const own: GlucoseTargets = {
      ...diabetic,
      items: [
        { context: 'fasting', lowMgDl: 90, highMgDl: 120, highInclusive: true, setBy: 'doctor', note: null },
        { context: 'bedtime', lowMgDl: 100, highMgDl: 150, highInclusive: true, setBy: 'user', note: null },
      ],
    };
    expect(code(85, 'fasting', null, DEFAULT_THRESHOLDS, own)).toBe('below_target');
    expect(code(125, 'fasting', null, DEFAULT_THRESHOLDS, own)).toBe('above_target');
    expect(code(150, 'bedtime', null, DEFAULT_THRESHOLDS, own)).toBe('in_target');
    expect(code(160, 'bedtime', null, DEFAULT_THRESHOLDS, diabetic)).toBe('high');
    expect(targetBand('random', own)).toBeNull();
    expect(code(150, 'random', null, DEFAULT_THRESHOLDS, own)).toBe('high');
  });

  it('not diabetic: targets are ignored', () => {
    const notDiabetic = { ...diabetic, diabetic: false };
    expect(code(131, 'fasting', 'glucometer', DEFAULT_THRESHOLDS, notDiabetic)).toBe('very_high');
    expect(code(95, 'fasting', 'glucometer', DEFAULT_THRESHOLDS, undefined)).toBe('in_range');
  });

  it('spec = 1: the method decides the post-meal rule', () => {
    expect(code(150, 'after_meal', 'glucometer', spec, null)).toBe('discuss');
    expect(code(150, 'after_meal', null, spec, null)).toBe('discuss');
    expect(code(150, 'after_meal', 'lab', spec, null)).toBe('high');
    expect(code(210, 'after_meal', 'lab', spec, null)).toBe('very_high');
    expect(code(150, 'random', 'glucometer', spec, null)).toBeNull();
    expect(code(150, 'after_meal', 'glucometer', spec, diabetic)).toBe('in_target');
  });
});
