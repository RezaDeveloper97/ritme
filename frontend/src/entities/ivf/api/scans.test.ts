import { describe, expect, it } from 'vitest';

import { ivfScansSchema } from './scans';

// Shape of contract/golden/ivf/scans_flow (GET /ivf/scans).
const cycle = {
  id: 1,
  number: 1,
  protocol: null,
  stage: 'stim',
  status: 'open',
  started_on: '2026-09-10',
  stim_started_on: '2026-09-17',
  beta_on: null,
  notify_companion: false,
  stage_day: 7,
  days_to_beta: null,
  timeline: [],
};

describe('ivfScansSchema', () => {
  it('maps the golden view', () => {
    const view = ivfScansSchema.parse({
      cycle,
      scans: [
        {
          date: '2026-09-23',
          stim_day: 7,
          right: { lt_10: 3, '10_14': 4, '15_17': 2, '18_plus': 0, total: 9 },
          left: { lt_10: 2, '10_14': 5, '15_17': 3, '18_plus': 1, total: 11 },
          total: { lt_10: 5, '10_14': 9, '15_17': 5, '18_plus': 1, total: 20 },
          endometrium_mm: '8.5',
          e2: '1250.00',
          e2_unit: 'pg_ml',
          notes: null,
        },
      ],
      growth: [{ date: '2026-09-23', stim_day: 7, follicles_10_14: 9, follicles_15_plus: 6 }],
    });
    expect(view.cycle?.stage).toBe('stim');
    expect(view.stimStartedOn).toBe('2026-09-17');
    expect(view.scans[0]).toMatchObject({ stimDay: 7, endometriumMm: '8.5', e2: '1250.00', e2Unit: 'pg_ml' });
    expect(view.scans[0]?.left).toEqual({ lt_10: 2, '10_14': 5, '15_17': 3, '18_plus': 1 });
    expect(view.growth).toEqual([{ date: '2026-09-23', stimDay: 7, mid: 9, lead: 6 }]);
  });

  it('tolerates an empty view and malformed items', () => {
    expect(ivfScansSchema.parse({ cycle: null, scans: [], growth: [] })).toEqual({
      cycle: null,
      stimStartedOn: null,
      scans: [],
      growth: [],
    });
    const view = ivfScansSchema.parse({ cycle, scans: [{ date: 'bad' }, { date: '2026-09-21', right: null }], growth: 'x' });
    expect(view.scans).toHaveLength(1);
    expect(view.scans[0]?.right).toEqual({ lt_10: 0, '10_14': 0, '15_17': 0, '18_plus': 0 });
    expect(view.scans[0]?.endometriumMm).toBeNull();
    expect(view.growth).toEqual([]);
  });
});
