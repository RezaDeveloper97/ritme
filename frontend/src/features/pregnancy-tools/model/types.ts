/*
 * Shapes of `/api/v1/pregnancy/kick-sessions` and `/api/v1/pregnancy/contractions` (B-N5-03,
 * backend-go/api/openapi.yaml `KickSession` / `ContractionSession`). Timestamps stay the server's
 * ISO strings (Tehran offset); `receivedAt` is the client clock when the payload arrived, so a
 * running timer is derived from server time and survives a reload or a skewed device clock.
 */

/** One kick-count session («شمارش حرکات جنین»). */
export interface KickSession {
  id: number;
  startedAt: string;
  endedAt: string | null;
  isActive: boolean;
  kicks: number;
  target: number;
  windowMinutes: number;
  /** Server-side elapsed time at the moment it answered (ended sessions: their length). */
  elapsedSeconds: number;
  timeToTargetSeconds: number | null;
  reachedTarget: boolean;
  lowCount: boolean;
  lastKickAt: string | null;
  pregnancyWeek: number | null;
  receivedAt: number;
}

export interface KickOverview {
  target: number;
  windowMinutes: number;
  active: KickSession | null;
  today: { date: string; kicks: number; sessions: number };
  history: KickSession[];
}

/** One contraction of a session; the server lists them newest first. */
export interface Contraction {
  id: number;
  startedAt: string;
  endedAt: string | null;
  durationSeconds: number | null;
  /** Start-to-start gap to the previous contraction; `null` for the first one. */
  intervalSeconds: number | null;
}

export interface FiveOneOne {
  met: boolean;
  count: number;
  spanMinutes: number;
  avgIntervalSeconds: number | null;
  avgDurationSeconds: number | null;
}

export interface ContractionSession {
  id: number;
  startedAt: string;
  endedAt: string | null;
  isActive: boolean;
  elapsedSeconds: number;
  /** Completed contractions. */
  count: number;
  running: { id: number; startedAt: string; elapsedSeconds: number } | null;
  /** «میانگین مدت / میانگین فاصله» over the last `runMinutes` (server maths, `labor.Recent`). */
  avgDurationSeconds: number | null;
  avgIntervalSeconds: number | null;
  fiveOneOne: FiveOneOne;
  alertAt: string | null;
  /** Present on the active session / single-session reads; empty on history rows. */
  contractions: Contraction[];
  receivedAt: number;
}

/** The live (admin-editable) 5-1-1 thresholds. */
export interface LaborParams {
  intervalMaxMinutes: number;
  durationMinSeconds: number;
  runMinutes: number;
}

export interface ContractionOverview {
  params: LaborParams;
  active: ContractionSession | null;
  history: ContractionSession[];
}
