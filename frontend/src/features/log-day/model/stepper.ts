/**
 * The next value on the step grid: an off-grid value (58.3, carried over from an older reading) snaps to the
 * nearest grid line in the pressed direction (58.5 / 58.0) instead of drifting by a full step (58.8 / 57.8).
 */
export function stepOnGrid(current: number, step: number, dir: 1 | -1): number {
  const units = current / step;
  const eps = 1e-6;
  const next = dir > 0 ? Math.floor(units + eps) + 1 : Math.ceil(units - eps) - 1;
  return Math.round(next * step * 1000) / 1000;
}
