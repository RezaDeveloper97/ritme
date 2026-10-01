// Public API of the `contraception` entity (CB-CONTRA-01/02/03). Import only from here (CLAUDE.md §3.3).

// ── Model ──────────────────────────────────────────────────────
export {
  CONTRACEPTION_METHODS,
  PACK_TYPES,
  type ContraceptionMethod,
  type ContraceptionMethodCode,
  type ContraceptionOverview,
  type MethodPayload,
  type MethodReminder,
  type MethodReminderKind,
  type PackDay,
  type PackType,
  type PillDay,
  type PillPack,
  type PillReminder,
} from './model/types';
export {
  hasMethodReminders,
  isPillMethod,
  IUD_DEFAULT_YEARS,
  IUD_YEARS_RANGE,
  methodToPayload,
  PACKS_LEFT_RANGE,
  packCellState,
  packWeeks,
  type PackCellState,
} from './model/pack';
export { METHOD_LOOK } from './model/look';
export {
  daysUntil,
  INJECTION_DONE_WINDOW,
  INJECTION_WEEKS,
  methodReminder,
  missedGuide,
  type MissedGuide,
  type MissedPillRule,
  type MissedRuleSeverity,
} from './model/missed';

// ── API ────────────────────────────────────────────────────────
export { contraceptionKeys } from './api/keys';
export { contraceptionOverviewSchema, missedRulesSchema } from './api/schema';
export {
  fetchContraception,
  fetchMissedPillRules,
  useContraception,
  useLogPill,
  useMissedPillRules,
  useSaveContraceptionMethod,
  useStopContraception,
  useUndoPill,
} from './api/queries';
