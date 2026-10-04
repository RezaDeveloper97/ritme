import type { IvfCycle } from './types';

/*
 * IVF ultrasound scans (CB-IVF-01 `/ivf/scans*`, CB-IVF-04 screen). One scan
 * per cycle day: follicle counts per ovary per size bin, endometrium and E2.
 * Health data — never log it (CLAUDE.md §11).
 */

/** Follicle size bins, smallest first (API keys). */
export const IVF_FOLLICLE_BINS = ['lt_10', '10_14', '15_17', '18_plus'] as const;
export type IvfFollicleBin = (typeof IVF_FOLLICLE_BINS)[number];

/** Ovaries in the board's reading order (right first). */
export const IVF_OVARIES = ['right', 'left'] as const;
export type IvfOvary = (typeof IVF_OVARIES)[number];

export const IVF_E2_UNITS = ['pg_ml', 'pmol_l'] as const;
export type IvfE2Unit = (typeof IVF_E2_UNITS)[number];

/** API limits (validateScan): 0–60 follicles per bin, endometrium 0–40 mm, E2 0–999999. */
export const IVF_SCAN_LIMITS = { follicles: 60, endometriumMm: 40, e2: 999_999 } as const;

export type IvfOvaryCounts = Record<IvfFollicleBin, number>;

export interface IvfScan {
  /** `Y-m-d`. */
  date: string;
  /** Day of stimulation on that date («روز ۷»), null before stimulation. */
  stimDay: number | null;
  right: IvfOvaryCounts;
  left: IvfOvaryCounts;
  /** Decimal as text («8.5»), null when not entered. */
  endometriumMm: string | null;
  e2: string | null;
  e2Unit: IvfE2Unit | null;
  notes: string | null;
}

/** One point of the growth chart: mid (10–14 mm) and lead (≥ 15 mm) follicles, both ovaries. */
export interface IvfGrowthPoint {
  date: string;
  stimDay: number | null;
  mid: number;
  lead: number;
}

export interface IvfScansView {
  cycle: IvfCycle | null;
  /** `Y-m-d` stimulation start of the open cycle, for the «روز N تحریک» of a new scan. */
  stimStartedOn: string | null;
  /** Oldest first. */
  scans: IvfScan[];
  growth: IvfGrowthPoint[];
}

export interface IvfScanInput {
  date: string;
  right: IvfOvaryCounts;
  left: IvfOvaryCounts;
  endometriumMm: number | null;
  e2: number | null;
  e2Unit: IvfE2Unit | null;
  notes: string | null;
}
