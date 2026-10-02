import { describe, expect, it } from 'vitest';

import { menopausePatternsSchema, menopauseScoreHistorySchema, menopauseScoreQuestionsSchema } from './score';

// Trimmed from backend-go/contract/golden/menopause/{score_flow,patterns}.en.json.
const HISTORY = {
  months: 6,
  max: 44,
  bands: [
    { code: 'none', title: 'No symptoms', min: 0, max: 4 },
    { code: 'moderate', title: 'Moderate', min: 9, max: 16 },
  ],
  latest: {
    month: '2026-09-23',
    jalali_year: 1405,
    jalali_month: 7,
    total: 14,
    max: 44,
    band: { code: 'moderate', title: 'Moderate', min: 9, max: 16 },
    domains: [{ code: 'somatic', score: 7, max: 16 }],
    answers: { hot_flashes: 3 },
    delta: -4,
    previous_month: '2026-08-23',
  },
  trend: [
    { month: '2026-08-23', jalali_year: 1405, jalali_month: 6, total: 18, band: 'severe' },
    { month: '2026-09-23', jalali_year: 1405, jalali_month: 7, total: 14, band: 'moderate' },
  ],
  items: [],
  hrt: { started_on: '2026-07-23', baseline_month: '2026-06-22', baseline_total: 22, latest_month: '2026-09-23', latest_total: 14, change: -8 },
};

describe('menopause score schemas (CB-MENO-08)', () => {
  it('parses the score history with the HRT note', () => {
    const h = menopauseScoreHistorySchema.parse(HISTORY);
    expect(h.latest).toMatchObject({ total: 14, band: { code: 'moderate' }, delta: -4, answers: { hot_flashes: 3 } });
    expect(h.trend.map((p) => p.total)).toEqual([18, 14]);
    expect(h.hrt).toEqual({ startedOn: '2026-07-23', baselineTotal: 22, latestTotal: 14, change: -8 });
    expect(menopauseScoreHistorySchema.parse({ ...HISTORY, latest: null, hrt: null }).hrt).toBeNull();
  });

  it('parses patterns with the disclaimer', () => {
    const p = menopausePatternsSchema.parse({
      days_logged: 4,
      min_days: 20,
      found: 0,
      disclaimer: { code: 'patterns_disclaimer', title: 'Patterns we noticed', body: 'not a medical diagnosis' },
      items: [{ key: 'night_sweats_fatigue', trigger: null, found: false, text: null, status: 'not_enough_data' }],
    });
    expect(p).toMatchObject({ daysLogged: 4, minDays: 20, found: 0, disclaimer: { title: 'Patterns we noticed' } });
    expect(p.items[0]).toEqual({ key: 'night_sweats_fatigue', trigger: null, found: false, text: null });
  });

  it('reads the questions from the catalog meta and drops untitled ones', () => {
    const qs = menopauseScoreQuestionsSchema.parse({
      group: 'meno_score_items',
      items: [
        { code: 'hot_flashes', title: 'Hot flashes', body: 'Waves of heat', meta: { domain: 'somatic', max: 4, log: [] } },
        { code: 'anxiety', title: 'Anxiety', body: null, meta: null },
        { code: 'blank', title: null, body: null, meta: null },
      ],
    });
    expect(qs).toEqual([
      { code: 'hot_flashes', title: 'Hot flashes', body: 'Waves of heat', domain: 'somatic', max: 4 },
      { code: 'anxiety', title: 'Anxiety', body: null, domain: 'somatic', max: 4 },
    ]);
  });
});
