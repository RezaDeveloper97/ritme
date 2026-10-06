import { describe, expect, it } from 'vitest';

import { parseContractionOverview, parseKickOverview, parseStopResult } from '../api/schema';
import {
  clockTime,
  contractionRows,
  contractionView,
  formatDuration,
  isStaleContraction,
  kickView,
  liveElapsedMs,
  serverOffsetMs,
} from './timing';

const START = '2026-09-23T21:15:00+03:30';
const startMs = Date.parse(START);

describe('formatDuration', () => {
  it('prints m:ss under an hour and h:mm:ss above, like the artboards', () => {
    expect(formatDuration(0)).toBe('0:00');
    expect(formatDuration(48_000)).toBe('0:48');
    expect(formatDuration((24 * 60 + 10) * 1000)).toBe('24:10');
    expect(formatDuration((3600 + 5 * 60 + 3) * 1000)).toBe('1:05:03');
  });

  it('clamps negatives and drops sub-second remainders', () => {
    expect(formatDuration(-5000)).toBe('0:00');
    expect(formatDuration(59_999)).toBe('0:59');
  });
});

describe('clockTime', () => {
  it('reads the wall clock the server wrote (Tehran), whatever the device zone', () => {
    expect(clockTime(START)).toBe('21:15');
    expect(clockTime('2026-09-23T07:05:59+03:30')).toBe('07:05');
  });

  it('is empty for missing or malformed values', () => {
    expect(clockTime(null)).toBe('');
    expect(clockTime('nope')).toBe('');
  });
});

describe('server clock offset', () => {
  it('measures server minus client from a running timer', () => {
    // The server said 120 s had elapsed; the client clock read 30 s *behind* the server.
    const serverNow = startMs + 120_000;
    const receivedAt = serverNow - 30_000;
    expect(serverOffsetMs(START, 120, receivedAt)).toBe(30_000);
  });

  it('derives live elapsed time from server timestamps, not from the device clock', () => {
    const receivedAt = startMs + 120_000 + 5 * 60_000; // device clock 5 min fast
    const offset = serverOffsetMs(START, 120, receivedAt);
    expect(liveElapsedMs(START, receivedAt, offset)).toBe(120_000);
    expect(liveElapsedMs(START, receivedAt + 10_000, offset)).toBe(130_000);
  });

  it('never goes negative and survives a malformed anchor', () => {
    expect(liveElapsedMs(START, startMs - 1000, 0)).toBe(0);
    expect(serverOffsetMs('bad', 10, 1)).toBe(0);
    expect(liveElapsedMs('bad', 1, 0)).toBe(0);
  });
});

// ── kick counter ───────────────────────────────────────────────

const kickPayload = (over: Record<string, unknown> = {}) => ({
  id: 1,
  started_at: START,
  ended_at: null,
  is_active: true,
  kicks: 8,
  target: 10,
  window_minutes: 120,
  elapsed_seconds: 1260,
  time_to_target_seconds: null,
  reached_target: false,
  low_count: false,
  last_kick_at: '2026-09-23T21:34:00+03:30',
  pregnancy_week: 26,
  ...over,
});

const overview = (active: unknown, history: unknown[] = []) => ({
  target: 10,
  window_minutes: 120,
  active,
  today: { date: '2026-09-23', kicks: 8, sessions: 1 },
  history,
});

describe('kick session restore', () => {
  it('restores a running count after a reload and keeps the timer going from the server start', () => {
    const receivedAt = 1_000_000;
    const o = parseKickOverview(overview(kickPayload()), receivedAt);
    expect(o.active?.kicks).toBe(8);
    const v = kickView(o.active!, receivedAt + 30_000);
    expect(v.count).toBe(8);
    expect(v.elapsedMs).toBe((1260 + 30) * 1000);
    expect(v.progress).toBeCloseTo(0.8);
    expect(v.reached).toBe(false);
    expect(v.lowCount).toBe(false);
  });

  it('shows queued taps and undos on top of the server count, never below zero', () => {
    const s = parseKickOverview(overview(kickPayload({ kicks: 1 })), 0).active!;
    expect(kickView(s, 0, 2).count).toBe(3);
    expect(kickView(s, 0, -3).count).toBe(0);
  });

  it('turns on the low-count advice once the 2-hour window passes below target', () => {
    const s = parseKickOverview(overview(kickPayload({ kicks: 6, elapsed_seconds: 7190 })), 0).active!;
    expect(kickView(s, 9_000).lowCount).toBe(false);
    expect(kickView(s, 10_000).lowCount).toBe(true);
    expect(kickView(s, 10_000, 4).lowCount).toBe(false); // the 10th tap is on its way
  });

  it('caps the ring at the target and freezes an ended session at its length', () => {
    const ended = parseKickOverview(
      overview(null, [kickPayload({ is_active: false, ended_at: '2026-09-23T21:40:00+03:30', kicks: 12, elapsed_seconds: 1500 })]),
      0,
    ).history[0];
    const v = kickView(ended, 99_999_999);
    expect(v.progress).toBe(1);
    expect(v.reached).toBe(true);
    expect(v.elapsedMs).toBe(1_500_000);
  });

  it('treats a missing overview session as idle and skips malformed history rows', () => {
    const o = parseKickOverview(overview(null, [{ id: 'x' }, kickPayload({ id: 2, is_active: false })]), 0);
    expect(o.active).toBeNull();
    expect(o.history.map((h) => h.id)).toEqual([2]);
  });
});

