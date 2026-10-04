import { existsSync, readFileSync } from 'node:fs';
import { resolve } from 'node:path';

import { describe, expect, it } from 'vitest';

import { babyActionError, toManualFeedBody } from './queries';
import { babyLogSummarySchema, diaperDaySchema, feedDaySchema, feedSchema, parseBabyToday, sleepDaySchema } from './schema';

/* Boundary contract of /api/v1/children/{id}/feeds|sleeps|diapers|baby-logs against the Go goldens (B-N5-03). */
const GOLDEN_DIR = resolve(process.cwd(), '../backend-go/contract/golden/babylog');
interface Step {
  request: { method: string; url: string; now?: string };
  status: number;
  body: { data?: unknown };
}
const steps = (name: string): Step[] =>
  (JSON.parse(readFileSync(resolve(GOLDEN_DIR, `${name}.json`), 'utf8')) as { steps: Step[] }).steps;
const data = (name: string, method: string, url: string, nth = 0): unknown =>
  steps(name).filter((s) => s.request.method === method && s.request.url === url)[nth]?.body.data;

describe.skipIf(!existsSync(GOLDEN_DIR))('baby-log parsers', () => {
  it('parses a running breast feed with its side state', () => {
    const f = feedSchema.parse(data('breast_feed_timer.fa', 'POST', '/api/v1/children/1/feeds/1/side'));
    expect(f).toMatchObject({ type: 'breast', isActive: true, activeSide: 'left', lastSide: 'right', leftSeconds: 0, rightSeconds: 420 });
    expect(f.sideStartedAt).toBe('2026-09-23T14:07:00+03:30');
  });

  it('parses the feed day: active, last, next side and totals', () => {
    const running = feedDaySchema.parse(data('breast_feed_timer.fa', 'GET', '/api/v1/children/1/feeds'));
    expect(running.active?.id).toBe(1);
    expect(running.last).toBeNull();
    expect(running.summary).toMatchObject({ count: 1, leftSeconds: 462, rightSeconds: 420 });
    const ended = feedDaySchema.parse(data('breast_feed_timer.fa', 'GET', '/api/v1/children/1/feeds', 1));
    expect(ended.active).toBeNull();
    expect(ended.last).toMatchObject({ lastSide: 'left', durationSeconds: 882, note: 'خوب شیر خورد' });
    expect(ended.nextSide).toBe('right');
    expect(ended.items).toHaveLength(1);
  });

  it('parses bottle and pump feeds with ml', () => {
    const pump = feedSchema.parse(data('bottle_and_pump', 'POST', '/api/v1/children/1/feeds/2/stop'));
    expect(pump).toMatchObject({ type: 'pump', amountMl: 120, durationSeconds: 900, activeSide: null });
  });

  it('parses sleep and diaper days', () => {
    const sleep = sleepDaySchema.parse(data('sleep_and_diapers.en', 'GET', '/api/v1/children/1/sleeps'));
    expect(sleep.active).toBeNull();
    expect(sleep.summary).toEqual({ count: 1, seconds: 27000, longestSeconds: 5400 });
    expect(sleep.items.map((s) => s.id)).toEqual([2, 1]);
    const diapers = diaperDaySchema.parse(data('sleep_and_diapers.en', 'GET', '/api/v1/children/1/diapers'));
    expect(diapers.summary).toEqual({ count: 2, wet: 1, dirty: 1, both: 0 });
    expect(diapers.items[0]).toMatchObject({ kind: 'dirty', changedAt: '2026-09-23T11:45:00+03:30' });
  });

  it('parses the summary and the child home today card', () => {
    const s = babyLogSummarySchema.parse(data('sleep_and_diapers.en', 'GET', '/api/v1/children/1/baby-logs?days=2'));
    expect(s.days.map((d) => d.date)).toEqual(['2026-09-22', '2026-09-23']);
    expect(s.averages).toEqual({ feedsPerDay: 0.5, sleepSecondsPerDay: 17100, diapersPerDay: 1, leftPercent: 57, rightPercent: 43 });
    expect(s.today.lastFeed?.lastSide).toBe('right');
    const home = data('sleep_and_diapers.en', 'GET', '/api/v1/children/1') as { today: unknown };
    const today = parseBabyToday(home.today);
    expect(today).toMatchObject({ feedingNow: false, sleepingNow: false, feeds: { count: 1, totalSeconds: 840 } });
    expect(parseBabyToday(null)).toBeNull();
    expect(parseBabyToday({ nope: true })).toBeNull();
  });

  it('drops a malformed row instead of failing the day', () => {
    const d = diaperDaySchema.parse({ date: '2026-09-23', summary: {}, items: [{ id: 1 }, { id: 2, changed_at: '2026-09-23T09:00:00+03:30', kind: 'wet', note: null }] });
    expect(d.items.map((x) => x.id)).toEqual([2]);
    expect(d.summary).toEqual({ count: 0, wet: 0, dirty: 0, both: 0 });
  });
});

describe('request bodies', () => {
  it('sends side minutes for a manual breast feed, minutes + ml otherwise', () => {
    expect(toManualFeedBody({ type: 'breast', startedAt: '2026-09-23 07:00', leftMinutes: 8, rightMinutes: 0 })).toEqual({
      type: 'breast',
      started_at: '2026-09-23 07:00',
      left_minutes: 8,
      right_minutes: 0,
    });
    expect(toManualFeedBody({ type: 'bottle', startedAt: '2026-09-23 08:00', durationMinutes: 10, amountMl: 90 })).toEqual({
      type: 'bottle',
      started_at: '2026-09-23 08:00',
      duration_minutes: 10,
      amount_ml: 90,
    });
  });

  it('has no message for a non-API error', () => {
    expect(babyActionError(new Error('offline'))).toBeUndefined();
  });
});
