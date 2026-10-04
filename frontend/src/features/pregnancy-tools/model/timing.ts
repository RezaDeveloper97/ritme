import type { ContractionSession, KickSession } from './types';

/*
 * Timer maths of the kick counter and the contraction timer. The server is the
 * source of truth: every live number is derived from its timestamps plus the
 * clock offset measured when the payload arrived, so a reload, a background tab
 * or a device clock that is minutes off never changes what the screen shows.
 */

/** Epoch ms of a server ISO timestamp (`2026-09-23T21:15:00+03:30`); NaN when malformed. */
export function isoMs(iso: string): number {
  return Date.parse(iso);
}

/**
 * Server clock minus client clock, from a running timer the server just
 * reported: at `receivedAt` (client) the server's now was `anchor + elapsed`.
 * Truncation on the server keeps the error under a second.
 */
export function serverOffsetMs(anchorIso: string, elapsedSeconds: number, receivedAt: number): number {
  const anchor = isoMs(anchorIso);
  if (!Number.isFinite(anchor)) return 0;
  return anchor + elapsedSeconds * 1000 - receivedAt;
}

/** Live elapsed ms since `startIso` at client time `nowMs`, corrected by `offsetMs`. */
export function liveElapsedMs(startIso: string, nowMs: number, offsetMs: number): number {
  const start = isoMs(startIso);
  if (!Number.isFinite(start)) return 0;
  return Math.max(0, nowMs + offsetMs - start);
}

/** `m:ss`, or `h:mm:ss` from an hour up — the artboards' «۲۴:۱۰» / «۰:۴۸» (Go `labor.Clock`). */
export function formatDuration(ms: number): string {
  const total = Math.max(0, Math.floor(ms / 1000));
  const h = Math.floor(total / 3600);
  const m = Math.floor((total % 3600) / 60);
  const s = total % 60;
  const pad = (n: number) => String(n).padStart(2, '0');
  return h > 0 ? `${h}:${pad(m)}:${pad(s)}` : `${m}:${pad(s)}`;
}

/**
 * Wall-clock `HH:mm` of a server timestamp, as the server wrote it (Tehran
 * time) — the «شروع ۲۱:۱۵» of the artboards. `''` when malformed.
 */
export function clockTime(iso: string | null): string {
  const m = iso ? /T(\d{2}):(\d{2})/.exec(iso) : null;
  return m ? `${m[1]}:${m[2]}` : '';
}

/** `YYYY-MM-DD` part of a server timestamp (its own calendar day). */
export function isoDay(iso: string): string {
  return iso.slice(0, 10);
}

// ── kick counter ───────────────────────────────────────────────

export interface KickView {
  /** Movements shown: the server's count plus taps still on their way (never below 0). */
  count: number;
  target: number;
  elapsedMs: number;
  /** Ring fill 0–1. */
  progress: number;
  reached: boolean;
  /** Fewer than the target once the window (2 h) has passed: the «تماس بگیر» advice. */
  lowCount: boolean;
  windowMs: number;
}

/**
 * What the counter shows for `session` at client time `nowMs`; `pending` are
 * taps (+) or undos (−) queued but not yet answered. An ended session freezes
 * at its server length.
 */
export function kickView(session: KickSession, nowMs: number, pending = 0): KickView {
  const count = Math.max(0, session.kicks + pending);
  const target = session.target > 0 ? session.target : 10;
  const windowMs = session.windowMinutes * 60_000;
  const elapsedMs = session.isActive
    ? liveElapsedMs(session.startedAt, nowMs, serverOffsetMs(session.startedAt, session.elapsedSeconds, session.receivedAt))
    : session.elapsedSeconds * 1000;
  const reached = count >= target;
  return {
    count,
    target,
    elapsedMs,
    progress: Math.min(1, count / target),
    reached,
    lowCount: !reached && windowMs > 0 && elapsedMs >= windowMs,
    windowMs,
  };
}

// ── contraction timer ──────────────────────────────────────────

export type ContractionPhase = 'idle' | 'resting' | 'contracting';

export interface ContractionView {
  phase: ContractionPhase;
  /** Length so far of the contraction being timed (0 unless contracting). */
  runningMs: number;
  /** Since the last contraction started — the interval building up (null before the first). */
  sinceLastStartMs: number | null;
}

/** Server-clock offset of a session payload (any running anchor works; else none). */
export function sessionOffsetMs(session: ContractionSession): number {
  if (session.running) return serverOffsetMs(session.running.startedAt, session.running.elapsedSeconds, session.receivedAt);
  if (session.isActive) return serverOffsetMs(session.startedAt, session.elapsedSeconds, session.receivedAt);
  return 0;
}

/** Phase and live numbers of the active session at client time `nowMs`. */
export function contractionView(session: ContractionSession | null, nowMs: number): ContractionView {
  if (!session || !session.isActive) return { phase: 'idle', runningMs: 0, sinceLastStartMs: null };
  const offset = sessionOffsetMs(session);
  const latest = session.contractions[0]?.startedAt ?? session.running?.startedAt ?? null;
  const sinceLastStartMs = latest ? liveElapsedMs(latest, nowMs, offset) : null;
  if (session.running) {
    return { phase: 'contracting', runningMs: liveElapsedMs(session.running.startedAt, nowMs, offset), sinceLastStartMs };
  }
  return { phase: 'resting', runningMs: 0, sinceLastStartMs };
}

export interface ContractionRow {
  id: number;
  /** `HH:mm` of the start. */
  start: string;
  /** Length (live while it runs). */
  durationMs: number | null;
  intervalMs: number | null;
  running: boolean;
}

/** The «شروع · مدت · فاصله» table, newest first, with the running one ticking. */
export function contractionRows(session: ContractionSession, nowMs: number): ContractionRow[] {
  const offset = sessionOffsetMs(session);
  return session.contractions.map((c) => {
    const running = c.endedAt === null;
    return {
      id: c.id,
      start: clockTime(c.startedAt),
      durationMs: running ? liveElapsedMs(c.startedAt, nowMs, offset) : c.durationSeconds === null ? null : c.durationSeconds * 1000,
      intervalMs: c.intervalSeconds === null ? null : c.intervalSeconds * 1000,
      running,
    };
  });
}

/** ms → whole seconds for a server average (`null` stays `null`). */
export function secondsToMs(seconds: number | null): number | null {
  return seconds === null ? null : seconds * 1000;
}
