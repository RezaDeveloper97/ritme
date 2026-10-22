import { describe, expect, it } from 'vitest';

import { DEFAULT_THRESHOLDS, bpScalePosition, classifyBp, classifyGlucose, classifyHr, isUrgentBp, uiTone } from './classify';

describe('client classification mirrors thresholds.go', () => {
  it('blood pressure (ACC/AHA 2017, the higher number decides)', () => {
    expect(classifyBp(118, 76)).toEqual({ code: 'normal', tone: 'ok' });
    expect(classifyBp(125, 79)).toEqual({ code: 'elevated', tone: 'watch' });
    expect(classifyBp(118, 82).code).toBe('stage1');
    expect(classifyBp(140, 70).code).toBe('stage2');
    expect(classifyBp(181, 90)).toEqual({ code: 'crisis', tone: 'urgent' });
    expect(classifyBp(170, 121).code).toBe('crisis');
    expect(isUrgentBp(180, 120)).toBe(false);
    expect(isUrgentBp(181, 80)).toBe(true);
  });

  it('glucose (ADA 2025, half-open bands per context)', () => {
    expect(classifyGlucose(53, 'fasting')?.code).toBe('urgent_low');
    expect(classifyGlucose(69, 'fasting')?.code).toBe('low');
    expect(classifyGlucose(99, 'fasting')?.code).toBe('in_range');
    expect(classifyGlucose(100, 'fasting')?.code).toBe('high');
    expect(classifyGlucose(126, 'fasting')?.code).toBe('very_high');
    expect(classifyGlucose(139, 'after_meal')?.code).toBe('in_range');
    expect(classifyGlucose(200, 'after_meal')?.code).toBe('very_high');
  });

  it('glucose with the SPEC v6 context rule (glucose.contexts.spec = 1)', () => {
    const th = { ...DEFAULT_THRESHOLDS, glucose: { ...DEFAULT_THRESHOLDS.glucose, contextsSpec: true } };
    expect(classifyGlucose(139, 'after_meal', th, 'glucometer')?.code).toBe('in_range');
    expect(classifyGlucose(140, 'after_meal', th, 'glucometer')).toEqual({ code: 'discuss', tone: 'watch' });
    expect(classifyGlucose(250, 'after_meal', th, null)?.code).toBe('discuss');
    expect(classifyGlucose(250, 'after_meal', th, 'lab')?.code).toBe('very_high');
    expect(classifyGlucose(199, 'random', th)).toBeNull();
    expect(classifyGlucose(200, 'bedtime', th)?.code).toBe('very_high');
    expect(classifyGlucose(60, 'random', th)?.code).toBe('low');
    expect(classifyGlucose(100, 'fasting', th)?.code).toBe('high');
  });

  it('heart rate: resting range only for resting / after waking', () => {
    expect(classifyHr(72, 'resting')?.code).toBe('normal');
    expect(classifyHr(100, 'after_waking')?.code).toBe('normal');
    expect(classifyHr(101, 'resting')?.code).toBe('high');
    expect(classifyHr(55, 'resting')?.code).toBe('low');
    expect(classifyHr(140, 'after_exercise')).toBeNull();
  });

  it('maps server tones to palette tones', () => {
    expect(uiTone('ok')).toBe('data');
    expect(uiTone('watch')).toBe('warm');
    expect(uiTone('urgent')).toBe('danger');
    expect(uiTone(null)).toBe('neutral');
  });

  it('places the scale marker inside its class segment', () => {
    expect(bpScalePosition(110, 70)).toBeLessThan(25);
    const s1 = bpScalePosition(135, 70);
    expect(s1).toBeGreaterThan(50);
    expect(s1).toBeLessThan(75);
    expect(bpScalePosition(190, 100)).toBeGreaterThan(75);
  });
});
