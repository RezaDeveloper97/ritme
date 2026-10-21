import { describe, expect, it } from 'vitest';

import type { PostpartumRecovery, PostpartumVisit } from '@/entities/postpartum';

import { VISIT_LOOK, visitAction, visitRows } from './visits';

const visit = (over: Partial<PostpartumVisit>): PostpartumVisit => ({
  kind: 'contact',
  dueOn: '2026-10-22',
  done: false,
  doneOn: null,
  status: 'upcoming',
  ...over,
});
const day = (postpartumVisit: boolean | null): PostpartumRecovery => ({
  date: '2026-10-21',
  lochiaAmount: null,
  lochiaColor: null,
  painLevel: null,
  painLocations: [],
  breasts: null,
  feedsCount: null,
  sleepHours: null,
  temperatureC: null,
  padSoakedWithinHour: null,
  severePain: null,
  breathless: null,
  postpartumVisit,
  alerts: [],
});

describe('postpartum visits card', () => {
  const three = [
    visit({ kind: 'contact', dueOn: '2026-10-22', done: true, doneOn: '2026-10-15', status: 'done' }),
    visit({ kind: 'reminder', dueOn: '2026-11-12', status: 'due' }),
    visit({ kind: 'comprehensive', dueOn: '2026-12-24', status: 'due' }),
  ];

  it('keeps the three dates in order', () => {
    expect(visitRows(three).map((v) => [v.kind, v.dueOn])).toEqual([
      ['contact', '2026-10-22'],
      ['reminder', '2026-11-12'],
      ['comprehensive', '2026-12-24'],
    ]);
    expect(visitRows([])).toEqual([]);
  });

  it('gives every status a tone (done green, overdue red)', () => {
    expect(VISIT_LOOK.done.tone).toBe('success');
    expect(VISIT_LOOK.overdue.tone).toBe('danger');
    expect(VISIT_LOOK.due.tone).toBe('warm');
  });

  it('offers to log today while a visit is open, undo once logged, nothing when all are done', () => {
    expect(visitAction(three, day(null))).toBe('log');
    expect(visitAction(three, null)).toBe('log');
    expect(visitAction(three, day(true))).toBe('undo');
    const allDone = three.map((v) => ({ ...v, done: true, status: 'done' as const }));
    expect(visitAction(allDone, day(false))).toBeNull();
  });
});
