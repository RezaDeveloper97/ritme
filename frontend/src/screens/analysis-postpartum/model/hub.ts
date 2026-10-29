import type { EpdsHistory, EpdsPoint } from '../api/epds';

/*
 * The postpartum analysis hub (An_Hub_Post): the cards are computed by
 * GET /analysis/postpartum (T-M10-27); what stays here is the EPDS trend
 * reading and the chart geometry, shared by the report and the Laravel
 * fallback (GET /postpartum/epds). Descriptive only — no score is
 * interpreted beyond the cut-offs the API echoes.
 */

// ── EPDS ───────────────────────────────────────────────────────
export interface EpdsTrend {
  kind: 'short' | 'full';
  /** Oldest first. */
  points: EpdsPoint[];
  max: number;
  /** Full: «possible» (10) and «likely» (13) cut-offs; short: the single «elevated» one (6). */
  lower: number;
  upper: number | null;
  latest: EpdsPoint;
  /** The highest earlier score at or above `lower`, when the latest is below it. */
  peak: EpdsPoint | null;
  status: 'urgent' | 'high' | 'down' | 'low';
}

/**
 * The trend card plots one questionnaire: the full EPDS when there is any
 * (its 10 / 13 bands are the ones the artboard draws), else the short one.
 */
export function epdsTrend(history: EpdsHistory): EpdsTrend | null {
  const full = history.checks.filter((c) => c.kind === 'full');
  const kind = full.length ? 'full' : 'short';
  const points = (full.length ? full : history.checks.filter((c) => c.kind === 'short')).slice().reverse();
  if (!points.length) return null;
  const th = history.thresholds;
  const lower = kind === 'full' ? th.fullPossible : th.shortElevated;
  const upper = kind === 'full' ? th.fullLikely : null;
  const latest = points[points.length - 1];
  const earlier = points.slice(0, -1).filter((p) => p.total >= lower);
  const peak = latest.total < lower && earlier.length ? earlier.reduce((a, b) => (b.total > a.total ? b : a)) : null;
  // Any urgent check in the history (self-harm item, or ≥ likely) keeps the card's call-your-doctor line.
  const urgent = latest.urgent || (upper !== null && latest.total >= upper);
  const status = urgent ? 'urgent' : latest.total >= lower ? 'high' : peak ? 'down' : 'low';
  return { kind, points, max: latest.max, lower, upper, latest, peak, status };
}

/** EPDS chart geometry: x per point (oldest left), y on a 0..max scale with the cut-off bands. */
export const EPDS_W = 320;
export function epdsGeometry(trend: EpdsTrend, height: number) {
  const padX = 24;
  const padTop = 8;
  const padBottom = 8;
  // The scale hugs the scores (the artboard puts 10 / 13 mid-chart), not the questionnaire's 0–30.
  const top = Math.min(trend.max, Math.max(...trend.points.map((p) => p.total), trend.upper ?? trend.lower) + 5);
  const bottom = Math.max(0, Math.min(...trend.points.map((p) => p.total)) - 4);
  const y = (v: number) => padTop + (height - padTop - padBottom) * (1 - (v - bottom) / (top - bottom || 1));
  const n = trend.points.length;
  const x = (i: number) => (n <= 1 ? EPDS_W / 2 : padX + ((EPDS_W - padX - 8) * i) / (n - 1));
  const dots = trend.points.map((p, i) => ({
    x: Math.round(x(i) * 10) / 10,
    y: Math.round(y(p.total) * 10) / 10,
    high: p.total >= trend.lower,
  }));
  const line = dots.map((d, i) => `${i ? 'L' : 'M'}${d.x} ${d.y}`).join('');
  return {
    dots,
    line,
    lowerY: y(trend.lower),
    upperY: trend.upper !== null ? y(trend.upper) : null,
    topY: padTop,
    padX,
  };
}
