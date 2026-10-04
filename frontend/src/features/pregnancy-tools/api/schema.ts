import { z } from 'zod';

import { type PregnancyAlertV2, pregnancyAlertV2Schema } from '@/entities/pregnancy';

import type {
  Contraction,
  ContractionOverview,
  ContractionSession,
  KickOverview,
  KickSession,
  LaborParams,
} from '../model/types';

/*
 * zod boundary for the pregnancy tools (CLAUDE.md §10): snake_case → camelCase
 * and the client receipt time stamped on every session (`receivedAt`), from
 * which the timers measure the server clock.
 */

const int = z.number().int();
const intOrNull = int.nullable().catch(null);
const iso = z.string().min(10);
const isoOrNull = iso.nullable().catch(null);

const kickRaw = z.object({
  id: int,
  started_at: iso,
  ended_at: isoOrNull,
  is_active: z.boolean(),
  kicks: int,
  target: int.catch(10),
  window_minutes: int.catch(120),
  elapsed_seconds: int.catch(0),
  time_to_target_seconds: intOrNull,
  reached_target: z.boolean().catch(false),
  low_count: z.boolean().catch(false),
  last_kick_at: isoOrNull,
  pregnancy_week: intOrNull,
});

function kick(r: z.infer<typeof kickRaw>, receivedAt: number): KickSession {
  return {
    id: r.id,
    startedAt: r.started_at,
    endedAt: r.ended_at,
    isActive: r.is_active,
    kicks: r.kicks,
    target: r.target,
    windowMinutes: r.window_minutes,
    elapsedSeconds: r.elapsed_seconds,
    timeToTargetSeconds: r.time_to_target_seconds,
    reachedTarget: r.reached_target,
    lowCount: r.low_count,
    lastKickAt: r.last_kick_at,
    pregnancyWeek: r.pregnancy_week,
    receivedAt,
  };
}

export function parseKickSession(raw: unknown, receivedAt = Date.now()): KickSession {
  return kick(kickRaw.parse(raw), receivedAt);
}

const kickOverviewRaw = z.object({
  target: int.catch(10),
  window_minutes: int.catch(120),
  active: kickRaw.nullable().catch(null),
  today: z
    .object({ date: z.string(), kicks: int.catch(0), sessions: int.catch(0) })
    .catch({ date: '', kicks: 0, sessions: 0 }),
  history: z.array(z.unknown()).catch([]),
});

export function parseKickOverview(raw: unknown, receivedAt = Date.now()): KickOverview {
  const r = kickOverviewRaw.parse(raw);
  const history: KickSession[] = [];
  for (const item of r.history) {
    const parsed = kickRaw.safeParse(item);
    if (parsed.success) history.push(kick(parsed.data, receivedAt));
  }
  return {
    target: r.target,
    windowMinutes: r.window_minutes,
    active: r.active ? kick(r.active, receivedAt) : null,
    today: r.today,
    history,
  };
}

// ── contractions ───────────────────────────────────────────────

const contractionRaw = z.object({
  id: int,
  started_at: iso,
  ended_at: isoOrNull,
  duration_seconds: intOrNull,
  interval_seconds: intOrNull,
});

const sessionRaw = z.object({
  id: int,
  started_at: iso,
  ended_at: isoOrNull,
  is_active: z.boolean(),
  elapsed_seconds: int.catch(0),
  count: int.catch(0),
  running: z.object({ id: int, started_at: iso, elapsed_seconds: int.catch(0) }).nullable().catch(null),
  avg_duration_seconds: intOrNull,
  avg_interval_seconds: intOrNull,
  five_one_one: z
    .object({
      met: z.boolean().catch(false),
      count: int.catch(0),
      span_minutes: int.catch(0),
      avg_interval_seconds: intOrNull,
      avg_duration_seconds: intOrNull,
    })
    .catch({ met: false, count: 0, span_minutes: 0, avg_interval_seconds: null, avg_duration_seconds: null }),
  alert_at: isoOrNull,
  contractions: z.array(z.unknown()).optional().catch(undefined),
});

function session(r: z.infer<typeof sessionRaw>, receivedAt: number): ContractionSession {
  const contractions: Contraction[] = [];
  for (const item of r.contractions ?? []) {
    const c = contractionRaw.safeParse(item);
    if (!c.success) continue;
    contractions.push({
      id: c.data.id,
      startedAt: c.data.started_at,
      endedAt: c.data.ended_at,
      durationSeconds: c.data.duration_seconds,
      intervalSeconds: c.data.interval_seconds,
    });
  }
  return {
    id: r.id,
    startedAt: r.started_at,
    endedAt: r.ended_at,
    isActive: r.is_active,
    elapsedSeconds: r.elapsed_seconds,
    count: r.count,
    running: r.running
      ? { id: r.running.id, startedAt: r.running.started_at, elapsedSeconds: r.running.elapsed_seconds }
      : null,
    avgDurationSeconds: r.avg_duration_seconds,
    avgIntervalSeconds: r.avg_interval_seconds,
    fiveOneOne: {
      met: r.five_one_one.met,
      count: r.five_one_one.count,
      spanMinutes: r.five_one_one.span_minutes,
      avgIntervalSeconds: r.five_one_one.avg_interval_seconds,
      avgDurationSeconds: r.five_one_one.avg_duration_seconds,
    },
    alertAt: r.alert_at,
    contractions,
    receivedAt,
  };
}

export function parseContractionSession(raw: unknown, receivedAt = Date.now()): ContractionSession {
  return session(sessionRaw.parse(raw), receivedAt);
}

const DEFAULT_PARAMS: LaborParams = { intervalMaxMinutes: 5, durationMinSeconds: 45, runMinutes: 60 };

const overviewRaw = z.object({
  params: z
    .object({
      interval_max_minutes: int.catch(DEFAULT_PARAMS.intervalMaxMinutes),
      duration_min_seconds: int.catch(DEFAULT_PARAMS.durationMinSeconds),
      run_minutes: int.catch(DEFAULT_PARAMS.runMinutes),
    })
    .nullable()
    .catch(null),
  active: sessionRaw.nullable().catch(null),
  history: z.array(z.unknown()).catch([]),
});

export function parseContractionOverview(raw: unknown, receivedAt = Date.now()): ContractionOverview {
  const r = overviewRaw.parse(raw);
  const history: ContractionSession[] = [];
  for (const item of r.history) {
    const parsed = sessionRaw.safeParse(item);
    if (parsed.success) history.push(session(parsed.data, receivedAt));
  }
  return {
    params: r.params
      ? {
          intervalMaxMinutes: r.params.interval_max_minutes,
          durationMinSeconds: r.params.duration_min_seconds,
          runMinutes: r.params.run_minutes,
        }
      : DEFAULT_PARAMS,
    active: r.active ? session(r.active, receivedAt) : null,
    history,
  };
}

/** `POST /contractions/stop` → the session and any alert the 5-1-1 rule raised. */
export function parseStopResult(
  raw: unknown,
  receivedAt = Date.now(),
): { session: ContractionSession; alerts: PregnancyAlertV2[] } {
  const r = z.object({ session: z.unknown(), alerts: z.array(z.unknown()).catch([]) }).parse(raw);
  const alerts: PregnancyAlertV2[] = [];
  for (const item of r.alerts) {
    const a = pregnancyAlertV2Schema.safeParse(item);
    if (a.success) alerts.push(a.data);
  }
  return { session: parseContractionSession(r.session, receivedAt), alerts };
}
