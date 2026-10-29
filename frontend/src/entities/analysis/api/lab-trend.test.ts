import { describe, expect, it } from 'vitest';

import { trendByMarker } from '../model/lab-trend';
import { labTrendListSchema } from './lab-trend';

/*
 * Boundary contract for GET /analysis/labs (backend-go/internal/analysis/labtrend.go,
 * T-M10-26): a trend needs the marker in 2 confirmed reports; below that the row
 * says how many more reports are needed instead of a direction.
 */
const note = { key: 'labs.trend.note', params: {}, text: 'This only shows how the numbers moved, not a diagnosis.' };

describe('labTrendListSchema', () => {
  it('parses an entitled list (ready and not-ready markers)', () => {
    const r = labTrendListSchema.parse({
      plus: true,
      locked: false,
      ready: true,
      data: {
        min_reports: 2,
        markers: [
          {
            marker: 'tsh',
            name: 'TSH',
            unit: 'mIU/L',
            reports: 3,
            ready: true,
            reports_needed: 0,
            direction: 'rising',
            finding: { key: 'labs.trend.rising', params: { count: 3 }, text: 'This marker went up across your last 3 labs.' },
          },
          {
            marker: 'ferritin',
            name: 'Ferritin',
            unit: null,
            reports: 1,
            ready: false,
            reports_needed: 1,
            direction: null,
            finding: { key: 'labs.trend.not_enough', params: { needed: 1 }, text: 'To see a trend, add 1 more lab result.' },
          },
          { marker: 'broken' },
        ],
        note,
      },
    });
    expect(r).toMatchObject({ locked: false, ready: true, minReports: 2, note });
    expect(r.markers.map((m) => m.marker)).toEqual(['tsh', 'ferritin']);
    expect(trendByMarker(r, 'tsh')).toMatchObject({ direction: 'rising', reportsNeeded: 0 });
    expect(trendByMarker(r, 'ferritin')).toMatchObject({ ready: false, direction: null, reportsNeeded: 1 });
    expect(trendByMarker(r, 'ldl')).toBeNull();
  });

  it('a locked (free) reader gets no markers', () => {
    const r = labTrendListSchema.parse({ plus: true, locked: true, ready: false, data: null });
    expect(r).toEqual({ locked: true, ready: false, minReports: 2, markers: [], note: null });
  });

  it('drops an unknown direction instead of failing', () => {
    const r = labTrendListSchema.parse({
      plus: true,
      locked: false,
      ready: true,
      data: {
        min_reports: 2,
        markers: [{ marker: 'tsh', name: 'TSH', unit: null, reports: 2, ready: true, reports_needed: 0, direction: 'zigzag', finding: null }],
        note: null,
      },
    });
    expect(r.markers[0].direction).toBeNull();
  });
});
