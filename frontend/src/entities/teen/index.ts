// Public API of the `teen` entity (CB-TEEN-02 on the CB-TEEN-01 API). Import only from here (CLAUDE.md §3.3).

export {
  NO_TEEN_GRANTS,
  TEEN_AGE_BANDS,
  TEEN_MENARCHE,
  TEEN_SHARE_KEYS,
  type TeenAgeBand,
  type TeenCatalogItem,
  type TeenGrants,
  type TeenKit,
  type TeenKitItem,
  type TeenMenarche,
  type TeenParentCard,
  type TeenParentInvite,
  type TeenParentLink,
  type TeenParentView,
  type TeenProfile,
  type TeenProfileInput,
  type TeenProfileState,
  type TeenShareKey,
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
// CB-TEEN-03: mother sharing (teen side) + the parent's read-only cards.
export {
  fetchTeenLinked,
  useForgetParentLink,
  useInviteParent,
  useRenewParentInvite,
  useSaveParentNote,
  useTeenLinked,
  useUpdateParentGrants,
} from './api/parent';
export { isEmptyParentView, maskParentView, primaryParentLink, sameTeenGrants, withShare } from './lib/parent-view';
export { ParentViewCard } from './ui/ParentViewCard';
