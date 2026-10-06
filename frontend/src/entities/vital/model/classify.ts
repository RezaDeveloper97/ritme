import type { Tone } from '@/shared/ui';

import type { GlucoseContext, VitalClass, VitalThresholds, VitalTone, VitalType } from './types';

/*
 * Live preview of the server's classification while a number is typed
 * (nbl_Vitals_AddBP «این عدد یعنی», AddGlucose / AddHR verdict pill). The bands
 * come from `GET /vitals/thresholds`; DEFAULT_THRESHOLDS mirrors
 * backend-go/internal/vitals/thresholds.go so the form works before that read
 * lands. The saved reading's classification is always the server's.
 */

export const DEFAULT_THRESHOLDS: VitalThresholds = {
  version: '2026-10-acc-aha-2017.ada-2025.v1',
  bp: {
    elevatedSystolic: 120,
    stage1: { systolic: 130, diastolic: 80 },
    stage2: { systolic: 140, diastolic: 90 },
    urgentAbove: { systolic: 180, diastolic: 120 },
  },
  glucose: {
    lowBelow: 70,
    urgentBelow: 54,
    mmolFactor: 18,
    contexts: {
      fasting: { targetMin: 70, targetMax: 100, veryHighFrom: 126 },
      before_meal: { targetMin: 70, targetMax: 100, veryHighFrom: 126 },
      after_meal: { targetMin: 70, targetMax: 140, veryHighFrom: 200 },
      bedtime: { targetMin: 70, targetMax: 140, veryHighFrom: 200 },
      random: { targetMin: 70, targetMax: 140, veryHighFrom: 200 },
    },
  },
  hr: { min: 60, max: 100, classifiedContexts: ['resting', 'after_waking'] },
};

const TONES: Record<VitalType, Record<string, VitalTone>> = {
  bp: { normal: 'ok', elevated: 'watch', stage1: 'high', stage2: 'high', crisis: 'urgent' },
  glucose: { urgent_low: 'urgent', low: 'high', in_range: 'ok', high: 'watch', very_high: 'high' },
  hr: { low: 'watch', normal: 'ok', high: 'watch' },
};

const cls = (type: VitalType, code: string): VitalClass => ({ code, tone: TONES[type][code] ?? 'watch' });

/** ACC/AHA category; the higher of the two numbers decides. */
export function classifyBp(systolic: number, diastolic: number, th: VitalThresholds = DEFAULT_THRESHOLDS): VitalClass {
  const b = th.bp;
  if (systolic > b.urgentAbove.systolic || diastolic > b.urgentAbove.diastolic) return cls('bp', 'crisis');
  if (systolic >= b.stage2.systolic || diastolic >= b.stage2.diastolic) return cls('bp', 'stage2');
  if (systolic >= b.stage1.systolic || diastolic >= b.stage1.diastolic) return cls('bp', 'stage1');
  if (systolic >= b.elevatedSystolic) return cls('bp', 'elevated');
  return cls('bp', 'normal');
}

/** ADA class of a mg/dL value in a context (half-open bands). */
export function classifyGlucose(mgdl: number, context: GlucoseContext, th: VitalThresholds = DEFAULT_THRESHOLDS): VitalClass {
  const band = th.glucose.contexts[context] ?? th.glucose.contexts.random;
  if (mgdl < th.glucose.urgentBelow) return cls('glucose', 'urgent_low');
  if (mgdl < band.targetMin) return cls('glucose', 'low');
  if (mgdl < band.targetMax) return cls('glucose', 'in_range');
  if (mgdl < band.veryHighFrom) return cls('glucose', 'high');
  return cls('glucose', 'very_high');
}

/** Resting range class; null for contexts the range does not apply to (after exercise, stress). */
export function classifyHr(bpm: number, context: string, th: VitalThresholds = DEFAULT_THRESHOLDS): VitalClass | null {
  if (!th.hr.classifiedContexts.includes(context)) return null;
  if (bpm < th.hr.min) return cls('hr', 'low');
  if (bpm > th.hr.max) return cls('hr', 'high');
  return cls('hr', 'normal');
}

/** Whether a reading would raise the urgent modal (the server decides; this only previews). */
export function isUrgentBp(systolic: number, diastolic: number, th: VitalThresholds = DEFAULT_THRESHOLDS): boolean {
  return systolic > th.bp.urgentAbove.systolic || diastolic > th.bp.urgentAbove.diastolic;
}

/** UI tone of a server tone: ok = turquoise data, watch / high = amber, urgent = danger. */
export function uiTone(tone: VitalTone | null | undefined): Tone {
  switch (tone) {
    case 'ok':
      return 'data';
    case 'watch':
    case 'high':
      return 'warm';
    case 'urgent':
      return 'danger';
    default:
      return 'neutral';
  }
}

/** The four segments of the «این عدد یعنی» scale (crisis sits at the end of stage 2). */
export const BP_SCALE = ['normal', 'elevated', 'stage1', 'stage2'] as const;

/**
 * Where the marker of the BP scale sits, 0–100 along the reading direction:
 * the class picks the segment, the systolic value the place inside it.
 */
export function bpScalePosition(systolic: number, diastolic: number, th: VitalThresholds = DEFAULT_THRESHOLDS): number {
  const c = classifyBp(systolic, diastolic, th).code;
  const idx = c === 'crisis' ? 3 : Math.max(0, BP_SCALE.indexOf(c as (typeof BP_SCALE)[number]));
  const bounds: Array<[number, number]> = [
    [90, th.bp.elevatedSystolic],
    [th.bp.elevatedSystolic, th.bp.stage1.systolic],
    [th.bp.stage1.systolic, th.bp.stage2.systolic],
    [th.bp.stage2.systolic, th.bp.urgentAbove.systolic],
  ];
  const [lo, hi] = bounds[idx];
  const within = c === 'crisis' ? 1 : Math.min(1, Math.max(0, (systolic - lo) / (hi - lo)));
  return Math.round((idx + 0.1 + within * 0.8) * 25);
}
