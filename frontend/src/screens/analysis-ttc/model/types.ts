import type { AnalysisPhrase, AnalysisSection } from '@/entities/analysis';

/*
 * Shapes of GET /analysis/ttc and /analysis/fertility (B-N3-11, backend-go
 * internal/analysis ttc.go — goldens testdata/golden/ttc_*.json). Temperatures
 * arrive as decimal strings and are parsed to numbers at the boundary.
 */

export type OvulationSource = 'bbt' | 'lh' | 'estimate';
export type LhValue = 'negative' | 'faint' | 'positive';
export type AgeBand = 'under_35' | 'from_35' | 'unknown';
export type LutealStatus = 'short' | 'normal' | 'long';
export type DotPhase = 'period' | 'fertile' | 'ovulation' | 'other';
export type RegularityStatus = 'regular' | 'irregular' | 'not_enough_data';

export interface BbtPoint {
  day: number;
  value: number;
}

export interface LhTest {
  day: number;
  value: LhValue;
}

export interface TtcTrying {
  since: string | null;
  cycles: number;
  months: number;
  confirmedCycles: number;
  age: number | null;
  referral: { ageBand: AgeBand; thresholdMonths: number; due: boolean };
  summary: AnalysisPhrase;
  advice: AnalysisPhrase;
}

export interface TtcBbt {
  cycleStart: string;
  points: BbtPoint[];
  coverline: number | null;
  shiftDay: number | null;
  ovulationDay: number | null;
  confirmed: boolean;
}

export interface TtcLh {
  cycleStart: string;
  tests: LhTest[];
  positiveDay: number | null;
  text: AnalysisPhrase;
}

export interface TimingDay {
  day: number;
  phase: DotPhase;
  intercourse: boolean;
  future: boolean;
}

export interface TtcTiming {
  cycleStart: string;
  windowFrom: number;
  windowTo: number;
  windowDays: number;
  windowSource: OvulationSource | null;
  inWindow: number;
  days: TimingDay[];
}

export interface TtcMucus {
  cyclesWithEggWhite: number;
  cyclesAnalysed: number;
  text: AnalysisPhrase;
}

export interface TtcLuteal {
  days: number | null;
  status: LutealStatus | null;
  cycles: number;
}

export interface RegularityBar {
  index: number;
  start: string;
  length: number;
  current: boolean;
}

export interface TtcRegularity {
  cycles: RegularityBar[];
  median: number | null;
  variation: number | null;
  status: RegularityStatus | null;
}

export interface TtcCycleSummary {
  index: number;
  start: string;
  end: string;
  current: boolean;
  length: number | null;
  ovulationDay: number | null;
  ovulationSource: OvulationSource | null;
  lhDay: number | null;
  lutealDays: number | null;
}

export interface TtcHub {
  trying: TtcTrying;
  bbt: AnalysisSection<TtcBbt>;
  lh: AnalysisSection<TtcLh>;
  timing: AnalysisSection<TtcTiming>;
  mucus: AnalysisSection<TtcMucus>;
  luteal: AnalysisSection<TtcLuteal>;
  regularity: AnalysisSection<TtcRegularity>;
  cycles: TtcCycleSummary[];
}

export interface FertilityCycle {
  cycle: { index: number; total: number; start: string; end: string; current: boolean; length: number | null; days: number; span: number };
  prevStart: string | null;
  nextStart: string | null;
  chart: { points: BbtPoint[]; coverline: number | null; shiftDay: number | null; highDays: number[]; confirmed: boolean };
  periodDays: number;
  fertileWindow: { from: number; to: number } | null;
  ovulation: { day: number | null; source: OvulationSource | null; confirmed: boolean };
  lh: { tests: LhTest[]; positiveDay: number | null; daysBeforeOvulation: number | null };
  intercourseDays: number[];
  luteal: AnalysisSection<{ days: number | null; status: LutealStatus | null }>;
  explanation: AnalysisPhrase;
  tip: AnalysisPhrase;
}
