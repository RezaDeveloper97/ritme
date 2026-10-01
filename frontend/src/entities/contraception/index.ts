// Public API of the `contraception` entity (CB-CONTRA-01/02). Import only from here (CLAUDE.md §3.3).

// ── Model ──────────────────────────────────────────────────────
export {
  CONTRACEPTION_METHODS,
  PACK_TYPES,
  type ContraceptionMethod,
  type ContraceptionMethodCode,
  type ContraceptionOverview,
  type MethodPayload,
  type MethodReminder,
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

// ── API ────────────────────────────────────────────────────────
export { contraceptionKeys } from './api/keys';
export { contraceptionOverviewSchema } from './api/schema';
export {
  fetchContraception,
  useContraception,
  useLogPill,
  useSaveContraceptionMethod,
  useStopContraception,
  useUndoPill,
} from './api/queries';
