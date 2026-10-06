import type { Tone } from '@/shared/ui';

import type { LabMarker, MarkerState } from './types';

/** Geometry of the range bar (nbl_Lab_Result / nbl_Lab_Marker): percentages along the track, oldest-left in LTR. */
export interface RangeGeometry {
  /** Start and end of the reference band, 0–100. */
  bandStart: number;
  bandEnd: number;
  /** Position of the value thumb, 0–100 (clamped so the thumb stays on the track). */
  marker: number;
}

const clamp = (v: number, lo: number, hi: number) => Math.min(hi, Math.max(lo, v));

/**
 * Where the reference band and the value sit on the track. The domain is the
 * band padded by 35 % of its width on each side, widened to include the value,
 * so an in-range value lands inside the band and an out-of-range one beside it.
 * One-sided ranges («< 5») get a band that runs to the track's edge. Null when
 * there is no number or no range at all (the row shows no bar).
 */
export function rangeGeometry(value: number | null, low: number | null, high: number | null): RangeGeometry | null {
  if (value === null || !Number.isFinite(value)) return null;
  if (low === null && high === null) return null;
  let lo = low ?? 0;
  let hi = high ?? (low as number) * 2;
  if (hi <= lo) hi = lo + Math.max(Math.abs(lo) * 0.2, 1);
  const pad = (hi - lo) * 0.35;
  let dMin = low === null ? Math.min(0, value) : lo - pad;
  let dMax = high === null ? hi + pad : hi + pad;
  dMin = Math.min(dMin, value - pad * 0.3);
  dMax = Math.max(dMax, value + pad * 0.3);
  if (low === null) lo = dMin;
  if (high === null) hi = dMax;
  const pos = (v: number) => ((v - dMin) / (dMax - dMin)) * 100;
  return {
    bandStart: clamp(pos(lo), 0, 100),
    bandEnd: clamp(pos(hi), 0, 100),
    marker: clamp(pos(value), 3, 97),
  };
}

/** Tone of a state: amber for anything that needs attention, turquoise for normal, neutral when unknown. */
export function stateTone(state: MarkerState): Tone {
  if (state === 'normal') return 'data';
  if (state === 'unknown') return 'neutral';
  return 'warm';
}

/** Markers that need attention first (low/high before borderline), then the rest in sheet order. */
export function splitMarkers(markers: readonly LabMarker[]): { attention: LabMarker[]; normal: LabMarker[]; unknown: LabMarker[] } {
  const rank: Record<MarkerState, number> = { low: 0, high: 0, borderline_low: 1, borderline_high: 1, normal: 2, unknown: 3 };
  const attention = markers.filter((m) => m.attention).sort((a, b) => rank[a.state] - rank[b.state]);
  return {
    attention,
    normal: markers.filter((m) => !m.attention && m.state === 'normal'),
    unknown: markers.filter((m) => !m.attention && m.state !== 'normal'),
  };
}

/**
 * Parses a typed number: Persian / Arabic digits and the Persian decimal
 * separator («۱۱٫۲») are accepted, like the server does. Null when empty or not a number.
 */
export function parseLabNumber(input: string): number | null {
  const ascii = input
    .trim()
    .replace(/[۰-۹]/g, (d) => String(d.charCodeAt(0) - 0x06f0))
    .replace(/[٠-٩]/g, (d) => String(d.charCodeAt(0) - 0x0660))
    .replace(/٫/g, '.');
  if (ascii === '' || !/^-?\d+(\.\d+)?$/.test(ascii)) return null;
  const n = Number(ascii);
  return Number.isFinite(n) ? n : null;
}