// ── contraction timer ──────────────────────────────────────────

const C1 = '2026-09-23T22:28:00+03:30';
const C2 = '2026-09-23T22:34:20+03:30';
const C3 = '2026-09-23T22:40:25+03:30';

const sessionPayload = (over: Record<string, unknown> = {}) => ({
  id: 7,
  started_at: C1,
  ended_at: null,
  is_active: true,
  elapsed_seconds: 760,
  count: 2,
  running: null,
  avg_duration_seconds: 52,
  avg_interval_seconds: 380,
  five_one_one: { met: false, count: 2, span_minutes: 7, avg_interval_seconds: 380, avg_duration_seconds: 52 },
  alert_at: null,
  contractions: [
    { id: 2, started_at: C2, ended_at: '2026-09-23T22:35:15+03:30', duration_seconds: 55, interval_seconds: 380 },
    { id: 1, started_at: C1, ended_at: '2026-09-23T22:28:49+03:30', duration_seconds: 49, interval_seconds: null },
  ],
  ...over,
});

describe('contraction timer', () => {
  it('is idle with no active session', () => {
    expect(contractionView(null, 0)).toEqual({ phase: 'idle', runningMs: 0, sinceLastStartMs: null });
  });

  it('restores a resting session and counts the interval up from the last start', () => {
    const receivedAt = 5_000;
    const o = parseContractionOverview({ params: null, active: sessionPayload(), history: [] }, receivedAt);
    // Server now = C1 + 760 s = 22:40:40; the last contraction started 22:34:20 → 380 s ago.
    const v = contractionView(o.active, receivedAt + 20_000);
    expect(v.phase).toBe('resting');
    expect(v.sinceLastStartMs).toBe((380 + 20) * 1000);
    expect(o.params).toEqual({ intervalMaxMinutes: 5, durationMinSeconds: 45, runMinutes: 60 });
  });

  it('restores a running contraction with its live length from the server start', () => {
    const payload = sessionPayload({
      running: { id: 3, started_at: C3, elapsed_seconds: 15 },
      contractions: [
        { id: 3, started_at: C3, ended_at: null, duration_seconds: null, interval_seconds: 365 },
        ...sessionPayload().contractions,
      ],
    });
    const receivedAt = 42;
    const o = parseContractionOverview({ params: { interval_max_minutes: 4, duration_min_seconds: 60, run_minutes: 90 }, active: payload, history: [] }, receivedAt);
    expect(o.params.runMinutes).toBe(90);
    const v = contractionView(o.active, receivedAt + 33_000);
    expect(v.phase).toBe('contracting');
    expect(v.runningMs).toBe(48_000);
    expect(v.sinceLastStartMs).toBe(48_000);

    const rows = contractionRows(o.active!, receivedAt + 33_000);
    expect(rows.map((r) => [r.start, r.durationMs, r.intervalMs, r.running])).toEqual([
      ['22:40', 48_000, 365_000, true],
      ['22:34', 55_000, 380_000, false],
      ['22:28', 49_000, null, false],
    ]);
  });

  it('parses the stop result with the raised 5-1-1 alert', () => {
    const r = parseStopResult(
      {
        session: sessionPayload({ alert_at: C3, five_one_one: { met: true, count: 12, span_minutes: 61, avg_interval_seconds: 290, avg_duration_seconds: 58 } }),
        alerts: [
          {
            id: 99,
            rule_key: 'contractions_511',
            level: 'urgent',
            title: '5-1-1',
            actions: [{ key: 'call', label: 'Call' }, { key: 'ack', label: 'Got it' }],
            contact: { text: 'Call 115', phone: null },
          },
          { id: 'broken' },
        ],
      },
      0,
    );
    expect(r.session.fiveOneOne.met).toBe(true);
    expect(r.alerts).toHaveLength(1);
    expect(r.alerts[0].ruleKey).toBe('contractions_511');
    expect(r.alerts[0].actions.map((a) => a.key)).toEqual(['call', 'ack']);
  });

  it('an ended session is idle on the timer', () => {
    const s = parseContractionOverview({ active: null, history: [sessionPayload({ is_active: false, ended_at: C3 })] }, 0);
    expect(s.history[0].contractions).toHaveLength(2);
    expect(contractionView(s.history[0], 0).phase).toBe('idle');
  });
});

describe('isStaleContraction (B-N5-10)', () => {
  it('flags a contraction left running for hours and a long-idle session', () => {
    expect(isStaleContraction({ phase: 'contracting', runningMs: 48_000, sinceLastStartMs: 48_000 })).toBe(false);
    expect(isStaleContraction({ phase: 'contracting', runningMs: 43 * 3_600_000, sinceLastStartMs: 43 * 3_600_000 })).toBe(true);
    expect(isStaleContraction({ phase: 'resting', runningMs: 0, sinceLastStartMs: 6 * 60_000 })).toBe(false);
    expect(isStaleContraction({ phase: 'resting', runningMs: 0, sinceLastStartMs: 7 * 3_600_000 })).toBe(true);
    expect(isStaleContraction({ phase: 'idle', runningMs: 0, sinceLastStartMs: null })).toBe(false);
  });
});
