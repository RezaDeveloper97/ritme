import { describe, expect, it } from 'vitest';

import { hubSchema, parseReport } from '@/entities/vital';

import { readingsNeeded } from './reports';

describe('not enough data', () => {
  it('reads the hub report states and asks for the missing readings', () => {
    const hub = hubSchema.parse({
      date: '2026-10-06',
      latest: {},
      plan: { from: '', to: '', planned: 0, done: 0, items: [] },
      recent: [],
      notifications: { enabled: false },
      reports: {
        bp: { range: '7d', status: 'not_enough_data', readings_needed: 1 },
        hr: { range: '7d', status: 'ok', readings_needed: 0 },
        glucose: { range: '14d', status: 'not_enough_data', readings_needed: 2 },
      },
    });
    expect(readingsNeeded(hub.reports.bp, true)).toBe(1);
    expect(readingsNeeded(hub.reports.hr, true)).toBeNull();
    expect(readingsNeeded(hub.reports.glucose, false)).toBeNull();
    expect(readingsNeeded(null, true)).toBeNull();
  });

  it('a hub without report states (older server) never shows the state', () => {
    const hub = hubSchema.parse({ date: '2026-10-06', latest: {}, plan: {}, recent: [], notifications: {} });
    expect(hub.reports).toEqual({ bp: null, hr: null, glucose: null });
  });

  it('parses a not_enough_data report with null statistics', () => {
    const r = parseReport('bp', {
      type: 'bp',
      range: { key: '7d', from: '2026-09-30', to: '2026-10-06', days: 7 },
      status: 'not_enough_data',
      readings_needed: 1,
      readings: 2,
      average: null,
      pulse: null,
      min: null,
      max: null,
      distribution: null,
      morning_vs_night: null,
      time_in_range: null,
      target: { systolic_max: 120, diastolic_max: 80 },
      series: [{ date: '2026-10-06', readings: 2, systolic: 121, diastolic: 78 }],
    });
    expect(r.status).toBe('not_enough_data');
    expect(r.readingsNeeded).toBe(1);
    expect(r.average).toBeNull();
    expect(r.distribution).toEqual([]);
  });
});
