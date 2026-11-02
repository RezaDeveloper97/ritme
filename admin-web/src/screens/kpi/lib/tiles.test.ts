import { describe, expect, it } from 'vitest';

import type { Kpi } from '../api/kpi';
import { ctrRows, kpiTiles } from './tiles';

const base = (over: Partial<Kpi>): Kpi =>
  ({
    generated_at: '2026-11-02T10:00:00+03:30',
    events_since: null,
    funnel: { days: 90, steps: [] },
    retention: { event: 'any', offsets: [], cohorts: [] },
    engine: null,
    data_quality: null,
    ai: { days: 30, labs_per_day: null, ocr_correction_pct: 8.5, cost_per_analysis_usd: null, assistant_msgs_per_day: null, referral_pct: null, suppressed: false },
    notes: {},
    ...over,
  }) as Kpi;

describe('KPI tiles', () => {
  it('builds the five new tiles with values and deltas', () => {
    const tiles = kpiTiles(
      base({
        rui_resolved_7d: { pct: 61.5, count: 40, of: 65, suppressed: false },
        ai_cost: { days: 30, share_pct: 12.2, suppressed: false },
        activity: { loggers_15d: { pct: 33, count: 90, of: 270, suppressed: false } },
        message_ctr: { days: 30, overall: { pct: 4.1, count: 10, of: 200, suppressed: false }, versions: [] },
        deltas: { rui_resolved_7d: { delta: 2.5 }, ocr_correction_rate: { delta: -1 } },
      } as Partial<Kpi>),
    );
    expect(tiles.map((t) => t.key)).toEqual(['rui7d', 'aiCostShare', 'loggers15d', 'ocrCorrection', 'ctr']);
    expect(tiles[0]).toMatchObject({ value: 61.5, delta: 2.5, suppressed: false });
    expect(tiles[3]).toMatchObject({ value: 8.5, delta: -1, days: 30 });
  });

  it('keeps suppressed values null and flagged, and missing ones plain null', () => {
    const tiles = kpiTiles(
      base({
        rui_resolved_7d: { pct: 50, suppressed: true },
        ai: { days: 30, labs_per_day: null, ocr_correction_pct: 9, cost_per_analysis_usd: null, assistant_msgs_per_day: null, referral_pct: null, suppressed: true },
        deltas: { rui_resolved_7d: { delta: 3 } },
      }),
    );
    expect(tiles[0]).toMatchObject({ value: null, suppressed: true, delta: null });
    expect(tiles[3]).toMatchObject({ value: null, suppressed: true });
    expect(tiles[1]).toMatchObject({ value: null, suppressed: false });
  });

  it('lists CTR per version newest first and strips suppressed rows', () => {
    const rows = ctrRows(
      base({
        message_ctr: {
          days: 30,
          overall: { pct: null, suppressed: true },
          versions: [
            { rule_id: 1, version: 2, shown: 500, clicked: 50, ctr_pct: 10, suppressed: false },
            { rule_id: 3, version: 1, shown: 4, clicked: 1, ctr_pct: 25, suppressed: true },
            { rule_id: 1, version: 3, shown: 300, clicked: 15, ctr_pct: 5, suppressed: false },
          ],
        },
      }),
    );
    expect(rows.map((r) => r.id)).toEqual(['3:1', '1:3', '1:2']);
    expect(rows[0]).toMatchObject({ ctr: null, shown: null, suppressed: true });
  });
});
