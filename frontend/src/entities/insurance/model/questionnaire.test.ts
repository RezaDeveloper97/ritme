import { describe, expect, it } from 'vitest';

import { buildTimeline, evidenceLabel } from './questionnaire';

describe('buildTimeline', () => {
  it('makes the first row current when nothing is recorded yet', () => {
    expect(buildTimeline([]).map((r) => r.state)).toEqual(['current', 'todo', 'todo', 'todo']);
  });

  it('marks recorded rows done and the next one current', () => {
    const rows = buildTimeline([
      { status: 'submitted', on: '2026-09-30' },
      { status: 'reviewing', on: '2026-10-01' },
    ]);
    expect(rows.map((r) => r.state)).toEqual(['done', 'done', 'current', 'todo']);
    expect(rows[1]?.on).toBe('2026-10-01');
  });

  it('counts earlier rows done when she skipped logging them, and ends on a result', () => {
    const rows = buildTimeline([{ status: 'conditional', on: '2026-10-05' }]);
    expect(rows.map((r) => r.state)).toEqual(['done', 'done', 'done', 'done']);
    expect(rows[3]?.status).toBe('conditional');
  });
});

describe('evidenceLabel', () => {
  it('reads server codes with a date', () => {
    expect(evidenceLabel('cesarean · 2024-03-01')).toEqual({ code: 'cesarean', text: 'cesarean', date: '2024-03-01' });
    expect(evidenceLabel('ivf_iui')).toEqual({ code: 'ivf_iui', text: 'ivf_iui', date: null });
  });

  it('keeps her own record text and splits only a trailing date', () => {
    expect(evidenceLabel('Appendectomy · 2023-05-02')).toEqual({ code: null, text: 'Appendectomy', date: '2023-05-02' });
    expect(evidenceLabel('Levothyroxine')).toEqual({ code: null, text: 'Levothyroxine', date: null });
  });
});
