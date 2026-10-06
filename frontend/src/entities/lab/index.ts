// Public API of the `lab` entity (B-N6-07 on the B-N6-06 API). Import only from here (CLAUDE.md §3.3).

export {
  LAB_CATEGORIES,
  LAB_CONSENT_CODE,
  LAB_PLUS_FEATURE,
  type CatalogMarker,
  type Lab,
  type LabCategory,
  type LabConsent,
  type LabCounts,
  type LabInterpretation,
  type LabLimits,
  type LabList,
  type LabListItem,
  type LabMarker,
  type LabStage,
  type LabStatus,
  type LabStatusPoll,
  type LabTrends,
  type MarkerDetail,
  type MarkerInput,
  type MarkerReference,
  type MarkerState,
  type MarkerTrend,
  type RedFlag,
  type TrendPoint,
  type TrendSeries,
} from './model/types';
export {
  isBusy,
  labHref,
  nextPollDelay,
  POLL_FAST_COUNT,
  POLL_FAST_MS,
  POLL_MAX_COUNT,
  POLL_SLOW_MS,
  PROCESS_STEPS,
  ringValue,
  stepStates,
  viewOf,
  type LabView,
  type ProcessStep,
  type StepProgress,
} from './model/status';
export { parseLabNumber, rangeGeometry, splitMarkers, stateTone, type RangeGeometry } from './model/range';
export { formatLabValue, formatReference, trimNumber } from './model/format';
export { labKeys } from './api/keys';
export {
  catalogSchema,
  consentSchema,
  labListSchema,
  labSchema,
  markerDetailSchema,
  statusSchema,
  trendsSchema,
} from './api/schema';
export {
  fetchLab,
  toMarkerBody,
  useAddLabMarker,
  useDeleteLab,
  useDeleteLabMarker,
  useLab,
  useLabCatalog,
  useLabConsent,
  useLabFeedback,
  useLabMarker,
  useLabs,
  useLabStatus,
  useLabTrends,
  useRefreshLab,
  useSetLabConsent,
  useUpdateLabMarker,
  useVerifyLab,
} from './api/queries';
export { MarkerStatePill } from './ui/MarkerStatePill';
export { RangeBar } from './ui/RangeBar';
