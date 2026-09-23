import { describe, expect, it } from 'vitest';

import { scheduleState } from './schedule';

const now = new Date('2026-09-23T10:00:00+03:30');

describe('scheduleState', () => {
  it('open window is live', () => expect(scheduleState(null, null, now)).toBe('live'));
  it('future start is scheduled', () => expect(scheduleState('2026-09-24T00:00:00+03:30', null, now)).toBe('scheduled'));
  it('past end has ended', () => expect(scheduleState(null, '2026-09-22T23:59:00+03:30', now)).toBe('ended'));
  it('inside the window is live', () =>
    expect(scheduleState('2026-09-20T00:00:00+03:30', '2026-09-30T00:00:00+03:30', now)).toBe('live'));
});
