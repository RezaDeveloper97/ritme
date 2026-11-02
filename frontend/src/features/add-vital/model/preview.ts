import {
  classifyGlucose,
  type GlucoseContext,
  type GlucoseTargets,
  type TargetContext,
  type VitalClass,
  type VitalThresholds,
} from '@/entities/vital';

/*
 * Live glucose class of the add form (T-M10-19b): the same rule the server
 * saves the reading with (backend-go/internal/vitals: Reading.Class →
 * classifyDiabetic, else ClassifyGlucoseReading). A user with diabetes
 * (`GET /vitals/targets` diabetic) has fasting / pre-meal / post-meal /
 * bedtime readings compared with her own target row, else the registry
 * default; hypoglycaemia wins. Everyone else gets the general rule with the
 * method (spec = 1 post-meal: a lab value keeps the 140 / 200 bands).
 */

/** The diabetic band a reading is compared with (null = no target: the general rule applies). */
export interface TargetBand {
  context: TargetContext;
  low: number | null;
  high: number | null;
  highInclusive: boolean;
}

const TARGET_OF: Partial<Record<GlucoseContext, TargetContext>> = {
  fasting: 'fasting',
  before_meal: 'pre_meal',
  after_meal: 'post_meal',
  bedtime: 'bedtime',
};

/** The user's row of the context (when it has a bound), else the registry default. */
export function targetBand(context: GlucoseContext, targets: GlucoseTargets | null | undefined): TargetBand | null {
  const tc = TARGET_OF[context];
  if (!tc || !targets?.diabetic) return null;
  const row = targets.items.find((i) => i.context === tc && ((i.lowMgDl ?? 0) > 0 || (i.highMgDl ?? 0) > 0));
  const src = row ?? targets.defaults[tc];
  if (!src) return null;
  return { context: tc, low: src.lowMgDl, high: src.highMgDl, highInclusive: src.highInclusive };
}

/** The class the server would save a glucose reading with (null = no class). */
export function previewGlucoseClass(
  mgdl: number,
  context: GlucoseContext,
  method: string | null,
  th: VitalThresholds,
  targets: GlucoseTargets | null | undefined,
): VitalClass | null {
  const band = targetBand(context, targets);
  if (!band) return classifyGlucose(mgdl, context, th, method);
  if (mgdl < th.glucose.urgentBelow) return { code: 'urgent_low', tone: 'urgent' };
  if (mgdl < th.glucose.lowBelow) return { code: 'low', tone: 'high' };
  if (band.low && mgdl < band.low) return { code: 'below_target', tone: 'watch' };
  if (band.high && (mgdl > band.high || (!band.highInclusive && mgdl === band.high))) return { code: 'above_target', tone: 'watch' };
  return { code: 'in_target', tone: 'ok' };
}
