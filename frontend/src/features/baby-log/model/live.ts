import type { BabyFeed, BabySleep, FeedSide } from './types';

/*
 * Live timer maths over the server's session (B-N5-07). The running session
 * lives on the server, so a reload or a second device shows the same timer:
 * the closed side totals come from the API and the running segment is
 * `now − side_started_at`. No React, no locale.
 */

/** Epoch ms of an API timestamp (`2026-09-23T14:00:00+03:30`); NaN when unparseable. */
export function stampMs(iso: string | null): number {
  return iso ? Date.parse(iso) : Number.NaN;
}

function since(iso: string | null, nowMs: number): number {
  const t = stampMs(iso);
  return Number.isFinite(t) ? Math.max(0, Math.floor((nowMs - t) / 1000)) : 0;
}

/** Seconds on `side` of a breast feed, the running segment included. */
export function sideSeconds(feed: BabyFeed, side: FeedSide, nowMs: number): number {
  const stored = side === 'left' ? feed.leftSeconds : feed.rightSeconds;
  if (!feed.isActive || feed.activeSide !== side) return stored;
  return stored + since(feed.sideStartedAt, nowMs);
}

/** Whole duration of a feed now: breast = left + right (pauses excluded); bottle / pump = wall time since start. */
export function feedSeconds(feed: BabyFeed, nowMs: number): number {
  if (!feed.isActive) return feed.durationSeconds;
  if (feed.type === 'breast') return sideSeconds(feed, 'left', nowMs) + sideSeconds(feed, 'right', nowMs);
  return since(feed.startedAt, nowMs);
}

export function sleepSeconds(sleep: BabySleep, nowMs: number): number {
  return sleep.isActive ? since(sleep.startedAt, nowMs) : sleep.durationSeconds;
}

/** `m:ss`, or `h:mm:ss` from an hour up — the Log_Feed side cards print «۷:۴۲». */
export function clockText(seconds: number): string {
  const total = Math.max(0, Math.floor(seconds));
  const h = Math.floor(total / 3600);
  const m = Math.floor((total % 3600) / 60);
  const s = total % 60;
  const pad = (n: number) => String(n).padStart(2, '0');
  return h > 0 ? `${h}:${pad(m)}:${pad(s)}` : `${m}:${pad(s)}`;
}

/** Whole minutes, rounded (a 30-second feed is «۱ دقیقه», never «۰»). */
export function minutesOf(seconds: number): number {
  if (seconds <= 0) return 0;
  return Math.max(1, Math.round(seconds / 60));
}

/** `HH:mm` of an API timestamp — the server always sends Tehran wall-clock with its offset. */
export function wallTime(iso: string): string {
  return iso.slice(11, 16);
}

/** Today and the time now on the Tehran wall clock (`Y-m-d`, `HH:mm`) — the API's manual-entry clock. */
export function tehranNow(now: Date = new Date()): { date: string; time: string } {
  const parts = new Intl.DateTimeFormat('en-GB', {
    timeZone: 'Asia/Tehran',
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
    hourCycle: 'h23',
  }).formatToParts(now);
  const get = (type: string) => parts.find((p) => p.type === type)?.value ?? '00';
  return { date: `${get('year')}-${get('month')}-${get('day')}`, time: `${get('hour')}:${get('minute')}` };
}

/** The `Y-m-d` before `date`. */
export function previousDay(date: string): string {
  const [y, m, d] = date.split('-').map(Number);
  const t = new Date(Date.UTC(y, m - 1, d) - 86_400_000);
  return t.toISOString().slice(0, 10);
}

const HHMM = /^([01]\d|2[0-3]):[0-5]\d$/;

/**
 * A manual sleep from two `HH:mm` times ending today: an end at or before the
 * start means the sleep began yesterday (22:00 → 06:00). Null on a bad time.
 */
export function manualSleepRange(start: string, end: string, today: string): { startedAt: string; endedAt: string } | null {
  if (!HHMM.test(start) || !HHMM.test(end)) return null;
  const startDay = end <= start ? previousDay(today) : today;
  return { startedAt: `${startDay} ${start}`, endedAt: `${today} ${end}` };
}

/** A manual feed's `started_at` from an `HH:mm` today (a time later than now is yesterday's). */
export function manualStart(time: string, now: { date: string; time: string }): string | null {
  if (!HHMM.test(time)) return null;
  return `${time > now.time ? previousDay(now.date) : now.date} ${time}`;
}
