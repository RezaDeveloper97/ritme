// Public API of the `fertility` entity (M5). Import only from here (CLAUDE.md §3.3).

// ── Model: types & enums ───────────────────────────────────────
export {
  BBT_PHASES,
  BBT_RANGES,
  CERVICAL_MUCUS,
  CHANCE_LEVELS,
  CHANCE_MAX_BARS,
  EVIDENCE_STRENGTHS,
  FERTILITY_SYMPTOMS,
  INSIGHT_CONFIDENCE,
  INTERCOURSE_TYPES,
  LH_RESULTS,
  type BbtCycle,
  type BbtPhase,
  type BbtPoint,
  type BbtRange,
  type BbtStats,
  type CervicalMucus,
  type ChanceLevel,
  type EvidenceStrength,
  type FertilityBbt,
  type FertilityChance,
  type FertilityDay,
  type FertilityDayInput,
  type FertilityInsights,
  type FertilitySymptom,
  type FertilityTip,
  type FertilityToday,
  type InsightConfidence,
  type InsightEvidence,
  type IntercourseType,
  type LabeledValue,
  type LhResult,
  type OvulationHistoryRow,
} from './model/types';
export {
  BBT_DEFAULT,
  BBT_MAX,
  BBT_MIN,
  BBT_STEP,
  clampBbt,
  formatBbt,
  isBbtInRange,
  parseBbt,
  roundBbt,
  stepBbt,
} from './model/bbt';

// ── API: keys, parsers, reads ──────────────────────────────────
export { fertilityKeys } from './api/keys';
export {
  fertilityBbtSchema,
  fertilityDaySchema,
  fertilityInsightsSchema,
  fertilityTodaySchema,
} from './api/schema';
export {
  fetchFertilityBbt,
  fetchFertilityDay,
  fetchFertilityInsights,
  fetchFertilityToday,
  useFertilityBbt,
  useFertilityDay,
  useFertilityInsights,
  useFertilityToday,
} from './api/queries';
