import { describe, expect, it } from 'vitest';

import { clockText, feedSeconds, manualSleepRange, manualStart, minutesOf, previousDay, sideSeconds, sleepSeconds, tehranNow, wallTime } from './live';
import { feedingHref, sectionOf } from './links';
import type { BabyFeed } from './types';

const feed = (over: Partial<BabyFeed>): BabyFeed => ({
  id: 1,
  type: 'breast',
  startedAt: '2026-09-23T14:00:00+03:30',
  endedAt: null,
  isActive: true,
  activeSide: 'left',
  sideStartedAt: '2026-09-23T14:07:00+03:30',
  lastSide: 'right',
  leftSeconds: 0,
  rightSeconds: 420,
  durationSeconds: 420,
  amountMl: null,
  note: null,
  ...over,
});
const at = (iso: string) => Date.parse(iso);

describe('live timer', () => {
  it('adds the running segment to its side only (the artboard 7:42 / 7:00)', () => {
    const f = feed({});
    const now = at('2026-09-23T14:14:42+03:30');
    expect(sideSeconds(f, 'left', now)).toBe(462);
    expect(sideSeconds(f, 'right', now)).toBe(420);
    expect(feedSeconds(f, now)).toBe(882);
  });

  it('does not count a paused breast feed, counts wall time for bottle / pump', () => {
    const now = at('2026-09-23T15:00:00+03:30');
    expect(feedSeconds(feed({ activeSide: null, sideStartedAt: null }), now)).toBe(420);
    expect(feedSeconds(feed({ type: 'pump', activeSide: null, sideStartedAt: null }), now)).toBe(3600);
    expect(feedSeconds(feed({ isActive: false, durationSeconds: 882 }), now)).toBe(882);
  });

  it('never goes negative on a clock behind the server', () => {
    expect(sideSeconds(feed({}), 'left', at('2026-09-23T14:06:00+03:30'))).toBe(0);
  });

  it('times a running sleep', () => {
    const s = { id: 2, startedAt: '2026-09-23T10:00:00+03:30', endedAt: null, isActive: true, durationSeconds: 0, note: null };
    expect(sleepSeconds(s, at('2026-09-23T11:30:00+03:30'))).toBe(5400);
  });
});

describe('formatting', () => {
  it('prints m:ss, then h:mm:ss', () => {
    expect(clockText(462)).toBe('7:42');
    expect(clockText(0)).toBe('0:00');
    expect(clockText(3725)).toBe('1:02:05');
  });

  it('rounds minutes, a short feed is at least one', () => {
    expect(minutesOf(0)).toBe(0);
    expect(minutesOf(20)).toBe(1);
    expect(minutesOf(882)).toBe(15);
  });

  it('reads the Tehran wall clock of a timestamp', () => {
    expect(wallTime('2026-09-23T14:20:00+03:30')).toBe('14:20');
    expect(tehranNow(new Date('2026-09-23T10:40:00Z'))).toEqual({ date: '2026-09-23', time: '14:10' });
    expect(tehranNow(new Date('2026-09-23T21:00:00Z'))).toEqual({ date: '2026-09-24', time: '00:30' });
  });
});

describe('manual entries', () => {
  it('puts a sleep that crosses midnight on yesterday', () => {
    expect(manualSleepRange('22:00', '06:00', '2026-09-23')).toEqual({ startedAt: '2026-09-22 22:00', endedAt: '2026-09-23 06:00' });
    expect(manualSleepRange('10:00', '11:30', '2026-09-23')).toEqual({ startedAt: '2026-09-23 10:00', endedAt: '2026-09-23 11:30' });
    expect(manualSleepRange('', '11:30', '2026-09-23')).toBeNull();
  });

  it('reads a feed time later than now as yesterday', () => {
    const now = { date: '2026-09-23', time: '08:00' };
    expect(manualStart('07:15', now)).toBe('2026-09-23 07:15');
    expect(manualStart('23:40', now)).toBe('2026-09-22 23:40');
    expect(manualStart('7:15', now)).toBeNull();
  });

  it('steps back over a month boundary', () => {
    expect(previousDay('2026-10-01')).toBe('2026-09-30');
  });
});

describe('links', () => {
  it('builds feeding links per section', () => {
    expect(feedingHref(4)).toBe('/children/4/feeding');
    expect(feedingHref(4, 'sleep')).toBe('/children/4/feeding?section=sleep');
    expect(feedingHref(null, 'diapers')).toBe('/children/feeding?section=diapers');
    expect(sectionOf('diapers')).toBe('diapers');
    expect(sectionOf('x')).toBe('feeding');
  });
});
