import { describe, expect, it } from 'vitest';

import { hubSchema, parseReport, savedSchema } from './schema';

const bpReading = {
  id: 51,
  source: 'vitals',
  type: 'bp',
  date: '2026-10-06',
  time: '08:10',
  measured_at: '2026-10-06T08:10:00+03:30',
  period: 'morning',
  blood_pressure: { systolic: 185, diastolic: 110, pulse: null, arm: 'left', position: 'sitting' },
  glucose: null,
  heart_rate: null,
  classification: { code: 'crisis', tone: 'urgent' },
  urgent: true,
  note: null,
  editable: true,
};

describe('vitals schemas', () => {
  it('keeps the urgent alert and its call action, dropping a malformed action', () => {
    const saved = savedSchema.parse({
      reading: bpReading,
      alert: {
        rule: 'bp_crisis',
        level: 'urgent',
        modal: true,
        title: 'فشار خونت در محدودهٔ بحرانی است',
        what_we_saw: 'x',
        advice: 'y',
        contact: 'z',
        actions: [{ key: 'call', label: 'تماس با ۱۱۵', phone: '115' }, { key: 'ack', label: 'باشه' }, { key: 'bad', phone: 'tel:evil' }],
      },
    });
    expect(saved.reading.urgent).toBe(true);
    expect(saved.alert?.actions).toEqual([
      { key: 'call', label: 'تماس با ۱۱۵', phone: '115' },
      { key: 'ack', label: 'باشه', phone: null },
    ]);
  });

  it('never loses an alert that fails to parse (falls back to bundled copy)', () => {
    const saved = savedSchema.parse({ reading: bpReading, alert: 'garbled' });
    expect(saved.alert).not.toBeNull();
    expect(saved.alert?.title).toBeNull();
  });

  it('has no alert under the thresholds', () => {
    expect(savedSchema.parse({ reading: bpReading, alert: null }).alert).toBeNull();
  });

  it('marks log-sheet values read-only and keeps the typed glucose unit', () => {
    const hub = hubSchema.parse({
      date: '2026-10-06',
      latest: {
        bp: null,
        hr: null,
        glucose: { ...bpReading, id: null, source: 'log', type: 'glucose', time: null, measured_at: null, period: null, blood_pressure: null, glucose: { mg_dl: 94, mmol_l: 5.2, value: 5.2, unit: 'mmol_l', context: 'fasting', method: null }, editable: false },
      },
      plan: { from: '2026-10-03', to: '2026-10-09', planned: 0, done: 0, items: [] },
      recent: [bpReading, { ...bpReading, type: 'weird' }],
      notifications: { category: 'vitals', enabled: false },
    });
    expect(hub.latest.glucose?.editable).toBe(false);
    expect(hub.latest.glucose?.glucose?.unit).toBe('mmol_l');
    expect(hub.latest.glucose?.glucose?.value).toBe(5.2);
    expect(hub.recent).toHaveLength(1);
  });

  it('parses a glucose report into camel case', () => {
    const r = parseReport('glucose', {
      type: 'glucose',
      range: { key: '14d', from: '2026-09-23', to: '2026-10-06', days: 14 },
      filter: 'all',
      readings: 2,
      average: { mg_dl: 101, mmol_l: 5.6, readings: 2 },
      by_context: [{ context: 'fasting', readings: 1, average: { mg_dl: 93, mmol_l: 5.2, readings: 1 }, above_target: 0, target: { min: 70, max: 100 } }],
      min: null,
      max: null,
      distribution: [],
      morning_vs_night: { morning: null, night: null, night_out_of_range: 0 },
      time_in_range: { in_range: 2, below: 0, above: 0, readings: 2, percent: 100 },
      series: [{ date: '2026-09-24', readings: 2, fasting: 99, after_meal: 122, other: null }],
    });
    expect(r.type).toBe('glucose');
    if (r.type !== 'glucose') return;
    expect(r.series[0].afterMeal).toBe(122);
    expect(r.byContext[0].aboveTarget).toBe(0);
    expect(r.timeInRange.inRange).toBe(2);
  });
});
