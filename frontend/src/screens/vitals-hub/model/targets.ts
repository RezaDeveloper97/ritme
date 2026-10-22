import { TARGET_CONTEXTS, type GlucoseTargetInput, type GlucoseTargets, type TargetContext } from '@/entities/vital';

/*
 * Diabetic glucose targets form rules, mirroring PUT /vitals/targets (one row
 * per context, low < high, each within the server's bounds; a row with neither
 * number is not saved and falls back to the default). Pure.
 */

/** One editable row per context, prefilled from the saved targets. */
export function targetRows(targets: GlucoseTargets): GlucoseTargetInput[] {
  return TARGET_CONTEXTS.map((context) => {
    const saved = targets.items.find((i) => i.context === context);
    return {
      context,
      lowMgDl: saved?.lowMgDl ?? null,
      highMgDl: saved?.highMgDl ?? null,
      setBy: saved?.setBy ?? 'doctor',
      note: saved?.note ?? null,
    };
  });
}

/** A typed number field: '' → null, else an integer. */
export function parseTarget(raw: string): number | null {
  const s = raw.trim();
  if (!s) return null;
  const n = Number(s.replace(/[۰-۹]/g, (d) => String('۰۱۲۳۴۵۶۷۸۹'.indexOf(d))));
  return Number.isFinite(n) ? Math.round(n) : null;
}

export type TargetIssue = 'bounds' | 'lowHigh';

/** Problems per context (empty = savable). */
export function targetIssues(rows: readonly GlucoseTargetInput[], bounds: { min: number; max: number }): Partial<Record<TargetContext, TargetIssue>> {
  const out: Partial<Record<TargetContext, TargetIssue>> = {};
  const outside = (v: number | null) => v !== null && (v < bounds.min || v > bounds.max);
  for (const r of rows) {
    if (outside(r.lowMgDl) || outside(r.highMgDl)) out[r.context] = 'bounds';
    else if (r.lowMgDl !== null && r.highMgDl !== null && r.lowMgDl >= r.highMgDl) out[r.context] = 'lowHigh';
  }
  return out;
}
