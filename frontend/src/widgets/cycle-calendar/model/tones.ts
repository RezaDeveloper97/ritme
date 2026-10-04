import type { CycleDayMarker } from '@/entities/cycle';

/**
 * How one calendar day is painted (Night & Bloom `Cycle_Calendar` / `TTC_Calendar`):
 * a logged period is a solid red band, an engine-predicted period a soft red one,
 * the fertile window (ovulation included) a solid amber band and PMS a lavender one.
 */
export type DayTone = 'period' | 'predicted' | 'fertile' | 'pms';

/** Legend order of the artboards; `ovulation` is drawn as a dot, only in the TTC variant. */
export type LegendKey = DayTone | 'ovulation';

export const CYCLE_LEGEND: readonly LegendKey[] = ['period', 'predicted', 'fertile', 'pms'];
export const TTC_LEGEND: readonly LegendKey[] = ['period', 'predicted', 'fertile', 'ovulation', 'pms'];
/** Teen / menopause (CB-TEEN-04b): no fertile window or ovulation anywhere on the calendar. */
export const NO_FERTILITY_LEGEND: readonly LegendKey[] = ['period', 'predicted', 'pms'];

/** A marker as a mode without fertility content sees it: the fertile window and ovulation are plain days. */
export function withoutFertility(marker: CycleDayMarker | null): CycleDayMarker | null {
  return marker === 'fertile' || marker === 'ovulation' ? null : marker;
}

/** Paint of a day, from its engine marker and whether the user actually logged the bleed. */
export function dayTone(marker: CycleDayMarker | null, logged: boolean): DayTone | null {
  switch (marker) {
    case 'period':
      return logged ? 'period' : 'predicted';
    case 'fertile':
    case 'ovulation':
      return 'fertile';
    case 'pms':
      return 'pms';
    default:
      // A logged day the engine hasn't caught up with yet still reads as period.
      return logged ? 'period' : null;
  }
}

/** Where a cell sits inside its band: rounded on the band's start and/or end. */
export interface BandEdge {
  start: boolean;
  end: boolean;
}

/**
 * Band edges for one week row (cells in reading order, `null` = padding / no tone).
 * Consecutive cells of the same tone merge into one pill; the row edge closes it,
 * so a band never bridges two weeks.
 */
export function bandEdges(row: readonly (DayTone | null)[]): (BandEdge | null)[] {
  return row.map((tone, i) => {
    if (!tone) return null;
    return { start: row[i - 1] !== tone, end: row[i + 1] !== tone };
  });
}
