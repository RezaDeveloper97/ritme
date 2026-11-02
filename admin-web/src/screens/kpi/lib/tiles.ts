/** View-models of the §5.6 KPI tiles (T-M10-34). Suppressed values are never shown as numbers or zero. */

import type { Kpi } from '../api/kpi';

export type TileKey = 'rui7d' | 'aiCostShare' | 'loggers15d' | 'ocrCorrection' | 'ctr';

export interface Tile {
  key: TileKey;
  /** Percent (0..100) or null. */
  value: number | null;
  /** True when the group is under the display threshold: show «کمتر از حد نمایش», not «—». */
  suppressed: boolean;
  delta: number | null;
  /** Context for the note under the value (window in days), if any. */
  days: number | null;
}

function tile(key: TileKey, value: number | null | undefined, suppressed: boolean, delta: number | null | undefined, days: number | null = null): Tile {
  const hidden = suppressed;
  return { key, value: hidden ? null : (value ?? null), suppressed: hidden, delta: hidden ? null : (delta ?? null), days };
}

export function kpiTiles(k: Kpi): Tile[] {
  const d = (key: string) => k.deltas?.[key]?.delta ?? null;
  const rui = k.rui_resolved_7d;
  const loggers = k.activity?.loggers_15d;
  return [
    tile('rui7d', rui?.pct, rui?.suppressed ?? false, d('rui_resolved_7d')),
    tile('aiCostShare', k.ai_cost?.share_pct, k.ai_cost?.suppressed ?? false, d('ai_cost_share'), k.ai_cost?.days ?? null),
    tile('loggers15d', loggers?.pct, loggers?.suppressed ?? false, d('loggers_15d')),
    tile('ocrCorrection', k.ai.ocr_correction_pct, k.ai.suppressed, d('ocr_correction_rate'), k.ai.days),
    tile('ctr', k.message_ctr?.overall.pct, k.message_ctr?.overall.suppressed ?? false, d('message_ctr'), k.message_ctr?.days ?? null),
  ];
}

export interface CtrRow {
  id: string;
  ruleId: number;
  version: number;
  shown: number | null;
  clicked: number | null;
  ctr: number | null;
  suppressed: boolean;
}

/** CTR per message rule version: newest rule and version first; a suppressed row keeps no numbers. */
export function ctrRows(k: Kpi): CtrRow[] {
  return (k.message_ctr?.versions ?? [])
    .map((v) => ({
      id: `${v.rule_id}:${v.version}`,
      ruleId: v.rule_id,
      version: v.version,
      shown: v.suppressed ? null : v.shown,
      clicked: v.suppressed ? null : v.clicked,
      ctr: v.suppressed ? null : v.ctr_pct,
      suppressed: v.suppressed,
    }))
    .sort((a, b) => b.ruleId - a.ruleId || b.version - a.version);
}
