// Public API of the `child` entity (B-N5-05 on the B-N5-02 API). Import only from here (CLAUDE.md §3.3).

export {
  BIRTH_RANGES,
  CHILD_DELIVERY_TYPES,
  CHILD_SEXES,
  MAX_CHILD_AGE_YEARS,
  MAX_CHILD_NAME,
  MAX_CHILDREN,
  type BirthField,
  type Child,
  type ChildAge,
  type ChildBirth,
  type ChildDeliveryType,
  type ChildHome,
  type ChildInput,
  type ChildMeasurement,
  type ChildRole,
  type ChildrenList,
  type ChildSex,
  type CompanionChild,
  type GrowthStatus,
  type GrowthVerdict,
  type IndicatorValue,
  type LearnTipPreview,
  type MilestoneSummary,
  type ThisWeek,
  type VaccineSummary,
  type VaccineVisit,
  type VaccineVisitStatus,
} from './model/types';
export { childTone, dueIn, isVisitUrgent, parseDecimalInput, roundPercentile, type DueIn } from './model/format';
export {
  CHILD_PHOTO_MAX_BYTES,
  CHILD_PHOTO_TYPES,
  checkChildPhoto,
  deleteChildPhoto,
  photoErrorCode,
  saveChildPhoto,
  useChildPhoto,
} from './model/photo';
export { childKeys } from './api/keys';
export { childHomeSchema, childrenListSchema, childSchema, parseCompanionChildren } from './api/schema';
export {
  childFieldError,
  fetchChild,
  fetchChildren,
  toChildBody,
  useChild,
  useChildren,
  useCreateChild,
  useDeleteChild,
  useUpdateChild,
} from './api/queries';
export { ChildAvatar } from './ui/ChildAvatar';
export { ChildStatusChips, useDueText } from './ui/ChildStatusChips';
export { ChildrenStrip } from './ui/ChildrenStrip';
