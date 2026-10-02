// Public API of the `teen` entity (CB-TEEN-02 on the CB-TEEN-01 API). Import only from here (CLAUDE.md §3.3).

export {
  TEEN_AGE_BANDS,
  TEEN_MENARCHE,
  type TeenAgeBand,
  type TeenCatalogItem,
  type TeenKit,
  type TeenKitItem,
  type TeenMenarche,
  type TeenParentLink,
  type TeenProfile,
  type TeenProfileInput,
  type TeenProfileState,
  type TeenToday,
} from './model/types';
export { teenKeys } from './api/keys';
export { teenKitSchema, teenProfileStateSchema, teenTodaySchema } from './api/schema';
export {
  fetchTeenProfile,
  fetchTeenToday,
  useSaveTeenProfile,
  useTeenProfile,
  useTeenToday,
  useToggleTeenKit,
  withKitTick,
} from './api/queries';
