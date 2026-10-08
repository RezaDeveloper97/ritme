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
