// Public API of the `condition` entity (CB-COND-02 on the CB-COND-01 / -06b API). Import only from here (CLAUDE.md §3.3).

export {
  CONDITION_CATALOG_GROUPS,
  CONDITION_NUDGE_KEYS,
  CONDITION_PROGRAMS,
  type ConditionCatalogGroup,
  type ConditionCatalogItem,
  type ConditionEnrolment,
  type ConditionNudge,
  type ConditionNudgeKey,
  type ConditionProgram,
  type ConditionsOverview,
} from './model/types';
export {
  CONDITION_LOOK,
  CONDITION_PATH,
  PROGRAMS_HUB_PATH,
  isConditionServiceTile,
  safeInAppPath,
} from './model/programs';
export { conditionKeys } from './api/keys';
export { conditionCatalogSchema, conditionNudgesSchema, conditionsOverviewSchema } from './api/schema';
export {
  fetchConditions,
  useConditionCatalog,
  useConditionNudges,
  useConditions,
  useEnrolCondition,
  useLeaveCondition,
} from './api/queries';

// CB-COND-05 — heavy-bleeding (PBAC) chart.
export {
  PBAC_CLOT_POINTS,
  PBAC_CLOTS,
  PBAC_FLOODING_POINTS,
  PBAC_LEVEL_POINTS,
  PBAC_LEVELS,
  PBAC_MAX_PADS,
  pbacScore,
  type PbacAlert,
  type PbacClots,
  type PbacDay,
  type PbacEntry,
  type PbacLevel,
  type PbacPeriod,
} from './model/pbac';
export { pbacDaySchema, pbacKeys, usePbacDay, useSavePbacDay } from './api/pbac';
